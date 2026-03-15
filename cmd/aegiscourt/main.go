package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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

// DummyCourtEngine implements CourtEngine with hardcoded dummy reviewers.
type DummyCourtEngine struct{}

func (e *DummyCourtEngine) ReviewProposal(ctx context.Context, prop Proposal) (CourtDecision, error) {
	// Simulate 6 reviewers
	personas := []string{"ciso", "mrm", "compliance", "ethics", "sre", "helpfulness"}
	totalScore := 0
	for range personas {
		// Random score 60-98
		score := 60 + (int(uuid.New().ID()) % 39)
		totalScore += score
	}
	avgScore := float64(totalScore) / float64(len(personas))
	approved := avgScore > 70.0

	decision := CourtDecision{
		ProposalID:     prop.ID,
		AggregateScore: avgScore,
		Approved:       approved,
	}
	if approved {
		decision.Conditions = []string{"Monitor for side effects", "Log all executions"}
	} else {
		decision.RejectionReason = "Insufficient aggregate score"
	}

	return decision, nil
}

// AuditStore provides tamper-evident logging.
type AuditStore interface {
	Append(entry json.RawMessage) error
	GetHistory(since time.Time) ([]json.RawMessage, error)
	VerifyIntegrity() error
}

// GvisorSandbox implements Sandbox using Docker with gVisor runtime.
type GvisorSandbox struct {
	containerID string
	running     bool
}

func (s *GvisorSandbox) Start(ctx context.Context, cmd []string) error {
	// Use Docker with gvisor runtime to run the command in a sandbox
	// Assume a base image like alpine for running arbitrary commands
	image := "alpine:latest"
	dockerCmd := []string{"docker", "run", "--runtime=runsc", "--rm", "-d", "--name", "aegis-sandbox-" + uuid.New().String()[:8], image}
	dockerCmd = append(dockerCmd, cmd...)
	
	execCmd := exec.CommandContext(ctx, dockerCmd[0], dockerCmd[1:]...)
	output, err := execCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to start gvisor sandbox: %w", err)
	}
	s.containerID = strings.TrimSpace(string(output))
	s.running = true
	return nil
}

func (s *GvisorSandbox) Stop() error {
	if !s.running || s.containerID == "" {
		return nil
	}
	cmd := exec.Command("docker", "stop", s.containerID)
	err := cmd.Run()
	s.running = false
	s.containerID = ""
	return err
}

func (s *GvisorSandbox) Exec(input string) (string, error) {
	if !s.running {
		return "", fmt.Errorf("sandbox not running")
	}
	// For simplicity, assume the command is running and we can exec into it
	// But since it's run with cmd, and -d, it's detached.
	// To exec, we need to docker exec
	// But for MVP, since Start runs the cmd, Exec can be used to send input if interactive, but here it's stub.
	// For full, perhaps make Start run a shell, and Exec sends commands to it.
	// But complex. For now, return stub.
	return "gvisor-exec: " + input, nil
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
type MerkleAuditEntry struct {
	Data []byte
}

func (a MerkleAuditEntry) CalculateHash() ([]byte, error) {
	return a.Data, nil
}

func (a MerkleAuditEntry) Equals(other merkletree.Content) (bool, error) {
	return string(a.Data) == string(other.(MerkleAuditEntry).Data), nil
}

// MerkleAuditStore implements AuditStore.
type MerkleAuditStore struct {
	tree   *merkletree.MerkleTree
	entries []MerkleAuditEntry
}

func NewMerkleAuditStore() *MerkleAuditStore {
	return &MerkleAuditStore{
		entries: []MerkleAuditEntry{},
	}
}

func (a *MerkleAuditStore) Append(entry json.RawMessage) error {
	ae := MerkleAuditEntry{Data: entry}
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

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	PrevHash    string          `json:"prev_hash"`
	PayloadHash string          `json:"payload_hash"`
	Data        json.RawMessage `json:"data"`
	Sig         string          `json:"sig"`
}

// FlatFileAuditStore implements AuditStore using flat files with signatures.
type FlatFileAuditStore struct {
	filePath  string
	prevHash  string
}

func NewFlatFileAuditStore(filePath string) *FlatFileAuditStore {
	return &FlatFileAuditStore{
		filePath: filePath,
		prevHash: "",
	}
}

func (a *FlatFileAuditStore) Append(entry json.RawMessage) error {
	payloadHash := sha256.Sum256(entry)
	payloadHashStr := hex.EncodeToString(payloadHash[:])

	auditEntry := AuditEntry{
		PrevHash:    a.prevHash,
		PayloadHash: payloadHashStr,
		Data:        entry,
		Sig:         "", // TODO: sign with kernel key
	}

	auditBytes, err := json.Marshal(auditEntry)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(a.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(string(auditBytes) + "\n"); err != nil {
		return err
	}

	// Update prevHash
	entryHash := sha256.Sum256(auditBytes)
	a.prevHash = hex.EncodeToString(entryHash[:])

	return nil
}

func (a *FlatFileAuditStore) GetHistory(since time.Time) ([]json.RawMessage, error) {
	data, err := os.ReadFile(a.filePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var history []json.RawMessage
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry AuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}
		history = append(history, entry.Data)
	}
	return history, nil
}

func (a *FlatFileAuditStore) VerifyIntegrity() error {
	data, err := os.ReadFile(a.filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	expectedPrev := ""
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry AuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return err
		}
		if entry.PrevHash != expectedPrev {
			return fmt.Errorf("chain broken at entry")
		}
		// Verify payload hash
		payloadHash := sha256.Sum256(entry.Data)
		if hex.EncodeToString(payloadHash[:]) != entry.PayloadHash {
			return fmt.Errorf("payload hash mismatch")
		}
		// Update expectedPrev
		entryBytes, _ := json.Marshal(entry)
		entryHash := sha256.Sum256(entryBytes)
		expectedPrev = hex.EncodeToString(entryHash[:])
	}
	return nil
}

