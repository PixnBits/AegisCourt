package court

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed reviewers/schema.json
var reviewerSchemaJSON []byte

//go:embed reviewers/*.md
var reviewerPromptsFS embed.FS

type ReviewerOutput struct {
	Score               int      `json:"score"`
	Recommendation      string   `json:"recommendation"`
	KeyConcerns         []string `json:"key_concerns"`
	RequiredMitigations []string `json:"required_mitigations"`
	Pros                []string `json:"pros"`
	Cons                []string `json:"cons"`
	Rationale           string   `json:"rationale"`
}

type ReviewerPersona struct {
	Name     string
	Filename string
	Prompt   string
	Weight   float64
}

var Reviewers = []ReviewerPersona{
	{Name: "CISO", Filename: "ciso.md", Weight: 1.2},
	{Name: "MRM", Filename: "mrm.md", Weight: 1.0},
	{Name: "Compliance", Filename: "compliance-regulatory.md", Weight: 1.0},
	{Name: "Ethics", Filename: "responsible-ai.md", Weight: 1.0},
	{Name: "SRE", Filename: "sre.md", Weight: 1.0},
	{Name: "Helpfulness", Filename: "helpfulness-evolution.md", Weight: 0.8},
}

func LoadReviewerPrompts() ([]ReviewerPersona, error) {
	loaded := make([]ReviewerPersona, len(Reviewers))
	copy(loaded, Reviewers)
	for i := range loaded {
		data, err := reviewerPromptsFS.ReadFile("reviewers/" + loaded[i].Filename)
		if err != nil {
			return nil, fmt.Errorf("failed to load reviewer prompt %s: %w", loaded[i].Filename, err)
		}
		loaded[i].Prompt = string(data)
	}
	return loaded, nil
}

func ValidateReviewerOutput(data []byte) (*ReviewerOutput, error) {
	var output ReviewerOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if output.Score < 0 || output.Score > 100 {
		return nil, fmt.Errorf("score must be 0–100, got %d", output.Score)
	}

	validRecs := map[string]bool{
		"Approve": true, "Approve with conditions": true,
		"Defer": true, "Reject": true,
	}
	if !validRecs[output.Recommendation] {
		return nil, fmt.Errorf("invalid recommendation: %q", output.Recommendation)
	}

	if len(output.KeyConcerns) > 5 {
		return nil, fmt.Errorf("key_concerns max 5 items, got %d", len(output.KeyConcerns))
	}
	if len(output.RequiredMitigations) > 5 {
		return nil, fmt.Errorf("required_mitigations max 5 items, got %d", len(output.RequiredMitigations))
	}

	if len(output.Rationale) < 10 {
		return nil, fmt.Errorf("rationale too short (min 10 chars)")
	}
	if len(output.Rationale) > 300 {
		return nil, fmt.Errorf("rationale too long (max 300 chars)")
	}

	// Check for additional properties by re-marshaling
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		allowed := map[string]bool{
			"score": true, "recommendation": true, "key_concerns": true,
			"required_mitigations": true, "pros": true, "cons": true, "rationale": true,
		}
		for key := range raw {
			if !allowed[key] {
				return nil, fmt.Errorf("unexpected field: %q (additionalProperties not allowed)", key)
			}
		}
	}

	return &output, nil
}

func ExtractJSON(response string) string {
	start := strings.Index(response, "{")
	if start == -1 {
		return response
	}
	depth := 0
	end := -1
	for i := start; i < len(response); i++ {
		switch response[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
		if end != -1 {
			break
		}
	}
	if end == -1 {
		return response[start:]
	}
	return response[start:end]
}
