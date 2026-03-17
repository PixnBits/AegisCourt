package sandbox

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/shirou/gopsutil/v3/mem"
)

type Resources struct {
	CPU    int
	Memory int
}

type SandboxID string

type Manager struct {
	// stub
}

func Spawn(task string, resources Resources) (SandboxID, error) {
	// Detect RAM
	vmem, err := mem.VirtualMemory()
	if err == nil && vmem.Total < 8*1024*1024*1024 { // <8GB
		// Force single-reviewer fallback
		fmt.Println("Low RAM detected, forcing fallback mode")
	}

	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("sandbox not supported on %s", runtime.GOOS)
	}

	// Enforce cgroup limits (stub)
	// Use cgroups to limit CPU and Memory

	// Stub: use runsc or gvisor
	// For now, just exec the task
	cmd := exec.Command("echo", task)
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	return SandboxID("stub-id"), nil
}

func DetectPlatform() string {
	return runtime.GOOS
}

func GetSandboxBackend() string {
	switch runtime.GOOS {
	case "linux":
		return "gvisor"
	case "darwin":
		return "seccomp-fallback"
	case "windows":
		return "docker-fallback"
	default:
		return "unsupported"
	}
}
