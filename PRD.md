---
title: AegisCourt Product Requirements Document
version: 0.3
date: 2026-03-14
status: Draft – Architect-reviewed
---

**Lean PRD – AegisCourt: Constitutional Self-Evolving Agent Framework**

### 1. Executive Summary / Product Vision
AegisCourt is a paranoid-by-design, open-source agentic framework. It begins with a minimal, cryptographically signed, immutable constitutional seed kernel that enforces strict isolation between agents, skills, and the host system. Agents can propose self-modifications (new behaviors, memory schemas, tool integrations) but **only** through an always-active Governance Court that simulates full enterprise review (CISO, MRM, Compliance, etc.).

The Court presents multi-viewpoint analysis, pros/cons, objective scoring against the user's "About Me" profile, interactive Q&A with simulated persona-agents, and a NASA-style all-hands go/no-go board. The user (even a solo hobbyist) acts as the final decision-maker. Every control remains on; deferrals require documented justification and automatic escalation.

**Core Value Proposition:** Deliver OpenClaw-level agentic power with Tier-0 financial-institution-grade security and governance — from a single <60-second local install. The same seed kernel supports hobbyist experimentation at home and scales (via guided evolution) to production zero-trust environments.

### 2. Problem Statement & Opportunity
OpenClaw demonstrated powerful autonomous agents but suffered from severe security gaps: lack of isolation, unvetted plugins, prompt injection, and privilege escalation. These issues eroded trust in agentic systems, especially in regulated sectors like finance.

**Opportunity:** Build from first principles with isolation and governance as invariants. Hobbyists gain safe, evolving agents; enterprises get auditable, bounded autonomy that maps directly to NIST AI Agent Standards and financial risk frameworks. No marketplace — focus purely on core kernel + user-controlled evolution.

### 3. Goals, Objectives & Success Metrics (KPIs)
**Primary Goal:** Launch a seed kernel + Governance Court that enforces isolation and trust at every layer.

**Success Metrics:**
- Security: Zero critical isolation/privilege violations in red-team audits (first 6 months).
- Trust: ≥95% of Governance Court decisions rated transparent & defensible in user feedback.
- Usability: <5 min setup (including LLM selection); <45 sec average Court review.
- Safety: All self-modifications reversible via one-click rollback; no host compromise vectors.
- Evolution: Agents show measurable improvement (e.g., +15% task success after 10 approved changes) without unsafe drift.
- Enterprise alignment: ≥85% coverage of NIST agent governance pillars.

### 4. Target Users & Personas
(Unchanged from v0.1 – the four personas still fit perfectly, with Alex Rivera as the default starting point.)


### 5. Key Features / Functional Requirements (MVP)
(Expanded with new sub-sections)

#### 5.1 Design Principles & Security Model
- Least Privilege + Ephemeral Everything: every agent/skill runs in its own isolated sandbox; zero shared filesystem/memory unless Court-approved and mediated.  
- Cryptographic Immutability: kernel + constitution + every mutation signed and versioned in an append-only log.  
- Bounded Autonomy: no action that touches host, network, or persistent storage without explicit Court go/no-go.  
- Defense-in-Depth: sandbox escape → kernel kill switch; LLM prompt injection → reviewer cross-check.  
- Threat Model Summary: Primary threats = prompt injection, sandbox escape, memory poisoning, supply-chain LLM backdoors, rogue self-modification. Mitigations are baked into every layer.

#### 5.2 Key User Stories
US-001: As Alex Rivera, I complete first-run setup (LLM selection + About Me) in <5 min.  
Acceptance: Ollama/local or cloud selection with supply-chain guidance; profile saved and used to calibrate Court thresholds.  

US-002: As any user, every proposed self-mod triggers Governance Court in <45 sec.  
Acceptance: Simulated reviewers, pros/cons, Q&A, NASA board, deferral path, immutable log entry.  

US-003: As hobbyist, I can safely add a new tool/skill that is isolated and reversible.  
Acceptance: Court approval required; sandboxed execution; one-click rollback if issues detected.

