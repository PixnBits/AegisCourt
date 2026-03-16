package llmrouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// OpenAIProvider implements LLMProvider for OpenAI.
type OpenAIProvider struct {
	apiKey string
	model  string
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider() *OpenAIProvider {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		// Stub
	}
	return &OpenAIProvider{
		apiKey: apiKey,
		model:  "gpt-3.5-turbo",
	}
}

// Generate sends a prompt to OpenAI and returns the response.
func (p *OpenAIProvider) Generate(prompt string, options map[string]interface{}) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}
	payload := map[string]interface{}{
		"model":    p.model,
		"messages": messages,
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
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("OpenAI API error: %s", resp.Status)
	}
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("invalid response format")
	}
	message, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}
	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}
	return content, nil
}

// Model returns the model name.
func (p *OpenAIProvider) Model() string {
	return p.model
}
