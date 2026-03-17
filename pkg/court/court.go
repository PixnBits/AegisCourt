package court

import (
	"AegisCourt/audit"
	"AegisCourt/pkg/court/reviewers"
	"AegisCourt/pkg/proposal"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CourtResult struct {
	ProposalID   string                        `json:"proposal_id"`
	Timestamp    string                        `json:"timestamp"`
	Reviews      map[string]*reviewers.ReviewerOutput `json:"reviews"`
	AggregatedScore int                         `json:"aggregated_score"`
	OverallRecommendation string                `json:"overall_recommendation"`
}

func getCourtDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".aegiscourt", "court")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func RunCourt(proposalID string) (*CourtResult, error) {
	draft, err := proposal.LoadDraft(proposalID)
	if err != nil {
		return nil, fmt.Errorf("failed to load proposal: %v", err)
	}

	proposalJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proposal: %v", err)
	}

	// List of personas
	personas := []string{"ciso", "compliance-regulatory", "helpfulness-evolution", "mrm", "responsible-ai", "sre"}

	reviews := make(map[string]*reviewers.ReviewerOutput)
	totalScore := 0
	count := 0

	for _, persona := range personas {
		output, err := reviewers.CallReviewer(persona, string(proposalJSON))
		if err != nil {
			return nil, fmt.Errorf("failed to call reviewer %s: %v", persona, err)
		}
		reviews[persona] = output
		totalScore += output.Score
		count++
	}

	avgScore := totalScore / count

	var overallRec string
	if avgScore >= 8 {
		overallRec = "approve"
	} else if avgScore >= 5 {
		overallRec = "conditional"
	} else {
		overallRec = "reject"
	}

	result := &CourtResult{
		ProposalID:            proposalID,
		Timestamp:             time.Now().UTC().Format(time.RFC3339),
		Reviews:               reviews,
		AggregatedScore:       avgScore,
		OverallRecommendation: overallRec,
	}

	// Save result
	dir, err := getCourtDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, proposalID+".json")
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(result); err != nil {
		return nil, err
	}

	return result, nil
}

func LoadCourtResult(proposalID string) (*CourtResult, error) {
	dir, err := getCourtDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, proposalID+".json")
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var result CourtResult
	if err := json.NewDecoder(file).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func ApplyProposal(proposalID string) error {
	result, err := LoadCourtResult(proposalID)
	if err != nil {
		return fmt.Errorf("failed to load court result: %v", err)
	}
	if result.OverallRecommendation != "approve" && result.OverallRecommendation != "conditional" {
		return fmt.Errorf("proposal not approved for application")
	}

	// "Apply" the change - in real, this would execute the proposed_change
	// For demo, just audit
	auditPayload := map[string]interface{}{
		"action":      "apply_proposal",
		"proposal_id": proposalID,
		"result":      result,
	}
	if auditErr := audit.Append(auditPayload); auditErr != nil {
		return fmt.Errorf("failed to audit apply: %v", auditErr)
	}

	fmt.Printf("Proposal %s applied successfully.\n", proposalID)
	return nil
}

func RollbackProposal(proposalID string) error {
	// "Rollback" - in real, revert the change
	// For demo, audit
	auditPayload := map[string]interface{}{
		"action":       "rollback_proposal",
		"proposal_id":  proposalID,
	}
	if auditErr := audit.Append(auditPayload); auditErr != nil {
		return fmt.Errorf("failed to audit rollback: %v", auditErr)
	}

	fmt.Printf("Proposal %s rolled back successfully.\n", proposalID)
	return nil
}