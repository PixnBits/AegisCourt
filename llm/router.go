package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"AegisCourt/audit"
)

type Config struct {
	Endpoint      string `json:"llm_endpoint"`
	PrimaryModel  string `json:"llm_model"`
	FallbackModel string `json:"fallback_model"`
}

var LoadConfig func() (Config, error) = loadConfig

func loadConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	configPath := filepath.Join(home, ".aegiscourt", "config.json")
	file, err := os.Open(configPath)
	if err != nil {
		// Default config
		return Config{
			Endpoint:      "http://localhost:11434",
			PrimaryModel:  "llama3.2:latest",
			FallbackModel: "llama3.2:latest",
		}, nil
	}
	defer file.Close()
	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return Config{}, err
	}
	if config.FallbackModel == "" {
		config.FallbackModel = "llama3.2:latest"
	}
	return config, nil
}

func healthCheck(endpoint string) error {
	resp, err := http.Get(endpoint + "/api/tags")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}
	return nil
}

func init() {
	config, err := LoadConfig()
	if err != nil {
		log.Printf("Failed to load LLM config: %v", err)
		return
	}
	if err := healthCheck(config.Endpoint); err != nil {
		log.Printf("LLM health check failed: %v", err)
	}
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func CallLLM(prompt string, model string) (string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", err
	}

	endpoint := config.Endpoint
	if model == "" {
		model = config.PrimaryModel
	}

	reqBody := GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(endpoint+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// Try fallback if primary failed
		if model == config.PrimaryModel {
			log.Printf("Primary model failed, trying fallback")
			return CallLLM(prompt, config.FallbackModel)
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if model == config.PrimaryModel {
			log.Printf("Primary model failed with status %d, trying fallback", resp.StatusCode)
			return CallLLM(prompt, config.FallbackModel)
		}
		return "", fmt.Errorf("LLM call failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var genResp GenerateResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return "", err
	}

	response := genResp.Response

	// Audit log
	auditPayload := map[string]interface{}{
		"action":   "call_llm",
		"model":    model,
		"prompt":   prompt,
		"response": response,
	}
	if auditErr := audit.Append(auditPayload); auditErr != nil {
		log.Printf("Failed to audit LLM call: %v", auditErr)
	}

	return response, nil
}