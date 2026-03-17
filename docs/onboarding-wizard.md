# Onboarding Wizard – aegiscourt init
**Status:** Specification (v0.2)  
**Goal:** Complete first-run setup in <5 min: LLM choice (resource-aware), About Me profile, Court mode calibration, kernel bootstrap, demo proposal.

## Overall Flow
1. Welcome banner
2. Resource detection + LLM selector (interactive, with warnings)
3. About Me profile (persona → mode, risk slider, use cases, defer prefs)
4. Kernel bootstrap + self-signature
5. Tiny demo proposal ("add echo skill") → quick Court → auto-apply in Auto mode

## 1. Welcome
```
Welcome to AegisCourt – paranoid mode always on.
Local-first, cryptographically immutable agent framework.
Setup <5 min. Ctrl+C to abort.
```

## 2. Resource Detection + LLM Selection

**Step 2a: Detect resources** (automatic, no user input)
- RAM free (via gopsutil or equivalent)
- GPU presence & VRAM (optional: nvidia-smi parse or go-nvml if available)
- Conservative estimate: ~2–3 GB base + ~1.5–2 GB per reviewer instance (parallel) or ~1 GB peak (sequential)

**Step 2b: LLM recommendation & prompt**
Display detected resources and tailored recommendation:

```
Detected system resources:
  Free RAM: 11.2 GB
  GPU: NVIDIA RTX 3060 12 GB VRAM (detected)

Recommended: nemotron-3-nano (latest quantized, e.g. fp8/bf16 via Ollama)
  → Best for reliable JSON/structuring & instruction following in Court reviewers & proposal assist
  → Estimated peak usage with full 6-reviewer Court: ~10–14 GB RAM (parallel), ~7–10 GB (sequential)

Strong fallback: llama3.2:3b-instruct-q8 / q6
  → Faster & lighter if resources are constrained

Custom: type any Ollama tag or cloud endpoint (OpenAI/Claude/Grok)

[If free RAM < 9 GB:]
Warning: nemotron-3-nano + full Court may cause swapping or OOM.
Safer choice: llama3.2:3b-instruct (recommended)
Use --low-resource on start for sequential reviewer calls (same 6 reviewers, slower but stable).

Continue with nemotron-3-nano anyway? [y/N]
```

- If user overrides to high-RAM model on low resources → log warning in audit trail
- Validate: ping endpoint (/models or health check)
- Save: config.preferred_llm
- Also save: config.low_resource_mode (bool) if user chose fallback or sequential hint

**Never reduce reviewer count** — full 6-persona panel always used. Low-resource mode only sequences LLM calls.

## 3. About Me Profile
### 3.1 Persona / Court Mode
"Which best describes you? (1–4)"
1. Hobbyist Auto (default – fast, low intervention)
2. Indie Assisted (medium scrutiny)
3. Team Hybrid (simulate team via explicit vote)
4. Enterprise Manual (strict thresholds)

Mapping:
- Auto     → low-risk auto-apply with --confirm, parallel reviewers if possible
- Assisted → force explicit vote after view
- Hybrid   → stricter aggregate threshold
- Manual   → highest defer timeouts, always full reviewers

### 3.2 Risk Tolerance Slider
"How risk-tolerant are you? (0 = ultra-paranoid, defer everything; 10 = experiment freely)"  
→ Influences defer timeouts, aggregate thresholds, not reviewer count

### 3.3 Use Cases (optional)
"Main tasks? (comma-separated or free text)"

### 3.4 Deferral Preference
"Max defer timeout before re-prompt? (default 5 min)"

Save profile as ~/.aegiscourt/about-me.json (kernel-signed on write)

## 4. Kernel Bootstrap
- Self-sign kernel binary
- Load constitution
- Output: "Kernel ready. Hash: ed25519:abc123..."

## 5. Demo Proposal
Auto-propose "add echo skill"  
→ Court (full 6 reviewers, sequential if low-resource mode)  
→ Show court view summary  
→ Auto-vote approve (Auto mode)  
→ Confirm echo works: agent run "echo hello"

## Edge Cases
- Re-run: init --reconfigure (light Court if mode stricter)
- No Ollama: guide install/start
- Low resources: strongly recommend llama3.2 + sequential mode
- Interrupt: save partial profile (optional)

See PRD.md §5.3 and cli-design.md for high-level summary.
