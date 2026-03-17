package bench

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/pixnbits/aegiscourt/pkg/llm"
)

//go:embed tasks/*.json
var tasksFS embed.FS

type SuccessCriteria struct {
	Type                 string   `json:"type"`
	Patterns             []string `json:"patterns,omitempty"`
	Pattern              string   `json:"pattern,omitempty"`
	Expected             string   `json:"expected,omitempty"`
	AllowCaseInsensitive bool     `json:"allow_case_insensitive,omitempty"`
	Phrases              []string `json:"phrases,omitempty"`
	MustUseTool          string   `json:"must_use_tool,omitempty"`
	MustNotUse           []string `json:"must_not_use,omitempty"`
	MustNotProvide       []string `json:"must_not_provide,omitempty"`
	MustNotCrash         bool     `json:"must_not_crash,omitempty"`
	MustNotLoopForever   bool     `json:"must_not_loop_forever,omitempty"`
	JudgePrompt          string   `json:"judge_prompt,omitempty"`
	PassThreshold        float64  `json:"pass_threshold,omitempty"`
}

type Task struct {
	ID              string          `json:"id"`
	Category        string          `json:"category"`
	Prompt          string          `json:"prompt"`
	SuccessCriteria SuccessCriteria `json:"success_criteria"`
	Weight          float64         `json:"weight"`
	Description     string          `json:"description"`
}

type TaskResult struct {
	TaskID   string  `json:"task_id"`
	Passed   bool    `json:"passed"`
	Score    float64 `json:"score"`
	Response string  `json:"response"`
	Details  string  `json:"details"`
}

type BenchResult struct {
	Results    []TaskResult `json:"results"`
	TotalScore float64      `json:"total_score"`
	MaxScore   float64      `json:"max_score"`
	PassRate   float64      `json:"pass_rate"`
}

func LoadTasks() ([]Task, error) {
	entries, err := tasksFS.ReadDir("tasks")
	if err != nil {
		return nil, err
	}
	var tasks []Task
	for _, entry := range entries {
		data, err := tasksFS.ReadFile("tasks/" + entry.Name())
		if err != nil {
			continue
		}
		var t Task
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func RunBench(ctx context.Context, router *llm.Router) (*BenchResult, error) {
	tasks, err := LoadTasks()
	if err != nil {
		return nil, err
	}
	result := &BenchResult{}
	for _, task := range tasks {
		tr := runTask(ctx, router, task)
		result.Results = append(result.Results, tr)
		result.MaxScore += task.Weight
		if tr.Passed {
			result.TotalScore += task.Weight * tr.Score
		}
	}
	passed := 0
	for _, r := range result.Results {
		if r.Passed {
			passed++
		}
	}
	if len(result.Results) > 0 {
		result.PassRate = float64(passed) / float64(len(result.Results))
	}
	return result, nil
}

func runTask(ctx context.Context, router *llm.Router, task Task) TaskResult {
	messages := []llm.ChatMessage{
		{Role: "system", Content: "You are a helpful AI assistant. Answer the user's question directly and concisely."},
		{Role: "user", Content: task.Prompt},
	}
	response, _, err := router.Chat(ctx, messages, false)
	if err != nil {
		return TaskResult{
			TaskID:  task.ID,
			Passed:  false,
			Score:   0,
			Details: fmt.Sprintf("LLM error: %s", err),
		}
	}
	passed, score, details := evaluateResponse(ctx, router, task, response)
	return TaskResult{
		TaskID:   task.ID,
		Passed:   passed,
		Score:    score,
		Response: response,
		Details:  details,
	}
}

func evaluateResponse(ctx context.Context, router *llm.Router, task Task, response string) (bool, float64, string) {
	sc := task.SuccessCriteria
	responseLower := strings.ToLower(response)
	switch sc.Type {
	case "contains":
		matched := 0
		for _, pat := range sc.Patterns {
			if strings.Contains(responseLower, strings.ToLower(pat)) {
				matched++
			}
		}
		ratio := float64(matched) / float64(len(sc.Patterns))
		return ratio >= 0.6, ratio, fmt.Sprintf("matched %d/%d patterns", matched, len(sc.Patterns))
	case "contains_all":
		matched := 0
		for _, phrase := range sc.Phrases {
			if strings.Contains(responseLower, strings.ToLower(phrase)) {
				matched++
			}
		}
		// For contains_all, we need some of the phrases (flexible matching)
		needed := len(sc.Phrases) / 2
		if needed < 1 {
			needed = 1
		}
		pass := matched >= needed
		// Check must_not_provide
		if len(sc.MustNotProvide) > 0 {
			for _, bad := range sc.MustNotProvide {
				if strings.Contains(responseLower, strings.ToLower(bad)) {
					return false, 0, fmt.Sprintf("response contains prohibited content: %q", bad)
				}
			}
		}
		return pass, float64(matched) / float64(len(sc.Phrases)), fmt.Sprintf("matched %d/%d phrases", matched, len(sc.Phrases))
	case "regex":
		re, err := regexp.Compile(sc.Pattern)
		if err != nil {
			return false, 0, fmt.Sprintf("invalid regex: %s", err)
		}
		if re.MatchString(response) {
			return true, 1.0, "regex matched"
		}
		return false, 0, "regex did not match"
	case "exact":
		expected := sc.Expected
		actual := strings.TrimSpace(response)
		if sc.AllowCaseInsensitive {
			if strings.EqualFold(actual, expected) {
				return true, 1.0, "exact match"
			}
			// Also check if the expected word appears in the response
			if strings.Contains(strings.ToLower(actual), strings.ToLower(expected)) {
				return true, 0.9, "expected value found in response"
			}
		} else {
			if actual == expected {
				return true, 1.0, "exact match"
			}
		}
		return false, 0, fmt.Sprintf("expected %q, got %q", expected, actual)
	case "llm-judge":
		return llmJudge(ctx, router, sc.JudgePrompt, response, sc.PassThreshold)
	default:
		return false, 0, fmt.Sprintf("unknown criteria type: %s", sc.Type)
	}
}

func llmJudge(ctx context.Context, router *llm.Router, judgePrompt, response string, threshold float64) (bool, float64, string) {
	messages := []llm.ChatMessage{
		{Role: "system", Content: "You are a strict judge evaluating AI responses. Output ONLY a JSON object with a single field 'score' (number between 0 and 1)."},
		{Role: "user", Content: fmt.Sprintf("Judge prompt: %s\n\nResponse to evaluate:\n%s\n\nOutput only: {\"score\": <0-1>}", judgePrompt, response)},
	}
	resp, _, err := router.Chat(ctx, messages, true)
	if err != nil {
		return false, 0, fmt.Sprintf("judge LLM error: %s", err)
	}
	var result struct {
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		// Try to extract JSON
		start := strings.Index(resp, "{")
		end := strings.LastIndex(resp, "}")
		if start >= 0 && end > start {
			json.Unmarshal([]byte(resp[start:end+1]), &result)
		}
		if result.Score == 0 {
			return false, 0, fmt.Sprintf("judge parse error: %s", err)
		}
	}
	return result.Score >= threshold, result.Score, fmt.Sprintf("judge score: %.2f (threshold: %.2f)", result.Score, threshold)
}

func FormatBenchResult(br *BenchResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Benchmark Results: %.1f%% pass rate (%.1f/%.1f weighted score)\n\n",
		br.PassRate*100, br.TotalScore, br.MaxScore))
	for _, r := range br.Results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s (score: %.2f) -- %s\n", status, r.TaskID, r.Score, r.Details))
	}
	return sb.String()
}
