package sandbox

import (
	"testing"
)

func TestRunSandboxed(t *testing.T) {
	stdout, stderr, exitCode, err := RunSandboxed("echo", []string{"hello", "world"}, 100, 1)
	if err != nil {
		t.Fatalf("RunSandboxed failed: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("Exit code %d, stderr: %s", exitCode, stderr)
	}
	expected := "hello world\n"
	if stdout != expected {
		t.Errorf("Expected stdout %q, got %q", expected, stdout)
	}
	if stderr != "" {
		t.Errorf("Expected no stderr, got %q", stderr)
	}
}

// Test suggestion for mocked runsc call:
// To mock runsc, you can use an interface for the executor.
// Define an interface Executor { Run(cmd string, args []string) (stdout, stderr string, exitCode int, err error) }
// Then, have a var executor Executor = &realExecutor{}
// In test, set executor = &mockExecutor{}
// Where mockExecutor returns predefined output.
// For runsc, mock the LookPath and the Run.