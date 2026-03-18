package agent

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultTools(t *testing.T) {
	tools := defaultTools()
	if len(tools) != 1 {
		t.Errorf("expected 1 default tool, got %d", len(tools))
	}
	if tools[0].Name != "echo" {
		t.Errorf("expected echo tool, got %s", tools[0].Name)
	}
}

func TestRegistryLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("AEGISCOURT_HOME", tmpDir)
	defer os.Unsetenv("AEGISCOURT_HOME")

	reg, err := LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(reg.Tools))
	}
}

func TestAddRemoveTool(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("AEGISCOURT_HOME", tmpDir)
	defer os.Unsetenv("AEGISCOURT_HOME")

	reg, _ := LoadRegistry()

	err := reg.AddTool(Tool{Name: "utc_time", Description: "UTC time", Handler: "builtin:utc_time"})
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(reg.Tools))
	}

	// Duplicate add should fail
	err = reg.AddTool(Tool{Name: "utc_time", Description: "duplicate", Handler: "builtin:utc_time"})
	if err == nil {
		t.Error("expected error on duplicate add")
	}

	// Remove
	err = reg.RemoveTool("utc_time")
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Tools) != 1 {
		t.Errorf("expected 1 tool after remove, got %d", len(reg.Tools))
	}

	// Remove nonexistent
	err = reg.RemoveTool("nonexistent")
	if err == nil {
		t.Error("expected error on nonexistent remove")
	}
}

func TestFindTool(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("AEGISCOURT_HOME", tmpDir)
	defer os.Unsetenv("AEGISCOURT_HOME")

	reg, _ := LoadRegistry()

	found := reg.FindTool("echo")
	if found == nil {
		t.Fatal("echo tool not found")
	}
	if found.Name != "echo" {
		t.Errorf("expected echo, got %s", found.Name)
	}

	notFound := reg.FindTool("nonexistent")
	if notFound != nil {
		t.Error("should not find nonexistent tool")
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("AEGISCOURT_HOME", tmpDir)
	defer os.Unsetenv("AEGISCOURT_HOME")

	reg, _ := LoadRegistry()
	prompt := reg.BuildSystemPrompt()

	if !strings.Contains(prompt, "echo") {
		t.Error("prompt should mention echo tool")
	}
	if !strings.Contains(prompt, "AegisCourt Agent") {
		t.Error("prompt should identify as AegisCourt Agent")
	}
}

func TestExecuteToolEcho(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("AEGISCOURT_HOME", tmpDir)
	defer os.Unsetenv("AEGISCOURT_HOME")

	reg, _ := LoadRegistry()
	result, err := ExecuteTool(reg, "echo", "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
}

func TestExecuteToolUTCTime(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("AEGISCOURT_HOME", tmpDir)
	defer os.Unsetenv("AEGISCOURT_HOME")

	reg, _ := LoadRegistry()
	reg.AddTool(Tool{Name: "utc_time", Description: "UTC", Handler: "builtin:utc_time"})

	result, err := ExecuteTool(reg, "utc_time", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "T") || !strings.Contains(result, "Z") {
		t.Errorf("expected ISO8601 time, got %q", result)
	}
}

func TestExecuteToolUnknown(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("AEGISCOURT_HOME", tmpDir)
	defer os.Unsetenv("AEGISCOURT_HOME")

	reg, _ := LoadRegistry()
	_, err := ExecuteTool(reg, "nonexistent", "")
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}
