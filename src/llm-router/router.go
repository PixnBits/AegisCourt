package llmrouter

import "log"

// LLMProvider interface for LLM backends.
type LLMProvider interface {
	Generate(prompt string, options map[string]interface{}) (string, error)
	Model() string
}

// Router routes LLM calls to appropriate providers.
type Router struct {
	primaryProvider  LLMProvider
	personaProviders map[string]LLMProvider
}

// NewRouter creates a new router with default providers.
func NewRouter(endpoint, model string) *Router {
	// For MVP, use Ollama as primary
	primary := &OllamaProvider{endpoint: endpoint, model: model}
	personaProviders := map[string]LLMProvider{
		"default": primary,
		// Add more later
	}
	return &Router{
		primaryProvider:  primary,
		personaProviders: personaProviders,
	}
}

// CallPersona calls the LLM for a specific persona.
func (r *Router) CallPersona(persona string, prompt string) (string, error) {
	provider, exists := r.personaProviders[persona]
	if !exists {
		provider = r.primaryProvider
	}
	// Check model risk
	level, reason := FlagModelRisk(provider.Model())
	if level == "High" {
		log.Printf("Warning: High-risk model %s used: %s", provider.Model(), reason)
		// TODO: force multi-reviewer mode
	}
	return provider.Generate(prompt, nil)
}
