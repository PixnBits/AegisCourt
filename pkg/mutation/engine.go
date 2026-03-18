package mutation

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pixnbits/aegiscourt/pkg/audit"
	"github.com/pixnbits/aegiscourt/pkg/court"
	"github.com/pixnbits/aegiscourt/pkg/proposal"
)

// Engine orchestrates atomic mutation apply and rollback.
type Engine struct {
	AuditLog *audit.Log
	Handlers map[string]Applier
}

func NewEngine(auditLog *audit.Log) *Engine {
	return &Engine{
		AuditLog: auditLog,
		Handlers: make(map[string]Applier),
	}
}

func (e *Engine) RegisterHandler(mutationType string, h Applier) {
	e.Handlers[mutationType] = h
}

// Apply executes the full atomic mutation flow for an approved proposal.
func (e *Engine) Apply(proposalID string) (*Mutation, error) {
	// 1. Load the court result
	cr, err := court.LoadResult(proposalID)
	if err != nil {
		return nil, fmt.Errorf("load court result: %w", err)
	}
	if cr.Status != court.StatusApproved {
		return nil, fmt.Errorf("proposal %s status is %s, not approved", proposalID, cr.Status)
	}

	// 2. Load the proposal draft
	draftID := cr.DraftID
	if draftID == "" {
		draftID = proposalID
	}
	draft, err := proposal.LoadDraft(draftID)
	if err != nil {
		return nil, fmt.Errorf("load draft: %w", err)
	}

	// 3. Build mutation
	mutID := GenerateMutationID(proposalID)
	patch, err := buildPatch(draft)
	if err != nil {
		return nil, fmt.Errorf("build patch: %w", err)
	}

	m := &Mutation{
		ID:         mutID,
		ProposalID: proposalID,
		Type:       draft.Type,
		Title:      draft.Title,
		Patch:      patch,
		Status:     StatusPrepared,
	}

	// 4. Find handler
	handler, ok := e.Handlers[m.Type]
	if !ok {
		return nil, fmt.Errorf("no handler registered for mutation type %q", m.Type)
	}

	// 5. Validate
	if err := handler.Validate(m); err != nil {
		m.Status = StatusFailed
		m.Error = fmt.Sprintf("validation failed: %v", err)
		SaveMutation(m)
		return m, fmt.Errorf("validation: %w", err)
	}

	// 6. Create snapshot before applying
	snapPath, err := CreateSnapshot(mutID)
	if err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}
	m.BeforeSnapshot = snapPath

	// 7. Apply
	if err := handler.Apply(m); err != nil {
		// Auto-rollback on failure
		m.Status = StatusFailed
		m.Error = fmt.Sprintf("apply failed: %v", err)
		RestoreSnapshot(snapPath)
		SaveMutation(m)
		e.auditEvent("mutation_apply_failed", mutID, err.Error())
		return m, fmt.Errorf("apply: %w", err)
	}

	// 8. Commit
	m.Status = StatusApplied
	m.AppliedAt = time.Now().UTC()
	if err := SaveMutation(m); err != nil {
		return nil, fmt.Errorf("save mutation record: %w", err)
	}

	e.auditEvent("mutation_applied", mutID, m.Title)
	return m, nil
}

// Rollback restores a specific mutation by ID.
func (e *Engine) Rollback(mutationID string) error {
	m, err := LoadMutation(mutationID)
	if err != nil {
		return fmt.Errorf("load mutation: %w", err)
	}
	if m.Status != StatusApplied {
		return fmt.Errorf("mutation %s status is %s, not applied", mutationID, m.Status)
	}
	if m.BeforeSnapshot == "" {
		return fmt.Errorf("no snapshot found for mutation %s", mutationID)
	}

	if err := RestoreSnapshot(m.BeforeSnapshot); err != nil {
		return fmt.Errorf("restore snapshot: %w", err)
	}

	// Also call handler-specific rollback for files not in snapshot
	if handler, ok := e.Handlers[m.Type]; ok {
		handler.Rollback(m)
	}

	m.Status = StatusRolled
	m.RolledBackAt = time.Now().UTC()
	if err := SaveMutation(m); err != nil {
		return fmt.Errorf("save mutation record: %w", err)
	}

	// Update court result status
	cr, err := court.LoadResult(m.ProposalID)
	if err == nil {
		cr.Status = court.StatusDeferred
		cr.VoteAction = "rollback"
		cr.VoteNotes = fmt.Sprintf("Rolled back mutation %s", mutationID)
		court.SaveResult(cr)
	}

	e.auditEvent("mutation_rolled_back", mutationID, m.Title)
	return nil
}

// RollbackLast rolls back the most recently applied mutation.
func (e *Engine) RollbackLast() error {
	m, err := LastAppliedMutation()
	if err != nil {
		return fmt.Errorf("list mutations: %w", err)
	}
	if m == nil {
		return fmt.Errorf("no applied mutations to roll back")
	}
	return e.Rollback(m.ID)
}

func (e *Engine) auditEvent(event, id, detail string) {
	if e.AuditLog != nil {
		e.AuditLog.Append(fmt.Sprintf("%s: %s — %s", event, id, detail))
	}
}

func buildPatch(draft *proposal.Draft) (json.RawMessage, error) {
	switch v := draft.ProposedChange.(type) {
	case map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal proposed_change map: %w", err)
		}
		return json.RawMessage(data), nil
	case string:
		// Wrap string content in a simple JSON object
		wrapper := map[string]string{"content": v}
		data, err := json.Marshal(wrapper)
		if err != nil {
			return nil, fmt.Errorf("marshal proposed_change string: %w", err)
		}
		return json.RawMessage(data), nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal proposed_change: %w", err)
		}
		return json.RawMessage(data), nil
	}
}
