package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Domain Types & Interfaces
// -----------------------------------------------------------------------------

// Proposal represents a self-modification request from an agent.
type Proposal struct {
	ID          string          `json:"id"`
	Timestamp   time.Time       `json:"timestamp"`
	Diff        json.RawMessage `json:"diff"`        // JSON Patch or custom delta
	Description string          `json:"description"` // Human-readable intent
	Proposer    string          `json:"proposer"`    // Agent ID or "kernel"
}

// CourtDecision is the aggregated result from the Governance Court.
type CourtDecision struct {
	ProposalID      string   `json:"proposal_id"`
	AggregateScore  float64  `json:"aggregate_score"`
	Approved        bool     `json:"approved"`
	Conditions      []string `json:"conditions,omitempty"`
	RejectionReason string   `json:"rejection_reason,omitempty"`
}

// KernelConfig holds runtime configuration (loaded from file + env).
type KernelConfig struct {
	DataDir          string   `json:"data_dir"`
	ConstitutionPath string   `json:"constitution_path"`
	LLMEndpoints     []string `json:"llm_endpoints"` // e.g. ["http://localhost:11434", "https://api.groq.com"]
	AboutMe          AboutMe  `json:"about_me"`
	Debug            bool     `json:"debug"`
}

// AboutMe captures user risk profile (calibrates Court strictness).
type AboutMe struct {
	RiskTolerance string `json:"risk_tolerance"` // "hobbyist", "financial", "paranoid"
	UseCase       string `json:"use_case"`       // "personal", "research", "production"
	MaxDeferHours int    `json:"max_defer_hours"`
}

// Sandbox interface (to be implemented with gVisor, seccomp fallback, etc.)
type Sandbox interface {
	Start(ctx context.Context, cmd []string) error
	Stop() error
	Exec(input string) (output string, err error)
}

// LLMRouter routes prompts to selected models.
type LLMRouter interface {
	Dispatch(ctx context.Context, prompt string, model string) (string, error)
}

// CourtEngine orchestrates reviewers and aggregates decisions.
type CourtEngine interface {
	ReviewProposal(ctx context.Context, prop Proposal) (CourtDecision, error)
}

// AuditStore provides tamper-evident logging.
type AuditStore interface {
	Append(entry json.RawMessage) error
	GetHistory(since time.Time) ([]json.RawMessage, error)
	VerifyIntegrity() error
}

// -----------------------------------------------------------------------------
// Kernel – the immutable root of trust
// -----------------------------------------------------------------------------
type Kernel struct {
	config      KernelConfig
	privateKey  ed25519.PrivateKey // kernel's signing key (generated or loaded)
	publicKey   ed25519.PublicKey
	sandboxMgr  Sandbox     // placeholder
	llmRouter   LLMRouter   // placeholder
	courtEngine CourtEngine // placeholder
	auditStore  AuditStore  // placeholder
	shutdown    chan struct{}
}

func NewKernel(config KernelConfig) (*Kernel, error) {
	// In real MVP: load or generate Ed25519 keypair (persist encrypted in data dir)
	// For skeleton: generate ephemeral key (replace with secure persistence)
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	k := &Kernel{
		config:     config,
		privateKey: priv,
		publicKey:  pub,
		shutdown:   make(chan struct{}),
	}

	// TODO: initialize real components (gVisor, LLM router, BadgerDB audit, etc.)
	// k.sandboxMgr = gvisor.NewManager(...)
	// k.llmRouter = ollama.NewRouter(config.LLMEndpoints...)
	// k.courtEngine = court.NewEngine(k.llmRouter, constitutionPath)
	// k.auditStore = audit.NewMerkleStore(filepath.Join(config.DataDir, "audit.db"))

	return k, nil
}

// Sign signs arbitrary data with the kernel's private key.
func (k *Kernel) Sign(data []byte) ([]byte, error) {
	return ed25519.Sign(k.privateKey, data), nil
}

// VerifySignature verifies data against the kernel's public key.
func (k *Kernel) VerifySignature(data, sig []byte) bool {
	return ed25519.Verify(k.publicKey, data, sig)
}

// SubmitProposal is the entry point for agent-proposed changes.
func (k *Kernel) SubmitProposal(ctx context.Context, desc string, diff json.RawMessage) error {
	prop := Proposal{
		ID:          uuid.New().String(),
		Timestamp:   time.Now().UTC(),
		Diff:        diff,
		Description: desc,
		Proposer:    "agent-runtime", // TODO: real agent ID
	}

	propBytes, err := json.Marshal(prop)
	if err != nil {
		return err
	}

	// Immediate audit log (before any processing)
	if err := k.auditStore.Append(propBytes); err != nil {
		return fmt.Errorf("audit append failed: %w", err)
	}

	// Activate Governance Court
	decision, err := k.courtEngine.ReviewProposal(ctx, prop)
	if err != nil {
		return err
	}

	if !decision.Approved {
		log.Printf("Proposal %s rejected: %s", prop.ID, decision.RejectionReason)
		return fmt.Errorf("court rejected: %s", decision.RejectionReason)
	}

	// TODO: apply the diff atomically (patch constitution, tools, memory schema)
	// - Sign the applied diff
	// - Append signed application to audit log
	// - Restart affected sandboxes with new config

	log.Printf("Proposal %s approved with conditions: %v", prop.ID, decision.Conditions)
	return nil
}

// Run starts the kernel main loop (proposal listener, health checks, etc.)
func (k *Kernel) Run(ctx context.Context) error {
	log.Printf("AegisCourt kernel starting – paranoid mode always on")

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Println("Shutdown signal received")
	case <-k.shutdown:
		log.Println("Emergency halt triggered")
	case <-ctx.Done():
		log.Println("Context cancelled")
	}

	// TODO: graceful shutdown – stop sandboxes, flush audit log, etc.
	return nil
}

// EmergencyHalt immediately stops everything and enters read-only mode.
func (k *Kernel) EmergencyHalt() {
	close(k.shutdown)
	// TODO: kill all sandboxes, freeze state
}

// -----------------------------------------------------------------------------
// Main entry point
// -----------------------------------------------------------------------------
func main() {
	configPath := flag.String("config", "config.json", "path to config file")
	flag.Parse()

	data, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("read config: %v", err)
	}

	var cfg KernelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	kernel, err := NewKernel(cfg)
	if err != nil {
		log.Fatalf("init kernel: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := kernel.Run(ctx); err != nil {
		log.Printf("kernel exited: %v", err)
		os.Exit(1)
	}
}
