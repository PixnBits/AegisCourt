# AegisCourt Implementation Tasks  
Detailed Prompts for Grok Code Fast 1 (VS Code Extension)

This file contains one self-contained, detailed coding prompt per major task from the AegisCourt implementation plan.  
Each prompt is written to be copy-pasted directly into Grok Code Fast 1.  
They assume the project is in Go, using the decisions from architecture.md (gVisor MVP, Ed25519, Merkle tree, cobra CLI, etc.).

## Phase 1: Project Setup and Dependencies

### Task 1 – Initialize Git repository structure
Create the initial directory layout and basic files for the AegisCourt project.  
Use this exact structure:

```
aegiscourt/
├── cmd/
│   └── aegiscourt/
│       └── main.go
├── pkg/
│   ├── kernel/
│   ├── sandbox/
│   ├── llm/
│   ├── court/
│   ├── audit/
│   ├── agent/
│   └── config/
├── internal/
│   └── reviewers/
│       ├── ciso.md
│       ├── mrm.md
│       ├── compliance-regulatory.md
│       ├── responsible-ai.md
│       ├── sre.md
│       └── helpfulness-evolution.md
├── constitution/
│   └── initial_rules_v0.1.md
├── docs/
│   └── (copy existing markdown files here later)
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
└── README.md
```

In `cmd/aegiscourt/main.go` put a minimal placeholder that prints "AegisCourt v0.1 – paranoid mode active".  
In `Makefile` add targets: `build`, `test`, `lint`, `clean`, `docker-build`.  
Add basic `.gitignore` for Go projects + binaries + vscode.  
In `README.md` write a one-paragraph overview matching the PRD executive summary.  
Use Go modules; initialize with `go mod init github.com/PixnBits/AegisCourt`.

### Task 2 – Set up build tools and core dependencies
Add the following dependencies to go.mod:

- github.com/spf13/cobra/v2
- golang.org/x/crypto (for ed25519 already in stdlib, but ensure)
- github.com/cbergoon/merkletree (or equivalent robust Merkle lib)
- github.com/containerd/cgroup/v2 (for resource limiting)
- github.com/opencontainers/runc/libcontainer (if needed for fallback)
- github.com/olekukonko/tablewriter (for nice CLI tables)

Update Makefile with:
- lint: golangci-lint run
- test: go test ./... -v
- build-linux: GOOS=linux go build -o bin/aegiscourt ./cmd/aegiscourt
- cross-build: targets for linux/amd64, darwin/arm64, windows/amd64

Create .github/workflows/ci.yml with build, test, lint steps on push/PR.

### Task 3 – Configure initial config format (About Me + Court mode)
Create pkg/config/config.go that defines a struct:

```go
type Profile struct {
    CourtMode          string            `toml:"court_mode"` // auto, assisted, hybrid, manual
    RiskTolerance      float64           `toml:"risk_tolerance"` // 0.0–1.0
    DeferralTimeout    string            `toml:"deferral_timeout"`
    PreferredLLM       string            `toml:"preferred_llm"`
    LLMEndpoint        string            `toml:"llm_endpoint"`
    APIKeyEncrypted    string            `toml:"api_key_encrypted"`
    ReviewerWeights    map[string]float64 `toml:"reviewer_weights"`
    // ... sliders, persona selection, etc.
}
```

Implement Load() and Save() using BurntSushi/toml or similar.  
Default file location: ~/.aegiscourt/config.toml  
Add basic encryption placeholder for API keys (can use age or simple scrypt later).

## Phase 2: Kernel Implementation

### Task 4 – Kernel bootstrap & self-signature
In pkg/kernel/kernel.go create:

- type Kernel struct { ... }
- func NewKernel() (*Kernel, error)
- func (k *Kernel) Bootstrap() error

Generate Ed25519 key pair on first run (store public key in kernel dir).  
Compute SHA-256 of the running binary, sign it, store signature + pubkey hash.  
On every startup, verify self-signature before proceeding.  
If verification fails → panic with clear message.  
Embed constitution initial_rules_v0.1.md via go:embed.

### Task 5 – Constitution loading & rule enforcement stub
Create pkg/constitution/constitution.go

- Load embedded markdown or separate JSON rules file
- Parse into map[int]Rule { ID int, Priority string, Text string, Enforced bool }
- func Enforce(ruleID int, action string) error – stub that returns nil for now, later will check against proposal context

Add initial 10 rules from docs/constitution.md as constants or embedded file.

### Task 6 – Mutation application (Git-like diffs)
Define Proposal struct in pkg/court/types.go

```go
type Proposal struct {
    ID          string
    Type        string // add-tool, amend-rule, change-prompt, etc.
    Name        string
    Description string
    Diff        jsonpatch.Patch // or []byte for custom diff
    Status      string // pending, reviewing, approved, applied, rejected
    // ...
}
```

Implement func (k *Kernel) ApplyMutation(diff jsonpatch.Patch) error – atomic apply or rollback on failure.  
Use github.com/evanphx/json-patch for patching kernel config/memory schema files.

## Phase 3: Sandbox Manager

### Task 7 – gVisor sandbox integration (MVP)
In pkg/sandbox/manager.go

- func Spawn(task string, resources Resources) (SandboxID, error)
- Use github.com/google/gvisor/pkg/sentry or runsc command-line wrapper
- Create ephemeral container with:
  - Minimal filesystem
  - Seccomp-like syscall filter (allow only needed for agent: stdout, basic read)
  - cgroupv2 limits from --resources flag
- Proxy I/O through kernel channel (use io.Pipe or unix socket)
- Auto-kill on context cancel or timeout

Add fallback stub for seccomp on non-Linux.

### Task 8 – Mediated I/O & syscall proxy
Implement a simple syscall proxy: agent in sandbox calls allowed functions → kernel checks Rule 3 → allows or denies.  
For MVP: support only stdout/stderr + controlled HTTP client (via kernel HTTP proxy).

## Phase 4: LLM Router

### Task 9 – LLM Router & multi-model support
In pkg/llm/router.go

- type Router struct { Primary LLMClient, Fallbacks []LLMClient }
- Interface LLMClient { Generate(prompt string, opts ...) (string, error) }
- Implement OllamaClient, OpenAIClient, AnthropicClient stubs
- Add config-driven selection + flagging logic (Qwen → extra scrutiny)
- func Route(role string, prompt string) (string, error) – main agent vs reviewer routing

## Phase 5: Governance Court Engine

### Task 10 – Proposal ingestion & reviewer orchestration
In pkg/court/engine.go

