# AegisCourt CLI Design & Specification
**Version:** 0.1-draft  
**Status:** Proposed
**Goal:** Define a secure, minimalist, Unix-inspired CLI that serves as the primary (and initially only) operator interface for AegisCourt. This document lives alongside PRD.md §5.4 and will evolve via Court-approved changes.

## 1. Design Principles
- Unix / git / docker / kubectl style: `aegiscourt <subcommand> [args]`
- Paranoid-by-default: confirmations for mutations, verbose risk messaging, full audit logging
- Progressive disclosure: simple defaults for hobbyists; flags and deep subcommands for enterprise / scripting
- Mode-aware behavior: output, prompts, and confirmation requirements adapt to active Court mode (set during onboarding or via config)
- Human-first output: readable tables/lists by default; `--json` for automation
- Errors reference constitution rules (e.g. "Blocked: Rule 3 – Unauthorized host access")
- Every state-changing command emits signed audit entry
- Lightweight: <1s startup, no heavy runtime deps beyond kernel
- Cross-platform: Linux primary (gVisor), macOS/Windows via seccomp / Docker fallback

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
- `--mode-info`           Show current Court mode + human review implications for this session
- `--help` / `-h`          Show help (rich per-subcommand help + examples)

## 3. Subcommands by Category

### 3.1 Setup & Configuration
- `init`  
  Interactive first-run wizard: kernel bootstrap, LLM selection, About Me profile  
  Flags:
    - `--llm <ollama|openai|anthropic|grok|...>`
    - `--profile-template <hobbyist|indie|enterprise|financial>`  
    - `--court-mode <auto|assisted|hybrid|manual>` (overrides wizard default; hobbyist auto if omitted)
  Output: signed kernel hash, setup summary

- `config get <key>`  
  `config set <key> <value>`  
  `config list`  
  View / edit runtime configuration (LLM endpoint, risk sliders, etc.)  
  Sensitive changes route through Court
  Keys supported:
   - `court.mode`                (auto|assisted|hybrid|manual)
   - `court.pending_human_signoffs` (read-only view of outstanding approvals)
  Changing `court.mode` to a stricter level (e.g. auto → manual) requires Court proposal & approval.

### 3.2 Runtime Control
- `start [--detached] [--resources <ram=4GB,cpu=2>] [--sandbox <gvisor|seccomp>]`  
  Launch kernel + agent runtime

- `stop`  
  Graceful shutdown of all sandboxes and kernel

- `agent run <task description>`  
  One-shot agent execution in ephemeral sandbox  
  Flags: `--agent-id <uuid>`, `--timeout <30s>`, `--output-format <text|json>`

- `halt [--no-confirm]`  
  Emergency: freeze all agents, rollback last mutation, enter read-only mode

### 3.3 Governance & Evolution
- `propose <type> <name>`  
  Submit self-modification proposal  
  Types: `add-tool`, `add-skill`, `change-prompt`, `amend-rule`, `add-reviewer`, etc.  
  Flags: `--description <text>`, `--diff-file <path>`, `--impact-assessment <text>`, `--benchmark-plan <text>`  
  Output: Proposal ID, immediate Court start notification

- `court list [--status <pending|active|completed|deferred>]`  
  List proposals and their current status
  Additions in output table:
    - Column: Mode
    - Column: Human Review Status (None / Pending / Partial / Complete)

- `court view <proposal-id>`  
  Show full Governance Court output: reviewer JSONs, pros/cons, scores, NASA-style board, aggregate recommendation
  - Display current Court mode at top
  - Human review section:
    - In Hobbyist Auto: "No mandatory human review – your vote is final."
    - In Indie Assisted: "Review reports below; single-user approval required."
    - In Team Hybrid: "Assigned reviewers: CISO → @samchen, MRM → @platform-lead"
    - In Enterprise Manual: "Multi-signature required: CISO, MRM, Compliance. Use court signoff <id> --domain <ciso|mrm|...> --user <handle>"
  - Visual NASA board now includes human sign-off indicators (e.g. ✅ / ⏳ per domain)

- `court qa <proposal-id> <question>`  
  Ask interactive clarification question routed to one or more reviewer personas

- `court signoff <proposal-id> --domain <ciso|mrm|compliance|ethics|sre|helpfulness> [--notes <text>]`
  - Used in Hybrid / Manual modes to record human approval for a specific reviewer domain
  - Requires authentication context if multi-user (future Phase 2; MVP uses CLI user identity or --user flag)
  - Logs signed entry; proposal advances only when required domains are signed

