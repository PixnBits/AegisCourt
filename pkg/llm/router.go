package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Router struct {
	Endpoint      string
	PrimaryModel  string
	FallbackModel string
	Timeout       time.Duration
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format,omitempty"`
}

type GenerateResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Format   string        `json:"format,omitempty"`
}

type ChatResponse struct {
	Model   string      `json:"model"`
	Message ChatMessage `json:"message"`
	Done    bool        `json:"done"`
}

type ModelInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type ModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

func NewRouter(endpoint, primary, fallback string) *Router {
	return &Router{
		Endpoint:      endpoint,
		PrimaryModel:  primary,
		FallbackModel: fallback,
		Timeout:       10 * time.Minute,
	}
}

func (r *Router) Generate(ctx context.Context, prompt string, jsonFormat bool) (string, string, error) {
	resp, model, err := r.doGenerate(ctx, r.PrimaryModel, prompt, jsonFormat)
	if err != nil && r.FallbackModel != "" && r.FallbackModel != r.PrimaryModel {
		resp, model, err = r.doGenerate(ctx, r.FallbackModel, prompt, jsonFormat)
	}
	return resp, model, err
}

func (r *Router) Chat(ctx context.Context, messages []ChatMessage, jsonFormat bool) (string, string, error) {
	resp, model, err := r.doChat(ctx, r.PrimaryModel, messages, jsonFormat)
	if err != nil && r.FallbackModel != "" && r.FallbackModel != r.PrimaryModel {
		resp, model, err = r.doChat(ctx, r.FallbackModel, messages, jsonFormat)
	}
	return resp, model, err
}

func (r *Router) doGenerate(ctx context.Context, model, prompt string, jsonFormat bool) (string, string, error) {
	req := GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}
	if jsonFormat {
		req.Format = "json"
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", model, err
	}
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", r.Endpoint+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", model, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", model, fmt.Errorf("LLM request failed (%s): %w", model, err)
	}
	defer httpResp.Body.Close()
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", model, fmt.Errorf("failed to read LLM response: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return "", model, fmt.Errorf("LLM returned status %d: %s", httpResp.StatusCode, string(respBody))
	}
	var genResp GenerateResponse
	if err := json.Unmarshal(respBody, &genResp); err != nil {
		return "", model, fmt.Errorf("failed to parse LLM response: %w", err)
	}
	return genResp.Response, model, nil
}

func (r *Router) doChat(ctx context.Context, model string, messages []ChatMessage, jsonFormat bool) (string, string, error) {
	req := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}
	if jsonFormat {
		req.Format = "json"
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", model, err
	}
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", r.Endpoint+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", model, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", model, fmt.Errorf("LLM chat request failed (%s): %w", model, err)
	}
	defer httpResp.Body.Close()
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", model, fmt.Errorf("failed to read LLM chat response: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return "", model, fmt.Errorf("LLM returned status %d: %s", httpResp.StatusCode, string(respBody))
	}
	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", model, fmt.Errorf("failed to parse LLM chat response: %w", err)
	}
	return chatResp.Message.Content, model, nil
}

func (r *Router) ListModels() ([]ModelInfo, error) {
	resp, err := http.Get(r.Endpoint + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Ollama at %s: %w", r.Endpoint, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var models ModelsResponse
	if err := json.Unmarshal(body, &models); err != nil {
		return nil, err
	}
	return models.Models, nil
}

func (r *Router) Ping() error {
	resp, err := http.Get(r.Endpoint + "/api/tags")
	if err != nil {
		return fmt.Errorf("cannot connect to Ollama at %s: %w", r.Endpoint, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}
	return nil
}
