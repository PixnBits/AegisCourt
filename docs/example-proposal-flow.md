# Example Proposal Flow – AegisCourt v0.2
**Status:** Illustrative / Living Example  
**Purpose:** Show a realistic end-to-end workflow using the guided proposal tools (`propose agent-help`, `propose guide`, `propose submit`, Court review, vote, apply/rollback).  
This example assumes Hobbyist Auto mode (default) on a single-user setup.

## Scenario
Alex Rivera has noticed their agent frequently asks for the current time in tasks like "schedule a reminder for tomorrow at 9 AM" or "what's the weather forecast for this afternoon?", but Alex doesn't want to grant broad web/network access yet.  
They want a **safe, mediated tool** that returns only the current UTC time (no timezone conversion, no extra data) via the kernel's controlled channel.

## Step-by-Step Flow

### 1. Start the runtime
```bash
aegiscourt start
# Kernel running, agent ready
```

### 2. Observe the need (optional but realistic trigger)
Alex runs a few tasks and sees repeated "I need current time" patterns in agent logs.

### 3. Generate initial draft with agent-help
Alex gives a short description:

```bash
aegiscourt propose agent-help "add a safe mediated tool that returns only current UTC time without full network access"
```

**What happens internally:**
- The main agent (using Nemotron-3-Nano) generates a draft proposal JSON.
- It suggests:
  - Type: `add-tool`
  - Name: `utc_time`
  - Description: "Mediated syscall to kernel clock for current UTC timestamp"
  - Motivation: "Repeated agent queries for time cause inefficiency and risk unnecessary network calls"
  - Proposed change: New tool definition (JSON schema for params/output), kernel-mediated implementation stub
  - Impact: +15–25% success on scheduling/time-sensitive tasks; negligible RAM/CPU
  - Risks: Low (read-only clock access, no network)
  - Rollback: Remove tool registration from agent config
  - Validation: Run 5 time-related canned tasks before/after

**Output:**
```
Draft proposal generated.
Reasoning: Focused on minimal, read-only access to avoid Rule 1 violation.
Launching refinement wizard...
```

### 4. Refine in the guided wizard (`propose guide`)
The wizard opens automatically (or manually via `aegiscourt propose guide --draft <uuid>`):

**Wizard steps (simplified example interaction):**

1. **Proposal Type** (pre-filled: add-tool)  
   → Confirm or change

2. **Title** (pre-filled: "Add mediated UTC time tool")  
   → Alex edits to "Add safe utc_time tool (kernel clock only)"

3. **Motivation / Problem** (pre-filled from agent)  
   → Alex adds: "Agent often needs time for reminders/planning but I don't want open web_search yet."

4. **Proposed Change** (multi-line editor opens)  
   → Agent pre-filled a draft schema:
     ```json
     {
       "name": "utc_time",
       "description": "Returns current UTC timestamp as ISO string",
       "parameters": {},
       "output": {"type": "string", "format": "date-time"}
     }
     ```
   → Alex confirms (no code changes needed yet — kernel will implement stub)

5. **Expected Impact**  
   → Pre-filled: "+20% on time-aware tasks"  
   → Alex accepts

6. **Risk & Security Self-Check**  
   → Wizard asks: "How does this preserve Rules 1–5?"  
   → Alex types: "Read-only, no network, mediated by kernel → no Rule 1/2 violation"

7. **Rollback Plan** (pre-filled)  
   → "Remove tool from agent tool list; revert config diff"

8. **Validation / Benchmark Plan**  
   → Pre-filled: "Run 5 scheduling tasks before/after; measure success rate"  
   → Alex adds: "Use canned bench tasks in repo/bench/time-sensitive.json"

9. **LLM Assist?** (optional)  
   → Alex chooses "light" → agent suggests clearer wording for risk section

**On completion:**
```
Draft refined and saved as ~/.aegiscourt/proposals/draft-20260317T0123.json
Ready to submit? Use: propose submit draft-20260317T0123
```

#### Example
For input: "add a safe way to get current UTC time without full web access"

