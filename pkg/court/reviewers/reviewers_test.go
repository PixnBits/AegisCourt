package reviewers

import (
	"strings"
	"testing"
)

func TestLoadPrompt(t *testing.T) {
	prompt, err := LoadPrompt("ciso")
	if err != nil {
		t.Fatalf("LoadPrompt failed: %v", err)
	}
	if prompt == "" {
		t.Error("Prompt is empty")
	}
	if !strings.Contains(prompt, "CISO") {
		t.Error("Prompt does not contain CISO")
	}
}

func TestReviewerOutputValidate(t *testing.T) {
	// Valid output
	valid := ReviewerOutput{
		Score:                85,
		Recommendation:       "Approve",
		KeyConcerns:          []string{"Minor concern"},
		RequiredMitigations:  []string{},
		Pros:                 []string{"Good security"},
		Cons:                 []string{"Some risk"},
		Rationale:            "This proposal is secure enough.",
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("Valid output failed validation: %v", err)
	}

	// Invalid: score out of range
	invalid := valid
	invalid.Score = 150
	if err := invalid.Validate(); err == nil {
		t.Error("Invalid output passed validation")
	}

	// Invalid: missing required field
	invalid2 := valid
	invalid2.Recommendation = "Invalid"
	if err := invalid2.Validate(); err == nil {
		t.Error("Invalid recommendation passed validation")
	}
}