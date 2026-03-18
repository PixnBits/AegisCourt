package court

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/pixnbits/aegiscourt/pkg/audit"
	"github.com/pixnbits/aegiscourt/pkg/llm"
)

type ProposalStatus string

const (
	StatusPending   ProposalStatus = "pending"
	StatusActive    ProposalStatus = "active"
	StatusCompleted ProposalStatus = "completed"
	StatusDeferred  ProposalStatus = "deferred"
	StatusRejected  ProposalStatus = "rejected"
	StatusApproved  ProposalStatus = "approved"
)

type CourtResult struct {
	ProposalID      string           `json:"proposal_id"`
	DraftID         string           `json:"draft_id,omitempty"`
	ProposalTitle   string           `json:"proposal_title"`
	Status          ProposalStatus   `json:"status"`
	CourtMode       string           `json:"court_mode"`
	AggregateScore  float64          `json:"aggregate_score"`
	Recommendation  string           `json:"recommendation"`
	ReviewerResults []ReviewerResult `json:"reviewer_results"`
	Conditions      []string         `json:"conditions"`
	StartedAt       string           `json:"started_at"`
	CompletedAt     string           `json:"completed_at"`
	VoteAction      string           `json:"vote_action,omitempty"`
	VoteNotes       string           `json:"vote_notes,omitempty"`
}

type ReviewerResult struct {
	Persona  string          `json:"persona"`
	Output   *ReviewerOutput `json:"output"`
	Model    string          `json:"model_used"`
	Error    string          `json:"error,omitempty"`
	Duration string          `json:"duration"`
}

type Engine struct {
	Router      *llm.Router
	AuditLog    *audit.Log
	LowResource bool
}

func NewEngine(router *llm.Router, auditLog *audit.Log, lowResource bool) *Engine {
	return &Engine{
		Router:      router,
		AuditLog:    auditLog,
		LowResource: lowResource,
	}
}