```json
{
  "type": "add-tool",
  "title": "Add mediated UTC time tool (kernel clock only)",
  "motivation": "Agent frequently needs current time for scheduling/reminders but open web_search risks unnecessary network exposure and Rule 1 violations.",
  "proposed_change": "New tool: utc_time\nParameters: none\nOutput: ISO8601 UTC string (e.g. 2026-03-17T12:05:00Z)\nImplemented via kernel clock proxy, no network allowed.",
  "expected_impact": {
    "success_gain_percent": 20,
    "resource_delta": "negligible",
    "other_benefits": ["Reduces reliance on external APIs", "Faster response time"]
  },
  "risk_level": "low",
  "risks_and_mitigations": ["Risk of clock API abuse → mitigated by read-only proxy and output cap"],
  "rollback_plan": "Remove tool registration from agent config; revert kernel mediation diff.",
  "validation_plan": "Run 5 time-sensitive scheduling tasks before/after; compare success rate and latency.",
  "constitution_check": "Uses kernel-mediated read-only access, no host write or network → fully preserves Rules 1, 2, and 3.",
  "llm_assist_used": "full"
}
```

### 5. Review the draft (optional)
```bash
aegiscourt court view --draft 20260317T0123
# Shows full JSON, highlights any missing/weak sections
```

### 6. Submit the proposal
```bash
aegiscourt propose submit draft-20260317T0123
# → Proposal ID: 0008 assigned
# Court starts automatically (Hobbyist Auto mode)
```

### 7. Monitor Court progress
```bash
aegiscourt court list
# Shows 0008 pending

aegiscourt status --watch
# Live updates as reviewers finish
```

### 8. View full Court results (rich reasoning)

#### Default view (clean & usable)
```bash
aegiscourt court view 0008
```

**Example output:**
```
Proposal 0008: Add safe utc_time tool (kernel clock only)
Court Mode: Hobbyist Auto
Aggregate Score: 91/100   Recommendation: Approve

NASA-style All-Hands Board:
CISO          🟢 90/100   Mediated read-only clock – preserves Rules 1–3
MRM           🟢 88/100   No behavior drift; easy to benchmark
Compliance    🟢 92/100   Fully auditable via kernel mediation
Ethics        🟢 95/100   No misuse vectors
SRE           🟢 87/100   Negligible resource impact
Helpfulness   🟢 96/100   Major unlock for time-sensitive tasks

Conditions: Implement kernel clock proxy + output length cap.
Your vote? [Approve] [Reject] [Defer] [Q&A]
```

#### Dig deeper: per-reviewer reasoning
```bash
aegiscourt court view 0008 --detailed
```

**Example output (shows nuance – line-broken for terminal readability):**
```
Reviewer Breakdown

CISO (Score 90)
  Key concerns:       Potential clock API misuse if not strictly read-only
  Required mitigations:
                      Kernel proxy only, no direct syscall
                      Output capped at ISO string
  Pros:
                      Eliminates need for network calls
                      Fully reversible
  Cons:
                      Slightly increases mediated I/O surface
  Recommendation:     Approve with conditions

Helpfulness & Evolution (Score 96)
  Key concerns:       None significant
  Pros:
                      +20% expected success on scheduling tasks
                      Addresses clear pain point from repeated time queries
  Cons:               None
  Recommendation:     Strongly approve

MRM (Score 88)
  Drift risk:         Low
  Explainability:     Change is transparent; benchmark plan included
  Evaluation gaps:    Post-apply test suite recommended
  Recommendation:     Approve

(Use `court view 0008 --reviewer ciso` for one persona only, or `--json` for raw data)
```

#### Zoom into a specific reviewer
```bash
aegiscourt court view 0008 --reviewer ciso
# Shows full JSON + raw reasoning text from that reviewer
```

#### Ask for clarification
```bash
aegiscourt court qa 0008 "Why did CISO flag clock API misuse?"
# Routes question to CISO persona → answer in <15s
```

### 9. Vote & apply
```bash
aegiscourt court vote 0008 approve --confirm
```

### 10. Verify & rollback if needed
```bash
aegiscourt agent run "What time is it right now?"
aegiscourt rollback last   # if anything feels off
```

## Key Takeaways
- **Default view** remains concise and scannable.
- **--detailed** uses vertical line breaks for pros/cons/mitigations so nothing wraps awkwardly in most terminals.
- Every score has traceable, human-readable reasoning — users can see exactly **why** a reviewer gave 90 vs 96.
- Drill-down commands (`--reviewer`, `--verbose`, `--json`) give power users full control.

This flow takes ~5–10 minutes end-to-end and produces a high-quality, reversible change.