// -----------------------------------------------------------------------------
// Kernel – the immutable root of trust
// -----------------------------------------------------------------------------
func loadConfig(path string) (KernelConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return KernelConfig{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg KernelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return KernelConfig{}, fmt.Errorf("parse config: %w", err)
	}

	// Apply defaults
	if cfg.DataDir == "" {
		cfg.DataDir = "./aegis-data"
	}
	if cfg.ConstitutionPath == "" {
		cfg.ConstitutionPath = "./docs/constitution.md"
	}
	if len(cfg.LLMEndpoints) == 0 {
		cfg.LLMEndpoints = []string{"http://localhost:11434"}
	}
	if cfg.AboutMe.RiskTolerance == "" {
		cfg.AboutMe.RiskTolerance = "hobbyist"
	}
	if cfg.AboutMe.UseCase == "" {
		cfg.AboutMe.UseCase = "personal"
	}
	if cfg.AboutMe.MaxDeferHours == 0 {
		cfg.AboutMe.MaxDeferHours = 24
	}

	// Validate required fields
	if cfg.DataDir == "" || cfg.ConstitutionPath == "" {
		return KernelConfig{}, fmt.Errorf("data_dir and constitution_path are required")
	}

	return cfg, nil
}

func loadConstitution(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type Kernel struct {
	config       KernelConfig
	constitution string
	privateKey   ed25519.PrivateKey // kernel's signing key (generated or loaded)
	publicKey    ed25519.PublicKey
	sandboxMgr   Sandbox     // placeholder
	llmRouter    LLMRouter   // placeholder
	courtEngine  CourtEngine // placeholder
	auditStore   AuditStore  // placeholder
	shutdown     chan struct{}
	currentState map[string]interface{} // dummy state
}

func NewKernel(configPath string) (*Kernel, error) {
	// Resolve config path to absolute so relative paths inside the
	// config file are interpreted relative to the config file location.
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("abs config path: %w", err)
	}

	cfg, err := loadConfig(absConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Resolve relative paths in config (data_dir, constitution_path)
	// relative to the directory containing the config file.
	configDir := filepath.Dir(absConfigPath)
	if !filepath.IsAbs(cfg.DataDir) {
		cfg.DataDir = filepath.Clean(filepath.Join(configDir, cfg.DataDir))
	}
	if !filepath.IsAbs(cfg.ConstitutionPath) {
		cfg.ConstitutionPath = filepath.Clean(filepath.Join(configDir, cfg.ConstitutionPath))
	}

	// Create data directory
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Load constitution
	consti, err := loadConstitution(cfg.ConstitutionPath)
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
		config:       cfg,
		constitution: consti,
		privateKey:   priv,
		publicKey:    pub,
		shutdown:     make(chan struct{}),
		currentState: make(map[string]interface{}),
	}

	// Initialize components
	k.sandboxMgr = &GvisorSandbox{}
	k.llmRouter = NewOllamaRouter(cfg.LLMEndpoints)
	k.courtEngine = &DummyCourtEngine{}
	k.auditStore = NewFlatFileAuditStore(filepath.Join(cfg.DataDir, "audit.log"))

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

	// Apply the approved proposal
	if err := k.ApplyApproved(decision, prop); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	log.Printf("Proposal %s approved with conditions: %v", prop.ID, decision.Conditions)
	return nil
}

