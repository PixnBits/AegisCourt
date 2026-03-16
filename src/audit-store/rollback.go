package auditstore

// RollbackTo rolls back to a specific entry ID.
func (as *AuditStore) RollbackTo(id string) error {
	// For MVP, just log the rollback event
	// In full implementation, find the mutation entry and apply inverse
	payload := map[string]interface{}{
		"target_id": id,
		"reason":    "manual rollback",
	}
	return as.LogEvent("rollback", payload)
}

// EmergencyHalt performs an emergency halt and rollback.
func (as *AuditStore) EmergencyHalt() error {
	// Log halt
	as.LogEvent("emergency_halt", map[string]string{"action": "freeze_all"})

	// Rollback last mutation (stub)
	as.LogEvent("rollback_last", map[string]string{"reason": "emergency"})

	return nil
}
