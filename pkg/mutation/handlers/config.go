package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/pixnbits/aegiscourt/pkg/config"
	"github.com/pixnbits/aegiscourt/pkg/mutation"
)

// SkillHandler handles add-skill mutations.
type SkillHandler struct{}

func (h *SkillHandler) Validate(m *mutation.Mutation) error {
	var sp mutation.SkillPatch
	if err := json.Unmarshal(m.Patch, &sp); err != nil {
		return fmt.Errorf("invalid skill patch: %w", err)
	}
	if sp.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if sp.Description == "" {
		return fmt.Errorf("skill description is required")
	}
	return nil
}

func (h *SkillHandler) Apply(m *mutation.Mutation) error {
	var sp mutation.SkillPatch
	if err := json.Unmarshal(m.Patch, &sp); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.UseCases == "" {
		cfg.UseCases = "skill:" + sp.Name
	} else {
		cfg.UseCases += ", skill:" + sp.Name
	}
	return cfg.Save()
}

func (h *SkillHandler) Rollback(m *mutation.Mutation) error {
	return nil
}

// MemoryHandler handles upgrade-memory mutations.
type MemoryHandler struct{}

func (h *MemoryHandler) Validate(m *mutation.Mutation) error {
	var raw map[string]any
	if err := json.Unmarshal(m.Patch, &raw); err != nil {
		return fmt.Errorf("invalid memory patch: %w", err)
	}
	return nil
}

func (h *MemoryHandler) Apply(m *mutation.Mutation) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.UseCases == "" {
		cfg.UseCases = "memory-upgrade"
	} else {
		cfg.UseCases += ", memory-upgrade"
	}
	return cfg.Save()
}

func (h *MemoryHandler) Rollback(m *mutation.Mutation) error {
	return nil
}

// GenericHandler handles "other" type mutations via config key-value updates.
type GenericHandler struct{}

func (h *GenericHandler) Validate(m *mutation.Mutation) error {
	var gp mutation.GenericPatch
	if err := json.Unmarshal(m.Patch, &gp); err != nil {
		return fmt.Errorf("invalid generic patch: %w", err)
	}
	if len(gp.Updates) == 0 {
		return fmt.Errorf("at least one update is required")
	}
	return nil
}

func (h *GenericHandler) Apply(m *mutation.Mutation) error {
	var gp mutation.GenericPatch
	if err := json.Unmarshal(m.Patch, &gp); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	for k, v := range gp.Updates {
		if err := cfg.Set(k, v); err != nil {
			return fmt.Errorf("set %s=%s: %w", k, v, err)
		}
	}
	return cfg.Save()
}

func (h *GenericHandler) Rollback(m *mutation.Mutation) error {
	return nil
}
