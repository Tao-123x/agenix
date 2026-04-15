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

- Persisted trace files must be written through runtime redaction.
- Runtime applies built-in redaction rules for common secret-bearing keys and
  text patterns before writing trace JSON.
- Skills may append additional redaction rules through a top-level
  `redaction.keys` and `redaction.patterns` manifest block.
- Redaction should preserve surrounding audit context and replace only the
  secret value with `[REDACTED]` when possible.
- If trace redaction fails, the runtime must fail closed and refuse to write the
  trace.

## Replay

- A replay runner may:
  - re‑execute
  - or replay tool results deterministically from trace (if supported)

## Implemented minimum validation

The current skeleton implements a lightweight contract check, not full JSON
Schema validation. `ReadTrace` returns `InvalidInput` when these fields are
missing:

- `run_id`
- `skill`
- `model_profile`
- `final.status`
- each event's `type`
- each event's `name`

This keeps `verify` and `replay` from accepting obviously malformed traces. The
validator intentionally does not yet validate timestamp presence or format,
policy shape, allowed event type values, request/result/error payload schemas,
status enum values, redaction rules, or deterministic replay completeness.
