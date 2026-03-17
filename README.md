# AegisCourt

AegisCourt is a paranoid-by-design, open-source agentic framework. It begins with a minimal, cryptographically signed, immutable constitutional seed kernel that enforces strict isolation between agents, skills, and the host system. Agents can propose self-modifications (new behaviors, memory schemas, tool integrations) but **only** through an always-active Governance Court that simulates full enterprise review (CISO, MRM, Compliance, etc.).

The Court presents multi-viewpoint analysis, pros/cons, objective scoring against the user's "About Me" profile, interactive Q&A with simulated persona-agents, and a NASA-style all-hands go/no-go board. The user (even a solo hobbyist) acts as the final decision-maker. Every control remains on; deferrals require documented justification and automatic escalation.

**Core Value Proposition:** Deliver OpenClaw-level agentic power with Tier-0 financial-institution-grade security and governance — from a single <60-second local install. The same seed kernel supports hobbyist experimentation at home and scales (via guided evolution) to production zero-trust environments.

## Installation

### Binary Download
Download from GitHub releases for Linux, macOS, Windows.

### Build from Source
```bash
git clone https://github.com/PixnBits/AegisCourt.git
cd AegisCourt
make build
```

### Quick Start
```bash
./bin/aegiscourt init  # Run onboarding wizard
./bin/aegiscourt start  # Start kernel
./bin/aegiscourt agent run "hello world"  # Run agent task
```