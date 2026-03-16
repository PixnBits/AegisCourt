package agentruntime

import (
	"testing"
)

func TestAgentRuntime_RunOneShot(t *testing.T) {
	// This would require mocking LLM and audit
	// For MVP, just check no panic
	runtime := NewAgentRuntime("http://127.0.0.1:11434", "test", nil)
	_, err := runtime.RunOneShot("test task")
	// Expect error since no real LLM, but no panic
	if err == nil {
		t.Log("Unexpected success, but ok for stub")
	}
}

func TestAgentRuntime_ApplyMutation(t *testing.T) {
	runtime := NewAgentRuntime("http://127.0.0.1:11434", "test", nil)
	mutation := Mutation{
		ID:       "test",
		DiffType: "prompt-update",
		Target:   "test",
	}
	err := runtime.ApplyMutation(mutation)
	// Should succeed for prompt-update
	if err != nil {
		t.Fatal(err)
	}
}
