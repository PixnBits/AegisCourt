package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/pixnbits/aegiscourt/pkg/agent"
	"github.com/pixnbits/aegiscourt/pkg/mutation"
)

// ToolHandler handles add-tool mutations.
type ToolHandler struct{}

func (h *ToolHandler) Validate(m *mutation.Mutation) error {
	var td mutation.ToolDef
	if err := json.Unmarshal(m.Patch, &td); err != nil {
		return fmt.Errorf("invalid tool definition: %w", err)
	}
	if td.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if td.Description == "" {
		return fmt.Errorf("tool description is required")
	}
	if td.Handler == "" {
		return fmt.Errorf("tool handler is required")
	}
	// Only allow known handler prefixes
	allowed := []string{"builtin:", "custom:"}
	valid := false
	for _, prefix := range allowed {
		if len(td.Handler) >= len(prefix) && td.Handler[:len(prefix)] == prefix {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("handler must start with builtin: or custom:, got %q", td.Handler)
	}
	return nil
}

func (h *ToolHandler) Apply(m *mutation.Mutation) error {
	var td mutation.ToolDef
	if err := json.Unmarshal(m.Patch, &td); err != nil {
		return err
	}

	reg, err := agent.LoadRegistry()
	if err != nil {
		return fmt.Errorf("load tool registry: %w", err)
	}

	t := agent.Tool{
		Name:        td.Name,
		Description: td.Description,
		Handler:     td.Handler,
		Parameters:  td.Parameters,
		OutputType:  td.OutputType,
	}

	if err := reg.AddTool(t); err != nil {
		return fmt.Errorf("add tool to registry: %w", err)
	}

	return nil
}

func (h *ToolHandler) Rollback(m *mutation.Mutation) error {
	var td mutation.ToolDef
	if err := json.Unmarshal(m.Patch, &td); err != nil {
		return err
	}

	reg, err := agent.LoadRegistry()
	if err != nil {
		return fmt.Errorf("load tool registry: %w", err)
	}

	return reg.RemoveTool(td.Name)
}
