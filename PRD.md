---
title: AegisCourt Product Requirements Document
version: 0.5 (v0.2 real-features scope)
date: 2026-03-17
status: v0.2 Implemented
---

**Lean PRD – AegisCourt: Constitutional Self-Evolving Agent Framework (v0.2 – Real Seed Kernel + Guided Evolution)**

### 1. Executive Summary / Product Vision
AegisCourt is a paranoid-by-design, open-source agentic framework. It starts with a minimal, cryptographically signed, immutable constitutional seed kernel enforcing strict isolation. Agents and users propose changes (behaviors, tools, prompts, rules) only through an always-active Governance Court that simulates enterprise-grade review.

The Court delivers multi-viewpoint analysis, pros/cons, scoring against the user's About Me profile, interactive Q&A, and a NASA-style board. The single user makes the final decision. Every mutation is reversible, auditable, and bounded. No stubs or simulations — if a feature appears in CLI/docs, it works end-to-end.

**Core Value Proposition (v0.2):** Deliver powerful, local-first agentic autonomy with strong isolation + governance — install in <60 seconds, evolve safely via guided proposals. Hobbyists get safe experimentation; scales toward enterprise via future hardening.

### 2. Problem Statement & Opportunity
(Unchanged — OpenClaw gaps in isolation/governance → AegisCourt fixes with invariants.)

### 3. Goals, Objectives & Success Metrics (KPIs)
**Primary Goal (v0.2):** Ship seed kernel + real Governance Court + guided proposal tools + basic evolution loop.

**Success Metrics (v0.2 targets):**
- Security: Zero critical isolation violations in dogfood/red-team probes
- Trust: Court decisions transparent & defensible (user feedback ≥90%)
- Usability: <5 min setup; <60 sec Court round-trip; guided proposal <10 min end-to-end
- Safety: All mutations reversible; no host compromise vectors
- Evolution: Measurable improvement after approved changes (e.g. +10–20% on repeated task types)
- Reliability: Primary LLM outputs parseable JSON ≥98% on reviewer/proposal tasks

### 4. Target Users & Personas
(Unchanged — Alex Rivera default; fits single-user v0.2 perfectly.)

### 5. Key Features / Functional Requirements (v0.2 MVP)

#### 5.1 Design Principles & Security Model
(Unchanged — least privilege, ephemeral, cryptographic immutability, bounded autonomy, defense-in-depth.)

#### 5.2 Key User Stories

US-001 to US-013 unchanged.

**US-014 (New – Guided Proposal Creation)**
As Alex Rivera (or any user), I can use guided tools to create high-quality proposals without writing raw JSON/diffs from scratch, so that my changes are well-reasoned, constitution-aligned, and likely to pass Court review.

Acceptance:
- `propose guide` interactive wizard collects all required fields (motivation, change, impact, risks, rollback, validation)
- Optional LLM assist (light/full) refines sections using primary model
- `propose agent-help "<short desc>"` generates draft from brief input → opens wizard
- Drafts saved as JSON → submittable via `propose submit`
- Wizard questions adapt to court.mode (stricter in Manual)
- All LLM calls logged/mediated

#### 5.3 First-Run Onboarding Flow
1. Binary launch → paranoid welcome
2. Resource detection (RAM/GPU) + LLM selector:
   - Default/recommended: nemotron-3-nano (latest quantized variant, e.g. FP8/BF16 via Ollama) — superior instruction following, structured JSON output, reasoning for Court reviewers & proposal assist.
   - Strong fallback: llama3.2:3b-instruct (lightweight/fast)
   - Guidance shown: resource estimate + warning if nemotron + full Court likely to strain system
   - Never reduce reviewer count; offer sequential execution (--low-resource mode) if needed
3. About Me wizard (risk sliders, use-cases, persona → court.mode calibration)
4. Kernel bootstrap + self-signature
5. Demo proposal (e.g. add echo skill) → live Court (full 6 reviewers) + optional guided refinement demo

#### 5.4 Governance Court Modes & Human Review Requirements
All modes single-user in v0.2 (no multi-person signoff).

