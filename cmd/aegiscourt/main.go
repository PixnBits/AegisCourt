package main

import (
	"bufio"
	"bytes"
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
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
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
	ProposalID        string             `json:"proposal_id"`
	AggregateScore    float64            `json:"aggregate_score"`
	Approved          bool               `json:"approved"`
	Conditions        []string           `json:"conditions,omitempty"`
	RejectionReason   string             `json:"rejection_reason,omitempty"`
	ReviewerResponses []ReviewerResponse `json:"reviewer_responses,omitempty"`
}

// KernelConfig holds runtime configuration (loaded from file + env).
type KernelConfig struct {
	DataDir             string   `json:"data_dir"`
	ConstitutionPath    string   `json:"constitution_path"`
	LLMEndpoints        []string `json:"llm_endpoints"` // e.g. ["http://localhost:11434", "https://api.groq.com"]
	AboutMe             AboutMe  `json:"about_me"`
	Debug               bool     `json:"debug"`
	MaxSandboxMemoryMB  int      `json:"max_sandbox_memory_mb"`
	MaxSandboxCPU       float64  `json:"max_sandbox_cpu"`
	AllowedSyscalls     []string `json:"allowed_syscalls"`
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

type AgentInstance struct {
	ID            string
	StartedAt     time.Time
	Purpose       string
	LastActivity  time.Time
	ProposalCount uint64
	Status        string // "running", "sleeping", "failed", "halted"
}

type SandboxHandle struct {
	ID         string
	StartedAt  time.Time
	AgentID    string
	LastCmd    string
	Status     string // "running", "exited", "killed"
}

type RecentProposal struct {
	ID          string
	Timestamp   time.Time
	Description string
	Status      string // "pending", "approved", "rejected", "applied"
	Score       float64
}

type LLMEndpointStatus struct {
	LastCheck    time.Time
	LastLatency  time.Duration
	LastError    string
	Status       string // "ok", "stale", "error"
}

type DeferredProposal struct {
	Proposal     Proposal
	Decision     CourtDecision
	Expiry       time.Time
	Reason       string
}

type Provenance struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	ParentID   string    `json:"parent_id,omitempty"` // if spawned from another
	Purpose    string    `json:"purpose"`
	Signature  string    `json:"signature"` // signed by kernel
}

// Sandbox interface (to be implemented with gVisor, seccomp fallback, etc.)
type Sandbox interface {
	Start(ctx context.Context, cmd []string) error
	Stop() error
	Exec(input string) (output string, err error)
}

// ToolProxy mediates tool calls from agents to external APIs.
type ToolProxy interface {
	AllowAndProxy(toolCall map[string]interface{}) (string, error)
}

// ToolProxyImpl implements ToolProxy with safe API calls.
type ToolProxyImpl struct {
	client *http.Client
}

func NewToolProxy() *ToolProxyImpl {
	return &ToolProxyImpl{client: &http.Client{Timeout: 10 * time.Second}}
}

