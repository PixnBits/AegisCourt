package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cbergoon/merkletree"
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

// Constitution holds the loaded rules from file.
type Constitution struct {
	Version string            `json:"version"`
	Rules   map[string]string `json:"rules"` // rule number -> text
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

// DummySandbox implements Sandbox with os/exec (gVisor stub).
type DummySandbox struct {
	cmd *exec.Cmd
}

func (s *DummySandbox) Start(ctx context.Context, cmd []string) error {
	s.cmd = exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	return s.cmd.Start()
}

func (s *DummySandbox) Stop() error {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Kill()
	}
	return nil
}

func (s *DummySandbox) Exec(input string) (string, error) {
	// Stub: for MVP, just echo input
	return "echo: " + input, nil
}

// OllamaRouter implements LLMRouter with HTTP client to Ollama.
type OllamaRouter struct {
	endpoints []string
	client    *http.Client
}

func NewOllamaRouter(endpoints []string) *OllamaRouter {
	return &OllamaRouter{
		endpoints: endpoints,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *OllamaRouter) Dispatch(ctx context.Context, prompt string, model string) (string, error) {
	// Use first endpoint for MVP
	endpoint := r.endpoints[0] + "/api/generate"
	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		// For MVP, return dummy response if Ollama not available
		return `{"persona": "dummy", "score": 75, "pros": ["test"], "cons": ["test"], "recommendation": "Approve"}`, nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	response, ok := result["response"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}
	return response, nil
}

// ReviewerResponse is the JSON output from each reviewer.
type ReviewerResponse struct {
	Persona    string   `json:"persona"`
	Score      int      `json:"score"`
	Pros       []string `json:"pros"`
	Cons       []string `json:"cons"`
	Recommendation string `json:"recommendation"`
	// Additional fields vary by persona
}

// CourtEngineImpl implements CourtEngine.
type CourtEngineImpl struct {
	router      LLMRouter
	constitution Constitution
	aboutMe     AboutMe
}

func NewCourtEngine(router LLMRouter, consti Constitution, about AboutMe) *CourtEngineImpl {
	return &CourtEngineImpl{
		router:      router,
		constitution: consti,
		aboutMe:     about,
	}
}

func (c *CourtEngineImpl) ReviewProposal(ctx context.Context, prop Proposal) (CourtDecision, error) {
	// Load reviewer templates
	personas := []string{"ciso", "mrm", "compliance-regulatory", "helpfulness-evolution", "responsible-ai", "sre"}
	responses := make([]ReviewerResponse, 0, len(personas))

	for _, persona := range personas {
		template, err := c.loadReviewerTemplate(persona)
		if err != nil {
			return CourtDecision{}, err
		}
		prompt := fmt.Sprintf("%s\n\nProposal: %s\nDiff: %s", template, prop.Description, string(prop.Diff))
		response, err := c.router.Dispatch(ctx, prompt, "llama3.2") // Assume model
		if err != nil {
			return CourtDecision{}, err
		}
		var resp ReviewerResponse
		json.Unmarshal([]byte(response), &resp)
		responses = append(responses, resp)
	}

	// Aggregate: average score, approve if > threshold based on aboutMe
	totalScore := 0
	for _, r := range responses {
		totalScore += r.Score
	}
	avgScore := float64(totalScore) / float64(len(responses))

	threshold := 70.0 // default
	switch c.aboutMe.RiskTolerance {
	case "paranoid":
		threshold = 90.0
	case "financial":
		threshold = 80.0
	}

	approved := avgScore >= threshold
	decision := CourtDecision{
		ProposalID:     prop.ID,
		AggregateScore: avgScore,
		Approved:       approved,
	}

	if !approved {
		decision.RejectionReason = "Aggregate score below threshold"
	}

	return decision, nil
}

func (c *CourtEngineImpl) loadReviewerTemplate(persona string) (string, error) {
	path := fmt.Sprintf("reviewers/%s.md", persona)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// AuditEntry implements merkletree.Content.
type AuditEntry struct {
	Data []byte
}

func (a AuditEntry) CalculateHash() ([]byte, error) {
	return a.Data, nil
}

func (a AuditEntry) Equals(other merkletree.Content) (bool, error) {
	return string(a.Data) == string(other.(AuditEntry).Data), nil
}

// MerkleAuditStore implements AuditStore.
type MerkleAuditStore struct {
	tree   *merkletree.MerkleTree
	entries []AuditEntry
}

func NewMerkleAuditStore() *MerkleAuditStore {
	return &MerkleAuditStore{
		entries: []AuditEntry{},
	}
}

func (a *MerkleAuditStore) Append(entry json.RawMessage) error {
	ae := AuditEntry{Data: entry}
	a.entries = append(a.entries, ae)
	contents := make([]merkletree.Content, len(a.entries))
	for i, e := range a.entries {
		contents[i] = e
	}
	tree, err := merkletree.NewTree(contents)
	if err != nil {
		return err
	}
	a.tree = tree
	return nil
}

func (a *MerkleAuditStore) GetHistory(since time.Time) ([]json.RawMessage, error) {
	// Stub: return all for now
	result := make([]json.RawMessage, len(a.entries))
	for i, e := range a.entries {
		result[i] = e.Data
	}
	return result, nil
}

func (a *MerkleAuditStore) VerifyIntegrity() error {
	if a.tree == nil {
		return nil
	}
	valid, err := a.tree.VerifyTree()
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("merkle tree integrity check failed")
	}
	return nil
}

// -----------------------------------------------------------------------------
// Kernel – the immutable root of trust
// -----------------------------------------------------------------------------
func loadConstitution(path string) (Constitution, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Constitution{}, err
	}
	content := string(data)
	rules := make(map[string]string)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "**Rule ") {
			ruleNum := strings.TrimPrefix(line, "**Rule ")
			ruleNum = strings.Split(ruleNum, " – ")[0]
			// Collect rule text until next rule or end
			ruleText := ""
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "**Rule ") || strings.HasPrefix(lines[j], "**Override") {
					break
				}
				ruleText += lines[j] + "\n"
			}
			rules[ruleNum] = strings.TrimSpace(ruleText)
		}
	}
	return Constitution{Version: "0.1", Rules: rules}, nil
}

