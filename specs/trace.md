# Trace Specification (v0.1 Draft)

[English](trace.md) | [简体中文](trace.zh-CN.md)

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

The current reference CLI replay path is trace-driven only: it reads the stored
trace, prints the event sequence in order, and prints the final output payload.
It does not re-execute tools or verifiers.

## Implemented minimum validation

The reference runtime now publishes a schema file at `specs/trace.schema.json`.
The current implementation still treats `ReadTrace` as the authoritative runtime
parser and minimum semantic validator, and `agenix validate` applies the
published schema-backed document check on top of that. `ReadTrace` returns
`InvalidInput` when these fields are missing:

- `run_id`
- `skill`
- `model_profile`
- `final.status`
- each event's `type`
- each event's `name`

This keeps `verify` and `replay` from accepting obviously malformed traces. The
validator intentionally does not yet validate timestamp presence or format,
policy shape, allowed event type values, request/result/error payload schemas,
status enum values, or deterministic replay completeness.