| Mode                  | Persona Mapping                  | LLM Reviewers              | Human Review (v0.2)                          | Final Decision Flow                                      | Typical Use Case                     |
|-----------------------|----------------------------------|----------------------------|----------------------------------------------|----------------------------------------------------------|--------------------------------------|
| **Hobbyist Auto**     | Alex Rivera (default)            | Full or reduced panel      | None — user sees board + recommendation      | User vote (approve/reject/defer); auto-apply low-risk with --confirm | Solo hobbyist                        |
| **Indie Assisted**    | Jordan Hale                      | Full panel                 | Required — user reads reports, asks Q&A      | Explicit user vote after review                          | Indie dev                            |
| **Team Hybrid**       | Sam Chen                         | Full panel                 | Required — user acts as all domains          | User vote after review (simulate team)                   | Small team                           |
| **Enterprise Manual** | Dr. Lena Moreau                  | Full panel as evidence     | Required — user must carefully review/vote   | Explicit vote; stricter thresholds                       | Regulated pilot                      |

**v0.2 Note:** Hybrid/Manual simulate "human review" by forcing explicit user vote after reading reports. Real multi-domain multi-user signoff deferred to v1.0.

#### 5.5 CLI Interface (Operator-Facing)
(Unchanged core design goals.)

**High-Level Command Structure** (additions highlighted)

**Governance & Evolution additions:**
- `propose guide [--type <type>] [--llm-assist <none|light|full>] ...` — interactive wizard
- `propose agent-help "<short request>"` — agent-generated draft → wizard
- `propose submit <draft-uuid>` — submit finalized draft

**Mutation application (v0.2 real):**
- `court vote <id> approve` — triggers atomic mutation: snapshot → validate → apply → audit
- `rollback <mutation-id | last>` — restores snapshot, reverts handler state
- `halt` — emergency rollback + HALTED marker
- `status` — shows last mutation, halt state, Court mode

(Full list otherwise matches previous cli-design.md — no `court signoff` in v0.2.)

### 6. High-Level Architecture
(Unchanged diagram and components.)

### 7. Initial Constitutional Ruleset v0.1
(Unchanged — 10 rules.)

### 8. Non-Functional Requirements
**Security:** Sandbox escape protection (gVisor primary)
**Observability:** Audit log + basic status/watch
**Resource Constraints:** <4 GB baseline (single-reviewer fallback); <12 GB full Court
**Performance:** Court ≤60 sec; guided wizard responsive
**Maintainability:** Versioned in Git; SBOM in snapshots

### 9. Scope (In / Out for v0.2)
**In:**
- Seed kernel, strict isolation (gVisor), always-on Governance Court
- LLM setup (Nemotron-3-Nano default), About Me + mode calibration
- Basic + guided self-mod proposals (wizard/agent-help)
- Local execution, reversible mutations, tamper-evident audit
- One-shot agent tasks, mediated tools (e.g. echo + basic web_search)
- Single-user graduated modes (explicit vote in stricter modes)
- **Atomic mutation application** — Court approve triggers real apply with snapshot rollback
- **Dynamic tool registry** — approved add-tool mutations immediately available to agent
- **Type-specific handlers** — add-tool, change-prompt, amend-rule, add-skill, memory, generic
- **Snapshot-based rollback** — tar.gz of full state before each mutation; `rollback` restores

**Out (deferred to v1.0+):**
- Multi-user / cryptographic signoff / quorum
- TUI/web UI
- Webhook/email notifications (file append only in v0.2)
- External skill verification / marketplace
- Vector memory proposals
- Full NIST compliance report generator
- Kubernetes-native

### 10. Assumptions, Risks & Dependencies
(Unchanged + add risk: "LLM JSON reliability — mitigated by Nemotron-3-Nano default.")

### 11. Roadmap & Milestones
- **v0.2 (Now–Q2 2026):** Real seed kernel + isolation + Court + guided proposals + basic evolution; OSS release.
- **v1.0 (Q3–Q4 2026):** Multi-user signoff, notifications, compliance exports, zero-trust hardening.

### 12. Open Questions & Next Steps
- Finalize primary Nemotron tag for Ollama (e.g. nemotron-3-nano:30b-reasoning or 4b/8b quantized)
- ~~Prototype guided wizard UX (prompt flow, assist levels)~~ → Implemented in v0.2
- ~~Dogfood first guided proposal → web_search tool~~ → Dogfooded utc_time tool end-to-end in v0.2
- Extend mutation handlers for more complex types (multi-file changes, LLM config swaps)

Link: See `docs/cli-design.md` for detailed command specs.
