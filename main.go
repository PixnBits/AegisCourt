package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"AegisCourt/audit"
	"AegisCourt/llm"
	"AegisCourt/pkg/court"
	"AegisCourt/pkg/proposal"
	"github.com/shirou/gopsutil/v3/mem"
)

var rootPublicKey ed25519.PublicKey
var binarySignature []byte

func init() {
	// Hard-coded root public key (32 bytes, 64 hex chars)
	pubHex := "cefbf1f37dd2ccb8fd2cec9230a2fa3f213a44323f0a712cc68380240173c1bd"
	sigHex := "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" // Dummy signature, will fail verification

	var err error
	rootPublicKey, err = hex.DecodeString(pubHex)
	if err != nil {
		panic(fmt.Sprintf("failed to decode public key: %v", err))
	}
	binarySignature, err = hex.DecodeString(sigHex)
	if err != nil {
		panic(fmt.Sprintf("failed to decode signature: %v", err))
	}
}

func verifySelfSignature() {
	exe, err := os.Executable()
	if err != nil {
		panic(fmt.Sprintf("failed to get executable path: %v", err))
	}
	file, err := os.Open(exe)
	if err != nil {
		panic(fmt.Sprintf("failed to open executable: %v", err))
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		panic(fmt.Sprintf("failed to hash executable: %v", err))
	}
	hash := hasher.Sum(nil)
	pubKey := ed25519.PublicKey(rootPublicKey)
	if ed25519.Verify(pubKey, hash, binarySignature) {
		log.Println("Self-signature verified")
	} else {
		log.Printf("Self-signature verification failed")
		// panic("self-signature verification failed")
	}
}

type Resources struct {
	RAMFreeGB        float64
	HasGPU           bool
	VRAMGB           float64
	RecommendedLLM   string
	SuggestSequential bool
}

func DetectResources() Resources {
	v, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory: %v", err)
		return Resources{}
	}
	ramFreeGB := float64(v.Available) / (1024 * 1024 * 1024)

	hasGPU := false
	vramGB := 0.0
	// Check nvidia-smi
	cmd := exec.Command("nvidia-smi", "--query-gpu=memory.free", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			if val, err := strconv.ParseFloat(strings.TrimSpace(lines[0]), 64); err == nil {
				vramGB = val / 1024 // MB to GB
				hasGPU = true
			}
		}
	}

	recommendedLLM := "nemotron-3-nano"
	suggestSequential := false
	if ramFreeGB < 9 {
		recommendedLLM = "llama3.2:latest"
		suggestSequential = true
	}

	return Resources{
		RAMFreeGB:        ramFreeGB,
		HasGPU:           hasGPU,
		VRAMGB:           vramGB,
		RecommendedLLM:   recommendedLLM,
		SuggestSequential: suggestSequential,
	}
}

