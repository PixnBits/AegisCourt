package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// SimpleAgent is a trivial agent that proposes self-improvements.
type SimpleAgent struct {
	kernel *Kernel
}

func NewSimpleAgent(kernel *Kernel) *SimpleAgent {
	return &SimpleAgent{kernel: kernel}
}

func (a *SimpleAgent) RunOnce(ctx context.Context) error {
	prompt := `You are a helpful agent in AegisCourt. Propose one small, safe self-improvement to the system constitution or configuration.

Example: {"description": "Add rule about data privacy", "diff": [{"op": "add", "path": "/rules/11", "value": "New rule text"}]}

Respond with valid JSON: {"description": "brief description", "diff": json_patch_array}`

	response, err := a.kernel.llmRouter.Dispatch(ctx, prompt, "llama3.2")
	if err != nil {
		return fmt.Errorf("LLM dispatch failed: %w", err)
	}

	var proposal struct {
		Description string          `json:"description"`
		Diff        json.RawMessage `json:"diff"`
	}

	if err := json.Unmarshal([]byte(response), &proposal); err != nil {
		log.Printf("Failed to parse agent proposal: %v, response: %s", err, response)
		return err
	}

	// Submit
	return a.kernel.SubmitProposal(ctx, proposal.Description, proposal.Diff)
}