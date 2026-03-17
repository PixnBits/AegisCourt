# AegisCourt

Paranoid-by-design AI governance framework. Local-first, cryptographically immutable, single-user agent framework with a multi-reviewer Governance Court.

## Quick Start

```bash
# Build
go build -o aegiscourt ./cmd/aegiscourt/

# First-run setup (detects resources, configures LLM, generates keys)
./aegiscourt init

# Start the kernel
./aegiscourt start

# Run a one-shot agent task
./aegiscourt agent run "What time is it?"

# Create a proposal via AI-assisted drafting
./aegiscourt propose agent-help "add a safe mediated tool that returns UTC time"

# Or use the interactive wizard
./aegiscourt propose guide

# Submit a draft for Court review
./aegiscourt propose submit <draft-id>

# View Court results
./aegiscourt court list
./aegiscourt court view <id>
./aegiscourt court view <id> --detailed
./aegiscourt court view <id> --reviewer ciso

# Vote on a proposal
./aegiscourt court vote <id> approve -y

# Rollback if needed
./aegiscourt rollback last

# Check system status
./aegiscourt status

# Verify audit chain integrity
./aegiscourt audit verify

# Run benchmark suite
./aegiscourt bench run

# Emergency freeze
./aegiscourt halt
```

## Requirements

- Go 1.21+
- [Ollama](https://ollama.com) with a model pulled (e.g. `ollama pull nemotron-3-nano:latest`)
- Linux (primary), macOS (partial — no GPU detection)

## Architecture

- **Ed25519 keypair** for kernel self-signature and audit entry signing
- **Merkle-chain audit log** — append-only JSONL with SHA-256 hash chain, tamper-evident
- **6 reviewer personas** — CISO, MRM, Compliance, Responsible AI, SRE, Helpfulness & Evolution
- **Weighted aggregate scoring** with NASA-style board display
- **4 Court modes** — Auto, Assisted, Hybrid, Manual
- **Guided proposal wizard** with optional LLM assist
- **Benchmark suite** — 8 canned tasks with pattern matching, regex, and LLM-as-judge evaluation

See [docs/architecture.md](docs/architecture.md) for full details.

## Court Modes

| Mode | Behavior |
|------|----------|
| Auto | Low-risk proposals auto-apply with `--confirm` |
| Assisted | Court runs, user reviews and votes |
| Hybrid | Simulates team via explicit vote after review |
| Manual | Strict thresholds, mandatory vote |

## Project Structure

```
cmd/aegiscourt/    CLI binary entry point
pkg/
  keys/            Ed25519 keypair management
  config/          Configuration (JSON, ~/.aegiscourt/config.json)
  audit/           Merkle-chain append-only audit log
  llm/             Ollama LLM router with fallback
  court/           Governance Court engine, reviewers, storage
  proposal/        Draft schema, validation, storage
  resources/       RAM/GPU detection, LLM recommendation
  notify/          File-based notification system
bench/             Benchmark runner and canned tasks
reviewers/         Reviewer persona prompts (.md)
prompts/           Agent-help prompt templates
docs/              Architecture, CLI design, constitution, roadmap
```

## Constitution (Invariants)

1. **Isolation** — Agent runs in ephemeral sandbox, all I/O mediated by kernel
2. **Governance** — Every mutation requires Court review
3. **Auditability** — All actions logged in tamper-evident Merkle chain
4. **Reversibility** — Every change can be rolled back
5. **Transparency** — Court reasoning is always visible and traceable

## License

See [LICENSE](LICENSE) for details.