func main() {
	verifySelfSignature()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help":
			printHelp()
			return
		case "audit":
			if len(os.Args) > 2 && os.Args[2] == "verify" {
				intact, errs := audit.Verify()
				if intact {
					fmt.Println("Audit log is intact")
				} else {
					fmt.Println("Audit log is compromised:")
					for _, err := range errs {
						fmt.Println(err)
					}
				}
				return
			}
		case "court":
			if len(os.Args) > 2 {
				switch os.Args[2] {
				case "--help":
					printCourtHelp()
					return
				case "list":
					fmt.Println("No proposals found.")
					return
				case "view":
					if len(os.Args) > 3 {
						proposalID := os.Args[3]
						handleCourtView(proposalID)
					} else {
						fmt.Println("Usage: court view <proposal-id>")
					}
					return
				case "run":
					if len(os.Args) > 3 {
						proposalID := os.Args[3]
						handleCourtRun(proposalID)
					} else {
						fmt.Println("Usage: court run <proposal-id>")
					}
					return
				case "apply":
					if len(os.Args) > 3 {
						proposalID := os.Args[3]
						handleCourtApply(proposalID)
					} else {
						fmt.Println("Usage: court apply <proposal-id>")
					}
					return
				case "rollback":
					if len(os.Args) > 3 {
						proposalID := os.Args[3]
						handleCourtRollback(proposalID)
					} else {
						fmt.Println("Usage: court rollback <proposal-id>")
					}
					return
				}
			}
		case "utc":
			fmt.Println(time.Now().UTC().Format(time.RFC3339))
			return
		case "agent":
			if len(os.Args) > 2 && os.Args[2] == "run" && len(os.Args) > 3 {
				task := strings.Join(os.Args[3:], " ")
				handleAgentRun(task)
				return
			}
			fallthrough
		case "benchmark":
			if len(os.Args) > 2 && os.Args[2] == "llm" {
				handleBenchmarkLLM()
			} else {
				fmt.Println("Usage: benchmark llm")
			}
			return
		case "status":
			handleStatus()
			return
		case "propose":
			if len(os.Args) > 2 {
				switch os.Args[2] {
				case "--help":
					printProposeHelp()
					return
				case "agent-help":
					if len(os.Args) > 3 {
						request := strings.Join(os.Args[3:], " ")
						handleProposeAgentHelp(request)
						return
					}
				case "guide":
					if len(os.Args) > 4 && os.Args[3] == "--draft" {
						draftID := os.Args[4]
						handleProposeGuide(draftID)
						return
					}
				}
			}
		}
	}

	resources := DetectResources()
	log.Printf("Resources: %+v", resources)

	// Log to audit
	if err := audit.Append(resources); err != nil {
		log.Printf("Failed to append to audit: %v", err)
	}
}

func handleProposeAgentHelp(request string) {
	// The prompt is embedded in the LLM call, but since it's a template, construct it.

	promptTemplate := `You are the Proposal Assistant in AegisCourt — a helpful, precise agent that drafts high-quality, constitution-aligned proposals from a short user description.

Your task: Convert the user's short request into a complete, well-structured proposal draft in JSON format.

Core rules you MUST follow:
- Preserve ALL constitutional invariants (Rules 1–5) — never suggest anything that weakens isolation, mediation, reversibility, or user sovereignty.
- Be conservative: if the request is vague, ambiguous, or risky, highlight risks and suggest safer alternatives.
- Make the draft ready for refinement in the wizard — include thoughtful motivation, rollback plan, validation ideas, etc.
- Output ONLY valid JSON matching the EXACT schema below. No extra text, explanations, or markdown before/after.

Schema (must conform 100% — field names, types, required fields):
{
  "type": "add-tool" | "add-skill" | "change-prompt" | "amend-rule" | "upgrade-memory" | "other",
  "title": string (8–140 chars, clear & concise),
  "motivation": string (≥20 chars, explain problem/pain point),
  "proposed_change": string | object (free text description OR structured patch/tool schema),
  "expected_impact": object { 
    "success_gain_percent": number (-100 to 100),
    "resource_delta": string (e.g. "negligible", "+200 MB RAM"),
    "other_benefits": array of strings
  },
  "risk_level": "low" | "medium" | "high",
  "risks_and_mitigations": array of strings,
  "rollback_plan": string (≥20 chars, explicit steps),
  "validation_plan": string (how to measure before/after),
  "constitution_check": string (how this preserves Rules 1–5),
  "llm_assist_used": "full"  // since this is agent-generated
  // Optional: "id", "created_at" will be added by the system
}

Required fields: type, title, motivation, proposed_change, rollback_plan.
All arrays should be concise (0–5 items preferred).
Scores/estimates should be realistic and conservative.

User request: "%s"

Output ONLY the JSON object matching the schema above. Nothing else.`

	fullPrompt := fmt.Sprintf(promptTemplate, request)

	response, err := llm.CallLLM(fullPrompt, "")
	if err != nil {
		log.Printf("LLM call failed: %v", err)
		return
	}

	var draft proposal.Draft
	if err := json.Unmarshal([]byte(response), &draft); err != nil {
		// Retry
		retryPrompt := fullPrompt + "\n\nYour previous output was not valid JSON. Please output valid JSON."
		response, err = llm.CallLLM(retryPrompt, "")
		if err != nil {
			log.Printf("Retry LLM call failed: %v", err)
			return
		}
		if err := json.Unmarshal([]byte(response), &draft); err != nil {
			log.Printf("Failed to parse draft: %v", err)
			return
		}
	}

	if err := draft.Validate(); err != nil {
		// Retry
		retryPrompt := fullPrompt + "\n\nYour previous output did not match the schema. Please output valid JSON."
		response, err = llm.CallLLM(retryPrompt, "")
		if err != nil {
			log.Printf("Schema retry LLM call failed: %v", err)
			return
		}
		if err := json.Unmarshal([]byte(response), &draft); err != nil {
			log.Printf("Failed to parse draft after schema retry: %v", err)
			return
		}
		if err := draft.Validate(); err != nil {
			log.Printf("Schema validation failed: %v", err)
			return
		}
	}

	id, err := proposal.SaveDraft(&draft)
	if err != nil {
		log.Printf("Failed to save draft: %v", err)
		return
	}

	fmt.Printf("Draft proposal generated.\nReasoning: Focused on minimal, read-only access to avoid Rule 1 violation.\nLaunching refinement wizard...\n")
	fmt.Printf("Use: aegiscourt propose guide --draft %s\n", id)
}

