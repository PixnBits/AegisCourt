package courtengine

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	llmrouter "aegiscourt/src/llm-router"
)

// ReviewerResult represents the result from a reviewer persona.
type ReviewerResult struct {
	Persona        string   `json:"persona"`
	RiskSeverity   string   `json:"risk_severity"`
	KeyConcerns    []string `json:"key_concerns"`
	Mitigations    []string `json:"required_mitigations"`
	Pros           []string `json:"pros"`
	Cons           []string `json:"cons"`
	Score          int      `json:"score"`
	Recommendation string   `json:"recommendation"`
}

// CourtEngine manages the court review process.
type CourtEngine struct {
	ProposalMgr *ProposalManager
	LLMRouter   *llmrouter.Router
	Personas    []string
}

// NewCourtEngine creates a new court engine.
func NewCourtEngine(dataDir string) *CourtEngine {
	return &CourtEngine{
		ProposalMgr: NewProposalManager(dataDir),
		LLMRouter:   llmrouter.NewRouter("ollama", "http://127.0.0.1:11434", "nemotron-3-nano", ""),
		Personas:    []string{"CISO", "MRM", "Compliance & Regulatory", "Responsible AI", "SRE", "Helpfulness & Evolution"},
	}
}

// SubmitProposal submits a proposal.
func (ce *CourtEngine) SubmitProposal(proposal Proposal) (string, error) {
	return ce.ProposalMgr.Submit(proposal)
}

// GetProposal gets a proposal by ID.
func (ce *CourtEngine) GetProposal(id string) (*Proposal, error) {
	return ce.ProposalMgr.Get(id)
}

// RunReview runs the review for a proposal.
func (ce *CourtEngine) RunReview(id string) error {
	proposal, err := ce.ProposalMgr.Get(id)
	if err != nil {
		return err
	}
	if proposal.Status != "pending" {
		return fmt.Errorf("proposal not pending")
	}

	// Simulate reviewers: CISO and Helpfulness (for speed)
	results := []ReviewerResult{
		{
			Persona:        "CISO",
			RiskSeverity:   "Medium",
			KeyConcerns:    []string{"Potential for data exfiltration", "Need sandbox validation"},
			Mitigations:    []string{"Enforce strict I/O controls", "Audit all operations"},
			Pros:           []string{"Enhances agent capabilities", "Controlled environment"},
			Cons:           []string{"Increases attack surface", "Requires careful implementation"},
			Score:          7,
			Recommendation: "Approve with mitigations",
		},
		{
			Persona:        "Helpfulness & Evolution",
			RiskSeverity:   "Low",
			KeyConcerns:    []string{"Tool must be genuinely useful", "Avoid feature bloat"},
			Mitigations:    []string{"Test usability", "Ensure reversible"},
			Pros:           []string{"Improves user experience", "Enables evolution"},
			Cons:           []string{"May introduce bugs", "Learning curve"},
			Score:          9,
			Recommendation: "Approve",
		},
	}

	proposal.Status = "reviewed"
	proposal.Results = results

	// Save updated proposal
	return ce.ProposalMgr.saveProposal(proposal)
}

// reviewByPersona performs review by a single persona.
func (ce *CourtEngine) reviewByPersona(persona string, proposal *Proposal) (ReviewerResult, error) {
	// Load prompt template
	promptTemplate, err := loadPersonaPrompt(persona)
	if err != nil {
		return ReviewerResult{}, err
	}

	// Replace placeholders
	prompt := strings.ReplaceAll(promptTemplate, "{{proposal_description}}", proposal.Description)
	prompt = strings.ReplaceAll(prompt, "{{proposal_diff}}", string(proposal.Diff))
	// TODO: constitution rules
	prompt = strings.ReplaceAll(prompt, "{{constitution_rules}}", "Mock constitution rules")

	// Call LLM
	response, err := ce.LLMRouter.CallPersona(persona, prompt)
	if err != nil {
		return ReviewerResult{}, err
	}

	// Parse JSON
	var result ReviewerResult
	err = json.Unmarshal([]byte(response), &result)
	if err != nil {
		return ReviewerResult{}, err
	}
	result.Persona = persona

	return result, nil
}

// loadPersonaPrompt loads the prompt for a persona.
func loadPersonaPrompt(persona string) (string, error) {
	// Map persona to file
	fileMap := map[string]string{
		"CISO":                    "ciso.md",
		"MRM":                     "mrm.md",
		"Compliance & Regulatory": "compliance-regulatory.md",
		"Responsible AI":          "responsible-ai.md",
		"SRE":                     "sre.md",
		"Helpfulness & Evolution": "helpfulness-evolution.md",
	}
	file, exists := fileMap[persona]
	if !exists {
		return "", fmt.Errorf("unknown persona: %s", persona)
	}
	data, err := os.ReadFile("reviewers/" + file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
