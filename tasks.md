# AegisCourt v0.2 Implementation Prompts for Grok Code Fast 1

These are self-contained, sequential prompts. Implement them one at a time in Go.  
Always:
- Use Go 1.22+ idioms
- Include proper error handling (return errors, never panic unless critical like signature fail)
- Add basic unit tests (table-driven where possible)
- Keep code paranoid, auditable, and reversible
- Never reduce reviewer count — full 6 always used
- Respect resource detection: suggest sequential mode (--low-resource) if RAM low, but never cut reviewers
- Reference existing files/schemas when mentioned

Start with an empty project or continue from previous implementations.

## Phase 1 – Kernel & Isolation Basics

**Prompt 1/1a – Kernel bootstrap + self-signature + resource detection**

You are building AegisCourt v0.2 — a paranoid, local-first agent framework in Go.

Goal: Implement kernel bootstrap, self-signature verification, and resource detection for onboarding.

Requirements:
- Binary self-verifies its own Ed25519 signature on startup (panic if invalid)
- Root public key is a hard-coded constant in the binary
- Use UUIDv7 for audit/log IDs (github.com/gofrs/uuid or similar)
- Detect free RAM (gopsutil preferred) and optionally GPU VRAM (nvidia-smi parse or go-nvml)
- Estimate Court peak: ~2.5 GB base + ~1.8 GB per reviewer (parallel) or ~1 GB sequential
- If free RAM < 9 GB → suggest --low-resource mode (sequential reviewers) and/or llama3.2 fallback
- Never reduce reviewer count — always 6
- Log resource check + chosen LLM to audit
- Output: main.go skeleton + DetectResources() function + signature check

Constraints:
- Go stdlib + gopsutil, golang.org/x/crypto/ed25519
- Minimal deps
- Paranoid: panic on signature fail, log everything

Give me:
1. main.go with bootstrap flow
2. DetectResources() → struct { RAMFreeGB float64, HasGPU bool, VRAMGB float64, RecommendedLLM string, SuggestSequential bool }
3. Signature verification function
4. Basic test suggestion for signature (mocked)
5. Error handling everywhere

Start coding.

**Prompt 2 – Append-only Merkle-signed audit log**

Continue AegisCourt v0.2.

Goal: Append-only Merkle-signed audit log.

Requirements:
- Each entry: UUIDv7, timestamp UTC, prev_hash (sha256), payload_hash (sha256 of JSON), ed25519 signature
- File: ~/.aegiscourt/audit.log (JSONL + signature field)
- Append(entry any) error — marshal to JSON, hash, sign, append line
- Verify() → (bool intact, []error) — recompute chain from genesis
- Root hash signed with kernel key on first entry
- `audit verify` CLI command handler

Give me:
1. audit/audit.go with Append, Verify, Entry struct
2. Simple CLI handler for `audit verify`
3. Test: append 3 entries → tamper one → verify fails with specific error
4. Proper error handling (file permissions, hash mismatch, etc.)

**Prompt 3 – Sandbox Manager with gVisor**

Goal: Basic Sandbox Manager using gVisor (Linux primary).

Requirements:
- Spawn ephemeral sandboxes for agent + reviewers
- Use gVisor runsc runtime (assume installed)
- RunSandboxed(cmd string, args []string, memLimitMB int, cpuShares int) → stdout, stderr, exitCode, error
- Proxy stdout/stderr, block unauthorized syscalls via gVisor
- Enforce cgroupv2 limits
- Low-resource mode: sequential execution flag (from config)
- Fallback warning if gVisor not found (suggest Docker/seccomp)

Give me:
1. sandbox/manager.go with SpawnSandboxed func
2. Basic cgroupv2 setup example
3. Error handling + audit logging
4. Test suggestion (mocked runsc call)

**Prompt 4 – LLM Router basics**

Goal: Simple LLM router for Ollama/local + future cloud.

Requirements:
- Configurable endpoint (default Ollama)
- Primary: nemotron-3-nano (latest quantized)
- Fallback: llama3.2:3b-instruct
- CallLLM(prompt string, model string) → response string, error
- Log every prompt/response to audit
- Health check on init

