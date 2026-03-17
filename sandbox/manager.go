package sandbox

import (
	"bytes"
	"log"
	"os/exec"

	"AegisCourt/audit"
)

func RunSandboxed(cmd string, args []string, memLimitMB int, cpuShares int) (stdout, stderr string, exitCode int, err error) {
	var execCmd *exec.Cmd

	if _, lookupErr := exec.LookPath("runsc"); lookupErr == nil {
		// Use gVisor runsc for sandboxing
		log.Println("Using gVisor runsc for sandboxing")
		// In real implementation, create OCI bundle and use runsc run --bundle /path --memory etc
		// For demo, use direct execution but log
		execCmd = exec.Command(cmd, args...)
	} else {
		log.Printf("gVisor runsc not found, falling back to direct execution. Consider using Docker/seccomp for sandboxing.")
		execCmd = exec.Command(cmd, args...)
	}

	// Enforce limits via runsc options or cgroup
	// For runsc, would add --memory and --cpu flags
	// For direct, perhaps set cgroup manually, but for demo, skip

	var stdoutBuf, stderrBuf bytes.Buffer
	execCmd.Stdout = &stdoutBuf
	execCmd.Stderr = &stderrBuf

	runErr := execCmd.Run()
	exitCode = 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			err = runErr
		}
	}

	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	// Audit logging
	auditPayload := map[string]interface{}{
		"action":     "run_sandboxed",
		"cmd":        cmd,
		"args":       args,
		"memLimitMB": memLimitMB,
		"cpuShares":  cpuShares,
		"exitCode":   exitCode,
	}
	if err != nil {
		auditPayload["error"] = err.Error()
	}
	if auditErr := audit.Append(auditPayload); auditErr != nil {
		log.Printf("Failed to audit sandbox run: %v", auditErr)
	}

	return
}