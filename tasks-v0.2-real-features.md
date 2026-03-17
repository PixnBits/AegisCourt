# Tasks – v0.2 Real Features
**Status:** Active / Implementation Target  
**Scope:** This file lists **only** the must-have, end-to-end working tasks for v0.2 release.  
No stubs, no partial features, no multi-user/crypto signoff, no TUI, no webhook, no external plugins, no full NIST reports, no Kubernetes operator.  
Everything here must ship complete and usable by a single user in Hobbyist Auto / Indie Assisted modes.

**Replaces:** Old tasks.md (archived or deleted after migration)

**Estimated effort:** ~8–12 weeks single dev (with strong LLM coding assistance in 2026)

## Phase 1: Kernel & Isolation Basics (Weeks 1–2)

1. Bootstrap Go kernel binary with self-signature (Ed25519) verification on startup  
   - Panic if signature invalid  
   - Store root public key in embedded constant

2. Implement append-only Merkle-signed audit log  
   - Each entry: UUIDv7, timestamp, prev hash, payload hash, signature  
   - `audit verify` command recomputes chain & reports tampering

3. Sandbox Manager with gVisor (Linux primary)  
   - Spawn ephemeral sandboxes for agent + each reviewer call  
   - Mediate all I/O (file/network/process) via kernel proxy  
   - Enforce cgroupv2 limits (RAM/CPU from `--resources`)  
   - Low-RAM fallback: reduce reviewers to 1–2

4. LLM Router basics  
   - Configurable endpoint (Ollama default)  
   - Primary model: nemotron-3-nano (latest quantized tag)  
   - Fallback: llama3.2:3b-instruct  
   - Log every prompt/response (audit entry)

5. Basic agent loop (ephemeral sandbox)  
   - `agent run <task>` → spawn sandbox, route to primary LLM, mediate tool calls  
   - Support echo tool (trivial mediated output)

## Phase 2: Governance Court Engine (Weeks 3–4)

6. Load reviewer personas from embedded .md files (CISO, MRM, Compliance, Ethics, SRE, Helpfulness)  
   - Each has strict JSON output schema (score, concerns, pros, cons, mitigations, recommendation)

7. Court orchestration  
   - Parallel LLM calls to reviewers (or sequential on low RAM)  
   - Weighted aggregate score (configurable per mode)  
   - NASA board text/table generation  
   - Deferral timer (profile-based, background check)

8. Court CLI commands  
   - `court list` — pending/active/completed  
   - `court view <id>` — default clean summary + board  
   - `court view <id> --detailed` — per-reviewer breakdown with line-broken pros/cons/mitigations  
   - `court view <id> --reviewer <persona>` — full JSON + raw reasoning for one  
   - `court view <id> --json` — raw output  
   - `court qa <id> <question>` — route to reviewers  
   - `court vote <id> approve|reject|defer [--notes] [--conditions] [--confirm]`

9. Mode calibration  
   - Auto: low-risk auto-apply with `--confirm`  
   - Assisted/Hybrid/Manual: force explicit vote after view  
   - Set via `config set court.mode` (lightweight Court gate if stricter)

## Phase 3: Guided Proposal Creation (Weeks 5–6)

10. Draft JSON schema & storage  
    - Fields: id, type, title, motivation, change (diff/text), impact, risks, rollback, validation, llm_assist_level, etc.  
    - Save/load from `~/.aegiscourt/proposals/draft-<uuid>.json`

11. `propose agent-help "<short desc>"`  
    - Prompt primary LLM to generate draft JSON + reasoning  
    - Auto-launch `propose guide --draft <uuid>`

12. `propose guide` interactive wizard  
    - Step-by-step prompts (type, title, motivation, change, impact, risks, rollback, validation)  
    - Multi-line editor for change/rollback/validation  
    - Optional LLM assist (light/full) on any section  
    - Adapt questions by court.mode (stricter risk checks in Manual)  
    - Save draft on completion

13. `propose submit <draft-uuid>`  
    - Validate required fields  
    - Create real proposal → trigger Court

## Phase 4: Mutations, Tools, Observability & Polish (Weeks 7–8+)

14. Mutation application & rollback  
    - Git-like diffs (JSON patch + code/rules)  
    - Atomic apply/rollback with audit entries  
    - `rollback <id|last>`

15. Basic mediated tool: utc_time (kernel clock proxy)  
    - Read-only ISO timestamp  
    - No network, strict output cap

16. Post-approval regression flagging  
    - Run 3–5 canned bench tasks (repo/bench/)  
    - Compare success delta  
    - Flag in `status` if > threshold drop → suggest rollback

17. File-based notifications  
    - Append-only `~/.aegiscourt/notifications.jsonl` for proposal events

18. Resource & status polish  
    - `status --watch` real-time sandboxes, resources, pending votes  
    - Low-RAM detection & reviewer reduction

19. Dogfood & acceptance tests  
    - Run full example-proposal-flow.md scenario  
    - Verify rich Court reasoning visible & parseable  
    - Test rollback restores state  
    - Measure: setup <5 min, Court <60 s, guided proposal <10 min

20. Documentation & release prep  
    - Finalize README quick-start  
    - Update PRD.md, cli-design.md, example-proposal-flow.md  
    - Tag v0.2.0 + archive PRD snapshot

## Success Criteria for v0.2 Release
- Zero sandbox escapes in probes
- Court decisions traceable & defensible
- Guided proposals produce high-quality, approvable changes
- All CLI commands in docs/cli-design.md work end-to-end
- Single-user modes behave as documented (no multi-user illusion)

Deferred to v0.5/v1.0: multi-user signoff, webhook, TUI, vector memory, skill manifests, NIST reports, k8s operator, etc.

This list is prioritized top-to-bottom. Dependencies are minimal within phases.
