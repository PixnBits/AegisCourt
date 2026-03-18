package mutation

import (
	"encoding/json"
	"time"
)

// Status tracks the lifecycle of a mutation.
type Status string

const (
	StatusPrepared Status = "prepared"
	StatusApplied  Status = "applied"
	StatusRolled   Status = "rolled_back"
	StatusFailed   Status = "failed"
)

// Mutation records one atomic change applied (or attempted) by the kernel.
type Mutation struct {
	ID             string          `json:"id"`
	ProposalID     string          `json:"proposal_id"`
	Type           string          `json:"type"`
	Title          string          `json:"title"`
	Patch          json.RawMessage `json:"patch"`
	BeforeSnapshot string          `json:"before_snapshot"`
	Status         Status          `json:"status"`
	AppliedAt      time.Time       `json:"applied_at,omitempty"`
	RolledBackAt   time.Time       `json:"rolled_back_at,omitempty"`
	Error          string          `json:"error,omitempty"`
}

// Applier is the interface that each mutation type handler must implement.
type Applier interface {
	Validate(m *Mutation) error
	Apply(m *Mutation) error
	Rollback(m *Mutation) error
}

// ToolDef is the schema for an add-tool patch.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Handler     string         `json:"handler"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	OutputType  string         `json:"output_type,omitempty"`
}

// PromptPatch is the schema for a change-prompt patch.
type PromptPatch struct {
	Target  string `json:"target"`
	Content string `json:"content"`
}

// RulePatch is the schema for an amend-rule patch.
type RulePatch struct {
	RuleNumber int    `json:"rule_number"`
	NewText    string `json:"new_text"`
	Rationale  string `json:"rationale"`
}

// SkillPatch is the schema for an add-skill patch.
type SkillPatch struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Config      any    `json:"config,omitempty"`
}

// GenericPatch is the schema for an "other" type patch.
type GenericPatch struct {
	Updates map[string]string `json:"updates"`
}
