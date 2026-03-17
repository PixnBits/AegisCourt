package resources

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type SystemResources struct {
	TotalRAMGB float64
	FreeRAMGB  float64
	GPUName    string
	GPUVramGB  float64
	CPUCount   int
}

func Detect() *SystemResources {
	res := &SystemResources{
		CPUCount: runtime.NumCPU(),
	}
	res.TotalRAMGB, res.FreeRAMGB = detectRAM()
	res.GPUName, res.GPUVramGB = detectGPU()
	return res
}

func detectRAM() (total, free float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		total = float64(ms.Sys) / (1024 * 1024 * 1024)
		free = float64(ms.Sys-ms.Alloc) / (1024 * 1024 * 1024)
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = val / (1024 * 1024) // kB to GB
		case "MemAvailable:":
			free = val / (1024 * 1024)
		}
	}
	return
}

func detectGPU() (name string, vramGB float64) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		return "", 0
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return "", 0
	}
	// Can have multiple GPUs; take first
	lines := strings.Split(line, "\n")
	parts := strings.SplitN(lines[0], ", ", 2)
	if len(parts) == 2 {
		name = strings.TrimSpace(parts[0])
		mb, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err == nil {
			vramGB = mb / 1024
		}
	}
	return
}

func (r *SystemResources) RecommendLLM() (primary, fallback string, lowResource bool, warning string) {
	primary = "nemotron-3-nano:latest"
	fallback = "llama3.2:latest"
	lowResource = false

	// Estimate: base ~2-3 GB + ~1.5-2 GB per reviewer (parallel), ~7-10 GB sequential
	estimatedPeakParallel := 14.0
	estimatedPeakSequential := 10.0

	if r.FreeRAMGB < 9 {
		lowResource = true
		warning = fmt.Sprintf("Warning: Free RAM %.1f GB is below recommended %.0f GB for full parallel Court.\n"+
			"Using --low-resource mode (sequential reviewers). All 6 reviewers will still be used.\n"+
			"Consider llama3.2 if nemotron-3-nano causes swapping.",
			r.FreeRAMGB, estimatedPeakParallel)
		if r.FreeRAMGB < 6 {
			primary = fallback
			warning += "\nStrongly recommend llama3.2 due to very low available RAM."
		}
	} else if r.FreeRAMGB < estimatedPeakParallel {
		lowResource = true
		warning = fmt.Sprintf("Free RAM %.1f GB -- using sequential reviewer mode for stability (estimated peak: %.0f GB parallel).",
			r.FreeRAMGB, estimatedPeakSequential)
	}

	return
}

func (r *SystemResources) String() string {
	s := fmt.Sprintf("  Free RAM: %.1f GB (Total: %.1f GB)\n  CPUs: %d", r.FreeRAMGB, r.TotalRAMGB, r.CPUCount)
	if r.GPUName != "" {
		s += fmt.Sprintf("\n  GPU: %s %.0f GB VRAM (detected)", r.GPUName, r.GPUVramGB)
	} else {
		s += "\n  GPU: not detected"
	}
	return s
}
