**Lean PRD v0.2 – AegisCourt: Constitutional Self-Evolving Agent Framework**  
**Version:** 0.2 (Incorporating security emphasis, isolation mandates, no marketplace, and LLM selection during setup – March 14, 2026)  

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

### 5. Key Features / Functional Requirements (Prioritized MVP)
**MVP Scope (Phase 1 – Seed Kernel Launch):**
- Immutable constitutional seed kernel (read-only bootstrap; cryptographically signed).
- Strict isolation enforcement:
  - Each agent/skill runs in its own ephemeral, hardened sandbox (e.g., seccomp + namespaces + minimal privileges).
  - No shared memory/state between agents/skills unless explicitly mediated & approved via Court.
  - Kernel mediates all host interactions (file, network, process) with least-privilege enforcement.
- "About Me" onboarding wizard (risk profile, use-case, deferral preferences) — used to calibrate Court strictness.
- Governance Court core (always on):
  - Simulated reviewer agents present viewpoints (CISO: "Isolation breach risk?", MRM: "Drift impact?", etc.).
  - Evidence-based pros/cons + scored recommendation (0–100 vs. profile).
  - Interactive Q&A with persona-agents.
  - NASA-style go/no-go board: thumbs per role + user final vote.
  - Deferral: Timed hold with justification log + auto-escalation (e.g., after 24h → higher threshold or reject).
- LLM selection during first-run setup:
  - Prompt user to select one or more primary LLMs for kernel reasoning & reviewer simulation.
  - Support: Cloud APIs (e.g., Grok, Claude, OpenAI) + local via Ollama (user provides endpoint or model tag).
  - Guidance in UI/CLI: "For minimal supply-chain risk, prefer models like nemotron-3-nano (NVIDIA), llama3.1/3.2/3.3 (Meta), or gemma (Google). Avoid or scrutinize Qwen variants due to reported jailbreak & supply-chain concerns in 2026 analyses."
  - Allow multiple LLMs (e.g., one for main agent, others for specific reviewers).
- Basic self-modification: Add/update isolated skills/tools, memory schema changes — all reversible.
- Immutable audit trail (tamper-proof log of every proposal, Court output, decision).
- Local-first install (<60 sec via single binary/Docker).

**Phase 2 (Guided Evolution to Vision 3):**
- Agent proposes hardening steps (e.g., container/pod isolation, mTLS, policy-as-code).
- Court reviews & applies approved changes progressively.
- Enterprise mode: Certified snapshots, SBOM export, regulatory mappings.

**Non-Functional Requirements:**
- **Security (paramount)**: Cryptographic verification of kernel & mutations; ephemeral sandboxes per agent/skill; no default outbound network; host privilege separation; mandatory bounds on agent capabilities.
- **Performance**: Court review <45 sec on consumer hardware (optimize reviewer prompts).
- **Reliability**: All changes atomic & rollback-able; graceful failure on isolation breach.
- **Accessibility**: CLI-first + optional minimal web UI for Court visualization.
- **Open-source**: MIT/Apache 2.0; no marketplace dependencies.

### 6. Scope (In / Out for MVP)
**In:** Seed kernel, strict isolation, always-on Governance Court, LLM selection/setup, basic self-mod, local execution, "About Me" + simulated reviewers.  
**Out:** Marketplace or community skill sharing/verification; multi-user collaboration (Phase 2+); full Kubernetes-native zero-trust (Phase 2+); managed hosting.

### 7. Assumptions, Risks & Dependencies
**Assumptions:** Ollama remains reliable for local LLM integration; frontier LLMs support reliable role-playing for reviewers.  
**Risks:**  
- Overly strict Court slows hobbyist iteration (mitigate: profile-based sliders + deferral).  
- LLM hallucination in Court reasoning (mitigate: evidence grounding, user Q&A, multi-LLM cross-check).  
- Local LLM performance bottlenecks (mitigate: guidance toward efficient models like nemotron-3-nano).  
**Dependencies:** User-provided LLM access (API keys or Ollama endpoint); Python/Go runtime.

### 8. Roadmap & Milestones
- **v0.1–0.2 (Now–Q2 2026):** Seed kernel + isolation + Governance Court + LLM setup MVP; OSS release.
- **v0.5 (Q3 2026):** Guided evolution; enterprise snapshot mode.
- **v1.0 (Q4 2026):** Full Vision 3 hardening support.
- **Ongoing:** Refine constitution from real usage; track NIST updates.

### 9. Open Questions & Next Steps
- Initial default constitution ruleset (e.g., "Never allow cross-agent memory sharing without explicit Court approval & isolation boundary")?  
- Number of default simulated reviewer personas (start with 4–6 matching key enterprise roles)?  
- Preferred isolation tech in MVP (e.g., gVisor sandbox, firejail, or lightweight containers)?  
- Exact "About Me" questions + how they map to Court thresholds.  
- First dogfood self-mod example (e.g., agent proposing its own better tool-calling prompt)?