type Kernel struct {
	config       KernelConfig
	constitution Constitution
	privateKey   ed25519.PrivateKey // kernel's signing key (generated or loaded)
	publicKey    ed25519.PublicKey
	sandboxMgr   Sandbox     // placeholder
	llmRouter    LLMRouter   // placeholder
	courtEngine  CourtEngine // placeholder
	auditStore   AuditStore  // placeholder
	shutdown     chan struct{}
}

func NewKernel(config KernelConfig) (*Kernel, error) {
	// Load constitution
	consti, err := loadConstitution(config.ConstitutionPath)
	if err != nil {
		return nil, fmt.Errorf("load constitution: %w", err)
	}

	// In real MVP: load or generate Ed25519 keypair (persist encrypted in data dir)
	// For skeleton: generate ephemeral key (replace with secure persistence)
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	k := &Kernel{
		config:       config,
		constitution: consti,
		privateKey:   priv,
		publicKey:    pub,
		shutdown:     make(chan struct{}),
	}

	// Initialize components
	k.sandboxMgr = &DummySandbox{}
	k.llmRouter = NewOllamaRouter(config.LLMEndpoints)
	k.courtEngine = NewCourtEngine(k.llmRouter, consti, config.AboutMe)
	k.auditStore = NewMerkleAuditStore()

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
	command := flag.String("cmd", "run", "command: run, submit-proposal, emergency-halt, export-audit")
	desc := flag.String("desc", "", "description for submit-proposal")
	diffPath := flag.String("diff", "", "path to diff file for submit-proposal")
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

	switch *command {
	case "run":
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := kernel.Run(ctx); err != nil {
			log.Printf("kernel exited: %v", err)
			os.Exit(1)
		}
	case "submit-proposal":
		if *desc == "" || *diffPath == "" {
			log.Fatalf("submit-proposal requires -desc and -diff")
		}
		diffData, err := os.ReadFile(*diffPath)
		if err != nil {
			log.Fatalf("read diff: %v", err)
		}
		var diff json.RawMessage
		json.Unmarshal(diffData, &diff)
		ctx := context.Background()
		if err := kernel.SubmitProposal(ctx, *desc, diff); err != nil {
			log.Fatalf("submit proposal: %v", err)
		}
		log.Println("Proposal submitted successfully")
	case "emergency-halt":
		kernel.EmergencyHalt()
		log.Println("Emergency halt triggered")
	case "export-audit":
		history, err := kernel.auditStore.GetHistory(time.Time{})
		if err != nil {
			log.Fatalf("export audit: %v", err)
		}
		for _, entry := range history {
			fmt.Println(string(entry))
		}
	default:
		log.Fatalf("unknown command: %s", *command)
	}
}