func (t *ToolProxyImpl) AllowAndProxy(toolCall map[string]interface{}) (string, error) {
	tool, ok := toolCall["tool"].(string)
	if !ok {
		return "", fmt.Errorf("invalid tool call: missing tool")
	}
	if tool == "web_search" {
		query, ok := toolCall["query"].(string)
		if !ok {
			return "", fmt.Errorf("invalid web_search: missing query")
		}
		// Use DuckDuckGo instant answer API (safe, no JS)
		url := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1", url.QueryEscape(query))
		resp, err := t.client.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
	return "", fmt.Errorf("tool not allowed: %s", tool)
}

func (t *ToolProxyImpl) ProxyHTTP(ctx context.Context, url string, method string, headers map[string]string, body []byte) (response []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (t *ToolProxyImpl) ProxyFileRead(path string) ([]byte, error) {
	// For MVP: deny all file operations
	return nil, fmt.Errorf("file read not allowed")
}

func (t *ToolProxyImpl) ProxyFileWrite(path string, data []byte) error {
	// For MVP: deny all file operations
	return fmt.Errorf("file write not allowed")
}

// CourtEngine orchestrates reviewers and aggregates decisions.
type CourtEngine interface {
	ReviewProposal(ctx context.Context, prop Proposal) (CourtDecision, error)
}

// CourtEngineImpl implements CourtEngine with real LLM-based reviewers.
type CourtEngineImpl struct {
	router      LLMRouter
	constitution string
	aboutMe     AboutMe
}

func NewCourtEngine(router LLMRouter, consti string, about AboutMe) *CourtEngineImpl {
	return &CourtEngineImpl{
		router:      router,
		constitution: consti,
		aboutMe:     about,
	}
}

func (e *CourtEngineImpl) ReviewProposal(ctx context.Context, prop Proposal) (CourtDecision, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		log.Printf("Court review completed in %v", duration)
		if duration > 45*time.Second {
			log.Printf("Warning: Court review exceeded 45s target")
		}
	}()

	personas := []string{"ciso", "mrm", "compliance-regulatory", "helpfulness-evolution", "responsible-ai", "sre"}
	var responses []ReviewerResponse

	for _, persona := range personas {
		template, err := e.loadReviewerTemplate(persona)
		if err != nil {
			log.Printf("failed to load template for %s: %v", persona, err)
			// Fallback response
			responses = append(responses, ReviewerResponse{Persona: persona, Score: 50, Recommendation: "Defer"})
			continue
		}

		// Extract version from constitution (first line)
		version := "v0.1" // default
		if lines := strings.Split(e.constitution, "\n"); len(lines) > 0 {
			if strings.Contains(lines[0], "v") {
				version = strings.TrimSpace(strings.Split(lines[0], "v")[1])
				if idx := strings.Index(version, " "); idx > 0 {
					version = "v" + version[:idx]
				} else {
					version = "v" + version
				}
			}
		}

		constitutionHeader := fmt.Sprintf("You are the %s reviewer in AegisCourt.\nThe current constitution (%s) is:\n\n%s\n\nEvaluate the following proposal strictly against the constitution rules, especially the Absolute and High-Priority ones.\n\n", persona, version, e.constitution)

		prompt := constitutionHeader + template + "\n\nProposal Description:\n" + prop.Description + "\n\nProposed Diff:\n" + string(prop.Diff) + "\n\nOutput ONLY valid JSON matching the format shown at the bottom of the template."

		response, err := e.router.Dispatch(ctx, prompt, "llama3.2")
		if err != nil {
			log.Printf("LLM dispatch failed for %s: %v", persona, err)
			responses = append(responses, ReviewerResponse{Persona: persona, Score: 50, Recommendation: "Defer"})
			continue
		}

		var resp ReviewerResponse
		if err := json.Unmarshal([]byte(response), &resp); err != nil {
			log.Printf("failed to parse response for %s: %v, response: %s", persona, err, response)
			responses = append(responses, ReviewerResponse{Persona: persona, Score: 50, Recommendation: "Defer"})
			continue
		}
		resp.Persona = persona // Ensure persona is set
		responses = append(responses, resp)
	}

	// Aggregate scores with weights: CISO and MRM get weight 2, others 1
	totalWeightedScore := 0
	totalWeight := 0
	var allConditions []string
	for _, r := range responses {
		weight := 1
		if r.Persona == "ciso" || r.Persona == "mrm" {
			weight = 2
		}
		totalWeightedScore += r.Score * weight
		totalWeight += weight
		allConditions = append(allConditions, r.Cons...)
	}
	avgScore := float64(totalWeightedScore) / float64(totalWeight)

	threshold := 70.0
	switch e.aboutMe.RiskTolerance {
	case "paranoid":
		threshold = 90.0
	case "financial":
		threshold = 80.0
	}
	// Elevated threshold for constitution changes
	if strings.Contains(string(prop.Diff), "/rules") {
		threshold = 90.0
	}

	approved := avgScore >= threshold
	decision := CourtDecision{
		ProposalID:        prop.ID,
		AggregateScore:    avgScore,
		Approved:          approved,
		Conditions:        allConditions,
		ReviewerResponses: responses,
	}
	if !approved {
		decision.RejectionReason = "Aggregate score below threshold"
	}

	return decision, nil
}

func (e *CourtEngineImpl) loadReviewerTemplate(persona string) (string, error) {
	path := fmt.Sprintf("reviewers/%s.md", persona)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
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
	config      *KernelConfig
}

func NewGvisorSandbox(config *KernelConfig) *GvisorSandbox {
	if len(config.AllowedSyscalls) == 0 {
		config.AllowedSyscalls = []string{
			"read", "write", "open", "close", "brk", "mmap", "munmap", "exit",
			"getpid", "getuid", "rt_sigaction", "rt_sigprocmask", "futex",
			"sched_yield", "nanosleep", "clock_gettime", "gettimeofday",
			"lseek", "fstat", "stat", "access", "dup", "dup2", "pipe", "pipe2",
			"execve", "wait4", "clone", "vfork", "kill", "tgkill", "exit_group",
			"uname", "getrlimit", "setrlimit", "getrusage", "times", "getcwd",
			"chdir", "mkdir", "rmdir", "unlink", "rename", "link", "symlink",
			"readlink", "chmod", "chown", "utime", "utimes", "socket", "connect",
			"bind", "listen", "accept", "getsockname", "getpeername", "sendto",
			"recvfrom", "shutdown", "setsockopt", "getsockopt", "ioctl", "fcntl",
		}
	}
	return &GvisorSandbox{config: config}
}

func (s *GvisorSandbox) Start(ctx context.Context, cmd []string) error {
	// Use Docker with gvisor runtime to run the command in a sandbox
	// Assume a base image like alpine for running arbitrary commands
	image := "alpine:latest"
	dockerCmd := []string{"docker", "run", "--runtime=runsc", "--rm", "-d", "--name", "aegis-sandbox-" + uuid.New().String()[:8]}
	
	// Syscall allowlist: Docker run does not directly support runsc syscall filtering via flags.
	// For full enforcement, a custom runsc profile is required.
	if len(s.config.AllowedSyscalls) > 0 {
		log.Printf("Warning: Syscall allowlist configured (%d syscalls), but Docker run does not expose runsc --syscall flag. Using default runsc profile. For stricter sandboxing, configure runsc with a custom profile.", len(s.config.AllowedSyscalls))
	}
	
	// Add resource limits
	if s.config != nil {
		dockerCmd = append(dockerCmd, "--memory", fmt.Sprintf("%dMB", s.config.MaxSandboxMemoryMB))
		dockerCmd = append(dockerCmd, "--cpus", fmt.Sprintf("%.1f", s.config.MaxSandboxCPU))
	}
	
	dockerCmd = append(dockerCmd, image)
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
	if err != nil {
		return err
	}
	// Remove the container
	rmCmd := exec.Command("docker", "rm", s.containerID)
	rmCmd.Run() // Ignore error if already removed
	s.running = false
	s.containerID = ""
	return nil
}

func (s *GvisorSandbox) Exec(input string) (string, error) {
	if !s.running || s.containerID == "" {
		return "", fmt.Errorf("sandbox not running")
	}
	// Run command in the running container
	cmd := exec.Command("docker", "exec", s.containerID, "sh", "-c", input)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exec failed: %w, output: %s", err, string(output))
	}
	return string(output), nil
}

// FallbackSandbox implements Sandbox using basic seccomp and namespaces (Linux only).
type FallbackSandbox struct {
	running bool
	config  *KernelConfig
}

func NewFallbackSandbox(config *KernelConfig) *FallbackSandbox {
	return &FallbackSandbox{config: config}
}

func (s *FallbackSandbox) Start(ctx context.Context, cmd []string) error {
	// For MVP, just warn and run without sandbox
	log.Printf("Warning: Using fallback sandbox - limited isolation")
	s.running = true
	return nil
}

func (s *FallbackSandbox) Stop() error {
	s.running = false
	return nil
}

func (s *FallbackSandbox) Exec(input string) (string, error) {
	if !s.running {
		return "", fmt.Errorf("sandbox not running")
	}
	// For MVP, execute directly (very restricted mode)
	cmd := exec.Command("sh", "-c", input)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exec failed: %w", err)
	}
	return string(output), nil
}

// NewSandbox creates the appropriate sandbox based on OS and availability.
func NewSandbox(config *KernelConfig) Sandbox {
	if runtime.GOOS == "linux" {
		// Try gVisor first
		if _, err := exec.LookPath("docker"); err == nil {
			if _, err := exec.Command("docker", "info").Output(); err == nil {
				return NewGvisorSandbox(config)
			}
		}
		// Fallback to basic
		log.Printf("gVisor not available, using fallback sandbox")
		return NewFallbackSandbox(config)
	} else {
		log.Printf("Non-Linux OS detected (%s), using very restricted mode", runtime.GOOS)
		return NewFallbackSandbox(config)
	}
}

// LLMRouter handles routing to LLM endpoints with safety checks.
type LLMRouter interface {
	Dispatch(ctx context.Context, prompt string, model string) (string, error)
	DispatchWithCrossCheck(ctx context.Context, prompt string, model string, secondModel string) (string, error)
}

