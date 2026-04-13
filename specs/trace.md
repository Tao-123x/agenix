# Trace Specification (v0.1 Draft)

## Required fields

- `run_id` (uuid)
- `skill`
- `model_profile`
- `timestamp`
- `policy` (applied permissions)
- `events[]`

## Event types

- `tool_call`: name, request, result, error, duration
- `checkpoint`: marker for resume
- `verifier`: name, result, output
- `final`: status

## Redaction

- No secrets in trace.
- If secrets appear in tool output, redaction policy must mask.

## Replay

- A replay runner may:
  - re‑execute
  - or replay tool results deterministically from trace (if supported)
