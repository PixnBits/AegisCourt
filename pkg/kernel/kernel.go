package kernel

import (
	"crypto/ed25519"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/PixnBits/AegisCourt/pkg/agent"
	"github.com/PixnBits/AegisCourt/pkg/audit"
	"github.com/PixnBits/AegisCourt/pkg/config"
	"github.com/PixnBits/AegisCourt/pkg/court"
	"github.com/PixnBits/AegisCourt/pkg/sandbox"
	jsonpatch "github.com/evanphx/json-patch"
)

//go:embed ../../constitution/initial_rules_v0.1.md
var constitution string

type Kernel struct {
	Config       *config.Profile
	SandboxMgr   *sandbox.Manager
	LLMRouter    interface{} // stub
	CourtEngine  interface{} // stub
	AuditStore   *audit.Store
	AgentRuntime interface{} // stub
	Constitution interface{} // stub	ApprovedTools map[string]Tool	mu          sync.RWMutex
	halted       bool
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
	// Load config, constitution, verify self-signature
	// Initialize sub-components
	// Start background goroutines
	// Enter main listen loop
	fmt.Println("Kernel started")
	return nil
}

func (k *Kernel) HandleProposal(p court.Proposal) error {
	// Append to audit
	// Trigger CourtEngine.RunReview
	// Based on mode: auto → wait for vote, etc.
	// On approval: ApplyMutation, log success
	fmt.Println("Handling proposal:", p.ID)
	return nil
}

func (k *Kernel) MediateAction(action interface{}) (interface{}, error) {
	// Check constitution rules
	// If allowed: proxy to sandbox / external
	// Log every mediated call
	fmt.Println("Mediating action")
	return nil, nil
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

	// Placeholder: assume signature is stored, but for now, skip verification
	// In real, load stored signature and pubkey, verify
	if !ed25519.Verify(k.PubKey, hash[:], []byte("placeholder")) {
		panic("Self-signature verification failed")
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