#### 5.3 First-Run Onboarding Flow
1. Binary/Docker launch → “Welcome to AegisCourt – paranoid mode always on.”  
2. LLM selector (default suggestion: nemotron-3-nano via Ollama).  
3. About Me wizard (risk tolerance slider, use-cases, deferral prefs).  
4. Kernel bootstrap + cryptographic self-signature.  
5. First test proposal (e.g., “add echo skill”) → live Governance Court demo.

### 6. High-Level Architecture
```
Kernel (immutable, signed)
├── Sandbox Manager (gVisor / firejail / seccomp namespaces – MVP choice: gVisor for strong isolation)
├── LLM Router (routes main agent vs. reviewer personas; supports multi-LLM)
├── Governance Court Engine (stateless, prompt-orchestrated reviewers)
├── Audit & Rollback Store (append-only, Merkle-tree style)
└── Agent Runtime (ephemeral containers; mediated I/O only)
```

All communication through kernel-mediated channels only. No direct agent-to-agent access. Self-modifications are proposed as Git-like diffs, reviewed, then atomically applied or rolled back.

### 7. Initial Constitutional Ruleset v0.1
1. Never execute any host write, network call, or process spawn without explicit Court approval + 2-factor user confirmation.  
2. All agents/skills run in separate, memory-isolated sandboxes; no shared state unless Court explicitly creates a mediated channel.  
3. Every mutation (code, memory schema, rule) must be cryptographically signed and stored with before/after diff.  
4. Supply-chain rule: Prefer models with public weights and known provenance (nemotron-3-nano, llama3.x, gemma); flag Qwen-family for extra Court scrutiny.  
5. Human override: User can always invoke “emergency halt” that freezes all agents and rolls back last change.  
(Ruleset itself can only be changed via Court + user vote; stored in kernel.)

### 8. Non-Functional Requirements
**Security (paramount):** Sandbox escape protection verified by quarterly red-team.  
**Observability:** All Court decisions, mutations, and sandbox activity exported to local Prometheus + tamper-proof JSONL log.  
**Resource Constraints (Hobbyist-Critical):** <4 GB RAM baseline (single reviewer mode); <12 GB with full 5-reviewer Court; CPU throttling on low-power devices.  
**Performance:** Court round-trip ≤45 sec on consumer hardware.  
**Maintainability:** Constitution + kernel versioned in Git; every release ships SBOM.

### 9. Scope (In / Out for MVP)  
**In:** Seed kernel, strict isolation, always-on Governance Court, LLM selection/setup, basic self-mod, local execution, "About Me" + simulated reviewers.  
**Out:** Marketplace or community skill sharing/verification; multi-user collaboration (Phase 2+); full Kubernetes-native zero-trust (Phase 2+); managed hosting.

### 10. Assumptions, Risks & Dependencies
**Assumptions:** Ollama remains reliable for local LLM integration; frontier LLMs support reliable role-playing for reviewers.  
**Risks:**  
- Overly strict Court slows hobbyist iteration (mitigate: profile-based sliders + deferral).  
- LLM hallucination in Court reasoning (mitigate: evidence grounding, user Q&A, multi-LLM cross-check).  
- Local LLM performance bottlenecks (mitigate: guidance toward efficient models like nemotron-3-nano).
- Multi-LLM Court latency on low-end hardware (mitigate: mitigated by single-reviewer fallback mode).
**Dependencies:** User-provided LLM access (API keys or Ollama endpoint); Python/Go runtime.

### 11. Roadmap & Milestones  
- **v0.1–0.2 (Now–Q2 2026):** Seed kernel + isolation + Governance Court + LLM setup MVP; OSS release.
- **v0.5 (Q3 2026):** Guided evolution; enterprise snapshot mode.
- **v1.0 (Q4 2026):** Full Vision 3 hardening support.
- **Ongoing:** Refine constitution from real usage; track NIST updates.

### 12. Open Questions & Next Steps
- Confirm isolation tech for MVP (gVisor vs. lightweight containerd vs. firejail/seccomp)?
- Expand starter constitution to 10 rules?
- Generate seed kernel code skeleton from this architecture?
- Number of default simulated reviewer personas (start with 4–6 matching key enterprise roles)?
- Exact "About Me" questions + how they map to Court thresholds.
- First dogfood self-mod example (e.g., agent proposing its own better tool-calling prompt)?
