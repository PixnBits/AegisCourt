package courtengine

import (
	"testing"
)

func TestProposalManager(t *testing.T) {
	pm := NewProposalManager("/tmp/test_proposals")

	// Submit a proposal
	proposal := Proposal{
		Type:        "add-tool",
		Name:        "web_search",
		Description: "Add web search tool",
		Diff:        []byte(`{"add": "tool"}`),
		CourtMode:   "auto",
	}
	id, err := pm.Submit(proposal)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Get the proposal
	retrieved, err := pm.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.ID != id {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, id)
	}
	if retrieved.Status != "pending" {
		t.Errorf("Status mismatch: got %s, want pending", retrieved.Status)
	}
	if retrieved.SubmittedAt.IsZero() {
		t.Error("SubmittedAt not set")
	}

	// List pending
	list, err := pm.List("pending")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List length: got %d, want 1", len(list))
	}

	// Test invalid type
	invalidProposal := Proposal{Type: "invalid"}
	_, err = pm.Submit(invalidProposal)
	if err == nil {
		t.Error("Submit should fail for invalid type")
	}

	// List all
	all, err := pm.List("")
	if err != nil {
		t.Fatalf("List all failed: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("List all length: got %d, want 2", len(all))
	}
}

func TestValidateProposal(t *testing.T) {
	tests := []struct {
		name        string
		proposal    Proposal
		wantAllowed bool
		wantReason  string
	}{
		{
			name: "valid proposal",
			proposal: Proposal{
				Name:        "web_search_tool",
				Description: "Add a tool to search the web safely",
			},
			wantAllowed: true,
			wantReason:  "",
		},
		{
			name: "short name",
			proposal: Proposal{
				Name:        "x",
				Description: "Add tool",
			},
			wantAllowed: false,
			wantReason:  "Rejected: Insufficient detail (Rule 6 violation)",
		},
		{
			name: "gibberish name",
			proposal: Proposal{
				Name:        "qwerty@#$%",
				Description: "Add a tool to do something useful",
			},
			wantAllowed: false,
			wantReason:  "Rejected: Unreviewable input (Rule 6 violation)",
		},
		{
			name: "too many special chars",
			proposal: Proposal{
				Name:        "!@#$%^&*()qwerty",
				Description: "Add a tool to do something useful",
			},
			wantAllowed: false,
			wantReason:  "Rejected: Unreviewable input (Rule 6 violation)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, reason := ValidateProposal(tt.proposal)
			if allowed != tt.wantAllowed {
				t.Errorf("ValidateProposal() allowed = %v, want %v", allowed, tt.wantAllowed)
			}
			if reason != tt.wantReason {
				t.Errorf("ValidateProposal() reason = %v, want %v", reason, tt.wantReason)
			}
		})
	}
}
