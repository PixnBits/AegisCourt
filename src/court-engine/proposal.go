package courtengine

import (
	"errors"
	"fmt"
	"time"
)

// Proposal represents a governance proposal.
type Proposal struct {
	ID          string
	Type        string
	Name        string
	Description string
	Diff        []byte
	SubmittedAt time.Time
	Status      string
	CourtMode   string
}

// ProposalManager manages proposals.
type ProposalManager struct {
	proposals map[string]*Proposal
	nextID    int
}

// NewProposalManager creates a new proposal manager.
func NewProposalManager() *ProposalManager {
	return &ProposalManager{
		proposals: make(map[string]*Proposal),
		nextID:    1,
	}
}

// Submit submits a new proposal.
func (pm *ProposalManager) Submit(proposal Proposal) (string, error) {
	// Validate type
	allowedTypes := []string{"add-tool", "change-prompt", "amend-rule"}
	valid := false
	for _, t := range allowedTypes {
		if proposal.Type == t {
			valid = true
			break
		}
	}
	if !valid {
		return "", errors.New("invalid proposal type")
	}

	// Generate ID
	id := fmt.Sprintf("%d", pm.nextID)
	pm.nextID++

	proposal.ID = id
	proposal.SubmittedAt = time.Now()
	proposal.Status = "pending"

	pm.proposals[id] = &proposal

	// TODO: log to audit

	return id, nil
}

// Get retrieves a proposal by ID.
func (pm *ProposalManager) Get(id string) (*Proposal, error) {
	proposal, exists := pm.proposals[id]
	if !exists {
		return nil, errors.New("proposal not found")
	}
	return proposal, nil
}

// List lists proposals by status. If status is empty, list all.
func (pm *ProposalManager) List(status string) ([]*Proposal, error) {
	var result []*Proposal
	for _, p := range pm.proposals {
		if status == "" || p.Status == status {
			result = append(result, p)
		}
	}
	return result, nil
}
