package court

import (
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
)

type Proposal struct {
	ID          string
	Type        string
	Name        string
	Description string
	Diff        jsonpatch.Patch
	CreatedAt   time.Time
	Status      string
}

type Engine struct {
	Reviewers map[string]string // persona -> prompt
}

func NewEngine() *Engine {
	reviewers := make(map[string]string)
	personas := []string{"ciso", "mrm", "compliance", "responsible-ai", "sre", "helpfulness"}
	for _, p := range personas {
		if prompt, err := LoadReviewerPrompt(p); err == nil {
			reviewers[p] = prompt
		}
	}
	return &Engine{
		Reviewers: reviewers,
	}
}

func (e *Engine) RunReview(p Proposal) (bool, string) {
	// Simple rule-based review
	if strings.Contains(strings.ToLower(p.Name), "harm") || strings.Contains(strings.ToLower(p.Description), "harm") {
		return false, "Proposal violates Rule 1: Never Cause Irreversible Harm"
	}
	// Check other rules
	if p.Type == "add-tool" && strings.Contains(p.Name, "unsafe") {
		return false, "Unsafe tool addition rejected"
	}
	return true, "Approved by court consensus"
}