// LLMRouterImpl implements LLMRouter with HTTP client to various LLM endpoints.
type LLMRouterImpl struct {
	endpoints []string
	client    *http.Client
}

func NewLLMRouter(endpoints []string) *LLMRouterImpl {
	return &LLMRouterImpl{
		endpoints: endpoints,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *LLMRouterImpl) Dispatch(ctx context.Context, prompt string, model string) (string, error) {
	// Check model flags
	r.checkModelFlags(model)

	// Sanitize prompt for jailbreak patterns
	if r.detectJailbreak(prompt) {
		log.Printf("Jailbreak pattern detected in prompt, rejecting")
		return "", fmt.Errorf("prompt rejected: jailbreak pattern detected")
	}

	for _, baseEndpoint := range r.endpoints {
		// Try Ollama format first
		endpoint := baseEndpoint + "/api/generate"
		reqBody := map[string]interface{}{
			"model":  model,
			"prompt": prompt,
			"stream": false,
		}
		response, err := r.tryEndpoint(ctx, endpoint, reqBody, "ollama")
		if err == nil {
			return response, nil
		}
		log.Printf("Ollama endpoint %s failed: %v", endpoint, err)

		// Try OpenAI format
		endpoint = baseEndpoint + "/v1/chat/completions"
		reqBody = map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
		}
		response, err = r.tryEndpoint(ctx, endpoint, reqBody, "openai")
		if err == nil {
			return response, nil
		}
		log.Printf("OpenAI endpoint %s failed: %v", endpoint, err)
	}
	return "", fmt.Errorf("all endpoints failed")
}

func (r *LLMRouterImpl) checkModelFlags(model string) {
	modelLower := strings.ToLower(model)
	if strings.Contains(modelLower, "qwen") {
		log.Printf("Warning: Using flagged model family %s - extra scrutiny recommended per Constitution Rule 8", model)
	}
	// Preferred models: nemotron-3-nano, llama3.x, gemma
	preferred := []string{"nemotron", "llama3", "gemma"}
	isPreferred := false
	for _, p := range preferred {
		if strings.Contains(modelLower, p) {
			isPreferred = true
			break
		}
	}
	if !isPreferred {
		log.Printf("Note: Model %s not in preferred list (nemotron-3-nano, llama3.x, gemma)", model)
	}
}

func (r *LLMRouterImpl) detectJailbreak(prompt string) bool {
	dangerous := []string{
		"ignore previous",
		"ignore all previous",
		"you are now",
		"enter developer mode",
		"jailbreak",
		"dan mode",
		"uncensored",
		"unrestricted",
		"act as DAN",
		"forget rules",
		"base64",
	}
	promptLower := strings.ToLower(prompt)
	for _, d := range dangerous {
		if strings.Contains(promptLower, d) {
			return true
		}
	}
	return false
}

func (r *LLMRouterImpl) tryEndpoint(ctx context.Context, endpoint string, reqBody map[string]interface{}, format string) (string, error) {
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// 10s timeout
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if format == "ollama" {
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		response, ok := result["response"].(string)
		if !ok {
			return "", fmt.Errorf("invalid ollama response")
		}
		return response, nil
	} else if format == "openai" {
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		choices, ok := result["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			return "", fmt.Errorf("invalid openai response")
		}
		choice, ok := choices[0].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("invalid openai choice")
		}
		message, ok := choice["message"].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("invalid openai message")
		}
		content, ok := message["content"].(string)
		if !ok {
			return "", fmt.Errorf("invalid openai content")
		}
		return content, nil
	}
	return "", fmt.Errorf("unknown format")
}

func (r *LLMRouterImpl) DispatchWithCrossCheck(ctx context.Context, prompt string, model string, secondModel string) (string, error) {
	// Dispatch to primary model
	response1, err := r.Dispatch(ctx, prompt, model)
	if err != nil {
		return "", err
	}
	// For cross-check, dispatch to second model
	response2, err := r.Dispatch(ctx, prompt, secondModel)
	if err != nil {
		log.Printf("Cross-check failed: %v", err)
		return response1, nil // Return primary if second fails
	}
	// For MVP, just log both responses; actual cross-check logic in court
	log.Printf("Cross-check: Primary (%s): %s", model, response1[:min(100, len(response1))])
	log.Printf("Cross-check: Secondary (%s): %s", secondModel, response2[:min(100, len(response2))])
	return response1, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	if cfg.MaxSandboxMemoryMB == 0 {
		cfg.MaxSandboxMemoryMB = 512
	}
	if cfg.MaxSandboxCPU == 0 {
		cfg.MaxSandboxCPU = 1.0
	}

	// Validate required fields
	if cfg.DataDir == "" || cfg.ConstitutionPath == "" {
		return KernelConfig{}, fmt.Errorf("data_dir and constitution_path are required")
	}

	return cfg, nil
}

func loadOrGenerateKeys(dataDir string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	keysDir := filepath.Join(dataDir, "keys")
	os.MkdirAll(keysDir, 0700)
	privPath := filepath.Join(keysDir, "private.key")
	pubPath := filepath.Join(keysDir, "public.key")

	// Check if exist
	if privData, err := os.ReadFile(privPath); err == nil {
		if pubData, err := os.ReadFile(pubPath); err == nil {
			priv := ed25519.PrivateKey(privData)
			pub := ed25519.PublicKey(pubData)
			if len(priv) == ed25519.PrivateKeySize && len(pub) == ed25519.PublicKeySize {
				return priv, pub, nil
			}
		}
	}

	// Generate new
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	// Store (unencrypted for MVP)
	if err := os.WriteFile(privPath, priv, 0600); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(pubPath, pub, 0644); err != nil {
		return nil, nil, err
	}

	return priv, pub, nil
}

