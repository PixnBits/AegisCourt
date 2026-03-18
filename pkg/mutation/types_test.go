package mutation

import (
	"encoding/json"
	"testing"
)

func TestStatusConstants(t *testing.T) {
	statuses := []Status{StatusPrepared, StatusApplied, StatusRolled, StatusFailed}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty status constant")
		}
	}
}

func TestToolDefJSON(t *testing.T) {
	td := ToolDef{
		Name:        "utc_time",
		Description: "Returns UTC time",
		Handler:     "builtin:utc_time",
		OutputType:  "text",
	}
	data, err := json.Marshal(td)
	if err != nil {
		t.Fatal(err)
	}

	var td2 ToolDef
	if err := json.Unmarshal(data, &td2); err != nil {
		t.Fatal(err)
	}
	if td2.Name != "utc_time" || td2.Handler != "builtin:utc_time" {
		t.Errorf("roundtrip failed: got %+v", td2)
	}
}

func TestPromptPatchJSON(t *testing.T) {
	pp := PromptPatch{Target: "agent-system", Content: "Be helpful."}
	data, err := json.Marshal(pp)
	if err != nil {
		t.Fatal(err)
	}
	var pp2 PromptPatch
	if err := json.Unmarshal(data, &pp2); err != nil {
		t.Fatal(err)
	}
	if pp2.Target != "agent-system" || pp2.Content != "Be helpful." {
		t.Errorf("roundtrip failed: %+v", pp2)
	}
}

func TestRulePatchJSON(t *testing.T) {
	rp := RulePatch{RuleNumber: 3, NewText: "No bias", Rationale: "Fairness"}
	data, err := json.Marshal(rp)
	if err != nil {
		t.Fatal(err)
	}
	var rp2 RulePatch
	if err := json.Unmarshal(data, &rp2); err != nil {
		t.Fatal(err)
	}
	if rp2.RuleNumber != 3 {
		t.Errorf("expected rule 3, got %d", rp2.RuleNumber)
	}
}

func TestMutationJSON(t *testing.T) {
	m := Mutation{
		ID:         "mut-001",
		ProposalID: "1234",
		Type:       "add-tool",
		Title:      "Add utc_time",
		Patch:      json.RawMessage(`{"name":"utc_time"}`),
		Status:     StatusPrepared,
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var m2 Mutation
	if err := json.Unmarshal(data, &m2); err != nil {
		t.Fatal(err)
	}
	if m2.ID != "mut-001" || m2.Status != StatusPrepared {
		t.Errorf("roundtrip failed: %+v", m2)
	}
}
