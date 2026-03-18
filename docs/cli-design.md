# AegisCourt CLI Design & Specification
**Version:** 0.2  
**Status:** Implemented (v0.2 scope -- real features, single-user)  
**Goal:** Define a secure, minimalist, Unix-inspired CLI that serves as the primary (and initially only) operator interface for AegisCourt. This document lives alongside PRD.md §5.4 and will evolve via Court-approved changes.

## 1. Design Principles
- Unix / git / docker / kubectl style: `aegiscourt <subcommand> [args]`
- Paranoid-by-default: confirmations for mutations, verbose risk messaging, full audit logging
- Progressive disclosure: simple defaults for hobbyists; flags and deep subcommands for advanced use
- Mode-aware behavior: output, prompts, and confirmation requirements adapt to active Court mode (set during onboarding or via config)
- Human-first output: readable tables/lists by default; `--json` for automation
- Errors reference constitution rules (e.g. "Blocked: Rule 3 – Unauthorized host access")
- Every state-changing command emits signed audit entry
- Lightweight: <1s startup, no heavy runtime deps beyond kernel
- Cross-platform: Linux primary (gVisor), macOS/Windows via seccomp / Docker fallback
- v0.2 limitation: All modes are single-user. No multi-person / multi-domain signoff yet.

## 2. Root Command & Global Flags

```bash
aegiscourt [global flags] <subcommand> [args]
```

**Global flags**
- `--verbose`              Detailed output (full reviewer JSON, internal state)
- `--json`                 Machine-readable JSON output (overrides human formatting)
- `--dry-run`              Simulate command without applying changes or writing audit log
- `--profile <path>`       Override About Me config file
- `--confirm` / `-y`       Bypass interactive confirmation prompts (dangerous; always audited)
- `--mode-info`            Show current Court mode + review implications for this session
- `--help` / `-h`          Show help (rich per-subcommand help + examples)
- `--low-resource`         Force sequential reviewer execution (full court, but one at a time)
                           Use if RAM constrained; preserves full Court quality

## 3. Subcommands by Category

### 3.1 Setup & Configuration

- `init`  
  Interactive first-run wizard: kernel bootstrap, LLM selection, About Me profile  
  Flags:
    - `--llm <ollama|openai|anthropic|grok|...>`
    - `--profile-template <hobbyist|indie|enterprise|financial>`  
    - `--court-mode <auto|assisted|hybrid|manual>` (overrides wizard default; hobbyist auto if omitted)  
  LLM guidance in wizard:  
  Default suggestion: **nemotron-3-nano** (latest quantized variant, e.g. FP8/BF16 via Ollama) — chosen for superior instruction following, structured JSON output, and reasoning reliability in Court reviewers and proposal assistance.  
  Strong fallback: **llama3.2:3b-instruct** (fast/lightweight on low RAM).  
  Output: signed kernel hash, setup summary

- `config get <key>`  
  `config set <key> <value>`  
  `config list`  
  View / edit runtime configuration (LLM endpoint, risk sliders, etc.)  
  Sensitive changes route through Court  
  Keys supported:
   - `court.mode`                (auto|assisted|hybrid|manual)
   - `court.pending_human_review` (read-only view of outstanding user votes in stricter modes)
   - `preferred_llm`             (e.g. "nemotron-3-nano:latest")

### 3.2 Runtime Control

- `start [--detached] [--resources <ram=4GB,cpu=2>] [--sandbox <gvisor|seccomp>]`  
  Launch kernel + agent runtime

- `stop`  
  Graceful shutdown of all sandboxes and kernel

- `agent run <task description>`  
  One-shot agent execution in ephemeral sandbox  
  Flags: `--agent-id <uuid>`, `--timeout <30s>`, `--output-format <text|json>`  
  **Dynamic tool registry:** The system prompt is built from `~/.aegiscourt/tools.json` at runtime.  
  Tools registered via approved mutations (e.g. `utc_time`) are available to the agent.  
  Tool calls use JSON format: `{"tool": "name", "args": {...}}`  
  Built-in tools: `echo` (default), `utc_time` (after proposal)

- `halt [--no-confirm]`  
  Emergency: freeze all agents, rollback last mutation, enter read-only mode  
  **Mechanism:** Rolls back the last applied mutation via the mutation engine,  
  then writes a `HALTED` marker file to `~/.aegiscourt/`. While halted, status  
  command shows the halt state. Remove the marker or re-init to resume.

### 3.3 Governance & Evolution

- `propose <type> <name>`  
  Submit self-modification proposal (low-level / direct)  
  Types: `add-tool`, `add-skill`, `change-prompt`, `amend-rule`, etc.  
  Flags: `--description <text>`, `--diff-file <path>`, `--impact-assessment <text>`, `--benchmark-plan <text>`  
  Output: Proposal ID, immediate Court start notification

- `propose guide`  
  Interactive step-by-step wizard to build a high-quality proposal aligned with the constitution.  
  Steps include:
  - Select proposal type
  - Describe observed problem/motivation
  - Define concrete proposed change (multi-line editor)
  - Self-assess impact, risks, preservation of Rules 1–5
  - Provide rollback plan
  - Outline validation/benchmark plan
  - Optional LLM-assisted refinement (light/full levels)  
  On completion: saves draft JSON in `~/.aegiscourt/proposals/draft-<uuid>.json`  
  Flags:
    - `--type <type>`               Pre-select type
    - `--editor <vim|nano|code>`    Override default editor
    - `--llm-assist <none|light|full>`  LLM help level (default: light)
    - `--from-recent-logs`          Suggest motivation from recent agent failures

  Example:
  ```bash
  aegiscourt propose guide --type add-tool --llm-assist light
  # → Wizard starts → draft saved
  ```