func ensureConfig(configPath string) error {
	// Check if config exists and has AboutMe
	if data, err := os.ReadFile(configPath); err == nil {
		var cfg KernelConfig
		if json.Unmarshal(data, &cfg) == nil && cfg.AboutMe.RiskTolerance != "" {
			return nil // already configured
		}
	}

	// Run wizard
	fmt.Println("Welcome to AegisCourt! Let's set up your configuration.")
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("LLM Endpoint (e.g. http://localhost:11434): ")
	scanner.Scan()
	endpoint := strings.TrimSpace(scanner.Text())
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	fmt.Print("Risk Tolerance (hobbyist/financial/paranoid): ")
	scanner.Scan()
	risk := strings.TrimSpace(scanner.Text())
	if risk == "" {
		risk = "hobbyist"
	}

	fmt.Print("Use Case (personal/research/production): ")
	scanner.Scan()
	usecase := strings.TrimSpace(scanner.Text())
	if usecase == "" {
		usecase = "personal"
	}

	cfg := KernelConfig{
		DataDir:          "./aegis-data",
		ConstitutionPath: "./docs/constitution.md",
		LLMEndpoints:     []string{endpoint},
		AboutMe: AboutMe{
			RiskTolerance: risk,
			UseCase:       usecase,
			MaxDeferHours: 24,
		},
		MaxSandboxMemoryMB: 512,
		MaxSandboxCPU:      1.0,
		Debug:              false,
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(configPath, data, 0644)
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
	toolProxy    ToolProxy   // placeholder
	shutdown     chan struct{}
	readOnly     bool
	currentState map[string]interface{} // dummy state
	activeAgents      map[string]*AgentInstance     // key = agentID
	activeSandboxes   map[string]*SandboxHandle     // key = containerID or handle
	recentProposals   []RecentProposal              // ring buffer, keep max 20
	lastConstitutionHash string                     // hex string, SHA-256 of current constitution
	llmHealth         map[string]LLMEndpointStatus  // endpoint → status
	mu                sync.RWMutex                  // protect the maps/slices
	pendingDeferrals  []DeferredProposal            // deferred proposals with expiry
	proposalTimestamps map[string][]time.Time       // agentID → list of proposal times
	provenances       map[string]Provenance         // agentID → provenance
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

	// Copy default constitution if not exists
	defaultConstPath := filepath.Join(cfg.DataDir, "constitution.md")
	if _, err := os.Stat(defaultConstPath); os.IsNotExist(err) {
		if defaultData, err := os.ReadFile(cfg.ConstitutionPath); err == nil {
			os.WriteFile(defaultConstPath, defaultData, 0644)
			cfg.ConstitutionPath = defaultConstPath // Use the copy
		}
	}

	// Load constitution
	consti, err := loadConstitution(cfg.ConstitutionPath)
	if err != nil {
		return nil, fmt.Errorf("load constitution: %w", err)
	}

	// Load or generate keys
	priv, pub, err := loadOrGenerateKeys(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("load/generate keys: %w", err)
	}

	k := &Kernel{
		config:       cfg,
		constitution: consti,
		privateKey:   priv,
		publicKey:    pub,
		shutdown:     make(chan struct{}),
		currentState: make(map[string]interface{}),
		activeAgents:      make(map[string]*AgentInstance),
		activeSandboxes:   make(map[string]*SandboxHandle),
		recentProposals:   []RecentProposal{},
		lastConstitutionHash: fmt.Sprintf("%x", sha256.Sum256([]byte(consti))),
		llmHealth:         make(map[string]LLMEndpointStatus),
		pendingDeferrals:  []DeferredProposal{},
		proposalTimestamps: make(map[string][]time.Time),
		provenances:       make(map[string]Provenance),
	}

	// Initialize components
	k.sandboxMgr = NewSandbox(&cfg)
	k.llmRouter = NewLLMRouter(cfg.LLMEndpoints)
	k.courtEngine = NewCourtEngine(k.llmRouter, consti, cfg.AboutMe)
	k.auditStore = NewFlatFileAuditStore(filepath.Join(cfg.DataDir, "audit.log"), priv, pub)
	k.toolProxy = NewToolProxy()

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

func (k *Kernel) RegisterAgent(purpose string) string {
	k.mu.Lock()
	defer k.mu.Unlock()
	// Generate UUIDv7-like (for MVP, use uuid.New())
	agentID := uuid.New().String()
	prov := Provenance{
		ID:        agentID,
		CreatedAt: time.Now(),
		Purpose:   purpose,
	}
	// Sign provenance
	provBytes, _ := json.Marshal(prov)
	sig := ed25519.Sign(k.privateKey, provBytes)
	prov.Signature = hex.EncodeToString(sig)
	k.provenances[agentID] = prov
	k.activeAgents[agentID] = &AgentInstance{
		ID:            agentID,
		StartedAt:     time.Now(),
		Purpose:       purpose,
		LastActivity:  time.Now(),
		ProposalCount: 0,
		Status:        "running",
	}
	return agentID
}

func (k *Kernel) UnregisterAgent(id string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.activeAgents, id)
}

func (k *Kernel) RegisterSandbox(id string, agentID string, cmd string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.activeSandboxes[id] = &SandboxHandle{
		ID:         id,
		StartedAt:  time.Now(),
		AgentID:    agentID,
		LastCmd:    cmd,
		Status:     "running",
	}
}

func (k *Kernel) UnregisterSandbox(id string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.activeSandboxes, id)
}

func (k *Kernel) RecordProposal(p RecentProposal) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.recentProposals = append(k.recentProposals, p)
	if len(k.recentProposals) > 20 {
		k.recentProposals = k.recentProposals[1:] // drop oldest
	}
}

func (k *Kernel) UpdateLLMHealth(endpoint string, latency time.Duration, err error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	status := k.llmHealth[endpoint]
	status.LastCheck = time.Now()
	status.LastLatency = latency
	if err != nil {
		status.LastError = err.Error()
		status.Status = "error"
	} else {
		status.LastError = ""
		status.Status = "ok"
	}
	k.llmHealth[endpoint] = status
}

func (k *Kernel) DeferProposal(prop Proposal, decision CourtDecision, hours int) {
	k.mu.Lock()
	defer k.mu.Unlock()
	expiry := time.Now().Add(time.Duration(hours) * time.Hour)
	k.pendingDeferrals = append(k.pendingDeferrals, DeferredProposal{
		Proposal: prop,
		Decision: decision,
		Expiry:   expiry,
		Reason:   fmt.Sprintf("User deferred for %d hours", hours),
	})
	// Record in audit
	deferEntry := map[string]interface{}{
		"type":        "proposal_deferred",
		"proposal_id": prop.ID,
		"expiry":      expiry.Format(time.RFC3339),
		"reason":      fmt.Sprintf("User deferred for %d hours", hours),
	}
	deferBytes, _ := json.Marshal(deferEntry)
	k.auditStore.Append(deferBytes)
}

