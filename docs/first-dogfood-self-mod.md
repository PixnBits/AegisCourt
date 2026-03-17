# First Dogfood Self-Mod Example

## Scenario
Agent repeatedly needs current date → proposes "get_current_time" tool

## Simulated Court Output
- CISO: Approved, low risk
- MRM: Approved
- etc.

## CLI Transcript
```
aegiscourt propose add-tool get_current_time
aegiscourt court vote approve
```

## Result
Tool added, agent uses it.