- `court vote <proposal-id> <approve|reject|defer>`  
  Cast final user decision  
  Flags: `--notes <text>`, `--conditions <json-string>`
  - Hobbyist Auto: vote applies immediately (unless overridden by conditions)
  - Assisted / Hybrid / Manual: vote only finalizes after all required human signoffs are collected
  - New flag: `--as-domain <ciso|mrm|...>` (in Hybrid/Manual; records vote as that reviewer)

### 3.4 Observability & Audit
- `status [--watch]`  
  Real-time overview: active sandboxes, resource usage, pending proposals, Court state  
  Output additions:
    - Current Court mode
    - Pending human sign-offs (if any) with domains/personas (e.g. "Awaiting CISO & MRM approval – Proposal 0012")
    - Mode-specific notes (e.g. "Hobbyist Auto: one-click approve enabled for low-impact changes")

- `log list [--filter <proposal-id|mutation-id|date-range>] [--export <path.jsonl>]`  
  View / export Merkle-signed audit trail entries

- `snapshot create [--name <label>] [--enterprise]`  
  Generate frozen state tarball + SBOM + regulatory mapping export

- `--pending-signoff` (show only proposals awaiting human approval)

### 3.5 Recovery & Maintenance
- `rollback <mutation-id | last>`  
  Revert specific (or most recent) applied mutation  
  Flags: `--all` (rollback to bootstrap), `--dry-run`

- `update [--channel <stable|edge>]`  
  Check for new kernel release and propose upgrade via Court

## 4. Example Happy Path (Alex Rivera – Hobbyist)

```bash
# 1. First run
aegiscourt init --llm ollama --profile-template hobbyist
# → Welcome wizard → "Kernel bootstrapped. Self-hash: ed25519:abc123…"

# 2. Start runtime
aegiscourt start

# 3. Run simple task
aegiscourt agent run "List my top 3 open GitHub issues"

# 4. Propose improvement after noticing repeated need
aegiscourt propose add-tool "web_search" \
  --description "Mediated search via user-configured API" \
  --diff-file ./proposals/web_search_v1.json

# → Proposal ID: 0007. Court reviewing… Complete in 38 seconds.

# 5. Review & interact
aegiscourt court view 0007
aegiscourt court qa 0007 "How are we preventing prompt injection via search results?"
aegiscourt court vote 0007 approve --confirm --notes "Mitigations look solid; adding length cap myself"

# 6. Verify
aegiscourt status
aegiscourt log list --filter 0007

# 7. Panic button if something feels off
aegiscourt halt
```

## 4.1 Example – Enterprise Manual Mode (Dr. Lena Moreau persona)
```bash
# Assume already in Manual mode
aegiscourt propose add-tool "external_api" --description "..."

# Court runs → Proposal ID: 0042

aegiscourt court view 0042
# → Shows full reports + "Pending: CISO, MRM, Compliance sign-off"

# CISO signs off
aegiscourt court signoff 0042 --domain ciso --notes "Syscall filters added; acceptable risk"

# Later, after all signoffs
aegiscourt court vote 0042 approve --notes "Board consensus reached"
```

## 5. Non-Functional Requirements
- Startup latency: <1 second (kernel already running checks are near-instant)
- Court round-trip target: ≤45 seconds on consumer hardware (shared with PRD KPI)
- Human sign-off latency: tracked separately; no hard timeout in MVP (escalation via notification hooks Phase 2)
- Output: ANSI tables for humans, clean JSON for `--json`
- Logging: Every mutating command → signed audit entry before + after
- Help: cobra-style rich help with examples per subcommand
- Error codes: Exit 1 + descriptive message + constitution rule reference when applicable

## 6. Future Evolution (Phase 2+)
- TUI / web UI wrapper (`aegiscourt ui`)
- Multi-user / team mode (`--user <id>`)
- Notification hooks for pending sign-offs (email/slack/webhook)
- Role-based access control for signoff domains
- Custom CLI extensions as Court-approved skills
- Scripting helpers (`aegiscourt script generate`)

See PRD.md §5.4 for high-level summary embedded in product requirements.

Next: Prototype using urfave/cli or spf13/cobra in Go; validate UX with Alex Rivera & Jordan Hale personas.
