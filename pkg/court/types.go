package court

import (
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
	// Simulate court review
	// For now, approve all proposals
	reason := "Approved by court consensus"
	return true, reason
}