// ApplyApproved applies the approved proposal.
func (k *Kernel) ApplyApproved(decision CourtDecision, prop Proposal) error {
	// Save current state as backup
	statePath := filepath.Join(k.config.DataDir, "current_state.json")
	backupPath := filepath.Join(k.config.DataDir, "versions", prop.ID+".json")
	os.MkdirAll(filepath.Dir(backupPath), 0700)
	if data, err := json.Marshal(k.currentState); err == nil {
		os.WriteFile(backupPath, data, 0600)
		os.WriteFile(statePath, data, 0600) // also update current
	}

	// "Apply" diff (stub: just log)
	log.Printf("Applying diff: %s", string(prop.Diff))

	// Sign new state
	stateBytes, _ := json.Marshal(k.currentState)
	sig, err := k.Sign(stateBytes)
	if err != nil {
		return err
	}
	appliedEntry := map[string]interface{}{
		"type":       "applied",
		"proposal_id": prop.ID,
		"state":      k.currentState,
		"sig":        sig,
	}
	appliedBytes, _ := json.Marshal(appliedEntry)
	return k.auditStore.Append(appliedBytes)
}

// Rollback reverts to a previous state.
func (k *Kernel) Rollback(proposalID string) error {
	versionsDir := filepath.Join(k.config.DataDir, "versions")
	_, err := os.ReadDir(versionsDir)
	if err != nil {
		return err
	}
	// Find the latest backup before this ID (stub: just use the one with ID)
	backupPath := filepath.Join(versionsDir, proposalID+".json")
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &k.currentState); err != nil {
		return err
	}
	statePath := filepath.Join(k.config.DataDir, "current_state.json")
	return os.WriteFile(statePath, data, 0600)
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
	command := flag.String("cmd", "run", "command: run, propose, submit-proposal, emergency-halt, halt, export-audit")
	desc := flag.String("desc", "", "description for submit-proposal")
	diffPath := flag.String("diff", "", "path to diff file for submit-proposal")
	// New flags for propose
	proposeDesc := flag.String("propose-desc", "", "description for propose command")
	proposeDiff := flag.String("propose-diff", "", "diff JSON for propose command")
	flag.Parse()

	kernel, err := NewKernel(*configPath)
	if err != nil {
		log.Fatalf("init kernel: %v", err)
	}

	// Print kernel public key fingerprint
	fingerprint := fmt.Sprintf("%x", kernel.publicKey[:8]) // first 8 bytes as hex
	log.Printf("Kernel public key fingerprint: %s", fingerprint)

	switch *command {
	case "run":
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := kernel.Run(ctx); err != nil {
			log.Printf("kernel exited: %v", err)
			os.Exit(1)
		}
	case "propose":
		if *proposeDesc == "" || *proposeDiff == "" {
			log.Fatalf("propose requires -propose-desc and -propose-diff")
		}
		prop := Proposal{
			ID:          uuid.New().String(),
			Timestamp:   time.Now().UTC(),
			Diff:        json.RawMessage(*proposeDiff),
			Description: *proposeDesc,
			Proposer:    "cli-user",
		}
		propBytes, err := json.Marshal(prop)
		if err != nil {
			log.Fatalf("marshal proposal: %v", err)
		}
		// Stub: append to audit
		if err := kernel.auditStore.Append(propBytes); err != nil {
			log.Fatalf("audit append: %v", err)
		}
		fmt.Printf("Proposal submitted: %s\n", prop.ID)
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
	case "halt":
		kernel.EmergencyHalt()
		log.Println("Halt command executed")
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
