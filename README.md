# AegisCourt – Constitutional Self-Evolving Agent Framework

AegisCourt provides a paranoid, constitutional framework for self-evolving agents with tamper-proof audit trails and governance controls.

## Quick Start
```bash
# Initialize with interactive setup
aegiscourt setup init

# Start the runtime
aegiscourt runtime start

# Run a one-shot task
aegiscourt runtime agent run "Hello world"

# Propose a change
aegiscourt governance propose add-tool web_search

# View proposal
aegiscourt governance court view 1
```

## Features
- **Constitutional AI**: Agents operate under strict rules with Court approval for changes
- **Tamper-Proof Audit**: Merkle-tree signed logs with rollback capabilities
- **Secure Sandboxing**: gVisor-based execution with resource limits
- **LLM Integration**: Router for Ollama/OpenAI with risk flagging
- **Governance CLI**: Propose, review, and approve agent modifications
- **Self-Evolution**: Agents can propose their own improvements

## Architecture
See [docs/architecture.md](docs/architecture.md) for detailed architecture.

## Warning
Paranoid mode always on – all changes require Court approval and are audited.