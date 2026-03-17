package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCallLLM(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			var req GenerateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("Failed to decode request: %v", err)
				return
			}
			resp := GenerateResponse{
				Response: "Mock response for " + req.Prompt,
				Done:     true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else if r.URL.Path == "/api/tags" {
			w.WriteHeader(200)
			w.Write([]byte(`{"models": []}`))
		} else {
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	// Override config
	originalLoad := LoadConfig
	LoadConfig = func() (Config, error) {
		return Config{
			Endpoint:      server.URL,
			PrimaryModel:  "test-model",
			FallbackModel: "fallback-model",
		}, nil
	}
	defer func() { LoadConfig = originalLoad }()

	response, err := CallLLM("test prompt", "")
	if err != nil {
		t.Fatalf("CallLLM failed: %v", err)
	}
	expected := "Mock response for test prompt"
	if response != expected {
		t.Errorf("Expected %q, got %q", expected, response)
	}
}