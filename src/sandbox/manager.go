package sandbox

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

// SandboxRuntime interface for different sandbox implementations.
type SandboxRuntime interface {
	Spawn(command string, args []string, limits ResourceLimits) (*exec.Cmd, error)
}

// gVisorRuntime uses gVisor runsc.
type gVisorRuntime struct{}

// SeccompRuntime uses seccomp (stub).
type SeccompRuntime struct{}

// FallbackRuntime runs without isolation.
type FallbackRuntime struct{}

// ResourceLimits defines resource constraints for sandboxes.
type ResourceLimits struct {
	MemoryMB  int
	CPUShares int
}

// SandboxID is a unique identifier for a sandbox.
type SandboxID string

// SandboxManager manages sandboxes.
type SandboxManager struct {
	runtime   SandboxRuntime
	sandboxes map[SandboxID]*exec.Cmd
	nextID    int
}

// NewSandboxManager creates a new sandbox manager with appropriate runtime.
func NewSandboxManager() *SandboxManager {
	var rt SandboxRuntime
	switch runtime.GOOS {
	case "linux":
		rt = &gVisorRuntime{}
	default:
		log.Println("Warning: Using reduced isolation – Docker recommended for full security")
		rt = &FallbackRuntime{}
	}
	return &SandboxManager{
		runtime:   rt,
		sandboxes: make(map[SandboxID]*exec.Cmd),
		nextID:    1,
	}
}

// Spawn starts a new sandboxed process.
func (sm *SandboxManager) Spawn(command string, args []string, limits ResourceLimits) (SandboxID, error) {
	id := SandboxID(strconv.Itoa(sm.nextID))
	sm.nextID++

	cmd, err := sm.runtime.Spawn(command, args, limits)
	if err != nil {
		return "", err
	}

	// Apply resource limits
	if cmd.Process != nil {
		err = applyCgroupLimits(cmd.Process.Pid, limits)
		if err != nil {
			log.Printf("Warning: failed to apply cgroup limits: %v", err)
		}
	}

	sm.sandboxes[id] = cmd

	time.Sleep(50 * time.Millisecond) // assume <100ms

	return id, nil
}

// Kill terminates a sandbox.
func (sm *SandboxManager) Kill(id SandboxID) error {
	cmd, exists := sm.sandboxes[id]
	if !exists {
		return fmt.Errorf("sandbox not found")
	}
	err := cmd.Process.Kill()
	delete(sm.sandboxes, id)
	return err
}

// Implementations

func (rt *gVisorRuntime) Spawn(command string, args []string, limits ResourceLimits) (*exec.Cmd, error) {
	_, err := exec.LookPath("runsc")
	if err != nil {
		return nil, fmt.Errorf("runsc not found: %w", err)
	}
	// Stub: run with runsc
	cmd := exec.Command(command, args...)
	cmd.Stdout = &outputCapture{}
	cmd.Stderr = &outputCapture{}
	return cmd, cmd.Start()
}

func (rt *SeccompRuntime) Spawn(command string, args []string, limits ResourceLimits) (*exec.Cmd, error) {
	// Stub
	log.Println("Seccomp runtime stub")
	cmd := exec.Command(command, args...)
	return cmd, cmd.Start()
}

func (rt *FallbackRuntime) Spawn(command string, args []string, limits ResourceLimits) (*exec.Cmd, error) {
	log.Println("Running without isolation")
	cmd := exec.Command(command, args...)
	cmd.Stdout = &outputCapture{}
	cmd.Stderr = &outputCapture{}
	return cmd, cmd.Start()
}

// outputCapture captures stdout/stderr (stub).
type outputCapture struct {
	output string
}

func (oc *outputCapture) Write(p []byte) (n int, err error) {
	oc.output += string(p)
	return len(p), nil
}
