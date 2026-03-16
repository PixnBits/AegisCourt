package kernel

// Mediator handles all I/O mediation for agents, tools, and sandboxes.
type Mediator struct {
	// TODO: add config, constitution, audit log references
}

// NewMediator creates a new mediator instance.
func NewMediator() *Mediator {
	return &Mediator{}
}

// AllowIO checks if an I/O operation is allowed based on constitution rules.
// For MVP, hard-codes checks for Rules 1,2,3.
func (m *Mediator) AllowIO(operation string, target string) (bool, string) {
	switch operation {
	case "file_write":
		return m.AllowFileWrite(target)
	case "network":
		return m.AllowNetwork(target)
	case "exec":
		return m.AllowExec(target)
	case "mutation":
		// For MVP, allow mutations (would check constitution)
		return true, ""
	default:
		return false, "Unknown operation"
	}
}

// AllowFileWrite checks file write permission.
// Rule 3: No unauthorized host access.
func (m *Mediator) AllowFileWrite(path string) (bool, string) {
	// For MVP, always deny without Court approval
	return false, "Blocked: Rule 3 violation – unauthorized host file write"
}

// AllowNetwork checks network access permission.
// Rule 3: No unauthorized external access.
func (m *Mediator) AllowNetwork(url string) (bool, string) {
	// For MVP, always deny
	return false, "Blocked: Rule 3 violation – unauthorized network access"
}

// AllowExec checks execution permission.
// Rule 3: No unauthorized process spawning.
func (m *Mediator) AllowExec(cmd string) (bool, string) {
	// For MVP, always deny
	return false, "Blocked: Rule 3 violation – unauthorized command execution"
}
