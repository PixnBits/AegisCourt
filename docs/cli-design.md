# AegisCourt CLI Design & Specification
**Version:** 0.1-draft  
**Status:** Proposed – for v0.2 CLI MVP  
**Date:** March 2026  
**Goal:** Define a secure, minimalist, Unix-inspired CLI that serves as the primary (and initially only) operator interface for AegisCourt. This document lives alongside PRD.md §5.4 and will evolve via Court-approved changes.

## 1. Design Principles
- Unix / git / docker / kubectl style: `aegiscourt <subcommand> [args]`
- Paranoid-by-default: confirmations for mutations, verbose risk messaging, full audit logging
- Progressive disclosure: simple defaults for hobbyists; flags and deep subcommands for enterprise / scripting
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
- `--help` / `-h`          Show help (rich per-subcommand help + examples)

## 3. Subcommands by Category

### 3.1 Setup & Configuration
- `init`  
  Interactive first-run wizard: kernel bootstrap, LLM selection, About Me profile  
  Flags: `--llm <ollama|openai|anthropic|grok|...>`, `--profile-template <hobbyist|indie|enterprise|financial>`  
  Output: signed kernel hash, setup summary

- `config get <key>`  
  `config set <key> <value>`  
  `config list`  
  View / edit runtime configuration (LLM endpoint, risk sliders, etc.)  
  Sensitive changes route through Court

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

- `court view <proposal-id>`  
  Show full Governance Court output: reviewer JSONs, pros/cons, scores, NASA-style board, aggregate recommendation

- `court qa <proposal-id> <question>`  
  Ask interactive clarification question routed to one or more reviewer personas

- `court vote <proposal-id> <approve|reject|defer>`  
  Cast final user decision  
  Flags: `--notes <text>`, `--conditions <json-string>`

### 3.4 Observability & Audit
- `status [--watch]`  
  Real-time overview: active sandboxes, resource usage, pending proposals, Court state  
  Output: formatted table

- `log list [--filter <proposal-id|mutation-id|date-range>] [--export <path.jsonl>]`  
  View / export Merkle-signed audit trail entries

- `snapshot create [--name <label>] [--enterprise]`  
  Generate frozen state tarball + SBOM + regulatory mapping export

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

## 5. Non-Functional Requirements
- Startup latency: <1 second (kernel already running checks are near-instant)
- Court round-trip target: ≤45 seconds on consumer hardware (shared with PRD KPI)
- Output: ANSI tables for humans, clean JSON for `--json`
- Logging: Every mutating command → signed audit entry before + after
- Help: cobra-style rich help with examples per subcommand
- Error codes: Exit 1 + descriptive message + constitution rule reference when applicable

## 6. Future Evolution (Phase 2+)
- TUI / web UI wrapper (`aegiscourt ui`)
- Multi-user / team mode (`--user <id>`)
- Custom CLI extensions as Court-approved skills
- Scripting helpers (`aegiscourt script generate`)

See PRD.md §5.4 for high-level summary embedded in product requirements.

Next: Prototype using urfave/cli or spf13/cobra in Go; validate UX with Alex Rivera & Jordan Hale personas.
