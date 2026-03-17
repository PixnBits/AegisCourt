package courtengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// QAEntry represents a Q&A session.
type QAEntry struct {
	Question  string            `json:"question"`
	Answers   map[string]string `json:"answers"`
	Timestamp time.Time         `json:"timestamp"`
}

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
	Results     []ReviewerResult
	QAs         []QAEntry
}

// ProposalManager manages proposals.
type ProposalManager struct {
	proposals map[string]*Proposal
	nextID    int
	filePath  string
}

// NewProposalManager creates a new proposal manager.
func NewProposalManager(dataDir string) *ProposalManager {
	filePath := filepath.Join(dataDir, "proposals.json")
	pm := &ProposalManager{
		proposals: make(map[string]*Proposal),
		nextID:    1,
		filePath:  filePath,
	}
	pm.load()
	return pm
}

// load loads proposals from file.
func (pm *ProposalManager) load() {
	data, err := os.ReadFile(pm.filePath)
	if err != nil {
		return // no file yet
	}
	var proposals []Proposal
	if err := json.Unmarshal(data, &proposals); err != nil {
		return
	}
	for _, p := range proposals {
		pm.proposals[p.ID] = &p
		if id := fmt.Sprintf("%d", pm.nextID); p.ID >= id {
			pm.nextID = parseInt(p.ID) + 1
		}
	}
}

// save saves proposals to file.
func (pm *ProposalManager) save() error {
	var proposals []Proposal
	for _, p := range pm.proposals {
		proposals = append(proposals, *p)
	}
	data, err := json.MarshalIndent(proposals, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pm.filePath, data, 0600)
}

// saveProposal saves a single updated proposal.
func (pm *ProposalManager) saveProposal(proposal *Proposal) error {
	return pm.save()
}

// ValidateProposal checks if a proposal meets basic quality standards.
func ValidateProposal(proposal Proposal) (bool, string) {
	// Check name and description length
	if len(proposal.Name) < 10 || len(proposal.Description) < 10 {
		return false, "Rejected: Insufficient detail (Rule 6 violation)"
	}

	// Check for gibberish in name: must consist of alphanum _ -
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !nameRegex.MatchString(proposal.Name) {
		return false, "Rejected: Unreviewable input (Rule 6 violation)"
	}

	return true, ""
}

// parseInt simple helper
func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// Submit submits a new proposal.
func (pm *ProposalManager) Submit(proposal Proposal) (string, error) {
	// Validate proposal quality
	allowed, reason := ValidateProposal(proposal)
	if !allowed {
		// Set status to rejected and save
		proposal.Status = "rejected"
		id := fmt.Sprintf("%d", pm.nextID)
		pm.nextID++
		proposal.ID = id
		proposal.SubmittedAt = time.Now()
		pm.proposals[id] = &proposal
		pm.save()
		// TODO: log to audit
		return "", errors.New(reason)
	}

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
	pm.save()

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