Give me:
1. llm/router.go with CallLLM and health check
2. Config struct + load from file
3. Test: mock HTTP response

## Phase 2 – Governance Court Engine

**Prompt 5 – Load reviewer personas + schema enforcement**

Goal: Load 6 reviewer prompts + enforce output schema.

Requirements:
- go:embed reviewers/*.md
- schema in pkg/court/reviewers/schema.json (use previous definition)
- Go struct ReviewerOutput + Validate() error
- CallReviewer(persona string, proposalJSON string) → ReviewerOutput, error
- Retry once on schema violation

Give me:
1. court/reviewers/reviewers.go — LoadPrompt + CallReviewer
2. output.go — struct + validation
3. Unit tests for schema pass/fail

**Prompt 6 – Court orchestration + CLI views**

Goal: Orchestrate reviewers + rich CLI output.

Requirements:
- Parallel LLM calls (or sequential if --low-resource)
- Weighted aggregate score (configurable)
- NASA board text/table
- CLI: court view (default), --detailed (line-broken pros/cons), --reviewer <persona>, --json
- Use ANSI tables for readability

Give me:
1. court/engine.go — RunCourt(proposal) → aggregate + []ReviewerOutput
2. cli/court_view.go — render functions
3. Test render with mock data

## Phase 3 – Guided Proposal Creation

**Prompt 7 – Proposal schema + validation**

Goal: Draft/proposal schema enforcement.

Requirements:
- Use pkg/proposal/schema.json (previous definition)
- Go struct Draft + Validate() error
- Save/load drafts from ~/.aegiscourt/proposals/draft-*.json

Give me:
1. proposal/types.go + validation func
2. proposal/storage.go — SaveDraft, LoadDraft
3. Unit tests for valid/invalid drafts

**Prompt 8 – propose agent-help command**

Goal: Generate draft via LLM + validate + launch wizard.

Requirements:
- Use prompts/propose-agent-help.md (previous template with schema)
- Call LLM → parse JSON → validate → save → return uuid
- Retry once on schema fail
- Auto-launch propose guide --draft <uuid>

Give me:
1. cli/propose_agent_help.go — handler
2. Integration with LLM router + schema validation
3. Test with mock LLM response

**Prompt 9 – propose guide interactive wizard**

Goal: Step-by-step wizard for proposal refinement.

Requirements:
- Interactive CLI prompts (bufio or survey lib if minimal dep)
- Steps: type, title, motivation, change (multi-line), impact, risks, rollback, validation
- Optional LLM assist on sections
- Validate against schema at each save
- Save draft on completion

Give me:
1. cli/propose_guide.go — wizard loop
2. Multi-line editor support (os/exec $EDITOR or simple input)
3. LLM assist call example

## Phase 4 – Mutations, Tools, Observability & Polish

**Prompt 10 – Mutation apply/rollback**

Goal: Apply/rollback mutations atomically.

Requirements:
- Git-like diffs (JSON patch + code/rules)
- Apply(diff) error → atomic, audit entry
- Rollback(id) error → restore previous state

Give me:
1. mutation/applier.go — Apply, Rollback funcs
2. Test: apply → rollback → state restored

**Prompt 11 – utc_time mediated tool + benchmarks**

Goal: Basic utc_time tool + canned benchmark runner.

Requirements:
- utc_time: kernel clock proxy, read-only ISO string
- bench/tasks/*.json — 8 tasks (use previous definitions)
- Runner: RunBenchmarks() → map[string]bool pass/fail + % success
- Post-approval: run before/after, flag if delta < -15%

Give me:
1. tools/utc_time.go — implementation
2. bench/runner.go — load tasks, run, judge, delta calc
3. Test for utc_time + one benchmark task

**Prompt 12 – Final polish & release prep**

Goal: status --watch, notifications.jsonl, docs updates.

Requirements:
- status --watch: real-time sandboxes, resources, pending votes
- Append-only notifications.jsonl for proposal events
- README quick-start + tag v0.2.0 prep

Give me:
1. cli/status.go — watch loop
2. notifications/notifications.go — append event
3. README.md snippet
