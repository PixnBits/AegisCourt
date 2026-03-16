package llmrouter

import (
	"testing"
)

// mockProvider is a mock LLM provider for testing.
type mockProvider struct {
	response string
}

func (m *mockProvider) Generate(prompt string, options map[string]interface{}) (string, error) {
	return m.response, nil
}

func (m *mockProvider) Model() string {
	return "mock-model"
}

func TestFlagModelRisk(t *testing.T) {
	tests := []struct {
		model  string
		level  string
		reason string
	}{
		{"nemotron-3-nano", "Low", "Trusted open-source model"},
		{"llama3.1", "Low", "Trusted open-source model"},
		{"gemma2", "Low", "Trusted open-source model"},
		{"qwen2", "High", "Extra Court scrutiny required due to supply-chain concerns"},
		{"gpt-4", "Unknown", "Model risk not assessed"},
	}
	for _, tt := range tests {
		level, reason := FlagModelRisk(tt.model)
		if level != tt.level || reason != tt.reason {
			t.Errorf("FlagModelRisk(%s) = (%s, %s), want (%s, %s)", tt.model, level, reason, tt.level, tt.reason)
		}
	}
}
