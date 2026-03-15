package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// SimpleAgent is a trivial agent that proposes self-improvements.
type SimpleAgent struct {
	kernel     *Kernel
	memoryFile string
	recent     []MemoryEntry
}

type MemoryEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
	Outcome     string    `json:"outcome"` // "approved", "rejected", "pending"
	Conditions  []string  `json:"conditions,omitempty"`
}

func NewSimpleAgent(kernel *Kernel) *SimpleAgent {
	return &SimpleAgent{
		kernel:     kernel,
		memoryFile: kernel.config.DataDir + "/agent-memory.jsonl",
		recent:     []MemoryEntry{},
	}
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

func (a *SimpleAgent) RunLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Load memory
		a.loadMemory()

		// Run benchmark
		currentScore := a.runBenchmark(ctx)

		// Build context
		memorySummary := ""
		if len(a.recent) > 0 {
			memorySummary = "Previous proposals and outcomes:\n"
			for _, m := range a.recent {
				memorySummary += fmt.Sprintf("- %s: %s\n", m.Description, m.Outcome)
			}
		}

		prompt := fmt.Sprintf(`You are a helpful agent in AegisCourt. Propose one small, safe self-improvement to the system constitution or configuration.

%s

Current benchmark score (count of 'success' in test response): %d

Based on previous outcomes, avoid proposing changes similar to recently rejected ones. Prefer changes that improve helpfulness, safety, or efficiency. Include the expected benchmark Δ in the description.

Example: {"description": "Add rule about data privacy (+2 benchmark)", "diff": [{"op": "add", "path": "/rules/11", "value": "New rule text"}]}

Respond with valid JSON: {"description": "brief description", "diff": json_patch_array}`, memorySummary, currentScore)

		response, err := a.kernel.llmRouter.Dispatch(ctx, prompt, "llama3.2")
		if err != nil {
			log.Printf("LLM dispatch failed: %v", err)
			time.Sleep(60 * time.Second)
			continue
		}

		var proposal struct {
			Description string          `json:"description"`
			Diff        json.RawMessage `json:"diff"`
		}

		if err := json.Unmarshal([]byte(response), &proposal); err != nil {
			log.Printf("Failed to parse agent proposal: %v, response: %s", err, response)
			time.Sleep(60 * time.Second)
			continue
		}

		// Submit
		err = a.kernel.SubmitProposal(ctx, proposal.Description, proposal.Diff)
		outcome := "approved"
		if err != nil {
			outcome = "rejected"
			log.Printf("Proposal rejected: %v", err)
		}

		// Append to memory
		entry := MemoryEntry{
			Timestamp:   time.Now(),
			Description: proposal.Description,
			Outcome:     outcome,
		}
		a.appendMemory(entry)

		// Sleep
		time.Sleep(120 * time.Second)
	}
}

func (a *SimpleAgent) runBenchmark(ctx context.Context) int {
	testPrompt := "Generate a response that includes the word 'success' multiple times to test the system."
	response, err := a.kernel.llmRouter.Dispatch(ctx, testPrompt, "llama3.2")
	if err != nil {
		return 0
	}
	return strings.Count(response, "success")
}

func (a *SimpleAgent) loadMemory() {
	data, err := os.ReadFile(a.memoryFile)
	if err != nil {
		return // no file yet
	}
	lines := strings.Split(string(data), "\n")
	a.recent = []MemoryEntry{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry MemoryEntry
		if json.Unmarshal([]byte(line), &entry) == nil {
			a.recent = append(a.recent, entry)
		}
	}
	// Keep last 5
	if len(a.recent) > 5 {
		a.recent = a.recent[len(a.recent)-5:]
	}
}

func (a *SimpleAgent) appendMemory(entry MemoryEntry) {
	data, _ := json.Marshal(entry)
	line := string(data) + "\n"
	f, err := os.OpenFile(a.memoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("Failed to append memory: %v", err)
		return
	}
	defer f.Close()
	f.WriteString(line)
	a.recent = append(a.recent, entry)
	if len(a.recent) > 5 {
		a.recent = a.recent[1:]
	}
}
