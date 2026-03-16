package llmrouter

import (
	"strings"
)

// FlagModelRisk assesses the risk level of a model.
func FlagModelRisk(model string) (level string, reason string) {
	model = strings.ToLower(model)
	switch {
	case strings.HasPrefix(model, "nemotron") || strings.HasPrefix(model, "llama3") || strings.HasPrefix(model, "gemma"):
		return "Low", "Trusted open-source model"
	case strings.HasPrefix(model, "qwen"):
		return "High", "Extra Court scrutiny required due to supply-chain concerns"
	default:
		return "Unknown", "Model risk not assessed"
	}
}
