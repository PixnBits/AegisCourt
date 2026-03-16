package sandbox

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"syscall"
)

// IsLowResourceMode checks if the machine has low resources.
func IsLowResourceMode() bool {
	// RAM < 8GB
	var info syscall.Sysinfo_t
	err := syscall.Sysinfo(&info)
	if err == nil {
		ramGB := info.Totalram / (1024 * 1024 * 1024)
		if ramGB < 8 {
			return true
		}
	}
	// CPU cores < 4
	if runtime.NumCPU() < 4 {
		return true
	}
	return false
}

// applyCgroupLimits applies resource limits using cgroups v2.
func applyCgroupLimits(pid int, limits ResourceLimits) error {
	if runtime.GOOS != "linux" {
		return nil // no-op
	}

	// Create cgroup path
	cgroupPath := fmt.Sprintf("/sys/fs/cgroup/aegiscourt/sandbox_%d", pid)

	// Create cgroup dir
	err := os.MkdirAll(cgroupPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create cgroup: %w", err)
	}

	// Set memory.max
	memoryMax := strconv.Itoa(limits.MemoryMB * 1024 * 1024) // bytes
	err = os.WriteFile(cgroupPath+"/memory.max", []byte(memoryMax), 0644)
	if err != nil {
		return fmt.Errorf("failed to set memory.max: %w", err)
	}

	// Set cpu.max (shares)
	// cpu.max is period quota, e.g. 100000 10000 for 10%
	// For simplicity, set to limits.CPUShares / 100 * 100000 or something
	quota := limits.CPUShares * 1000 // assume shares 100 = 10%
	err = os.WriteFile(cgroupPath+"/cpu.max", []byte(fmt.Sprintf("%d 100000", quota)), 0644)
	if err != nil {
		return fmt.Errorf("failed to set cpu.max: %w", err)
	}

	// Add process to cgroup
	err = os.WriteFile(cgroupPath+"/cgroup.procs", []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return fmt.Errorf("failed to add process to cgroup: %w", err)
	}

	return nil
}