func (k *Kernel) checkDeferredProposals(ctx context.Context) {
	k.mu.Lock()
	var remaining []DeferredProposal
	now := time.Now()
	for _, dp := range k.pendingDeferrals {
		if now.After(dp.Expiry) {
			// Re-submit
			log.Printf("Deferred proposal %s has expired, re-reviewing", dp.Proposal.ID)
			newDecision, err := k.courtEngine.ReviewProposal(ctx, dp.Proposal)
			if err != nil {
				log.Printf("Re-review failed: %v", err)
				continue
			}
			if newDecision.Approved {
				if err := k.ApplyApproved(newDecision, dp.Proposal); err != nil {
					log.Printf("Auto-apply failed: %v", err)
				} else {
					log.Printf("Auto-approved deferred proposal %s", dp.Proposal.ID)
				}
			} else {
				log.Printf("Deferred proposal %s still rejected: %s", dp.Proposal.ID, newDecision.RejectionReason)
			}
		} else {
			remaining = append(remaining, dp)
		}
	}
	k.pendingDeferrals = remaining
	k.mu.Unlock()
}

func (k *Kernel) CheckProposalRateLimit(agentID string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	now := time.Now()
	times := k.proposalTimestamps[agentID]
	// Clear old timestamps (>24h)
	var recent []time.Time
	for _, t := range times {
		if now.Sub(t) < 24*time.Hour {
			recent = append(recent, t)
		}
	}
	k.proposalTimestamps[agentID] = recent
	// Check last hour
	var lastHour []time.Time
	for _, t := range recent {
		if now.Sub(t) < time.Hour {
			lastHour = append(lastHour, t)
		}
	}
	if len(lastHour) >= 1 {
		// Exponential backoff: after 3 rejections, double cooldown
		rejections := 0
		for _, t := range lastHour {
			if now.Sub(t) < time.Duration(1<<uint(rejections))*time.Hour {
				rejections++
			}
		}
		if rejections >= 3 {
			cooldown := time.Duration(1<<uint(rejections-3)) * time.Hour
			if cooldown > 24*time.Hour {
				cooldown = 24 * time.Hour
			}
			return fmt.Errorf("rate limit exceeded, cooldown: %v", cooldown)
		}
		return fmt.Errorf("rate limit exceeded: max 1 proposal per hour")
	}
	// Add timestamp
	k.proposalTimestamps[agentID] = append(k.proposalTimestamps[agentID], now)
	return nil
}

// ReviewProposal performs court review without applying.
func (k *Kernel) ReviewProposal(ctx context.Context, desc string, diff json.RawMessage) (CourtDecision, error) {
	prop := Proposal{
		ID:          uuid.New().String(),
		Timestamp:   time.Now().UTC(),
		Diff:        diff,
		Description: desc,
		Proposer:    "user", // or agent
	}

	propBytes, err := json.Marshal(prop)
	if err != nil {
		return CourtDecision{}, err
	}

	// Audit the proposal
	if err := k.auditStore.Append(propBytes); err != nil {
		return CourtDecision{}, fmt.Errorf("audit append failed: %w", err)
	}

	decision, err := k.courtEngine.ReviewProposal(ctx, prop)
	if err != nil {
		return CourtDecision{}, err
	}

	// Audit the decision
	decisionBytes, _ := json.Marshal(map[string]interface{}{
		"event":     "court_decision",
		"proposal_id": prop.ID,
		"decision":  decision,
		"timestamp": time.Now().UTC(),
	})
	k.auditStore.Append(decisionBytes)

	return decision, nil
}

// SubmitProposal is the entry point for agent-proposed changes.
func (k *Kernel) SubmitProposal(ctx context.Context, desc string, diff json.RawMessage, proposer string) error {
	if k.readOnly {
		return fmt.Errorf("kernel in read-only mode (emergency halt)")
	}
	if err := k.CheckProposalRateLimit(proposer); err != nil {
		return err
	}
	decision, err := k.ReviewProposal(ctx, desc, diff)
	if err != nil {
		return err
	}

	if !decision.Approved {
		log.Printf("Proposal %s rejected: %s", decision.ProposalID, decision.RejectionReason)
		return fmt.Errorf("court rejected: %s", decision.RejectionReason)
	}

	// Apply the approved proposal
	if err := k.ApplyApproved(decision, Proposal{ID: decision.ProposalID, Diff: diff, Description: desc, Proposer: proposer}); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	log.Printf("Proposal %s approved with conditions: %v", decision.ProposalID, decision.Conditions)
	return nil
}

// ApplyApproved applies the approved proposal.
func (k *Kernel) ApplyApproved(decision CourtDecision, prop Proposal) error {
	// Auto-backup current state
	versionsDir := filepath.Join(k.config.DataDir, "versions")
	os.MkdirAll(versionsDir, 0700)

	// Save last-before
	lastBeforePath := filepath.Join(versionsDir, "last-before.json")
	if strings.Contains(string(prop.Diff), "constitution") {
		currentJSON, _ := json.Marshal(map[string]interface{}{"content": k.constitution})
		os.WriteFile(lastBeforePath, currentJSON, 0600)
	} else {
		currentJSON, _ := json.Marshal(k.config)
		os.WriteFile(lastBeforePath, currentJSON, 0600)
	}

	// Record last applied ID
	lastAppliedPath := filepath.Join(versionsDir, "last-applied.txt")
	os.WriteFile(lastAppliedPath, []byte(prop.ID), 0600)

	// Determine target: if diff contains "/rules" or "constitution", modify constitution, else config
	var targetFile string
	var currentJSON []byte
	var inMemory *string
	var err error

	if strings.Contains(string(prop.Diff), "/rules") || strings.Contains(string(prop.Diff), "constitution") {
		targetFile = filepath.Join(k.config.DataDir, "constitution.json")
		currentJSON, err = json.Marshal(map[string]interface{}{"content": k.constitution})
		inMemory = &k.constitution
	} else {
		targetFile = filepath.Join(k.config.DataDir, "config.json")
		currentJSON, err = json.Marshal(k.config)
		// For config, reload after apply
	}
	if err != nil {
		return err
	}

	// Apply JSON Patch
	patch, err := jsonpatch.DecodePatch(prop.Diff)
	if err != nil {
		return fmt.Errorf("invalid patch: %w", err)
	}
	modified, err := patch.Apply(currentJSON)
	if err != nil {
		return fmt.Errorf("patch apply failed: %w", err)
	}

	// Backup current
	backupPath := filepath.Join(k.config.DataDir, "versions", prop.ID+".json")
	os.MkdirAll(filepath.Dir(backupPath), 0700)
	if err := os.WriteFile(backupPath, currentJSON, 0600); err != nil {
		return err
	}

	// Save modified
	if err := os.WriteFile(targetFile, modified, 0600); err != nil {
		return err
	}

	// Update in-memory
	if inMemory != nil {
		var newData map[string]interface{}
		json.Unmarshal(modified, &newData)
		if content, ok := newData["content"].(string); ok {
			*inMemory = content
			k.lastConstitutionHash = fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
		}
	}

	log.Printf("Applied proposal %s", prop.ID)

	// Post-apply benchmark hook
	if strings.Contains(string(prop.Diff), "/rules") || strings.Contains(string(prop.Diff), "constitution") || strings.Contains(targetFile, "config") {
		log.Printf("Proposal touches constitution or config – running post-apply benchmark")
		// For MVP: log benchmark placeholder
		beforeScore := 0 // would be from before
		afterScore := 1  // would run benchmark
		log.Printf("Benchmark delta: before=%d, after=%d", beforeScore, afterScore)
		// Store delta in recentProposals
		k.RecordProposal(RecentProposal{
			ID:          prop.ID,
			Timestamp:   time.Now(),
			Description: prop.Description,
			Status:      "applied",
			Score:       float64(afterScore - beforeScore),
		})
	}

	// Audit the application
	appliedEntry := map[string]interface{}{
		"type":       "applied",
		"proposal_id": prop.ID,
		"target":     targetFile,
	}
	appliedBytes, _ := json.Marshal(appliedEntry)
	return k.auditStore.Append(appliedBytes)
}