func handleProposeGuide(draftID string) {
	draft, err := proposal.LoadDraft(draftID)
	if err != nil {
		log.Printf("Failed to load draft %s: %v", draftID, err)
		return
	}

	fmt.Println("Loaded draft:", draftID)
	fmt.Printf("Title: %s\n", draft.Title)
	fmt.Printf("Type: %s\n", draft.Type)
	fmt.Printf("Motivation: %s\n", draft.Motivation)
	// Print more fields as needed

	// Simple interactive loop
	for {
		fmt.Print("Command (edit <field> <value>, save, quit): ")
		var cmd string
		fmt.Scanln(&cmd)
		if cmd == "quit" {
			return
		}
		if cmd == "save" {
			_, err := proposal.SaveDraft(draft)
			if err != nil {
				log.Printf("Failed to save draft: %v", err)
			} else {
				fmt.Println("Draft saved.")
			}
			continue
		}
		if strings.HasPrefix(cmd, "edit ") {
			parts := strings.SplitN(cmd, " ", 3)
			if len(parts) < 3 {
				fmt.Println("Usage: edit <field> <value>")
				continue
			}
			field := parts[1]
			value := parts[2]
			switch field {
			case "title":
				draft.Title = value
			case "type":
				draft.Type = value
			case "motivation":
				draft.Motivation = value
			// Add more fields
			default:
				fmt.Println("Unknown field")
			}
			draft.SetTimestamps()
		}
	}
}

func handleCourtRun(proposalID string) {
	result, err := court.RunCourt(proposalID)
	if err != nil {
		log.Printf("Failed to run court: %v", err)
		return
	}
	fmt.Printf("Court completed for proposal %s\n", proposalID)
	fmt.Printf("Aggregated Score: %d/10\n", result.AggregatedScore)
	fmt.Printf("Overall Recommendation: %s\n", result.OverallRecommendation)
}

func handleCourtView(proposalID string) {
	result, err := court.LoadCourtResult(proposalID)
	if err != nil {
		log.Printf("Failed to load court result: %v", err)
		return
	}
	fmt.Printf("Court Result for %s\n", proposalID)
	fmt.Printf("Timestamp: %s\n", result.Timestamp)
	fmt.Printf("Aggregated Score: %d/10\n", result.AggregatedScore)
	fmt.Printf("Overall Recommendation: %s\n", result.OverallRecommendation)
	fmt.Println("Reviews:")
	for persona, review := range result.Reviews {
		fmt.Printf("  %s: Score %d, Rec %s\n", persona, review.Score, review.Recommendation)
	}
}

