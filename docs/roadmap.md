# AegisCourt Roadmap
**Status:** Living Document  
**Last updated:** March 2026  
**Goal:** Provide a clear, phased view of AegisCourt development — from the v0.2 seed release to the v1.0 zero-trust horizon and beyond.

This roadmap is **not** a commitment list; it is a directional guide. Major scope changes require Court proposals and user vote.

## Release Cadence Philosophy
- **v0.2** — real, single-user foundation (local-first, paranoid, guided evolution)  
- **v0.5** — enterprise foundations (multi-user basics, compliance artifacts)  
- **v1.0** — production-grade zero-trust hardening + ecosystem stubs  
- **v2+ horizon** — managed evolution, community registry maturity, formal proofs

Releases are tagged when a coherent, dogfood-tested set of features is ready. No fixed dates — progress-driven.

## v0.2 – Seed Kernel + Guided Evolution (Active / Targeting Q2 2026)

**Focus:** Ship a trustworthy, local-first agent executor with strong isolation, always-on governance, and safe self-evolution tools. Single-user only. No stubs or vaporware.

**Key Deliverables**
- Immutable kernel + self-signature
- gVisor-based isolation (Linux primary, fallbacks warned)
- LLM router with Nemotron-3-Nano default
- Governance Court (4–6 reviewers, mode-calibrated depth)
- CLI: init, start/stop, agent run, propose (basic + guide/agent-help/submit), court list/view/qa/vote, status/watch, log export, rollback, halt
- Guided proposal creation (wizard + agent-draft)
- Basic mediated tools (echo + simple web_search)
- Reversible mutations + tamper-evident Merkle audit log
- Resource bounding + low-RAM fallback
- File-based notifications append (~/.aegiscourt/notifications.jsonl)
- Basic post-approval regression flagging (no auto-rollback)

**Success Gates**
- Zero sandbox escapes in dogfood probes
- Court round-trip ≤60 s on consumer hardware
- Guided proposal end-to-end <10 min
- ≥98% parseable JSON from reviewers/proposal assist
- First guided proposal → approved web_search tool works

## v0.5 – Enterprise Foundations & Compliance Artifacts (Q3 2026 Target)

**Focus:** Add multi-user basics, observability, and exportable compliance evidence without compromising invariants.

**Key Features (Deferred from v0.2)**
- Cryptographic per-user signoff (Ed25519 keys, domain assignment)
- Pending signoff state machine + `court signoff` command
- Basic webhook notifications for Court events
- Enterprise snapshot: Merkle proofs + basic SBOM + NIST mapping table
- Post-approval automated benchmarks + regression detection → suggest rollback
- Proposal export/import (tamper-evident JSON for team sharing)
- Prometheus metrics endpoint (Court throughput, mutation success, alerts)

**Success Gates**
- Multi-user signoff demo (2–3 domains, one user per domain)
- Snapshot verifiable externally (proof check script)
- ≥85% NIST AI RMF pillar mapping in generated report

## v1.0 – Zero-Trust Production Hardening (Q4 2026 Target)

**Focus:** Full zero-trust deployment readiness + ecosystem foundations.

**Key Features**
- Kubernetes operator: CRDs (Kernel, AgentTask, Proposal, CourtReview), gVisor pods, network policies
- Multi-domain quorum + configurable approval workflows
- TUI preview (bubbletea-based dashboard for status/Court/logs)
- Vector memory proposal flow (local HNSW stub)
- Verified skill manifest install/verify (local path only)
- Community reviewer persona + proposal templates gallery
- Continuous red-team harness skeleton + fuzzing suite
- Full NIST AI Agent Standards ≥95% coverage claim (with evidence)
- Formal verification notes for core invariants (isolation, mediation, reversibility)

**Success Gates**
- Single-node k8s deployment (minikube) passes smoke tests
- Red-team suite blocks 100% critical vectors
- ≥95% pillar coverage verifiable in compliance report

## v2+ Horizon (2027+)

**Longer-term Directions**
- Managed evolution: agent proposes → Court auto-refines → user high-impact approve only
- Community skill registry foundations (IPFS pinning, signed manifests sharing)
- Reputation & discovery signals for shared skills (post-v1)
- On-chain hash commitments (optional, user-controlled)
- Advanced memory schemas (vector + graph hybrid proposals)
- Formal verification (TLA+ or similar for key invariants)
- Edge-device optimizations + offline-first modes

## Deferred / Out of Scope Indefinitely
- Centralized marketplace (contradicts user sovereignty)
- Mandatory cloud hosting / SaaS version
- Auto-apply without user vote (even in Auto mode for high-risk changes)
- Weakening core rules (1–5) via amendments

## Related Documents
- `PRD.md` — current active implementation scope (v0.2)
- `docs/vision-v1.0.md` — inspirational north star
- `docs/prd-archive/` — historical PRD snapshots per release

This roadmap is versioned in Git and can be proposed/amended via Court just like code.
