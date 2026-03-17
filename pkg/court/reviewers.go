package court

import (
	_ "embed"
	"fmt"
)

//go:embed ../../internal/reviewers/ciso.md
var cisoPrompt string

//go:embed ../../internal/reviewers/mrm.md
var mrmPrompt string

//go:embed ../../internal/reviewers/compliance-regulatory.md
var compliancePrompt string

//go:embed ../../internal/reviewers/responsible-ai.md
var responsibleAIPrompt string

//go:embed ../../internal/reviewers/sre.md
var srePrompt string

//go:embed ../../internal/reviewers/helpfulness-evolution.md
var helpfulnessPrompt string

func LoadReviewerPrompt(persona string) (string, error) {
	switch persona {
	case "ciso":
		return cisoPrompt, nil
	case "mrm":
		return mrmPrompt, nil
	case "compliance":
		return compliancePrompt, nil
	case "responsible-ai":
		return responsibleAIPrompt, nil
	case "sre":
		return srePrompt, nil
	case "helpfulness":
		return helpfulnessPrompt, nil
	default:
		return "", fmt.Errorf("unknown persona: %s", persona)
	}
}

func BuildReviewerPrompt(persona string, proposal Proposal, constitution string, profile interface{}) string {
	prompt, err := LoadReviewerPrompt(persona)
	if err != nil {
		return "Error loading prompt"
	}
	// Inject proposal, constitution, etc.
	return fmt.Sprintf("%s\n\nProposal: %+v\nConstitution: %s", prompt, proposal, constitution)
}
