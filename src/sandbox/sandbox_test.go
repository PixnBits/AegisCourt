package sandbox

import (
	"testing"
	"time"
)

func TestSpawn(t *testing.T) {
	sm := NewSandboxManager()

	// Spawn echo hello
	id, err := sm.Spawn("echo", []string{"hello"}, ResourceLimits{MemoryMB: 100, CPUShares: 100})
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	// For MVP, since output is not returned, just check that process started
	cmd := sm.sandboxes[id]
	if cmd == nil {
		t.Error("Command not stored")
	}

	// Kill
	err = sm.Kill(id)
	if err != nil {
		t.Errorf("Kill failed: %v", err)
	}
}
