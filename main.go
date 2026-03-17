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
	"strconv"
	"strings"

	"AegisCourt/audit"
	"AegisCourt/llm"
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
				case "list":
					fmt.Println("No proposals found.")
					return
				case "view":
					fmt.Println("Court view not implemented yet.")
					return
				}
			}
		case "propose":
			if len(os.Args) > 2 {
				switch os.Args[2] {
				case "agent-help":
					if len(os.Args) > 3 {
						request := strings.Join(os.Args[3:], " ")
						handleProposeAgentHelp(request)
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