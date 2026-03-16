package courtengine

import (
	"testing"
)

func TestProposalManager(t *testing.T) {
	pm := NewProposalManager()

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
}