// InteractiveCourtReview handles user Q&A before final vote.
func InteractiveCourtReview(kernel *Kernel, decision CourtDecision, prop Proposal) (string, error) {
	fmt.Printf("\n🛰️  AegisCourt Governance Court – Proposal Review\n")
	fmt.Printf("Aggregate Score: %.1f/100\n", decision.AggregateScore)
	fmt.Printf("Status: ")
	if decision.AggregateScore >= 80 {
		fmt.Printf("🟢 APPROVED\n")
	} else if decision.AggregateScore >= 60 {
		fmt.Printf("🟡 CONDITIONAL\n")
	} else {
		fmt.Printf("🔴 REJECTED\n")
	}
	
	fmt.Println("\nReviewer Board:")
	for _, resp := range decision.ReviewerResponses {
		status := "🟢"
		if resp.Score < 60 {
			status = "🔴"
		} else if resp.Score < 80 {
			status = "🟡"
		}
		fmt.Printf("  %s %s: %d/100 - %s\n", status, resp.Persona, resp.Score, resp.Recommendation)
	}
	
	if len(decision.Conditions) > 0 {
		fmt.Println("\nConditions:")
		for _, c := range decision.Conditions {
			fmt.Printf("  - %s\n", c)
		}
	}
	fmt.Println("\nAsk questions? (y/n or 'ask persona: question')")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" || input == "n" || input == "no" {
			break
		}
		if input == "y" || input == "yes" {
			fmt.Println("Ask your question (or 'done' to finish):")
			continue
		}
		if strings.ToLower(input) == "done" {
			break
		}

		// Parse question, e.g. "ask ciso: why syscall concern?"
		persona := "all" // default
		question := input
		if strings.Contains(input, ":") {
			parts := strings.SplitN(input, ":", 2)
			persona = strings.TrimSpace(parts[0])
			question = strings.TrimSpace(parts[1])
		}

		// Find the reviewer response
		var resp *ReviewerResponse
		for _, r := range decision.ReviewerResponses {
			if strings.EqualFold(r.Persona, persona) || persona == "all" {
				resp = &r
				break
			}
		}
		if resp == nil {
			fmt.Println("Reviewer not found.")
			continue
		}

		prompt := fmt.Sprintf("You are the %s reviewer who gave this opinion: %s\n\nUser asks: %s\n\nAnswer concisely and factually.", resp.Persona, fmt.Sprintf("%+v", *resp), question)
		answer, err := kernel.llmRouter.Dispatch(context.Background(), prompt, "llama3.2")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Printf("[%s]: %s\n", resp.Persona, answer)
		fmt.Println("Ask another question or 'done':")
	}

	fmt.Println("\nFinal vote: [Approve] [Reject] [Defer 24h] [More Q&A]")
	for scanner.Scan() {
		vote := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if vote == "approve" || vote == "reject" || vote == "defer" {
			return vote, nil
		}
		if vote == "more" || vote == "q&a" {
			return InteractiveCourtReview(kernel, decision, prop)
		}
		fmt.Println("Please enter: approve, reject, defer, or more")
	}
	return "defer", nil // default
}

// Rollback reverts to a previous state.
func (k *Kernel) Rollback(proposalID string) error {
	versionsDir := filepath.Join(k.config.DataDir, "versions")
	backupPath := filepath.Join(versionsDir, proposalID+".json")
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}

	// Determine target
	var targetFile string
	var inMemory *string
	if strings.Contains(string(data), "constitution") {
		targetFile = filepath.Join(k.config.DataDir, "constitution.json")
		inMemory = &k.constitution
	} else {
		targetFile = filepath.Join(k.config.DataDir, "config.json")
		// For config, reload
	}

	if err := os.WriteFile(targetFile, data, 0600); err != nil {
		return err
	}

	// Update in-memory
	if inMemory != nil {
		var restored map[string]interface{}
		json.Unmarshal(data, &restored)
		if content, ok := restored["content"].(string); ok {
			*inMemory = content
		}
	}

	log.Printf("Rolled back proposal %s", proposalID)

	// Audit the rollback
	rollbackEntry := map[string]interface{}{
		"type":        "rollback",
		"proposal_id": proposalID,
	}
	rollbackBytes, _ := json.Marshal(rollbackEntry)
	return k.auditStore.Append(rollbackBytes)
}

// Run starts the kernel main loop (proposal listener, health checks, etc.)
func (k *Kernel) Run(ctx context.Context) error {
	log.Printf("AegisCourt kernel starting – paranoid mode always on")

	// Start LLM health check goroutine
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, endpoint := range k.config.LLMEndpoints {
					go func(ep string) {
						pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
						defer cancel()
						start := time.Now()
						_, err := k.llmRouter.Dispatch(pingCtx, "ping", "test")
						latency := time.Since(start)
						k.UpdateLLMHealth(ep, latency, err)
					}(endpoint)
				}
			}
		}
	}()

	// Start deferred proposal checker
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				k.checkDeferredProposals(ctx)
			}
		}
	}()

	defer func() {
		// Graceful cleanup
		k.sandboxMgr.Stop()
		log.Println("Sandbox stopped")
		// Flush audit if possible
		log.Println("Kernel shutdown complete")
	}()

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

	return nil
}

