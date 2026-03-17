package agent

import (
	"fmt"
	"strings"
)

type AgentRunner interface {
	RunLoop(task string) string
}

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]string
	Execute(args map[string]interface{}) (string, error)
}

type WebSearchTool struct{}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "Search the web"
}

func (t *WebSearchTool) Parameters() map[string]string {
	return map[string]string{"query": "string"}
}

func (t *WebSearchTool) Execute(args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok {
		return "", fmt.Errorf("query must be string")
	}
	// Mock search
	return "Mock search results for: " + query, nil
}

type Runtime struct {
	Tools map[string]Tool
}

func NewRuntime() *Runtime {
	return &Runtime{
		Tools: map[string]Tool{
			"web_search": &WebSearchTool{},
		},
	}
}

func (r *Runtime) RunLoop(task string) string {
	// Simple implementation: if task contains "search", use web_search
	if strings.Contains(task, "search") {
		if tool, ok := r.Tools["web_search"]; ok {
			result, err := tool.Execute(map[string]interface{}{"query": task})
			if err != nil {
				return "Error: " + err.Error()
			}
			return result
		}
	}
	return "Task completed: " + task
}
