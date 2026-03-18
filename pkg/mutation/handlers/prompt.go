package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pixnbits/aegiscourt/pkg/keys"
	"github.com/pixnbits/aegiscourt/pkg/mutation"
)

// PromptHandler handles change-prompt mutations.
type PromptHandler struct{}

func (h *PromptHandler) Validate(m *mutation.Mutation) error {
	var pp mutation.PromptPatch
	if err := json.Unmarshal(m.Patch, &pp); err != nil {
		return fmt.Errorf("invalid prompt patch: %w", err)
	}
	if pp.Content == "" {
		return fmt.Errorf("prompt content is required")
	}
	if len(pp.Content) > 10000 {
		return fmt.Errorf("prompt content too long (max 10000 chars)")
	}
	if pp.Target == "" {
		pp.Target = "agent-system"
	}
	if pp.Target != "agent-system" {
		return fmt.Errorf("unsupported prompt target: %q (only agent-system supported)", pp.Target)
	}
	return nil
}

func (h *PromptHandler) Apply(m *mutation.Mutation) error {
	var pp mutation.PromptPatch
	if err := json.Unmarshal(m.Patch, &pp); err != nil {
		return err
	}

	dir, err := keys.AegisCourtDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "agent-prompt.txt")
	return os.WriteFile(path, []byte(pp.Content), 0600)
}

func (h *PromptHandler) Rollback(m *mutation.Mutation) error {
	// Snapshot restore handles the actual rollback.
	// We just need to confirm the prompt file can be reached.
	dir, err := keys.AegisCourtDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "agent-prompt.txt")
	os.Remove(path) // best effort — snapshot will overwrite anyway
	return nil
}

// ConstitutionHandler handles amend-rule mutations.
type ConstitutionHandler struct{}

func (h *ConstitutionHandler) Validate(m *mutation.Mutation) error {
	var rp mutation.RulePatch
	if err := json.Unmarshal(m.Patch, &rp); err != nil {
		return fmt.Errorf("invalid rule patch: %w", err)
	}
	if rp.RuleNumber < 1 || rp.RuleNumber > 5 {
		return fmt.Errorf("rule_number must be 1-5, got %d", rp.RuleNumber)
	}
	if rp.NewText == "" {
		return fmt.Errorf("new_text is required")
	}
	if rp.Rationale == "" {
		return fmt.Errorf("rationale is required for constitutional amendments")
	}
	return nil
}

func (h *ConstitutionHandler) Apply(m *mutation.Mutation) error {
	var rp mutation.RulePatch
	if err := json.Unmarshal(m.Patch, &rp); err != nil {
		return err
	}

	dir, err := keys.AegisCourtDir()
	if err != nil {
		return err
	}

	rulesPath := filepath.Join(dir, "constitution-rules.json")
	rules := make(map[string]string)

	data, err := os.ReadFile(rulesPath)
	if err == nil {
		json.Unmarshal(data, &rules)
	}

	key := fmt.Sprintf("rule_%d", rp.RuleNumber)
	rules[key] = rp.NewText

	out, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(rulesPath, out, 0600)
}

func (h *ConstitutionHandler) Rollback(m *mutation.Mutation) error {
	// Snapshot restore handles the rollback
	return nil
}