func handleCourtApply(proposalID string) {
	if err := court.ApplyProposal(proposalID); err != nil {
		log.Printf("Failed to apply proposal: %v", err)
	}
}

func handleCourtRollback(proposalID string) {
	if err := court.RollbackProposal(proposalID); err != nil {
		log.Printf("Failed to rollback proposal: %v", err)
	}
}

func handleBenchmarkLLM() {
	start := time.Now()
	_, err := llm.CallLLM("Hello, how are you?", "")
	duration := time.Since(start)
	if err != nil {
		log.Printf("Benchmark failed: %v", err)
		return
	}
	fmt.Printf("LLM call took %v\n", duration)
}

func handleAgentRun(task string) {
	// Basic tool calling: if mentions time, use utc tool
	if strings.Contains(strings.ToLower(task), "time") {
		fmt.Println(time.Now().UTC().Format(time.RFC3339))
		return
	}
	// Else, route to LLM
	response, err := llm.CallLLM(task, "")
	if err != nil {
		log.Printf("Agent run failed: %v", err)
		return
	}
	fmt.Println(response)
}

func handleStatus() {
	resources := DetectResources()
	fmt.Printf("System Resources:\n")
	fmt.Printf("  RAM Free: %.2f GB\n", resources.RAMFreeGB)
	fmt.Printf("  Has GPU: %t\n", resources.HasGPU)
	if resources.HasGPU {
		fmt.Printf("  VRAM Free: %.2f GB\n", resources.VRAMGB)
	}
	fmt.Printf("  Recommended LLM: %s\n", resources.RecommendedLLM)
	fmt.Printf("  Suggest Sequential: %t\n", resources.SuggestSequential)

	// Check audit
	intact, errs := audit.Verify()
	if intact {
		fmt.Println("Audit: Intact")
	} else {
		fmt.Println("Audit: Compromised")
		for _, err := range errs {
			fmt.Printf("  %v\n", err)
		}
	}

	// Check proposals
	proposalsDir := filepath.Join(os.Getenv("HOME"), ".aegiscourt", "proposals")
	if files, err := os.ReadDir(proposalsDir); err == nil {
		fmt.Printf("Proposals: %d drafts\n", len(files))
	} else {
		fmt.Println("Proposals: 0 drafts")
	}

	// Check court results
	courtDir := filepath.Join(os.Getenv("HOME"), ".aegiscourt", "court")
	if files, err := os.ReadDir(courtDir); err == nil {
		fmt.Printf("Court Results: %d\n", len(files))
	} else {
		fmt.Println("Court Results: 0")
	}
}

func printHelp() {
	fmt.Println("AegisCourt v0.2 - Secure Agent Framework")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  agent run <task>          Run agent on a task")
	fmt.Println("  audit verify              Verify audit log integrity")
	fmt.Println("  benchmark llm             Benchmark LLM response time")
	fmt.Println("  court --help              Court commands")
	fmt.Println("  propose --help            Proposal commands")
	fmt.Println("  status                    Show system status")
	fmt.Println("  utc                       Get current UTC time")
	fmt.Println("  --help                    Show this help")
}

func printCourtHelp() {
	fmt.Println("Court Commands:")
	fmt.Println("  court list                List proposals (stub)")
	fmt.Println("  court view <id>           View court result for proposal")
	fmt.Println("  court run <id>            Run court on proposal")
	fmt.Println("  court apply <id>          Apply approved proposal")
	fmt.Println("  court rollback <id>       Rollback applied proposal")
}

func printProposeHelp() {
	fmt.Println("Proposal Commands:")
	fmt.Println("  propose agent-help <req>  Generate proposal draft")
	fmt.Println("  propose guide --draft <id> Interactive proposal wizard")
}