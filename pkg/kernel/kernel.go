package kernel

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/PixnBits/AegisCourt/pkg/agent"
	"github.com/PixnBits/AegisCourt/pkg/audit"
	"github.com/PixnBits/AegisCourt/pkg/config"
	"github.com/PixnBits/AegisCourt/pkg/constitution"
	"github.com/PixnBits/AegisCourt/pkg/court"
	"github.com/PixnBits/AegisCourt/pkg/sandbox"
	jsonpatch "github.com/evanphx/json-patch"
)

type LLMRouter struct{}

type Kernel struct {
	Config          *config.Profile
	SandboxMgr      *sandbox.Manager
	LLMRouter       *LLMRouter
	CourtEngine     *court.Engine
	AuditStore      *audit.Store
	AgentRuntime    agent.AgentRunner
	Constitution    interface{}
	ApprovedTools   map[string]agent.Tool
	PubKey          ed25519.PublicKey
	StoredSignature []byte
	mu              sync.RWMutex
	halted          bool
}

func NewKernel(cfg *config.Profile) (*Kernel, error) {
	return &Kernel{
		Config:        cfg,
		SandboxMgr:    &sandbox.Manager{}, // stub
		AuditStore:    audit.NewStore(),
		ApprovedTools: make(map[string]agent.Tool),
	}, nil
}

func (k *Kernel) Start() error {
	// Bootstrap if needed
	if err := k.Bootstrap(); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	// Verify self
	if err := k.VerifySelf(); err != nil {
		return fmt.Errorf("self-verification failed: %w", err)
	}

	// Initialize sub-components
	k.Constitution = constitution.GetRules()
	k.CourtEngine = court.NewEngine()
	k.AgentRuntime = agent.NewRuntime()
	k.LLMRouter = &LLMRouter{} // stub

	// Register approved tools
	k.RegisterApprovedTool(&agent.WebSearchTool{})

	// Register approved tools (stub)
	// k.RegisterApprovedTool(&agent.WebSearchTool{})

	// Start background goroutines if needed
	// For now, just print
	fmt.Println("Kernel started successfully")
	return nil
}

func (k *Kernel) HandleProposal(p court.Proposal) error {
	// Create audit entry
	payload, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal proposal: %w", err)
	}
	entry := audit.AuditEntry{
		ID:        p.ID,
		Timestamp: time.Now(),
		Payload:   payload,
		// PrevHash, Signature, Proof will be set by Append
	}

	// Append to audit
	if err := k.AuditStore.Append(entry); err != nil {
		return fmt.Errorf("failed to append to audit: %w", err)
	}

	// Trigger CourtEngine.RunReview
	approved, reason := k.CourtEngine.RunReview(p)
	if !approved {
		fmt.Printf("Proposal %s rejected: %s\n", p.ID, reason)
		return nil // or error?
	}

	fmt.Printf("Proposal %s approved: %s\n", p.ID, reason)

	// Based on mode: auto → wait for vote, etc.
	// On approval: ApplyMutation, log success
	// Stub: always approve
	if p.Diff != nil {
		if err := k.ApplyMutation(p.Diff); err != nil {
			return fmt.Errorf("failed to apply mutation: %w", err)
		}
	}
	fmt.Printf("Proposal %s approved and applied\n", p.ID)
	return nil
}

func (k *Kernel) MediateAction(action interface{}) (interface{}, error) {
	actionStr := fmt.Sprintf("%v", action)
	// Check constitution rules
	if err := constitution.Enforce(1, actionStr); err != nil {
		return nil, fmt.Errorf("action denied: %w", err)
	}
	// If allowed: proxy to sandbox / external
	// For now, simulate
	result := "mediated: " + actionStr
	// Log every mediated call
	fmt.Println("Mediated action:", actionStr)
	return result, nil
}

func (k *Kernel) RegisterApprovedTool(tool agent.Tool) error {
	k.ApprovedTools[tool.Name()] = tool
	return nil
}

func (k *Kernel) Bootstrap() error {
	// Generate Ed25519 key pair
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return err
	}
	k.PubKey = pub

	// Compute SHA-256 of binary
	binPath := os.Args[0]
	binData, err := os.ReadFile(binPath)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(binData)

	// Sign the hash
	signature := ed25519.Sign(priv, hash[:])

	// Store signature
	k.StoredSignature = signature

	// Store signature and pubkey hash (placeholder: print for now)
	pubHash := sha256.Sum256(pub)
	fmt.Printf("Kernel bootstrapped. PubKey hash: %x, Signature: %x\n", pubHash, signature)

	return nil
}

func (k *Kernel) VerifySelf() error {
	// On startup, verify
	binPath := os.Args[0]
	binData, err := os.ReadFile(binPath)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(binData)

	if !ed25519.Verify(k.PubKey, hash[:], k.StoredSignature) {
		return fmt.Errorf("self-signature verification failed")
	}
	return nil
}

func (k *Kernel) ApplyMutation(diff jsonpatch.Patch) error {
	// Placeholder: apply to config
	configData, err := toml.Marshal(k.Config)
	if err != nil {
		return err
	}
	patched, err := diff.Apply(configData)
	if err != nil {
		return err
	}
	return toml.Unmarshal(patched, k.Config)
}

func (k *Kernel) Rollback(mutationID string) error {
	// Stub: revert last mutation
	// In real, find audit entry, reverse diff
	fmt.Println("Rolling back mutation:", mutationID)
	return nil
}
