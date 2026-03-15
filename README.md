# AegisCourt: Secure, Self-Evolving AI Agent Framework

[![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)](https://golang.org) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![Status: Alpha](https://img.shields.io/badge/Status-Alpha-orange.svg)](https://github.com/PixnBits/AegisCourt)

AegisCourt is an open-source framework for building secure, self-evolving AI agents. It starts with a tiny, cryptographically secure "seed kernel" that enforces strict isolation and governance for all agent actions and modifications. Think of it as a paranoid guardian for AI agents: every change (like adding a new tool or updating behavior) must pass through a simulated "Governance Court" of expert reviewers (e.g., CISO, Ethics Lead) before approval.

Inspired by enterprise risk frameworks (like NIST AI standards and financial regulations), AegisCourt makes advanced agentic AI safe for everyone—from hobbyists tinkering at home to teams in regulated industries.

## Purpose
AegisCourt provides a foundational kernel for AI agents that can autonomously propose improvements to themselves while maintaining unbreakable security invariants. The core idea is "bounded autonomy": agents can evolve, but only through transparent, auditable reviews that mimic a full enterprise approval process. This ensures no rogue changes, prompt injections, or privilege escalations slip through.

At its heart is an immutable constitution (a set of rules) enforced by the kernel, plus ephemeral sandboxes for running agent code. Users act as the final decision-maker in the Governance Court, with options for interactive Q&A and one-click rollbacks.

## Goals: What It Solves, When to Use It, and When Not To
### What Problems Does It Solve?
- **Security Gaps in AI Agents**: Tools like OpenClaw (or similar agent frameworks) often lack isolation, allowing prompt injections, unauthorized access, or unsafe plugins. AegisCourt enforces least-privilege sandboxes and cryptographic immutability from day one.
- **Trust in Self-Evolving Systems**: Agents that modify themselves can drift into unsafe behavior. The Governance Court simulates multi-stakeholder reviews (security, ethics, operations) for every change, providing pros/cons, scores, and conditions.
- **Scalability from Hobby to Enterprise**: Starts simple for personal use but evolves into production-grade setups without rewriting code. It aligns with standards like NIST AI Agent Governance, making it auditable for regulated sectors (e.g., finance, healthcare).
- **User Sovereignty**: Always-on human override, emergency halts, and reversible changes prevent lock-in or surprises.

Key metrics it targets:
- Zero critical security violations.
- Fast reviews (<45 seconds on consumer hardware).
- Measurable agent improvements without unsafe drift.

### When to Use It
- **You're Building Self-Improving AI Agents**: If your project involves agents that learn, add tools, or update their own code/memory, and you want built-in safeguards.
- **Security is Paramount**: Ideal for sensitive data, regulated environments, or when you can't afford breaches (e.g., personal finance trackers, research bots, or enterprise automations).
- **You Want Transparency**: Great for solo developers or teams needing auditable logs and simulated expert feedback without a real committee.
- **From Prototype to Production**: Start local on a laptop; evolve to zero-trust deployments.

Examples: A home AI for stock analysis that safely adds web search; a team bot that proposes efficiency tweaks with full risk assessments.

### When Not to Use It
- **You Need Ultra-Lightweight Agents**: If governance overhead (e.g., court reviews) feels too heavy for simple, non-evolving scripts—use plain libraries like LangChain instead.
- **No Self-Modification Needed**: For static AI tools without autonomy or evolution, this adds unnecessary complexity.
- **Performance-Critical Apps**: On very low-resource devices, the sandboxes and LLM-based reviews might be too slow (though it has fallbacks).
- **Fully Managed Services**: If you prefer hosted platforms (e.g., AWS SageMaker agents) over self-hosted open-source.

In short: Use AegisCourt when safety and evolution are key; skip it for quick-and-dirty prototypes without risks.

## Setup Instructions
AegisCourt is built in Go and requires a local LLM (via Ollama) for the Governance Court. It uses gVisor (via Docker) for sandboxes, so Docker must be installed with the gVisor runtime.

### Prerequisites
- **Go**: Version 1.22 or higher ([install guide](https://golang.org/doc/install)).
- **Ollama**: For local LLM inference. Install from [ollama.com](https://ollama.com) and pull a model like `llama3` (`ollama pull llama3`).
- **Docker**: With gVisor runtime. Install Docker ([docker.com](https://www.docker.com)), then add gVisor:
  ```
  curl -fsSL https://gvisor.dev/archive.key | sudo gpg --dearmor -o /usr/share/keyrings/gvisor-archive-keyring.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/gvisor-archive-keyring.gpg] https://storage.googleapis.com/gvisor/releases release main" | sudo tee /etc/apt/sources.list.d/gvisor.list
  sudo apt update && sudo apt install runsc
  ```
  Configure Docker to use `runsc` runtime (edit `/etc/docker/daemon.json` and restart Docker).
- **Hardware**: At least 4GB RAM (8GB+ recommended for full reviews).

### Installation
1. Clone the repo:
   ```
   git clone https://github.com/PixnBits/AegisCourt.git
   cd AegisCourt
   ```

2. Build the binary:
   ```
   cd cmd/aegiscourt
   go build -o aegiscourt main.go
   ```

3. Configure:
   - Edit `config.json` for your setup (e.g., LLM endpoint, risk tolerance).
   - Ensure Ollama is running (`ollama serve`).

4. Test setup:
   ```
   ./aegiscourt -cmd run
   ```
   You should see "AegisCourt kernel starting – paranoid mode always on".

## Usage
Run the binary with flags:
- `-cmd <command>`: Main action (e.g., `run`, `submit-proposal`).
- `-config <path>`: Path to `config.json` (default: `./config.json`).

Common commands:
- `run`: Starts the kernel (listens for proposals).
- `submit-proposal -desc "Brief desc" -diff "path/to/diff.json"`: Submits a change for review and application.
- `export-audit`: Dumps the tamper-evident audit log.
- `emergency-halt`: Immediate shutdown and freeze.

Add `-with-agent` to `run` for a demo agent that proposes changes.

## Example Runs
### 1. Basic Kernel Startup
```
./aegiscourt -cmd run
```
Output:
```
AegisCourt kernel starting – paranoid mode always on
Kernel public key fingerprint: [fingerprint]
```
Press Ctrl+C to stop.

### 2. Submit a Proposal (e.g., Add a New Rule)
Create a `diff.json`:
```json
[
  {
    "op": "add",
    "path": "/rules/11",
    "value": "Always prioritize user privacy in data handling."
  }
]
```
Run:
```
./aegiscourt -cmd submit-proposal -desc "Add privacy rule to constitution" -diff diff.json
```
Output (simplified):
```
Court Decision:
  Aggregate Score: 85.0/100
  Approved: true
  Conditions:
    - Add syscall filter for data access
  Reviewer Responses:
    CISO: Score 75, Approve with conditions
    ...
Proposal applied successfully
```
If rejected, it explains why.

### 3. Run with Demo Agent
```
./aegiscourt -cmd run -with-agent-loop
```
The agent will periodically propose improvements (e.g., "Add web search tool"), submit them to the Court, and log outcomes. Watch the logs for reviews.

### 4. Interactive Review
Add `-interactive-court` to `submit-proposal`:
```
./aegiscourt -cmd submit-proposal -desc "Test" -diff diff.json -interactive-court
```
After the Court decision, you can ask questions like "ask ciso: why the concern?" before voting approve/reject.

### 5. Rollback a Change
```
./aegiscourt -cmd rollback-last
```
Confirms and reverts the last applied proposal.

For more, check `docs/example-proposal-flow.md` for a full simulation.

## Contributing
We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) (TBD) for guidelines. Focus areas: Better sandboxes, more reviewer personas, Windows/macOS support.

## License
MIT License. See [LICENSE](LICENSE) for details.

Questions? Open an issue or join discussions on GitHub.
