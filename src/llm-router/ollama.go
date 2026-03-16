package llmrouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// OllamaProvider implements LLMProvider for Ollama.
type OllamaProvider struct {
	endpoint string
	model    string
}

// Generate sends a prompt to Ollama and returns the response.
func (p *OllamaProvider) Generate(prompt string, options map[string]interface{}) (string, error) {
	// Ollama API: POST /api/generate
	url := p.endpoint + "/api/generate"
	payload := map[string]interface{}{
		"model":  p.model,
		"prompt": prompt,
		"stream": false,
	}
	if options != nil {
		for k, v := range options {
			payload[k] = v
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Ollama API error: %s", resp.Status)
	}
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}
	response, ok := result["response"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}
	return response, nil
}

// Model returns the model name.
func (p *OllamaProvider) Model() string {
	return p.model
}
