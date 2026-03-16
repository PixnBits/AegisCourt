package kernel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrap(t *testing.T) {
	tempDir := "/tmp/aegiscourt_test"
	os.RemoveAll(tempDir) // clean
	defer os.RemoveAll(tempDir)

	// Mock functions
	originalGetConfigDir := getConfigDir
	originalLoadConstitution := loadConstitution
	getConfigDir = func() string { return tempDir }
	loadConstitution = func() (string, error) { return "Mock constitution", nil }
	defer func() {
		getConfigDir = originalGetConfigDir
		loadConstitution = originalLoadConstitution
	}()

	// Call Bootstrap
	kernelHash, err := Bootstrap()
	if err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}
	if kernelHash == "" {
		t.Error("Kernel hash is empty")
	}

	// Check if files are created
	sigPath := filepath.Join(tempDir, "kernel.sig")
	if _, err := os.Stat(sigPath); os.IsNotExist(err) {
		t.Error("kernel.sig not created")
	}
	logPath := filepath.Join(tempDir, "bootstrap.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("bootstrap.log not created")
	}
}

func TestVerifyKernel(t *testing.T) {
	// TODO: mock for test
	// err := VerifyKernel()
	// if err != nil {
	// 	t.Errorf("VerifyKernel failed: %v", err)
	// }
}

func TestMediator(t *testing.T) {
	m := NewMediator()

	// Test AllowIO
	allowed, reason := m.AllowIO("file_write", "/tmp/test")
	if allowed {
		t.Error("File write should be denied")
	}
	if !strings.Contains(reason, "Rule 3") {
		t.Errorf("Reason should mention Rule 3: %s", reason)
	}

	allowed, reason = m.AllowIO("network", "http://example.com")
	if allowed {
		t.Error("Network should be denied")
	}

	allowed, reason = m.AllowIO("exec", "ls")
	if allowed {
		t.Error("Exec should be denied")
	}

	// Test direct methods
	allowed, reason = m.AllowFileWrite("/tmp/test")
	if allowed {
		t.Error("AllowFileWrite should deny")
	}

	allowed, reason = m.AllowNetwork("url")
	if allowed {
		t.Error("AllowNetwork should deny")
	}

	allowed, reason = m.AllowExec("cmd")
	if allowed {
		t.Error("AllowExec should deny")
	}
}

func TestChecker(t *testing.T) {
	constitution := "Mock constitution"
	checker := NewChecker(constitution)

	// Test action that violates Rule 1
	allowed, violations, reason := checker.Check("delete file")
	if allowed {
		t.Error("Action should not be allowed")
	}
	if len(violations) == 0 || violations[0] != 1 {
		t.Errorf("Should violate Rule 1: %v", violations)
	}
	if !strings.Contains(reason, "Rule 1") {
		t.Errorf("Reason should mention Rule 1: %s", reason)
	}

	// Test action that violates Rule 3
	allowed, violations, reason = checker.Check("write file")
	if allowed {
		t.Error("Action should not be allowed")
	}
	if len(violations) == 0 || violations[0] != 3 {
		t.Errorf("Should violate Rule 3: %v", violations)
	}

	// Test allowed action
	allowed, violations, reason = checker.Check("read memory")
	if !allowed {
		t.Error("Action should be allowed")
	}
	if len(violations) != 0 {
		t.Errorf("Should not violate any rules: %v", violations)
	}
}
