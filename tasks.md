
```text
// Prompt 1 – FT-001: runtime tracking structures
You are working inside the AegisCourt Go codebase (cmd/aegiscourt/main.go and related files).
Add the following in-memory runtime visibility fields to the Kernel struct:

- activeAgents      map[string]*AgentInstance     // key = agentID
- activeSandboxes   map[string]*SandboxHandle     // key = containerID or handle
- recentProposals   []RecentProposal              // ring buffer, keep max 20
- lastConstitutionHash string                     // hex string, SHA-256 of current constitution
- llmHealth         map[string]LLMEndpointStatus  // endpoint → status
- mu                sync.RWMutex                  // protect the maps/slices

Define these new types cleanly:

type AgentInstance struct {
    ID            string
    StartedAt     time.Time
    Purpose       string
    LastActivity  time.Time
    ProposalCount uint64
    Status        string // "running", "sleeping", "failed", "halted"
}

type SandboxHandle struct {
    ID         string
    StartedAt  time.Time
    AgentID    string
    LastCmd    string
    Status     string // "running", "exited", "killed"
}

type RecentProposal struct {
    ID          string
    Timestamp   time.Time
    Description string
    Status      string // "pending", "approved", "rejected", "applied"
    Score       float64
}

type LLMEndpointStatus struct {
    LastCheck    time.Time
    LastLatency  time.Duration
    LastError    string
    Status       string // "ok", "stale", "error"
}

Initialize them in NewKernel() with empty maps, empty slice, and mu.
Show only the changed/added parts of the Kernel struct and NewKernel function.
Use sync.RWMutex properly.
```

```text
// Prompt 2 – FT-002: lifecycle registration hooks
In the AegisCourt kernel (cmd/aegiscourt), add these thread-safe methods to the Kernel type:

RegisterAgent(id string, purpose string) 
UnregisterAgent(id string)
RegisterSandbox(id string, agentID string, cmd string)
UnregisterSandbox(id string)
RecordProposal(p RecentProposal)   // append + keep only last 20
UpdateLLMHealth(endpoint string, latency time.Duration, err error)

Implement them using kernel.mu.Lock()/Unlock() or RLock() where appropriate.
For RecordProposal: append to recentProposals; if len > 20, drop oldest (ring buffer style).
For UpdateLLMHealth: create or update entry; set Status to "ok"/"error"/"stale" accordingly.

Also update lastConstitutionHash whenever constitution is modified (in ApplyApproved when constitution changes):
kernel.lastConstitutionHash = fmt.Sprintf("%x", sha256.Sum256([]byte(kernel.constitution)))

Show the new methods and any necessary imports (sync, crypto/sha256, encoding/hex, etc.).
```

```text
// Prompt 3 – FT-003: ps command (text + json)
Add a new CLI command "ps" to main.go switch block:

case "ps":
    // support --json flag (reuse or add new flag if needed)

Implement human-readable table output showing:
- Agents: short ID, Started time, Proposal count, Status, Purpose
- Sandboxes: short ID, Started, Status, "Last Cmd (short agentID)"

If nothing active → "No agents or sandboxes currently active."

For JSON: struct with Agents []AgentInstance, Sandboxes []SandboxHandle, Timestamp string (RFC3339)

Use kernel.mu.RLock() during read.
Show aligned columns using fmt.Printf with widths.
Only show changed parts + new case block.
```

```text
// Prompt 4 – FT-004: status command
Add new CLI command "status" in main.go:

case "status":

Print human-readable overview:
- Current time
- Kernel fingerprint (first 8 bytes hex)
- Constitution version (parse from first line if possible) + hash prefix (16 chars)
- Last applied proposal ID (read from aegis-data/versions/last-applied.txt)
- Risk profile (AboutMe.RiskTolerance / UseCase)
- Active agents count
- Active sandboxes count
- Recent proposals: last 5 with emoji (🟢 approved/applied, 🟡 pending, 🔴 rejected), time, score, short description (truncate at 60 chars)

Use kernel.mu.RLock() for reading.
Handle missing files gracefully.
Show the full case block + any helper functions.
```

```text
// Prompt 5 – FT-005: periodic LLM health check
In kernel.Run(ctx), after existing setup, start a background goroutine that:

Every 60 seconds:
- For each endpoint in kernel.config.LLMEndpoints
- Send tiny "ping" prompt to LLM (use router.Dispatch with short timeout)
- Measure latency
- Call kernel.UpdateLLMHealth(endpoint, latency, err)
- Use context.WithTimeout(5 * time.Second) per check

Use select { case <-ctx.Done(): return } to stop on shutdown.
Log errors but never block.
Show the new goroutine code + any needed imports.
```

```text
// Prompt 6 – FT-006: security hardening for visibility commands
Update the "ps" and "status" commands to:

- Immediately return error message and exit if kernel.readOnly == true
  ("Cannot view runtime status in emergency halt / read-only mode")
- Never show full proposal descriptions > 60 chars (truncate + …)
- Never include any prompt fragments, user About Me free-text, or keys
- Use only hash prefix for constitution (not full content)
- Add small comment above each command: // Visibility command – no sensitive data exposed

Show minimal diff / updated code snippets for both commands.
```