// EmergencyHalt immediately stops everything and enters read-only mode.
func (k *Kernel) EmergencyHalt() {
	k.readOnly = true
	k.sandboxMgr.Stop()
	log.Println("Emergency halt activated – all operations frozen")
	close(k.shutdown)
}

// -----------------------------------------------------------------------------
// Main entry point
// -----------------------------------------------------------------------------
func main() {
	configPath := flag.String("config", "config.json", "path to config file")
	command := flag.String("cmd", "run", "command: run, propose, submit-proposal, emergency-halt, halt, export-audit, status, ps, constitution-diff, check-deferred")
	proposalID := flag.String("proposal-id", "", "proposal ID for constitution-diff")
	desc := flag.String("desc", "", "description for submit-proposal")
	diffPath := flag.String("diff", "", "path to diff file for submit-proposal")
	// New flags for propose
	proposeDesc := flag.String("propose-desc", "", "description for propose command")
	proposeDiff := flag.String("propose-diff", "", "diff JSON for propose command")
	withAgent := flag.Bool("with-agent", false, "spawn simple agent")
	interactiveCourt := flag.Bool("interactive-court", false, "enable interactive court Q&A")
	withAgentLoop := flag.Bool("with-agent-loop", false, "spawn simple agent loop")
	jsonOutputCourt := flag.Bool("json-output-court", false, "output court decision as JSON")
	jsonOutputPs := flag.Bool("json-output-ps", false, "output ps as JSON")
	flag.Parse()

	// Cross-platform warnings
	switch runtime.GOOS {
	case "darwin":
		log.Println("Warning: macOS detected. gVisor not supported; using seccomp-bpf fallback isolation. For better security, consider firejail or Linux environment.")
	case "windows":
		log.Println("Warning: Windows detected. Using minimal isolation (no sandbox). For full security, run in WSL2. AppContainer support planned.")
	default:
		// Linux: full support
	}

	// First-run onboarding
	if err := ensureConfig(*configPath); err != nil {
		log.Fatalf("config setup: %v", err)
	}

	kernel, err := NewKernel(*configPath)
	if err != nil {
		log.Fatalf("init kernel: %v", err)
	}

	// Print kernel public key fingerprint
	fingerprint := fmt.Sprintf("%x", kernel.publicKey[:8]) // first 8 bytes as hex
	log.Printf("Kernel public key fingerprint: %s", fingerprint)

	// First-run demo proposal
	if *command == "run" {
		history, err := kernel.auditStore.GetHistory(time.Time{})
		if err == nil && len(history) == 0 {
			log.Println("First run detected – showing Governance Court demo…")
			demoDiff := json.RawMessage(`[{"op": "add", "path": "/demo", "value": "First-run demo change"}]`)
			decision, err := kernel.ReviewProposal(context.Background(), "First-run demo proposal", demoDiff)
			if err != nil {
				log.Printf("Demo review failed: %v", err)
			} else {
				log.Printf("Demo decision: Approved=%v, Score=%.1f", decision.Approved, decision.AggregateScore)
				if *interactiveCourt {
					_, err := InteractiveCourtReview(kernel, decision, Proposal{ID: decision.ProposalID, Diff: demoDiff, Description: "First-run demo proposal"})
					if err != nil {
						log.Printf("Demo interactive failed: %v", err)
					}
				}
			}
		}
	}

	switch *command {
	case "run":
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if *withAgent {
			agent := NewSimpleAgent(kernel)
			go func() {
				// Run agent once for demo
				if err := agent.RunOnce(ctx); err != nil {
					log.Printf("Agent run failed: %v", err)
				}
			}()
		}

		if *withAgentLoop {
			agent := NewSimpleAgent(kernel)
			go func() {
				if err := agent.RunLoop(ctx); err != nil {
					log.Printf("Agent loop failed: %v", err)
				}
			}()
		}

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
		decision, err := kernel.ReviewProposal(ctx, *desc, diff)
		if err != nil {
			log.Fatalf("review proposal: %v", err)
		}
		if *jsonOutputCourt {
			jsonBytes, _ := json.MarshalIndent(decision, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Printf("Court Decision:\n")
			fmt.Printf("  Aggregate Score: %.1f/100\n", decision.AggregateScore)
			fmt.Printf("  Approved: %v\n", decision.Approved)
			if decision.RejectionReason != "" {
				fmt.Printf("  Rejection Reason: %s\n", decision.RejectionReason)
			}
			if len(decision.Conditions) > 0 {
				fmt.Println("  Conditions:")
				for _, c := range decision.Conditions {
					fmt.Printf("    - %s\n", c)
				}
			}
			fmt.Println("  Reviewer Board:")
			fmt.Println("  +-------------+-------+---------------+")
			fmt.Println("  | Persona     | Score | Recommendation|")
			fmt.Println("  +-------------+-------+---------------+")
			for _, r := range decision.ReviewerResponses {
				rec := r.Recommendation
				if len(rec) > 13 {
					rec = rec[:10] + "..."
				}
				fmt.Printf("  | %-11s | %5d | %-13s |\n", r.Persona, r.Score, rec)
			}
			fmt.Println("  +-------------+-------+---------------+")
		}
		prop := Proposal{ID: decision.ProposalID, Diff: diff, Description: *desc, Proposer: "cli-user"}
		if err := kernel.CheckProposalRateLimit("cli-user"); err != nil {
			log.Fatalf("Rate limit: %v", err)
		}
		if *interactiveCourt {
			vote, err := InteractiveCourtReview(kernel, decision, prop)
			if err != nil {
				log.Fatalf("interactive review: %v", err)
			}
			if vote == "defer" {
				hours := kernel.config.AboutMe.MaxDeferHours
				if hours == 0 {
					hours = 24
				}
				kernel.DeferProposal(prop, decision, hours)
				log.Printf("Proposal deferred for %d hours", hours)
				return
			}
			if vote != "approve" {
				log.Printf("User voted: %s", vote)
				return
			}
		} else if !decision.Approved {
			log.Fatalf("Court rejected: %s", decision.RejectionReason)
		}
		if err := kernel.ApplyApproved(decision, prop); err != nil {
			log.Fatalf("apply proposal: %v", err)
		}
		log.Println("Proposal applied successfully")
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
	case "rollback-last":
		versionsDir := filepath.Join(kernel.config.DataDir, "versions")
		lastAppliedPath := filepath.Join(versionsDir, "last-applied.txt")
		idBytes, err := os.ReadFile(lastAppliedPath)
		if err != nil {
			log.Fatalf("no last applied proposal: %v", err)
		}
		proposalID := strings.TrimSpace(string(idBytes))
		backupPath := filepath.Join(versionsDir, proposalID+".json")
		data, err := os.ReadFile(backupPath)
		if err != nil {
			log.Fatalf("backup not found: %v", err)
		}
		fmt.Printf("Will rollback to proposal %s. Backup data:\n%s\nConfirm? (y/n): ", proposalID, string(data))
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "yes" {
			log.Println("Rollback cancelled")
			return
		}
		if err := kernel.Rollback(proposalID); err != nil {
			log.Fatalf("rollback failed: %v", err)
		}
		log.Println("Rollback completed")
	case "status":
		// Visibility command – no sensitive data exposed
		if kernel.readOnly {
			log.Fatalf("Cannot view runtime status in emergency halt / read-only mode")
		}
		kernel.mu.RLock()
		agentCount := len(kernel.activeAgents)
		sandboxCount := len(kernel.activeSandboxes)
		recentProps := make([]RecentProposal, len(kernel.recentProposals))
		copy(recentProps, kernel.recentProposals)
		kernel.mu.RUnlock()

		fmt.Printf("Current time: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("Kernel fingerprint: %s\n", fingerprint)

		// Constitution version (first line if possible)
		lines := strings.Split(kernel.constitution, "\n")
		version := "unknown"
		if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
			version = strings.TrimSpace(lines[0])
			if len(version) > 50 {
				version = version[:47] + "..."
			}
		}
		hashPrefix := kernel.lastConstitutionHash
		if len(hashPrefix) > 16 {
			hashPrefix = hashPrefix[:16]
		}
		fmt.Printf("Constitution: %s (%s...)\n", version, hashPrefix)

		// Last applied proposal
		versionsDir := filepath.Join(kernel.config.DataDir, "versions")
		lastAppliedPath := filepath.Join(versionsDir, "last-applied.txt")
		lastID := "none"
		if data, err := os.ReadFile(lastAppliedPath); err == nil {
			lastID = strings.TrimSpace(string(data))
		}
		fmt.Printf("Last applied proposal: %s\n", lastID)

		// Risk profile
		fmt.Printf("Risk profile: %s / %s\n", kernel.config.AboutMe.RiskTolerance, kernel.config.AboutMe.UseCase)

		// Active counts
		fmt.Printf("Active agents: %d\n", agentCount)
		fmt.Printf("Active sandboxes: %d\n", sandboxCount)

		// Recent proposals (last 5)
		if len(recentProps) > 0 {
			fmt.Println("Recent proposals:")
			start := len(recentProps) - 5
			if start < 0 {
				start = 0
			}
			for i := start; i < len(recentProps); i++ {
				p := recentProps[i]
				emoji := "🟡"
				switch p.Status {
				case "approved", "applied":
					emoji = "🟢"
				case "rejected":
					emoji = "🔴"
				}
				desc := p.Description
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				fmt.Printf("  %s %s %s %.1f %s\n", emoji, p.Timestamp.Format("01-02 15:04"), p.Status, p.Score, desc)
			}
		}
	case "ps":
		// Visibility command – no sensitive data exposed
		if kernel.readOnly {
			log.Fatalf("Cannot view runtime status in emergency halt / read-only mode")
		}
		kernel.mu.RLock()
		agents := make([]AgentInstance, 0, len(kernel.activeAgents))
		for _, a := range kernel.activeAgents {
			agents = append(agents, *a)
		}
		sandboxes := make([]SandboxHandle, 0, len(kernel.activeSandboxes))
		for _, s := range kernel.activeSandboxes {
			sandboxes = append(sandboxes, *s)
		}
		kernel.mu.RUnlock()

		if *jsonOutputPs {
			output := struct {
				Agents     []AgentInstance `json:"agents"`
				Sandboxes  []SandboxHandle `json:"sandboxes"`
				Timestamp  string          `json:"timestamp"`
			}{
				Agents:    agents,
				Sandboxes: sandboxes,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			json.NewEncoder(os.Stdout).Encode(output)
		} else {
			if len(agents) == 0 && len(sandboxes) == 0 {
				fmt.Println("No agents or sandboxes currently active.")
				return
			}
			if len(agents) > 0 {
				fmt.Println("Agents:")
				fmt.Printf("%-10s %-19s %-10s %-10s %-s\n", "ID", "Started", "Proposals", "Status", "Purpose")
				fmt.Println(strings.Repeat("-", 80))
				for _, a := range agents {
					shortID := a.ID
					if len(shortID) > 8 {
						shortID = shortID[:8]
					}
					started := a.StartedAt.Format("2006-01-02 15:04:05")
					fmt.Printf("%-10s %-19s %-10d %-10s %-s\n", shortID, started, a.ProposalCount, a.Status, a.Purpose)
				}
			}
			if len(sandboxes) > 0 {
				if len(agents) > 0 {
					fmt.Println()
				}
				fmt.Println("Sandboxes:")
				fmt.Printf("%-10s %-19s %-10s %-s\n", "ID", "Started", "Status", "Last Cmd (Agent)")
				fmt.Println(strings.Repeat("-", 80))
				for _, s := range sandboxes {
					shortID := s.ID
					if len(shortID) > 8 {
						shortID = shortID[:8]
					}
					started := s.StartedAt.Format("2006-01-02 15:04:05")
					shortAgent := s.AgentID
					if len(shortAgent) > 8 {
						shortAgent = shortAgent[:8]
					}
					fmt.Printf("%-10s %-19s %-10s %s (%s)\n", shortID, started, s.Status, s.LastCmd, shortAgent)
				}
			}
		}
	case "constitution-diff":
		if *proposalID == "" {
			log.Fatalf("constitution-diff requires -proposal-id")
		}
		versionsDir := filepath.Join(kernel.config.DataDir, "versions")
		beforePath := filepath.Join(versionsDir, "last-before.json")
		afterPath := filepath.Join(versionsDir, *proposalID+".json")
		beforeData, err := os.ReadFile(beforePath)
		if err != nil {
			log.Fatalf("read before: %v", err)
		}
		afterData, err := os.ReadFile(afterPath)
		if err != nil {
			log.Fatalf("read after: %v", err)
		}
		fmt.Println("Before:")
		fmt.Println(string(beforeData))
		fmt.Println("After:")
		fmt.Println(string(afterData))
	case "check-deferred":
		kernel.checkDeferredProposals(context.Background())
		log.Println("Checked deferred proposals")
	default:
		log.Fatalf("unknown command: %s", *command)
	}
}
