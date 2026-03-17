package sandbox

import (
	"fmt"
	"os/exec"
	"runtime"
)

type Resources struct {
	CPU    int
	Memory int
}

type SandboxID string

func Spawn(task string, resources Resources) (SandboxID, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("sandbox not supported on %s", runtime.GOOS)
	}

	// Stub: use runsc or gvisor
	// For now, just exec the task
	cmd := exec.Command("echo", task)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return "stub-id", nil
}

func MediateSyscall(syscall string, args ...interface{}) error {
	// Stub: check against allowed syscalls
	allowed := []string{"write", "read"}
	for _, a := range allowed {
		if syscall == a {
			return nil
		}
	}
	return fmt.Errorf("syscall %s not allowed", syscall)
}
