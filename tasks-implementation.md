# Tasks Implementation
Prompts given to Claude Opus 4.6

## First Round
```
Read #file:tasks.md  and implement the prompts one by one in order. After each prompt commit the changes. Do not track the compiled binaries in git. Provide a summary at the end when all the prompts are finished.
Please verify changes operate as expected by running the CLI, e.g. `$ ./aegiscourt --help`, use #file:cli-design.md for the flows and commands to run
note that some of the LLMs run in Ollama are large and take on the order of minutes for this machine to run
do not add stubs, TODOs, or "for now"s, all code must be real implementations
```

## Second Round
```
Great job! now the next part:

**AegisCourt – Mutation Application Implementation Task List**  
**Role:** Senior Software Architect (v0.2 scope)  
**Goal:** Deliver the **missing production-grade piece** — when a proposal reaches `approve`, the kernel actually applies the change safely, reversibly, and atomically, while preserving every constitutional invariant.  
**Current state (Claude Opus 4.6 codebase):** Court, voting, audit, and drafts work perfectly; **apply/rollback is still a stub**.  
**Target:** Fully working end-to-end flow by end of v0.2 (as described in PRD.md §5.4, docs/example-proposal-flow.md, tasks-v0.2-real-features.md item 14, and architecture.md ADR-004).

### Phase 0: Design & Architecture (1–2 days)
**MUT-001** – Create Mutation Application ADR  
- Document exact semantics for every proposal `type` (add-tool, add-skill, change-prompt, amend-rule, upgrade-memory, other).  
- Define “Git-like diff” format (JSON Patch + optional code delta).  
- Specify atomic apply contract, pre-apply snapshot, rollback point, and failure rollback.  
- Files: `docs/adr-007-mutation-application.md` (new).  
- Acceptance: Approved by you (user) via Court if you want to dogfood the design itself.

**MUT-002** – Define core mutation interfaces & types  
- `pkg/mutation/types.go` + `mutation.go`  
- `type Mutation struct { ID string; ProposalID string; Type string; Patch json.RawMessage; BeforeSnapshot string; ... }`  
- Interface: `Applier` with `Prepare`, `Apply`, `Rollback`, `Validate`.  
- Dependencies: none.

### Phase 1: Mutation Engine Core (3–4 days)
**MUT-003** – Implement Mutation Store & Snapshotting  
- `pkg/mutation/store.go` (Merkle-signed append-only, reuses existing audit pattern).  
- Before every apply: create tarball snapshot of `~/.aegiscourt/` (config, court results, proposals, tools registry, etc.).  
- Acceptance: `aegiscourt snapshot create` works and is referenced in audit log.

**MUT-004** – Build atomic Applier engine  
- `pkg/mutation/engine.go`  
- Flow:  
  1. Load approved CourtResult + Draft  
  2. Validate patch against constitution (reuse reviewer schema checks)  
  3. Snapshot → Apply → Commit (or Rollback on any error)  
  4. Sign every step in audit log  
- Support `--dry-run` flag on `court vote`.  
- Dependencies: MUT-002, MUT-003.

**MUT-005** – Hook `court vote approve` to trigger apply  
- Update `cmd/aegiscourt/main.go` in `cmdCourtVote`  
- If `action == "approve" && status == completed && aggregate >= threshold`: call `mutationEngine.Apply(proposalID)`  
- In Auto mode + `--confirm`: auto-trigger.  
- Acceptance: `aegiscourt court vote 0008 approve` now prints “Mutation applied” and audit shows it.

### Phase 2: Type-Specific Handlers (4–6 days) – prioritized by example flow
**MUT-006** – add-tool handler (first real feature)  
- Register new mediated tool in agent runtime (`pkg/agent/tools.go` – new package).  
- Example: `utc_time` → kernel clock proxy (already stubbed in `handleToolCall`).  
- Update system prompt dynamically.  
- Files: `pkg/mutation/handlers/tool.go`, `pkg/agent/runtime.go`.

**MUT-007** – change-prompt & amend-rule handlers  
- Hot-swap agent system prompt or reload constitution ruleset without restart (where possible).  
- Files: `pkg/mutation/handlers/prompt.go`, `pkg/mutation/handlers/constitution.go`.

**MUT-008** – add-skill & upgrade-memory handlers (stub for v0.2)  
- Minimal implementation: update config + restart agent loop.  
- Full vector memory deferred to v0.5.

**MUT-009** – Generic “other” fallback handler  
- Apply raw JSON Patch to config files + reload.

### Phase 3: Safety, Observability & Rollback (2–3 days)
**MUT-010** – Post-apply regression guard  
- After apply: run relevant bench tasks (`bench/tasks/` that match proposal type).  
- If pass rate drops >10%: auto-create rollback proposal or flag in `status`.  
- Reuse existing `bench/bench.go`.

**MUT-011** – Enhanced rollback command  
- `rollback <id|last>` now restores snapshot + updates CourtResult status.  
- `rollback --all` returns to bootstrap state.  
- Update `cmdRollback`.

**MUT-012** – Emergency halt integration  
- `halt` now forces immediate rollback of last mutation + read-only mode.

### Phase 4: Integration, Testing & Polish (3–4 days)
**MUT-013** – Update all CLI flows & status  
- `status` shows “Last mutation: utc_time (applied)”.  
- `log list` and `audit verify` include mutation events.  
- `court view` after approve shows “Applied at …”.

**MUT-014** – Dogfood first guided proposal  
- Use the exact UTC time example from `docs/example-proposal-flow.md`.  
- Verify: `agent run "What is the current UTC time?"` now succeeds without web_search.

**MUT-015** – Full test suite  
- Unit tests for every handler + engine (apply/rollback round-trip).  
- Integration: run entire `example-proposal-flow.md` end-to-end.  
- Red-team: attempt sandbox escape during apply → must fail.

**MUT-016** – Documentation & release artifacts  
- Update `README.md`, `docs/example-proposal-flow.md`, `tasks-v0.2-real-features.md` (mark item 14 complete).  
- Add `docs/mutation-flow.md` (mirrors example-proposal-flow).  
- Update PRD.md status.

### Phase 5: Optional / v0.5 Prep (nice-to-have)
- Multi-model cross-check before apply (high-risk proposals).  
- Prometheus metrics for mutation success rate.  
- Snapshot export with SBOM.

### Total Effort Estimate
- **Senior dev (you or Claude):** 2–3 weeks full-time (realistic for v0.2 closure).  
- Critical path: MUT-001 → MUT-005 → MUT-006 (first working self-mod in <1 week).

### Success Gates for v0.2 Release
- Zero failed applies in dogfood (all reversible).  
- `utc_time` tool works after guided proposal.  
- `rollback last` restores exact pre-apply state (audit-verifiable).  
- All mutations appear in `audit verify` with signatures.
```
