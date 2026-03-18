# ADR-007: Mutation Application

**Status:** Accepted  
**Date:** 2026-03-17  
**Context:** When a proposal reaches `approve` status via Court vote, the kernel must apply the change safely, reversibly, and atomically while preserving all constitutional invariants.

## Decision

### Mutation Types and Semantics

Each proposal `type` maps to a specific handler:

| Type | Handler | What it does |
|------|---------|-------------|
| `add-tool` | `ToolHandler` | Registers a new mediated tool in `~/.aegiscourt/tools.json`, updates agent system prompt |
| `add-skill` | `SkillHandler` | Adds skill config entry, restarts agent loop |
| `change-prompt` | `PromptHandler` | Hot-swaps agent system prompt file |
| `amend-rule` | `ConstitutionHandler` | Updates constitution rules with Court-gated write |
| `upgrade-memory` | `MemoryHandler` | Updates memory config (full vector memory deferred to v0.5) |
| `other` | `GenericHandler` | Applies raw JSON Patch to config files |

### Diff Format

Mutations use a structured format with:
- **Patch**: JSON object describing the change (type-specific schema)
- **Before snapshot**: Tarball of `~/.aegiscourt/` state before apply
- **Mutation ID**: Unique identifier linking to proposal ID

For `add-tool`, the patch is the tool definition:
```json
{
  "name": "utc_time",
  "description": "Returns current UTC timestamp as ISO8601 string",
  "handler": "kernel_clock",
  "parameters": {},
  "output_type": "string"
}
```

For `change-prompt`, the patch is the new prompt content.
For `amend-rule`, the patch includes the rule number and new text.
For `other`, the patch is a set of key-value updates to config.

### Atomic Apply Contract

1. **Prepare**: Validate patch against constitution, load draft + court result
2. **Snapshot**: Create tarball of `~/.aegiscourt/` to `~/.aegiscourt/snapshots/<mutation-id>.tar.gz`
3. **Apply**: Execute type-specific handler
4. **Commit**: Update mutation store, sign audit entry
5. **Verify** (optional): Run bench tasks if applicable, flag regression
6. **Rollback on error**: If any step 3-5 fails, restore from snapshot

Every step is signed in the audit log. Snapshots are retained for rollback.

### Rollback

- `rollback <id>` restores the snapshot associated with that mutation
- `rollback last` finds the most recent applied mutation and restores its snapshot
- `rollback --all` restores to the earliest snapshot (bootstrap state)
- `halt` forces immediate rollback of last mutation + enters read-only mode

### File Layout

```
~/.aegiscourt/
  mutations/          Mutation records (JSON)
  snapshots/          Pre-apply tarballs
  tools.json          Registered mediated tools
  agent-prompt.txt    Current agent system prompt (mutable via Court)
```

## Consequences

- All mutations are reversible via snapshot restore
- Audit trail captures every apply and rollback with signatures
- Tool registry is a simple JSON file, not compiled — hot-reloadable
- Constitution changes are the highest-risk mutation and require manual mode or high aggregate score
