# Capabilities (v0.1 Draft)

## Model capability requirements

- `tool_calling` support
- `structured_output` mode
- token budget (context window)
- latency preference (optional)
- reasoning level (heuristic)

## Negotiation

- Skill declares `requires`.
- Runtime checks model profile.
- Outcome:
  - **ok:** proceed
  - **degraded:** proceed with warnings
  - **fail:** runtime reports `unsupported`

## Failure reporting

- Must include: which requirement failed, what capability was missing, and suggestions.
