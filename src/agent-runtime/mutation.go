package agentruntime

import (
	"encoding/json"
	"fmt"
)

// Mutation represents a self-modification.
type Mutation struct {
	ID       string          `json:"id"`
	DiffType string          `json:"diff_type"` // "json-patch", "code-replace", "prompt-update"
	Target   string          `json:"target"`    // e.g. "agent-tool-calling-prompt"
	Patch    json.RawMessage `json:"patch"`
	Rollback json.RawMessage `json:"rollback"`
}

// ApplyMutation applies a mutation to the agent.
func (r *AgentRuntime) ApplyMutation(m Mutation) error {
	// Log before
	if r.Audit != nil {
		r.Audit.LogEvent("mutation_start", map[string]interface{}{
			"id":        m.ID,
			"diff_type": m.DiffType,
			"target":    m.Target,
		})
	}

	// Validate against constitution (stub)
	allowed, reason := r.Mediator.AllowIO("mutation", m.Target)
	if !allowed {
		if r.Audit != nil {
			r.Audit.LogEvent("mutation_denied", map[string]string{"reason": reason})
		}
		return fmt.Errorf("mutation denied: %s", reason)
	}

	// Apply based on type
	switch m.DiffType {
	case "prompt-update":
		// For MVP, just log
		fmt.Printf("Applying prompt update to %s\n", m.Target)
	case "code-replace":
		// Stub: would write to sandbox
		fmt.Printf("Applying code replace to %s\n", m.Target)
	default:
		return fmt.Errorf("unsupported diff type: %s", m.DiffType)
	}

	// Log after
	if r.Audit != nil {
		r.Audit.LogEvent("mutation_applied", map[string]interface{}{
			"id":        m.ID,
			"diff_type": m.DiffType,
			"target":    m.Target,
		})
	}

	return nil
}

// RollbackMutation rolls back a mutation.
func (r *AgentRuntime) RollbackMutation(id string) error {
	// Log rollback
	r.Audit.LogEvent("mutation_rollback", map[string]string{"id": id})

	// Stub: apply inverse
	fmt.Printf("Rolling back mutation %s\n", id)

	return nil
}