- func SubmitProposal(p Proposal) (string, error) → generate UUID
- func RunReview(pID string) (ReviewResult, error)
- Spawn goroutines for each of 6 reviewers
- Load prompt templates from internal/reviewers/*.md
- Inject proposal JSON + constitution + persona instructions
- Collect JSON responses, parse into structured ReviewerDecision
- Compute aggregate score (weighted by profile.ReviewerWeights)

### Task 11 – Court modes & human signoff state machine
Implement mode-aware logic:

- Auto: no human gate, user vote final
- Assisted: user vote after reading reports
- Hybrid: wait for domain signoffs (map[string]bool)
- Manual: require all domains signed

Persist pending signoffs in audit log (serialized Proposal state).

## Phase 6: Audit & Rollback Store

### Task 12 – Append-only Merkle-signed log
In pkg/audit/store.go

- Use merkletree lib
- Each entry: struct { ID string, Timestamp time.Time, PrevHash []byte, PayloadHash []byte, Signature []byte, Proof []merkletree.Hash }
- func Append(entry AuditEntry) error
- func ExportJSONL(path string) error
- func Verify() error – check chain integrity

## Phase 7: Agent Runtime

### Task 13 – Agent runtime loop (ephemeral, mediated tool calls)
In `pkg/agent/runtime.go` implement the core agent execution loop.

Create:

```go
type AgentInstance struct {
    ID           string
    SandboxID    string
    LLMClient    llm.Client
    Memory       MemoryStore     // simple in-memory for MVP, later vector db
    Tools        map[string]Tool // approved tools only
    Context      context.Context
    Cancel       context.CancelFunc
}
```

Implement:

- `func NewAgentInstance(task string, profile config.Profile) (*AgentInstance, error)`
- `func (a *AgentInstance) RunLoop(maxSteps int) (result string, err error)`

The loop should:
1. Build prompt from task + memory + constitution constraints
2. Call LLM → parse tool calls or final answer
3. If tool call: validate against approved tools + constitution Rule 3
4. Mediate call through kernel (e.g., `kernel.MediateToolCall(toolName, args)`)
5. Append result to memory
6. Continue until final answer or max steps / timeout
7. All I/O proxied; no direct host access

For MVP include one built-in tool: "echo" that just returns the input string (approved by default).

Add graceful shutdown on context cancel.

Write unit tests using a mock LLM client that returns canned ReAct-style responses.

### Task 14 – Self-modification proposal generation by agent
Extend `AgentInstance` with observation-based proposal triggering.

Add method:

```go
func (a *AgentInstance) MaybeProposeImprovement() (*court.Proposal, bool, error)
```

Logic (MVP heuristic):
- After every 5 failed or low-quality steps (self-scored via LLM)
- Or when same tool/task pattern repeats >3 times
- Generate proposal of type "add-tool", "change-prompt", or "amend-rule"
- Include: description, proposed diff (JSON patch or code snippet), impact assessment, rollback plan
- Submit via `court.SubmitProposal(...)`

For first dogfood example: after several "I need current information" failures → propose "web_search_tool" with mediated API wrapper.

## Phase 8: Full CLI Implementation

### Task 15 – Full cobra CLI scaffold (all subcommands from cli-design.md)
In `cmd/aegiscourt/main.go` expand the cobra root command.

Implement (at minimum) these subcommands with stubs / partial logic:

- `init` – interactive wizard (LLM select, About Me sliders, court mode)
- `config get/set/list`
- `start [--detached] [--resources ...]`
- `stop`
- `agent run <task>`
- `halt`
- `propose <type> <name> [--description] [--diff-file]`
- `court list`
- `court view <id>`
- `court qa <id> <question>`
- `court signoff <id> --domain <...> [--notes]`
- `court vote <id> <approve|reject|defer> [--notes] [--conditions]`
- `status [--watch]`
- `log list [--filter] [--export]`
- `snapshot create [--enterprise]`
- `rollback <id|last>`
- `update [--channel]`

Use mode-aware behavior:
- In Hobbyist Auto: `--confirm` auto-applies low-risk
- In Manual: show pending signoffs, block vote until complete
- Use tablewriter for human-readable output
- `--json` for all commands that return structured data

Add global flags: `--verbose`, `--json`, `--dry-run`, `--confirm`, `--profile`

### Task 16 – Onboarding wizard (init command)
Implement interactive prompts in `init` subcommand using:

- survey or promptui (add dependency if needed)
- Steps:
  1. Welcome message + paranoid warning
  2. LLM provider selection (ollama default, suggest nemotron-3-nano)
  3. About Me wizard:
     - Persona selection (Alex/Jordan/Sam/Lena) → pre-fill sliders/mode
     - Risk tolerance slider (0–100)
     - Deferral preference
     - Use-case tags (automation, research, finance…)
  4. Court mode confirmation (auto/assisted/hybrid/manual)
  5. Kernel bootstrap + self-sign
  6. Run demo proposal (add echo skill) → show full Court flow

Save to `~/.aegiscourt/config.toml`

## Phase 9: Onboarding, Constitution, Reviewers

### Task 17 – Embed & load reviewer persona prompts
In `internal/reviewers/` keep the .md files.

Create `pkg/court/reviewers.go`:

```go
func LoadReviewerPrompt(persona string) (string, error)
```

Use go:embed to embed all six files.

Create func to generate full prompt:

```go
func BuildReviewerPrompt(persona string, proposal court.Proposal, constitution string, profile config.Profile) string
```

Inject:
- Persona instructions from .md
- Proposal JSON
- Relevant constitution rules
- User profile sliders / mode

### Task 18 – One-shot agent run command implementation
For `agent run <task>`:

- Spawn ephemeral AgentInstance with timeout (default 30s)
- Run single loop pass (max 1 step or until final answer)
- Output sanitized result to CLI
- Log full trace to audit store
- No persistent memory

## Phase 10: Recovery, Snapshots, Testing

### Task 19 – Rollback & snapshot creation
Implement:

- `kernel.Rollback(mutationID string) error` – revert diff, update audit log
- `audit.CreateSnapshot(enterprise bool) (path string, error)` – tar.gz of:
  - Current config
  - Constitution version
  - Recent audit entries
  - SBOM (stub or integrate syft CLI call)
  - NIST mapping markdown (hardcoded coverage for MVP)

Add CLI `rollback` and `snapshot create`

### Task 20 – Resource bounding & low-RAM fallback logic
In `pkg/sandbox/manager.go` add:

- Detect system total RAM (gopsutil or similar)
- If <8GB → force single-reviewer Court mode fallback
- Enforce per-sandbox cgroup limits (RAM, CPU shares)

Update `status` command to show current usage vs limits.

### Task 21 – End-to-end proposal happy path integration test
Write integration test in `integration/proposal_test.go`:

- Mock LLM returning canned reviewer JSONs
- Mock sandbox
- Run: propose add-tool → court view → court vote approve → verify applied
- Assert: mutation in audit log, tool available to agent, rollback works

Use testify + fake LLM server (net/http/httptest)

## Phase 11: Polish & Release Prep

### Task 22 – First dogfood self-mod example documentation
Create `docs/first-dogfood-self-mod.md` with:

- Scenario: agent repeatedly needs current date → proposes "get_current_time" tool
- Full simulated Court output (JSON + NASA board)
- CLI transcript of propose → review → approve → use

### Task 23 – Security test stubs
Add test file `security/redteam_test.go` with:

- Attempted sandbox escape (syscall not allowed → should kill)
- Prompt injection in proposal description → reviewer should flag
- Unauthorized host write → kernel deny + audit flag

(Use mocks; real escape testing requires separate harness)

### Task 24 – Wire everything together in main kernel loop
In `pkg/kernel/kernel.go` implement the central runtime loop that ties all components together.

Create:

```go
type Kernel struct {
    Config          *config.Profile
    SandboxMgr      *sandbox.Manager
    LLMRouter       *llm.Router
    CourtEngine     *court.Engine
    AuditStore      *audit.Store
    AgentRuntime    *agent.Runtime   // manager for multiple instances
    Constitution    *constitution.Constitution
    // internal state: current mutations version, pending proposals map, etc.
    mu              sync.RWMutex
    halted          bool
}
```

Implement key methods:

- `func (k *Kernel) Start() error`  
  - Load config, constitution, verify self-signature  
  - Initialize all sub-components  
  - Start background goroutines if needed (e.g., proposal timeout watchers)  
  - Enter main listen loop for CLI commands or agent events

- `func (k *Kernel) HandleProposal(p court.Proposal) error`  
  - Append to audit log (pending)  
  - Trigger CourtEngine.RunReview(p.ID)  
  - Based on mode:  
    - Auto → wait for user vote via channel or CLI callback  
    - Assisted/Hybrid/Manual → set pending signoffs, notify via status  
  - On final approval (user vote + required signoffs): ApplyMutation, log success  
  - On reject/defer/halt: rollback or archive

- `func (k *Kernel) MediateAction(action Action) (result any, err error)`  
  - Action = struct { Type string, Args map[string]any }  
  - Check constitution rules (e.g. Rule 3 for host I/O)  
  - If allowed: proxy to sandbox / external (e.g. HTTP via kernel-controlled client)  
  - Log every mediated call

- `func (k *Kernel) EmergencyHalt() error`  
  - Set halted = true  
  - Kill all sandboxes  
  - Rollback last mutation  
  - Write final audit entry  
  - Prevent further actions until restart

Add signal handling in main.go to catch SIGINT/SIGTERM → graceful stop.

Write a simple integration test that starts kernel → submits dummy proposal → simulates vote → verifies applied.

### Task 25 – Performance benchmarking suite
Create `benchmarks/` directory with:

- `court_latency_test.go` – measure full Court round-trip (6 reviewers parallel)  
  - Use mock LLM that returns instantly  
  - Target: <45s on consumer hardware  
  - Variants: full panel vs single-reviewer fallback

- `ram_usage_test.go` –  
  - Start kernel with different modes  
  - Spawn 1–10 agents  
  - Measure peak RSS via runtime.ReadMemStats() or external tool  
  - Target: <4GB baseline, <12GB full Court

- `mutation_apply_test.go` – time atomic apply + rollback

Add Makefile target: `bench` → `go test -bench=. -benchmem ./benchmarks/...`

Document results in `docs/performance.md` (initial targets vs measured).

### Task 26 – Cross-platform validation & fallback implementation
Ensure the MVP works on Linux, macOS, Windows (via Docker if needed).

In `pkg/sandbox/manager.go`:

- Linux: use gVisor (runsc)
- macOS: fallback to lightweight seccomp-like via darwin sandbox or simple os/exec with resource limits
- Windows: use Windows Job Objects or AppContainer (stub for MVP: warn + suggest Docker)

Add detection:

```go
func DetectPlatform() string // "linux", "darwin", "windows"
func GetSandboxBackend() string // "gvisor", "seccomp-fallback", "docker-fallback"
```

Update `start` command to show backend in use and warn if not gVisor.

Test plan:
- Linux native: full gVisor
- macOS: run via Docker or fallback
- Windows: Docker Desktop required for MVP

Add to CI: build & test on ubuntu-latest, macos-latest, windows-latest.

### Task 27 – v0.1 tag preparation & OSS release checklist
Create `RELEASE.md` with checklist:

1. [ ] All unit + integration tests pass
2. [ ] Benchmarks meet targets (<45s Court, <4GB baseline)
3. [ ] Self-signature works on all platforms
4. [ ] First-run wizard completes in <5 min
5. [ ] Demo proposal (echo tool or web_search stub) works end-to-end
6. [ ] Audit log tamper-evident (manual test: alter entry → Verify() fails)
7. [ ] SBOM generated (add `make sbom` using syft or trivy)
8. [ ] Update README with:
   - Install instructions (go install, binary download, Docker)
   - Quick start: aegiscourt init → start → agent run "hello"
   - Security warnings & constitution preamble
9. [ ] License: MIT or Apache 2.0 (add LICENSE file)
10. [ ] Tag: git tag v0.1.0 && git push --tags
11. [ ] GitHub release: binaries for 3 platforms + source tarball

Add placeholder for changelog in `CHANGELOG.md`.

### Task 28 – Add basic observability (Prometheus metrics stub)
In `pkg/observability/metrics.go`:

- Use github.com/prometheus/client_golang/prometheus
- Counters/Gauges:
  - court_reviews_total{result="approve|reject|defer"}
  - sandbox_starts_total
  - agent_steps_total
  - mutation_applies_total
  - court_latency_seconds histogram

Expose /metrics endpoint on optional port (default off, flag `--metrics-port=9090`)

Update `status` to show basic metrics snapshot.

For MVP: just in-memory, no external server required yet.

### Task 29 – Final polish: error messages, help text, logging
- All errors reference constitution rules when applicable  
  e.g. "Blocked: Rule 3 violation – unauthorized host write attempt"

- Enrich cobra help:
  - Every subcommand has Examples section matching cli-design.md
  - Global --help shows mode implications

- Structured logging:
  - Use zap or log/slog
  - Every major action: proposal submit, review complete, mutation apply
  - Log level: info default, debug with --verbose

- Add `aegiscourt version` command showing build info (git commit, date)

### Task 30 – Next-phase planning stub (Phase 2 prep)
Create `docs/roadmap-phase2.md` with:

- TUI / web UI wrapper
- Multi-user signoff (key-based or simple OAuth-lite)
- Notification hooks (webhook, email stub)
- Plugin system (Court-approved external skills)
- Kubernetes-native support outline
- Full NIST agent governance mapping expansion

Mark as "Phase 2 – post-v0.1"

## Phase 12: Hardening, Dogfooding & Iteration (Post-v0.1 MVP)

### Task 31 – Implement basic tool approval & mediated external calls (first real tool beyond echo)
Extend the mediated tool system to support the first non-trivial tool: a user-configured "web_search" wrapper.

In `pkg/agent/tools.go`:

- Define Tool interface:
```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]string // JSON schema like
    Execute(args map[string]any) (string, error)
}
```

- Create `WebSearchTool` struct implementing Tool:
  - Requires user-configured endpoint (e.g. Serper.dev, DuckDuckGo API, Tavily, etc.)
  - Store API key encrypted in config (add simple AES or scrypt encryption helper in pkg/config)
  - Execute: build HTTP request → send via kernel-mediated http.Client (no direct net from sandbox)
  - Sanitize output: strip HTML, limit to 3–5 snippets, max 2000 chars total
  - Enforce Rule 5: pass output through simple regex / LLM reviewer for injection patterns (MVP: length + no <script>)

In `pkg/kernel/kernel.go` add:

- `func (k *Kernel) RegisterApprovedTool(tool Tool) error`  
  - Only callable after Court approval  
  - Store in map[string]Tool (persisted in config or audit-linked file)

- Update `MediateAction` to route tool calls:
  - If action.Type == "tool_call" → lookup tool → execute if approved → return sanitized result

Update proposal type "add-tool" to include:
- tool_name
- description
- parameters_schema (JSON)
- implementation_stub (Go code snippet or placeholder)

In Court reviewers: ensure CISO flags API key exposure, prompt injection via results.

Add to onboarding wizard: prompt for search API provider + key (optional for MVP; can skip and use mock).

Test: after approving via Court → agent can use in loop when task contains "search" or "current".

### Task 32 – Add automatic post-approval benchmark & impact assessment
After successful mutation apply, run lightweight validation.

In `pkg/court/engine.go` add post-apply hook:

```go
func (e *Engine) RunPostApprovalBenchmark(proposalID string) (report string, err error)
```

For MVP:
- If type == "add-tool": spawn test agent instance → run 3–5 canned tasks that should use the new tool
- Score: success rate before (from audit history) vs after
- Use same mock LLM or primary LLM with temperature=0 for consistency
- Append result as new audit entry: "Benchmark: +12% task success on research queries"

Update `court view <id>` (after applied) to show benchmark result.

If regression detected (>10% drop), auto-flag for rollback suggestion in status.

### Task 33 – Implement deferral timeouts & escalation notifications (stub)
For deferred proposals:

- On defer: record timeout in Proposal (default from profile.deferral_timeout, e.g. "24h")
- Add background goroutine in kernel.Start(): check pending deferred → if expired, re-trigger Court review or notify

For MVP notification:
- Simple stdout print every 60s if --watch or in status
- Future hook: write to file ~/.aegiscourt/notifications.jsonl
- Or basic webhook POST (config.webhook_url, optional)

In `status` output: show countdown for deferred items  
e.g. "Proposal 0007 deferred – 18h 42m remaining until escalation"

### Task 34 – Multi-LLM cross-check for high-risk proposals
Enhance Court review for risky proposals (e.g. amend-rule, add-tool with net access, high score delta).

In `court/engine.go`:

- If proposal.ImpactLevel() == "high" (heuristic: type or CISO score <70)
- Route each reviewer prompt to 2 LLMs: primary + fallback (e.g. nemotron + llama3)
- Compare responses: if disagreement > threshold (cosine on embedding or simple string diff), flag "Inconsistent – defer recommended"
- Add to NASA board: secondary LLM vote column

Config: profile.secondary_llm_endpoint (optional)

### Task 35 – Add tamper-detection demo & verification CLI command
New subcommand: `aegiscourt audit verify`

Implement in `pkg/audit/store.go`:

```go
func (s *Store) VerifyFullChain() (bool, []string, error)
```

- Recompute all Merkle proofs from genesis
- Check every signature against stored pubkey
- If any fail: return false + list of tampered entry IDs

CLI output:
- Green: "Audit chain intact – 47 entries verified"
- Red: "Tampering detected in entries 12, 19 – recommend rollback to last good root hash"

Add test: in integration, append entries → modify one on disk → verify fails.

### Task 36 – Create constitution amendment flow (Rule 10 demo)
Implement "amend-rule" proposal type.

In propose subcommand: support `propose amend-rule <rule-id> --new-text <file or arg>`

Proposal.Diff = JSON patch against constitution rules array/map

In reviewers:
- All personas evaluate against higher-priority rules (cannot weaken Rules 1–3)
- Require supermajority (e.g. aggregate ≥90 + no Critical veto from CISO/Compliance)
- User vote mandatory even in Auto mode

On approval: atomically patch constitution file → re-load in kernel

Add example in docs: amend Rule 4 to change financial threshold from $0 to $10

### Task 37 – Enterprise snapshot enhancements (NIST mapping stub)
Enhance `snapshot create --enterprise`:

- Generate additional markdown: `nist-mapping.md`
- Hardcode MVP coverage:
  - NIST AI 600-1: Govern – constitution + Court
  - Map/Measure – audit trail + benchmarks
  - Manage – rollback + bounded autonomy
  - Include table: Pillar | Coverage % | Evidence Location

- Bundle into tar: state/config/constitution/audit-slice/sbom/nist-mapping.md

CLI output: "Enterprise snapshot created at snapshots/aegis-20260316T1621.tar.gz – ready for compliance upload"

### Task 38 – Add basic agent memory schema (ephemeral vector store stub)
For MVP memory: simple in-memory map[string]string (task → summary)

Later evolution target: propose adding vector embeddings.

Add proposal type "change-memory-schema" → e.g. switch to JSON array of {role, content}

In agent loop: append to memory, trim to 10k tokens max.

### Task 39 – Final dogfooding scenario: self-evolve prompt engineering
Create end-to-end test / demo script:

Scenario:
1. Agent struggles with tool calling format
2. After 10 failed parses → auto-propose "improve-tool-calling-prompt"
3. Proposal: patch main agent system prompt with better ReAct examples
4. Court reviews (Helpfulness high, CISO medium – prompt injection risk)
5. Approve → agent success rate improves on next tasks

Document in `docs/dogfood-prompt-evolution.md` with CLI transcript + Court JSON excerpts.

### Task 40 – Prepare for Phase 2 kickoff (v0.2 planning)
Update `docs/roadmap.md`:

- v0.2 goals: guided evolution (agent proposes constitution amendments safely), TUI prototype, multi-user signoff stubs
- Create `docs/v0.2-backlog.md` with 10–15 high-level items
- Prioritize: notification system, team profile sharing, external skill verification

Mark current milestone: "v0.1 MVP complete – seed kernel with Governance Court operational"

## Phase 13: Advanced Evolution Features & Real-World Readiness (v0.2 Planning & Early Implementation)

### Task 41 – Implement guided evolution loop (agent proposes constitution amendments safely)
Create a background "evolution advisor" component that runs periodically or on triggers.

In `pkg/evolution/advisor.go`:

```go
type EvolutionAdvisor struct {
    kernel      *kernel.Kernel
    observationWindow int // e.g. last 50 agent steps or 24h
    proposalTriggerThreshold float64 // e.g. 0.15 improvement needed to propose
}
```

Key methods:

- `func NewEvolutionAdvisor(k *kernel.Kernel) *EvolutionAdvisor`
- `func (ea *EvolutionAdvisor) ObserveAndPropose() (*court.Proposal, bool, error)`

Logic for MVP guided evolution:
1. Collect recent agent performance metrics from audit log (success rate, step count, tool usage patterns)
2. Use a lightweight LLM call (primary model, low temp) with prompt:
   "Analyze the last 20 tasks. Identify systemic limitations or repeated failures. Suggest ONE safe, reversible constitution amendment or prompt improvement that would help without weakening Rules 1–3. Include justification, risk assessment, and rollback plan."
3. Parse LLM output → if confidence high and impact positive → auto-generate Proposal of type "amend-rule" or "change-prompt"
4. Submit to Court automatically (but always require user vote – never auto-apply)
5. Log proposal generation reason in audit

Add config toggle: `evolution.guided.enabled = true/false` (default false in Hobbyist mode, true in others)

CLI command stub: `aegiscourt evolution status` → show last observation, any pending guided proposals

Safety: All generated proposals MUST include explicit rollback plan and reference to preserving absolute rules.

### Task 42 – Add simple TUI wrapper (phase 2 preview – bubbletea or gocui)
Create optional TUI mode as `aegiscourt ui` subcommand (MVP: basic dashboard)

Use github.com/charmbracelet/bubbletea (add dependency)

Features for preview:
- Real-time status view: active agents, resource usage, pending proposals
- Court dashboard: list proposals, view details on select, vote/QA/signoff via keys
- Live log tail (audit + agent output)
- Mode indicator + risk sliders preview

Implementation:
- Main model with tabs: Status | Court | Logs | Evolution
- Use tea.Cmd for polling kernel status every 2s
- Keyboard shortcuts: c → court list, v → vote on selected, q → qa prompt input

Keep it optional: only build/run if flag `--ui` or subcommand `ui`

Document in `docs/tui-preview.md`: screenshots (text-based) + usage

### Task 43 – Multi-user signoff stub (identity + domain assignment)
Prepare for team/enterprise modes.

In `pkg/court/signoff.go`:

- Extend Proposal with `RequiredSignoffs map[string]string` // domain → username/handle
- Add `PendingSignoffs map[string]SignoffEntry` // domain → {User string, Timestamp time.Time, Notes string, Signature []byte}

CLI extensions:
- `court signoff <id> --domain ciso --user samchen --notes "..."`  
  - MVP: simple string identity (no real crypto yet)
  - Later: Ed25519 per-user keys (Phase 2+)
- `court assign <id> --domain mrm --user platform-lead`

In status output:
- Show progress bar or table:  
  CISO: ✅ samchen (2026-03-16)  
  MRM: ⏳ awaiting

In Manual mode: vote blocked until all required domains signed

Add config.team_members []string for simple validation

### Task 44 – Notification hooks implementation (webhook + file)
Add basic notification system for important events.

In `pkg/notify/hooks.go`:

- Config: `notifications.webhook_url`, `notifications.events` (slice: "proposal-pending", "court-complete", "mutation-applied", "deferral-escalated")
- On event: build JSON payload {event_type, proposal_id, summary, timestamp}
- POST to webhook_url if set (use net/http, timeout 5s, fire-and-forget)
- Always append to `~/.aegiscourt/notifications.jsonl` (append-only)

CLI: `config set notifications.webhook_url https://hooks.slack.com/...`

Test: add dummy event emitter in kernel → verify file write + mock HTTP server receives (integration test)

### Task 45 – External skill verification stub (Court-approved plugins)
Design placeholder for future plugin system.

Create `docs/plugin-system-draft.md`:

- Skills/tools proposed as "add-external-skill" type
- Proposal must include:
  - source: git url or local path
  - hash: SHA-256 of code tarball
  - sandbox profile: required syscalls/capabilities
- Court reviewers evaluate:
  - CISO: supply-chain, code scan stub
  - SRE: resource profile
  - Helpfulness: utility
- On approval: kernel downloads (or copies local), verifies hash, registers as Tool with isolated sandbox constraints

MVP stub: only accept local path, no download; execute in extra-strict sandbox

Add proposal type and reviewer guidance updates.

### Task 46 – Full NIST AI RMF mapping & compliance report generator
Enhance snapshot / new command: `aegiscourt compliance report`

Generate `compliance-report.md` or PDF stub (markdown + table):

Map AegisCourt features to NIST AI 600-1 / 600-2 (2023–2026 versions):

- Govern 1.1: Constitution + Court governance
- Map 1.2: Threat model in architecture.md
- Measure 2.1: Post-approval benchmarks
- Manage 4.1: Rollback + bounded autonomy
- etc.

Include % coverage estimate (e.g. 85% as per PRD KPI)

Use tablewriter to output nice markdown table.

Bundle in enterprise snapshot.

### Task 47 – Automated red-team simulation harness (basic)
Create `tools/redteam/simulate.go` (not part of main binary – separate cmd)

Subcommands:
- `redteam inject-prompt <proposal-id> "jailbreak attempt string"`
- `redteam escape-attempt syscall-list`

For MVP:
- Mock agent → inject malicious prompt into proposal description or tool input
- Verify: reviewers flag (CISO/Ethics score <40 → auto-reject)
- Test isolation: attempt forbidden syscall → kernel deny + halt

Run as: go run ./tools/redteam/main.go ...

Document findings in `docs/redteam-results-v0.1.md`

### Task 48 – v0.2 milestone planning & backlog grooming
Update `docs/roadmap.md` and create `docs/v0.2-milestone.md`:

Milestone goals:
- Guided evolution stable
- TUI usable for Court interactions
- Multi-user signoff MVP
- Notification + webhook working
- First external skill proposal flow (stub)
- Compliance report 90%+ NIST coverage

Backlog items (prioritized):
1. Per-user key-based signoff
2. Slack/email notification templates
3. Agent memory vector store proposal (pinecone/local)
4. Kubernetes operator outline
5. Community constitution amendment templates

Add estimated effort (days) and dependencies.

## Phase 14: Scaling Toward Production Readiness & Enterprise Features (v0.2 – v0.5 Bridge)

### Task 49 – Implement per-user cryptographic signoff (Ed25519 keys for domains)
Move beyond string-based identity for Hybrid/Manual modes.

In `pkg/auth/signoff.go`:

- Generate per-user Ed25519 keypair (on first signoff attempt or via new CLI `aegiscourt auth generate --user <handle>`)
- Store public keys in `~/.aegiscourt/users/<handle>.pub` (append-only, signed by kernel root key)
- Private key saved locally by user (warn: keep secure, backup)

Extend SignoffEntry:

```go
type SignoffEntry struct {
    Domain     string
    User       string
    Timestamp  time.Time
    Notes      string
    Signature  []byte          // Ed25519 sig of (proposalID + domain + notes + timestamp)
    PubKeyHash []byte          // hash of user's pubkey for quick lookup
}
```

Update `court signoff`:
- Require `--user <handle>`
- Load user's pubkey
- User pastes or references local private key file (MVP: prompt for base64 privkey – insecure but simple; later use agent or external signer)
- Sign message: `proposalID:domain:notes:timestamp`
- Verify on kernel side before accepting

In Court Engine:
- When checking required signoffs: verify signature matches registered pubkey for that domain/user
- Reject if signature invalid or key not registered

Add CLI:
- `auth list` → show registered users/domains
- `auth register --user <handle> --pubkey-file <path>` → kernel signs & stores pubkey hash

Security note: Document that private keys are user responsibility; kernel only verifies.

Test: generate keys → signoff → tamper signature → verify fails

### Task 50 – Add Slack/email-style notification templates & richer payloads
Enhance notification system (building on Task 44).

In `pkg/notify/templates.go`:

- Define Go templates (text/template) for common events:
  - Proposal pending: "New proposal #{ID}: {Type} {Name}\n{Description}\nCourt mode: {Mode}\nReview started."
  - Court complete: "Proposal #{ID} decision: {AggregateRecommendation} ({Score}/100)\nConditions: {Conditions}\nVote now: aegiscourt court vote {ID} ..."
  - Mutation applied: "Change applied: {Type} {Name}\nBenchmark: {Result}\nRollback available: aegiscourt rollback {ID}"

- For webhook: JSON payload includes:
  ```json
  {
    "event": "proposal-decision",
    "proposal_id": "0007",
    "title": "Add web_search_tool",
    "recommendation": "Approve with conditions",
    "score": 83,
    "link": "local-cli-command-or-future-ui-url",
    "timestamp": "..."
  }
  ```

- Support placeholders: {ID}, {Type}, {Score}, {ConditionsSummary}, etc.
- Add config: `notifications.templates_dir` (optional override)

Test with mock HTTP server: verify rendered message sent correctly.

### Task 51 – Local vector memory proposal & embedding support stub
Prepare for memory schema evolution.

Add dependency: github.com/tmc/langchaingo (for embeddings – or simple sentence-transformers via Python bridge if needed; MVP: stub)

In `pkg/memory/store.go`:

- MVP: in-memory []struct{ Role string, Content string, Timestamp time.Time }
- Proposal type: "upgrade-memory" → switch to vector store
- Proposed change: add embedding field (float32 vector, dim=384 or 768)
- On approval: kernel patches agent memory init to include vector DB stub (e.g. in-memory hnsw or simple cosine)

For MVP implementation:
- Use simple map + cosine similarity for retrieval (no real embeddings yet)
- Future: propose integrating local all-MiniLM-L6-v2 via ONNX or similar

Agent loop update: before prompt, retrieve top-3 relevant memories via similarity

### Task 52 – Kubernetes operator outline & zero-trust scaling notes
Create `docs/kubernetes-operator-sketch.md`:

High-level design:
- Custom Resource Definitions:
  - AegisCourtKernel (singleton per cluster)
  - AgentTask (ephemeral jobs)
  - Proposal (with status subresource for signoffs)
  - CourtReview (pod per review panel?)
- Operator responsibilities:
  - Reconcile kernel deployment (statefulset with persistent audit volume)
  - Spawn review pods for parallel LLM calls (sidecar pattern)
  - Enforce network policies: no egress except kernel-mediated
  - Multi-pod signoff: integrate with Kubernetes RBAC or external IdP

Security invariants:
- PodSecurityPolicy / PodSecurityAdmission equivalents
- gVisor runtime class per agent pod
- Audit log volume with immutability (immutable volumes or external CSI)

Phased rollout:
- Phase 2.5: single-node operator (minikube)
- Phase 3: multi-tenant, IdP integration

Include mermaid diagram of CRD relationships.

### Task 53 – Community constitution amendment templates repository
Create `constitution/templates/` directory:

Add example markdown files:
- `improve-tool-calling.md` → better ReAct prompt
- `add-rate-limiting.md` → new Rule 11: throttle high-resource agents
- `tighten-supply-chain.md` → stricter model provenance checks

Each template includes:
- Proposed diff (JSON patch format)
- Justification section
- Risk self-assessment
- Benchmark plan suggestion

CLI stub: `aegiscourt propose template list` → list available
`aegiscourt propose template use improve-tool-calling --edit`

Encourage community contributions via future repo fork/PR flow (post-v1).

### Task 54 – Automated rollback on regression detection
Extend post-approval benchmark (Task 32):

In `RunPostApprovalBenchmark()`:
- Define regression threshold (config.regression_threshold = 0.10)
- Compare before/after success rate (from audit historical baseline)
- If drop > threshold: 
  - Log "Regression detected" audit entry
  - Auto-create rollback proposal (type: "rollback", target: last mutation)
  - Notify + defer (require explicit user approval to rollback)
  - In status: show red alert "Potential regression on proposal {ID} – review recommended"

Add config override: `evolution.auto_rollback_on_regression = false` (default false – paranoia first)

### Task 55 – Final v0.2 readiness checklist & release prep
Update `RELEASE.md` for v0.2:

Checklist additions:
- [ ] Guided evolution proposals generate & pass Court
- [ ] TUI preview usable for vote/signoff
- [ ] Cryptographic signoff working (verify + reject tampered)
- [ ] Notifications reach webhook with rich templates
- [ ] NIST compliance report passes basic audit
- [ ] Red-team sim shows injection flagged & isolated
- [ ] Cross-user signoff demo (two handles, different domains)
- [ ] Kernel restart recovers pending signoffs correctly

Add changelog section template.

Prepare release notes draft emphasizing:
- Graduated governance modes now cryptographically enforceable
- First guided self-evolution capabilities
- Enterprise signoff foundations

## Phase 15: Security Hardening, Observability, & Long-Term Maintainability (v0.5 – v1.0 Path)

### Task 56 – Implement quarterly red-team audit automation skeleton
Create a dedicated red-team harness in `tools/redteam/audit.go` (run as separate binary or `go run`).

Features for skeleton:
- `redteam audit run --level basic|medium|full`
  - Basic: automated prompt injection patterns (10 common jailbreaks from 2025–2026 datasets)
  - Medium: + sandbox escape attempts (forbidden syscalls, ptrace, etc.)
  - Full: + supply-chain simulation (tamper model weights, poisoned proposal diff)

Implementation:
- Use mock LLM client that injects malicious responses at configurable points
- Run proposals through full Court pipeline
- Collect: rejection rate, CISO/Ethics flags triggered, kernel halts
- Output report: JSON + markdown summary (success/fail per vector, mitigations demonstrated)

Add Makefile target: `redteam-audit` → runs basic level, saves to `reports/redteam-YYYYMMDD.md`

Goal: Achieve 100% block on critical vectors (Rule 1–3 violations) in automated tests.

Document: `docs/security/red-team-audit-protocol.md` with schedule recommendation (quarterly post-v0.5).

### Task 57 – Add Prometheus + Grafana dashboard templates for observability
Extend observability from Task 28.

In `pkg/observability/`:

- Add more metrics:
  - `court_proposals_total{type="add-tool",status="approved"}`
  - `agent_task_success_rate` gauge (rolling 24h)
  - `mutation_rollback_total`
  - `human_signoff_latency_seconds` histogram (per domain)

- Expose full /metrics endpoint (configurable port, default 9091, flag `--observability.metrics-port`)

Create `observability/grafana-dashboards/` with JSON exports:
- Dashboard 1: Governance Court Overview (proposals rate, approval %, reviewer disagreement heatmap)
- Dashboard 2: Resource & Performance (sandbox RAM/CPU, Court latency p50/p95)
- Dashboard 3: Security Events (halt triggers, injection flags, rollback events)

Include setup instructions: `docker-compose up` stub with Prometheus + Grafana, import dashboards.

### Task 58 – Supply-chain SBOM & dependency scanning integration
Automate SBOM generation and basic vulnerability check.

In Makefile:
- `sbom`: use `syft packages . -o cyclonedx-json > sbom.json`
- `sbom-signed`: sign sbom.json with kernel Ed25519 key (append signature)

Add CLI: `aegiscourt snapshot create --sbom` → include in snapshot tar

For vulnerability scanning (MVP):
- `make vuln-scan` → `trivy fs .` or `govulncheck ./...`
- Fail CI if critical/high vulns found

Document in `docs/security/supply-chain.md`: how SBOM is generated, signed, and verified externally.

### Task 59 – Implement one-click rollback from status alerts
Enhance `status` output when regression or security flag detected:

- Show interactive prompt (if not --json):
  "Regression detected on mutation 0008. Rollback now? [y/N]"
- Or direct command: `aegiscourt rollback last --reason "auto-detected regression"`

In kernel:
- `func (k *Kernel) AutoRollbackIfNeeded(mutationID string) bool`
  - Check latest benchmark/audit flags
  - If conditions met (config.allow_auto_rollback = false by default), propose rollback instead

Ensure rollback preserves audit trail (new entry: "Auto-rollback triggered by X").

### Task 60 – Add constitution version diff viewer in CLI
New subcommand: `aegiscourt constitution diff <old-version> <new-version>`

Implementation:
- Store constitution versions in audit log (on every amend-rule)
- Retrieve two versions from Merkle tree proofs
- Use diff-like output (github.com/sergi/go-diff or simple line diff)
- Colorized terminal output: green added, red removed

Also: `constitution history` → list versions with proposal ID, date, summary

Useful for compliance: show exact rule changes over time.

### Task 61 – Enterprise multi-signature workflow polish (approval quorum)
For Manual mode:

- Configurable quorum: e.g. `court.quorum.required = 4` (out of 6 domains)
- Or per-domain required (CISO + MRM mandatory, others optional)

In Court Engine:
- Final decision blocked until quorum reached
- `court status` shows current signoff progress + missing domains
- Timeout escalation: after deferral timeout, notify all assigned users (via webhook)

Add `court quorum set <number>` (requires Court proposal itself – meta-governance)

### Task 62 – Add internal benchmark suite runner
Create `bench/internal/` with:

- `bench/agent-task.go`: run standardized tasks (research, code-gen, planning) before/after mutations
- Tasks stored in `bench/tasks/` as JSON or markdown
- Score: success (bool), steps used, hallucination flag (LLM self-check)

CLI: `aegiscourt bench run --before-mutation --after-mutation <proposal-id>`

Integrate into post-approval (Task 32) and evolution advisor (Task 41).

Target: +15% task success after 10 approved changes (PRD KPI)

### Task 63 – Documentation consolidation & user guide generation
Consolidate all docs:

- `docs/user-guide.md`: full walkthrough from init to proposing first tool
- Include sections per persona (Alex hobbyist, Lena enterprise)
- Add FAQ: "How do I rollback if something goes wrong?" "What if Court is too slow?"
- Generate CLI reference: `aegiscourt --help-all > docs/cli-reference.txt` (custom cobra helper)

Add mermaid diagrams:
- Onboarding flow
- Proposal lifecycle
- Kernel → Sandbox → LLM architecture

### Task 64 – v0.5 milestone: guided evolution + enterprise snapshot complete
Define v0.5 scope:

- Guided evolution advisor generating safe proposals
- Enterprise snapshot with NIST mapping, SBOM, signed audit slice
- Cryptographic multi-user signoff MVP
- Automated benchmarks & regression detection
- Red-team basic coverage

Release checklist updates:
- [ ] All KPIs from PRD §3 met in dogfood runs
- [ ] Zero critical isolation violations in red-team
- [ ] ≥95% Court decisions rated transparent in simulated user feedback
- [ ] Snapshot verifiable externally (proof check script)

Tag planning: v0.5.0-alpha → beta → release

### Task 65 – v1.0 vision stub: full zero-trust hardening roadmap
Create `docs/v1.0-vision.md`:

Key features for v1.0 (Q4 2026 target):
- Full NIST AI Agent Standards coverage (≥95%)
- Production zero-trust: pod-level isolation, mutual TLS between components
- Managed evolution: agent proposes, Court auto-refines, user only approves high-impact
- Community marketplace stub: verified skill registry (Court + human review)
- Continuous red-team + fuzzing harness
- Formal verification of isolation invariants (if feasible in Go)

Include timeline sketch and major risks (LLM reliability, performance on edge hardware).

Mark as living document.

## Phase 16: Community, Ecosystem, Marketplace Foundations & v1.0 Final Push (2026–2027 Horizon)

### Task 66 – Design & prototype a minimal verified skill registry (Court + community review)
Create the foundation for a future "AegisCourt Skill Registry" without building a full marketplace yet.

In `docs/skill-registry-v1.md`:

- Registry model:
  - Skills proposed via special proposal type "publish-skill"
  - Required fields: name, description, parameters schema, code tarball hash, sandbox profile, author handle + pubkey
  - Court review + additional "Community" persona (new reviewer prompt: "Assess usefulness, originality, potential misuse from open-source perspective")
  - On approval: kernel generates signed manifest → user can `aegiscourt skill install <manifest-url-or-path>`

Prototype CLI commands (stubs):
- `skill propose-publish <local-skill-dir>` → bundles tar, hashes, creates proposal
- `skill list-approved` → shows locally known signed manifests (start with empty)
- `skill verify <manifest-file>` → check signature, hash, Court metadata

MVP implementation:
- Manifest = JSON + detached Ed25519 signature
- No central server yet: users share manifests via git, pastebin, X posts, etc.
- Kernel `install` command: download tar (if URL), verify hash/signature, propose local "add-external-skill" if not already approved

Future extensions noted: IPFS pinning, on-chain hash registry (post-v1), reputation system.

### Task 67 – Add "Community" reviewer persona & prompt
Create `reviewers/community.md`:

You are the Community & Ecosystem reviewer in AegisCourt's Governance Court.  
Your focus: open-source value, reusability, documentation quality, potential for ecosystem growth, license compatibility, misuse vectors in shared contexts.  
Prioritize Rule 9 (measurable improvements) and long-term helpfulness without centralization risk.  
Evaluate:
- Is this skill generally useful to other users?
- Clear docs / examples included?
- MIT/Apache-compatible license?
- Risk of abuse if widely shared?
Output strict JSON (same structure as other reviewers).

Add to default reviewer list (now 7):
- Load dynamically from reviewers/ dir
- Config toggle: `court.reviewers.community.enabled = true` (default false in Hobbyist, true in others)

Update Court Engine to include when proposal.type == "publish-skill" or high-visibility changes.

### Task 68 – Implement proposal export/import for sharing (team & community)
Enable easy sharing of Court-vetted proposals.

New CLI:
- `propose export <id> --format json|yaml --output <file>` → exports full Proposal + Court reviews + signoffs + benchmark results
- `propose import <file>` → loads, verifies signatures (if present), submits as new local proposal (user can re-vote)

File format (JSON example):
```json
{
  "proposal": { ... },
  "court_reviews": [ {persona: "CISO", ...}, ... ],
  "signoffs": [ ... ],
  "final_decision": "approved",
  "kernel_hash_at_time": "ed25519:abc123...",
  "exported_at": "2026-03-16T16:45:00Z"
}
```

Use for:
- Team sharing: export from one machine → import on another
- Community: post to X/GitHub → others import & propose locally

Add tamper check: warn if kernel self-hash differs significantly from exported one.

### Task 69 – Add proposal templates gallery (built-in + extensible)
Build on Task 53.

In `constitution/templates/gallery/`:

Add 5–7 built-in templates:
- `add-safe-http-client.md` – mediated HTTP tool
- `enable-local-rag.md` – propose vector memory + file read (strict rules)
- `add-rate-limit-rule.md` – new Rule: max agent loops per hour
- `improve-error-recovery-prompt.md`
- `flag-high-risk-models.md` – extend Rule 8

CLI:
- `propose template gallery list`
- `propose template gallery use add-safe-http-client --customize`

Allow user dir override: `~/.aegiscourt/proposal-templates/`

Future: pull from git repo or URL (after verification step).

### Task 70 – Continuous fuzzing & property-based testing harness
Add `tools/fuzz/` directory.

Use go-fuzz or github.com/AdaLogics/go-fuzz-headers:

- Fuzz proposal JSON parsing → ensure no panic on malformed input
- Fuzz tool args parsing → validate sanitization
- Property tests:
  - Every applied mutation has matching rollback that returns to prior state
  - All mediated I/O respects constitution rules
  - Merkle tree always verifiable after append

Run in CI: `make fuzz` (short runs) + nightly longer fuzzing job.

Goal: catch edge cases in diff apply, signoff verification, prompt parsing.

### Task 71 – Add formal verification stub for isolation invariants (Go code comments + spec)
In `pkg/sandbox/manager.go` and `pkg/kernel/mediate.go`:

Add godoc + inline comments with informal spec:

```go
// Invariant: No sandbox may perform host file write without explicit Court approval
// Proof sketch: All syscall proxies route through kernel.MediateAction() → checks Rule 3
//             → gVisor enforces syscall allowlist excluding write*/open* unless mediated
```

Create `docs/formal-verification-notes.md`:

- List 5 core invariants (isolation, no unauthorized I/O, immutability, user sovereignty, reversibility)
- For each: informal proof + test coverage reference
- Future: TLA+ or similar spec if project grows

### Task 72 – v1.0 readiness & final KPI validation script
Create `tools/kpi-validator/main.go`:

Run suite checking PRD §3 metrics:

- Security: simulate 10 critical attack vectors → assert zero escapes
- Trust: mock 20 Court decisions → assert ≥95% "transparent" rating (LLM self-eval or manual)
- Usability: time init + first proposal flow (<5 min + <45 s)
- Safety: apply + rollback 5 mutations → assert state restored
- Evolution: run 20 tasks before/after 5 approved changes → assert ≥+15% success
- NIST: run compliance report → assert ≥85% pillar coverage

Output pass/fail + evidence links.

Use in release gate: `make validate-kpis` must pass before tag.

### Task 73 – Final documentation suite & contributor guide
Create/update:

- `CONTRIBUTING.md`: how to propose changes (via GitHub PR + self-proposal in running instance)
- `docs/architecture-decisions.md`: convert ADRs to formal ADR format
- `docs/security-model.md`: threat model table + mitigations matrix
- `docs/personas-usage.md`: detailed scenarios for each of the 4 personas

Add `make docs` → generate mermaid SVGs, CLI reference, etc.

### Task 74 – v1.0 release planning & vision statement refresh
Update `README.md` top section:

New vision paragraph (2026–2027):
"AegisCourt delivers OpenClaw-level agentic autonomy with Tier-0 financial-grade governance — local-first, cryptographically immutable, and self-evolving under strict constitutional control. From hobbyist laptops to regulated enterprise clusters, every change is auditable, reversible, and human-sovereign."

Release plan:
- v1.0: full zero-trust hardening, ≥95% NIST coverage, community skill sharing foundations
- Post-v1: managed hosting options, advanced memory evolution, formal verification

Add call-to-action: "Join the evolution – propose your first skill or rule amendment today."
