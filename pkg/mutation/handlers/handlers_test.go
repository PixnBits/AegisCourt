package handlers

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/pixnbits/aegiscourt/pkg/mutation"
)

func setup(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	os.Setenv("AEGISCOURT_HOME", tmpDir)
	t.Cleanup(func() { os.Unsetenv("AEGISCOURT_HOME") })
}

func makeMutation(typ string, patch any) *mutation.Mutation {
	data, _ := json.Marshal(patch)
	return &mutation.Mutation{
		ID:         "mut-test",
		ProposalID: "p-test",
		Type:       typ,
		Title:      "Test mutation",
		Patch:      json.RawMessage(data),
		Status:     mutation.StatusPrepared,
	}
}

func TestToolHandler_Validate(t *testing.T) {
	h := &ToolHandler{}

	// Valid
	m := makeMutation("add-tool", mutation.ToolDef{
		Name: "test", Description: "desc", Handler: "builtin:echo",
	})
	if err := h.Validate(m); err != nil {
		t.Errorf("valid tool should pass: %v", err)
	}

	// Missing name
	m = makeMutation("add-tool", mutation.ToolDef{Description: "desc", Handler: "builtin:echo"})
	if err := h.Validate(m); err == nil {
		t.Error("missing name should fail")
	}

	// Bad handler prefix
	m = makeMutation("add-tool", mutation.ToolDef{Name: "x", Description: "d", Handler: "unsafe:x"})
	if err := h.Validate(m); err == nil {
		t.Error("bad handler prefix should fail")
	}
}

func TestToolHandler_ApplyRollback(t *testing.T) {
	setup(t)
	h := &ToolHandler{}

	m := makeMutation("add-tool", mutation.ToolDef{
		Name: "utc_time", Description: "UTC time", Handler: "builtin:utc_time",
	})

	if err := h.Apply(m); err != nil {
		t.Fatal(err)
	}

	// Verify tool exists via rollback (which removes it)
	if err := h.Rollback(m); err != nil {
		t.Fatal(err)
	}
}

func TestPromptHandler_Validate(t *testing.T) {
	h := &PromptHandler{}

	// Valid
	m := makeMutation("change-prompt", mutation.PromptPatch{Target: "agent-system", Content: "Be helpful"})
	if err := h.Validate(m); err != nil {
		t.Errorf("valid prompt should pass: %v", err)
	}

	// Empty content
	m = makeMutation("change-prompt", mutation.PromptPatch{Target: "agent-system", Content: ""})
	if err := h.Validate(m); err == nil {
		t.Error("empty content should fail")
	}

	// Invalid target
	m = makeMutation("change-prompt", mutation.PromptPatch{Target: "invalid", Content: "hi"})
	if err := h.Validate(m); err == nil {
		t.Error("invalid target should fail")
	}
}

func TestPromptHandler_Apply(t *testing.T) {
	setup(t)
	h := &PromptHandler{}

	m := makeMutation("change-prompt", mutation.PromptPatch{
		Target: "agent-system", Content: "Be very helpful and safe.",
	})

	if err := h.Apply(m); err != nil {
		t.Fatal(err)
	}
}

func TestConstitutionHandler_Validate(t *testing.T) {
	h := &ConstitutionHandler{}

	// Valid
	m := makeMutation("amend-rule", mutation.RulePatch{RuleNumber: 1, NewText: "New rule", Rationale: "Better"})
	if err := h.Validate(m); err != nil {
		t.Errorf("valid rule should pass: %v", err)
	}

	// Invalid rule number
	m = makeMutation("amend-rule", mutation.RulePatch{RuleNumber: 6, NewText: "Bad", Rationale: "No"})
	if err := h.Validate(m); err == nil {
		t.Error("rule 6 should fail")
	}

	// Missing rationale
	m = makeMutation("amend-rule", mutation.RulePatch{RuleNumber: 1, NewText: "Ok", Rationale: ""})
	if err := h.Validate(m); err == nil {
		t.Error("empty rationale should fail")
	}
}

func TestSkillHandler_Apply(t *testing.T) {
	setup(t)
	h := &SkillHandler{}
	m := makeMutation("add-skill", mutation.SkillPatch{Name: "poetry", Description: "Write poems"})
	if err := h.Apply(m); err != nil {
		t.Fatal(err)
	}
}

func TestGenericHandler_Validate(t *testing.T) {
	h := &GenericHandler{}

	// Valid
	m := makeMutation("other", mutation.GenericPatch{Updates: map[string]string{"preferred_llm": "test"}})
	if err := h.Validate(m); err != nil {
		t.Errorf("valid generic should pass: %v", err)
	}

	// Empty updates
	m = makeMutation("other", mutation.GenericPatch{Updates: map[string]string{}})
	if err := h.Validate(m); err == nil {
		t.Error("empty updates should fail")
	}
}
