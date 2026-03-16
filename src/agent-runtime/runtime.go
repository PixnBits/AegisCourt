package agentruntime

import (
	"fmt"

	auditstore "aegiscourt/src/audit-store"
	"aegiscourt/src/kernel"
	llmrouter "aegiscourt/src/llm-router"
	"aegiscourt/src/sandbox"
)

// AgentRuntime manages ephemeral agent execution.
type AgentRuntime struct {
	SandboxMgr  *sandbox.SandboxManager
	Mediator    *kernel.Mediator
	LLM         *llmrouter.Router
	Audit       *auditstore.AuditStore
	llmEndpoint string
	llmModel    string
}

// NewAgentRuntime creates a new agent runtime.
func NewAgentRuntime(provider, llmEndpoint, llmModel, apiKey string, audit *auditstore.AuditStore) *AgentRuntime {
	return &AgentRuntime{
		SandboxMgr:  sandbox.NewSandboxManager(),
		Mediator:    kernel.NewMediator(),
		LLM:         llmrouter.NewRouter(provider, llmEndpoint, llmModel, apiKey),
		Audit:       audit,
		llmEndpoint: llmEndpoint,
		llmModel:    llmModel,
	}
}

// RunOneShot runs a one-shot agent task.
func (r *AgentRuntime) RunOneShot(task string) (string, error) {
	// Log start
	if r.Audit != nil {
		r.Audit.LogEvent("agent_run_start", map[string]string{"task": task})
	}

	response, err := r.LLM.CallPersona("default", task)
	if err != nil {
		if r.Audit != nil {
			r.Audit.LogEvent("agent_run_error", map[string]string{"error": err.Error()})
		}
		return "", err
	}

	// Log success
	if r.Audit != nil {
		r.Audit.LogEvent("agent_run_success", map[string]interface{}{
			"task":     task,
			"response": response,
		})
	}

	// TODO: spawn sandbox, mediate I/O, timeout
	return response, nil
}

// Start launches the background agent loop (stub for MVP).
func (r *AgentRuntime) Start() error {
	fmt.Println("Agent runtime started")
	return nil
}

// Stop stops the runtime.
func (r *AgentRuntime) Stop() error {
	fmt.Println("Agent runtime stopped")
	return nil
}