func (e *Engine) RunCourt(ctx context.Context, proposalID, proposalTitle, proposalJSON, courtMode string) (*CourtResult, error) {
	reviewers, err := LoadReviewerPrompts()
	if err != nil {
		return nil, fmt.Errorf("failed to load reviewer prompts: %w", err)
	}

	result := &CourtResult{
		ProposalID:    proposalID,
		ProposalTitle: proposalTitle,
		Status:        StatusActive,
		CourtMode:     courtMode,
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	e.AuditLog.Append(fmt.Sprintf("court_started: proposal=%s mode=%s", proposalID, courtMode))

	if e.LowResource {
		result.ReviewerResults = e.runSequential(ctx, reviewers, proposalJSON)
	} else {
		result.ReviewerResults = e.runParallel(ctx, reviewers, proposalJSON)
	}

	e.computeAggregate(result)
	result.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	result.Status = StatusCompleted

	e.AuditLog.Append(fmt.Sprintf("court_completed: proposal=%s score=%.0f recommendation=%s",
		proposalID, result.AggregateScore, result.Recommendation))

	return result, nil
}

func (e *Engine) runParallel(ctx context.Context, reviewers []ReviewerPersona, proposalJSON string) []ReviewerResult {
	results := make([]ReviewerResult, len(reviewers))
	var wg sync.WaitGroup

	for i, reviewer := range reviewers {
		wg.Add(1)
		go func(idx int, rev ReviewerPersona) {
			defer wg.Done()
			results[idx] = e.callReviewer(ctx, rev, proposalJSON)
		}(i, reviewer)
	}

	wg.Wait()
	return results
}

func (e *Engine) runSequential(ctx context.Context, reviewers []ReviewerPersona, proposalJSON string) []ReviewerResult {
	results := make([]ReviewerResult, len(reviewers))
	for i, reviewer := range reviewers {
		results[i] = e.callReviewer(ctx, reviewer, proposalJSON)
	}
	return results
}

func (e *Engine) callReviewer(ctx context.Context, reviewer ReviewerPersona, proposalJSON string) ReviewerResult {
	start := time.Now()

	prompt := reviewer.Prompt + "\n\n---\nProposal to review:\n" + proposalJSON + "\n\nRespond with ONLY valid JSON matching the required schema. No explanations or text outside the JSON."

	messages := []llm.ChatMessage{
		{Role: "system", Content: reviewer.Prompt},
		{Role: "user", Content: "Review this proposal and respond with ONLY valid JSON matching the required schema.\n\nProposal:\n" + proposalJSON},
	}

	response, model, err := e.Router.Chat(ctx, messages, true)
	if err != nil {
		// Fallback to generate API
		response, model, err = e.Router.Generate(ctx, prompt, true)
		if err != nil {
			return ReviewerResult{
				Persona:  reviewer.Name,
				Error:    err.Error(),
				Duration: time.Since(start).String(),
			}
		}
	}

	jsonStr := ExtractJSON(response)
	output, err := ValidateReviewerOutput([]byte(jsonStr))
	if err != nil {
		// Retry once with error feedback
		retryMessages := append(messages, llm.ChatMessage{
			Role:    "user",
			Content: fmt.Sprintf("Your previous response had a validation error: %s\nPlease output ONLY valid JSON matching the schema exactly. No text outside the JSON.", err),
		})
		response, model, err2 := e.Router.Chat(ctx, retryMessages, true)
		if err2 == nil {
			jsonStr = ExtractJSON(response)
			output, err = ValidateReviewerOutput([]byte(jsonStr))
		}
		if err != nil {
			e.AuditLog.Append(fmt.Sprintf("reviewer_schema_violation: persona=%s error=%s", reviewer.Name, err))
			return ReviewerResult{
				Persona:  reviewer.Name,
				Model:    model,
				Error:    fmt.Sprintf("schema validation failed: %s", err),
				Duration: time.Since(start).String(),
			}
		}
	}

	e.AuditLog.Append(fmt.Sprintf("reviewer_completed: persona=%s score=%d recommendation=%s model=%s",
		reviewer.Name, output.Score, output.Recommendation, model))

	return ReviewerResult{
		Persona:  reviewer.Name,
		Output:   output,
		Model:    model,
		Duration: time.Since(start).String(),
	}
}

func (e *Engine) computeAggregate(result *CourtResult) {
	var totalWeight, weightedSum float64
	var conditions []string
	rejectCount := 0
	deferCount := 0

	for i, rr := range result.ReviewerResults {
		if rr.Output == nil {
			continue
		}
		weight := Reviewers[i].Weight
		totalWeight += weight
		weightedSum += float64(rr.Output.Score) * weight

		if rr.Output.Recommendation == "Reject" {
			rejectCount++
		}
		if rr.Output.Recommendation == "Defer" {
			deferCount++
		}
		conditions = append(conditions, rr.Output.RequiredMitigations...)
	}

	if totalWeight > 0 {
		result.AggregateScore = math.Round(weightedSum / totalWeight)
	}

	result.Conditions = conditions

	switch {
	case rejectCount >= 2:
		result.Recommendation = "Reject"
	case result.AggregateScore < 60:
		result.Recommendation = "Reject"
	case deferCount >= 3:
		result.Recommendation = "Defer"
	case result.AggregateScore < 80:
		result.Recommendation = "Approve with conditions"
	default:
		result.Recommendation = "Approve"
	}
}

func (e *Engine) QA(ctx context.Context, proposalJSON, persona, question string) (string, error) {
	reviewers, err := LoadReviewerPrompts()
	if err != nil {
		return "", err
	}

	var found *ReviewerPersona
	for _, r := range reviewers {
		if matchesPersona(r.Name, persona) {
			found = &r
			break
		}
	}

	if found == nil {
		// Route to all reviewers and combine
		var combined string
		for _, r := range reviewers {
			messages := []llm.ChatMessage{
				{Role: "system", Content: r.Prompt},
				{Role: "user", Content: fmt.Sprintf("Regarding this proposal:\n%s\n\nQuestion: %s\n\nAnswer briefly from your perspective.", proposalJSON, question)},
			}
			resp, _, err := e.Router.Chat(ctx, messages, false)
			if err == nil {
				combined += fmt.Sprintf("[%s]: %s\n\n", r.Name, resp)
			}
		}
		return combined, nil
	}

	messages := []llm.ChatMessage{
		{Role: "system", Content: found.Prompt},
		{Role: "user", Content: fmt.Sprintf("Regarding this proposal:\n%s\n\nQuestion: %s\n\nAnswer from your perspective.", proposalJSON, question)},
	}
	resp, _, err := e.Router.Chat(ctx, messages, false)
	return resp, err
}

func matchesPersona(name, query string) bool {
	name = toLower(name)
	query = toLower(query)
	return name == query || startsWith(name, query)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func FormatNASABoard(result *CourtResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Proposal %s: %s\n", result.ProposalID, result.ProposalTitle))
	sb.WriteString(fmt.Sprintf("Court Mode: %s\n", result.CourtMode))
	sb.WriteString(fmt.Sprintf("Aggregate Score: %.0f/100   Recommendation: %s\n\n", result.AggregateScore, result.Recommendation))
	sb.WriteString("NASA-style All-Hands Board:\n")

	for _, rr := range result.ReviewerResults {
		indicator := "🟢"
		if rr.Output == nil {
			indicator = "🔴"
			sb.WriteString(fmt.Sprintf("%-14s %s ERROR   %s\n", rr.Persona, indicator, rr.Error))
			continue
		}
		if rr.Output.Score < 60 {
			indicator = "🔴"
		} else if rr.Output.Score < 80 {
			indicator = "🟡"
		}
		sb.WriteString(fmt.Sprintf("%-14s %s %d/100   %s\n", rr.Persona, indicator, rr.Output.Score, rr.Output.Rationale))
	}

	if len(result.Conditions) > 0 {
		sb.WriteString("\nConditions:\n")
		for _, c := range result.Conditions {
			sb.WriteString(fmt.Sprintf("  - %s\n", c))
		}
	}

	return sb.String()
}

func FormatDetailed(result *CourtResult) string {
	var sb strings.Builder
	sb.WriteString(FormatNASABoard(result))
	sb.WriteString("\n\nReviewer Breakdown\n")

	for _, rr := range result.ReviewerResults {
		if rr.Output == nil {
			sb.WriteString(fmt.Sprintf("\n%s: ERROR — %s\n", rr.Persona, rr.Error))
			continue
		}
		sb.WriteString(fmt.Sprintf("\n%s (Score %d)\n", rr.Persona, rr.Output.Score))
		if len(rr.Output.KeyConcerns) > 0 {
			sb.WriteString("  Key concerns:\n")
			for _, c := range rr.Output.KeyConcerns {
				sb.WriteString(fmt.Sprintf("                      %s\n", c))
			}
		}
		if len(rr.Output.RequiredMitigations) > 0 {
			sb.WriteString("  Required mitigations:\n")
			for _, m := range rr.Output.RequiredMitigations {
				sb.WriteString(fmt.Sprintf("                      %s\n", m))
			}
		}
		if len(rr.Output.Pros) > 0 {
			sb.WriteString("  Pros:\n")
			for _, p := range rr.Output.Pros {
				sb.WriteString(fmt.Sprintf("                      %s\n", p))
			}
		}
		if len(rr.Output.Cons) > 0 {
			sb.WriteString("  Cons:\n")
			for _, c := range rr.Output.Cons {
				sb.WriteString(fmt.Sprintf("                      %s\n", c))
			}
		}
		sb.WriteString(fmt.Sprintf("  Recommendation:     %s\n", rr.Output.Recommendation))
	}

	return sb.String()
}

func FormatReviewer(result *CourtResult, persona string) string {
	for _, rr := range result.ReviewerResults {
		if matchesPersona(rr.Persona, persona) {
			if rr.Output == nil {
				return fmt.Sprintf("%s: ERROR — %s", rr.Persona, rr.Error)
			}
			data, _ := json.MarshalIndent(rr.Output, "", "  ")
			return fmt.Sprintf("%s Reviewer Output:\n%s", rr.Persona, string(data))
		}
	}
	return fmt.Sprintf("no reviewer found matching %q", persona)
}