- `propose agent-help "<short request>"`  
  Uses the main agent to generate an initial proposal draft from a brief description.  
  Automatically launches `propose guide` on the resulting draft for user refinement.  
  Example:
  ```bash
  aegiscourt propose agent-help "add a mediated way to query current UTC time without full web access"
  ```

- `propose submit <draft-uuid>`  
  Submit a finalized draft proposal created via guide/agent-help.  
  Triggers normal Court review flow.

- `court list [--status <pending|active|completed|deferred>]`  
  List proposals and their current status  
  Columns include: Mode, Human Review Status (None / Pending User Vote / Complete)

- `court view <proposal-id>`  
  Show full Governance Court output: reviewer JSONs, pros/cons, scores, NASA-style board, aggregate recommendation  
  Displays current Court mode and human review implications  
  For drafts: `court view --draft <uuid>`

- `court qa <proposal-id> <question>`  
  Ask interactive clarification question routed to one or more reviewer personas

- `court vote <proposal-id> <approve|reject|defer>`  
  Cast final user decision  
  Flags: `--notes <text>`, `--conditions <json-string>`  
  In Auto mode with `--confirm`: low-risk proposals may auto-apply after Court  
  In Assisted/Hybrid/Manual: explicit vote required after reviewing reports  
  **On approve:** The mutation engine atomically applies the proposed change:  
  1. Loads the Court result and linked draft proposal  
  2. Builds a typed patch from the draft's `proposed_change` field  
  3. Validates the patch via the type-specific handler (add-tool, change-prompt, amend-rule, etc.)  
  4. Creates a tar.gz snapshot of `~/.aegiscourt/` (excluding snapshots/ and mutations/)  
  5. Applies the mutation (e.g. registers tool in `tools.json`, writes prompt file, updates constitution rules)  
  6. Records mutation metadata + audit entry  
  On failure at any step, the engine auto-rollbacks to the snapshot and marks the mutation as failed.

### 3.4 Observability & Audit

- `status [--watch]`  
  Real-time overview: active sandboxes, resource usage, pending proposals, Court state  
  Shows current Court mode and any pending user votes  
  **Mutation info:** Displays last applied mutation (ID, type, title, applied time)  
  **Halt detection:** Shows `HALTED` state if emergency halt marker exists

- `log list [--filter <proposal-id|mutation-id|date-range>] [--export <path.jsonl>]`  
  View / export Merkle-signed audit trail entries

- `snapshot create [--name <label>]`  
  Generate frozen state tarball + SBOM (basic)

- `audit verify`  
  Recompute Merkle chain and signatures; report intact or tampered entries

### 3.5 Recovery & Maintenance

- `rollback <mutation-id | last>`  
  Revert specific (or most recent) applied mutation  
  Flags: `--all` (rollback to bootstrap), `--dry-run`  
  **Mechanism:** Restores the tar.gz snapshot captured before the mutation was applied,  
  then calls the type-specific handler's Rollback method (e.g. removes tool from registry).  
  Updates mutation status to `rolled_back` and records audit entry.

- `update [--channel <stable|edge>]`  
  Check for new kernel release and propose upgrade via Court

## 4. Example Happy Path (Hobbyist – Alex Rivera)

```bash
# 1. First run
aegiscourt init --llm ollama --profile-template hobbyist
# → Wizard recommends nemotron-3-nano → kernel bootstrapped

# 2. Start runtime
aegiscourt start

# 3. Simple task
aegiscourt agent run "What time is it right now?"

# 4. Guided proposal to improve
aegiscourt propose agent-help "add safe way to get current time without network calls"
# → Draft generated → wizard opens for refinement
# ... complete wizard ...
aegiscourt propose submit draft-abc123

# 5. Review & approve
aegiscourt court view 0007
aegiscourt court vote 0007 approve --confirm
# → Mutation mut-0007-20260317T1234 applied
# → Snapshot: ~/.aegiscourt/snapshots/snap-mut-0007-20260317T1234.tar.gz

# 6. Verify the new tool works
aegiscourt agent run "What time is it right now?"
# → Agent calls utc_time tool → returns ISO 8601 UTC timestamp

# 7. Monitor & rollback if needed
aegiscourt status
# → Last mutation: mut-0007 | add-tool | utc_time | Applied 2026-03-17T12:34:56Z
aegiscourt rollback last    # reverts snapshot + removes tool from registry
aegiscourt log list --filter 0007
```

## 5. Non-Functional Requirements
- Startup latency: <1 second
- Court round-trip target: ≤60 seconds on consumer hardware
- Output: ANSI tables for humans, clean JSON for `--json`
- Logging: Every mutating command → signed audit entry
- Help: rich `--help` per subcommand with examples
- Error codes: Exit 1 + descriptive message + constitution rule reference
- Drafts and proposals conform to `pkg/proposal/schema.json`

## 6. Future Evolution (Phase 3+)
- TUI / web UI wrapper
- Multi-user / team signoff with cryptographic keys
- Notification hooks (webhook, email)
- Proposal templates gallery
- External skill verification

See PRD.md §5.4 for high-level summary embedded in product requirements.
