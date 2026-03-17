package courtengine

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	llmrouter "aegiscourt/src/llm-router"
	"aegiscourt/src/sandbox"
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
		LLMRouter:   llmrouter.NewRouter("ollama", "http://127.0.0.1:11434", "llama3.2:latest", ""),
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

	var results []ReviewerResult
	for _, persona := range ce.Personas {
		result, err := ce.reviewByPersona(persona, proposal)
		if err != nil {
			// Log error but continue with others
			fmt.Printf("Error reviewing by %s: %v\n", persona, err)
			continue
		}
		results = append(results, result)
	}

	proposal.Status = "reviewed"
	proposal.Results = results

	// Save updated proposal
	return ce.ProposalMgr.saveProposal(proposal)
}

// AskQuestion asks a clarifying question about a proposal.
func (ce *CourtEngine) AskQuestion(proposalID string, question string, targetPersonas []string) (map[string]string, error) {
	proposal, err := ce.ProposalMgr.Get(proposalID)
	if err != nil {
		return nil, err
	}

	if len(targetPersonas) == 0 {
		targetPersonas = ce.Personas
	}

	answers := make(map[string]string)
	for _, persona := range targetPersonas {
		prompt := fmt.Sprintf("You are the %s reviewer. Answer this clarifying question about the proposal '%s' (description: %s): %s", persona, proposal.Name, proposal.Description, question)
		response, err := ce.LLMRouter.CallPersona(persona, prompt, map[string]interface{}{"temperature": 0.2, "max_tokens": 300})
		if err != nil {
			answers[persona] = fmt.Sprintf("Error: %v", err)
		} else {
			answers[persona] = response
		}
	}

	// Add to proposal QAs
	qa := QAEntry{
		Question:  question,
		Answers:   answers,
		Timestamp: time.Now(),
	}
	proposal.QAs = append(proposal.QAs, qa)
	ce.ProposalMgr.saveProposal(proposal)

	return answers, nil
}

// reviewByPersona performs review by a single persona.
func (ce *CourtEngine) reviewByPersona(persona string, proposal *Proposal) (ReviewerResult, error) {
	promptTemplate, err := loadPersonaPrompt(persona)
	if err != nil {
		return ReviewerResult{}, err
	}

	// Load constitution
	constitutionData, err := os.ReadFile("docs/constitution.md")
	if err != nil {
		return ReviewerResult{}, fmt.Errorf("failed to load constitution: %w", err)
	}
	constitution := string(constitutionData)

	// Extract facts if not low resource
	facts := "{}"
	if !sandbox.IsLowResourceMode() {
		factsPrompt := fmt.Sprintf("Extract key facts from the proposal. Proposal name: %s\nDescription: %s\nDiff: %s\nOutput JSON: {\"summary\": \"...\", \"risks\": \"...\", \"benefits\": \"...\"}", proposal.Name, proposal.Description, string(proposal.Diff))
		factsResponse, err := ce.LLMRouter.CallPersona(persona, factsPrompt, map[string]interface{}{"temperature": 0.1, "max_tokens": 200})
		if err != nil {
			// Fallback to empty
			facts = "{}"
		} else {
			facts = factsResponse
		}
	}

	// Replace placeholders
	prompt := strings.ReplaceAll(promptTemplate, "{{proposal_name}}", proposal.Name)
	prompt = strings.ReplaceAll(prompt, "{{proposal_description}}", proposal.Description)
	prompt = strings.ReplaceAll(prompt, "{{proposal_diff}}", string(proposal.Diff))
	prompt = strings.ReplaceAll(prompt, "{{constitution_text}}", constitution)
	prompt = strings.ReplaceAll(prompt, "{{facts}}", facts)

	// Call LLM
	options := map[string]interface{}{
		"temperature": 0.3,
		"max_tokens":  500,
	}
	response, err := ce.LLMRouter.CallPersona(persona, prompt, options)
	if err != nil {
		return ReviewerResult{}, err
	}

	// Parse JSON
	var result ReviewerResult
	err = json.Unmarshal([]byte(response), &result)
	if err != nil {
		return ReviewerResult{}, fmt.Errorf("failed to parse LLM response as JSON: %s", response)
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
