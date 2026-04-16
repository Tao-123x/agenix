# 0010: Replay Event Stream Output

## Status

Accepted

## Context

`agenix replay` previously returned only a one-line summary with run id, skill,
final status, and event count. That proved the trace file could be read, but it
did not expose enough information for audit or operator inspection.

At the same time, v0 replay is still intentionally local and trace-driven. It
does not attempt to re-execute tools or re-run verifiers.

## Decision

Keep replay non-executing and extend it to expose stored trace details:

- `Replay(...)` now returns:
  - run metadata
  - ordered trace events
  - final output payload
  - final error text
- `agenix replay <trace>` now prints:
  - the existing summary line
  - one line per event in trace order
  - `final_output=...` when present
  - `final_error=...` when present

## Consequences

- replay is more useful for audit and debugging without changing runtime side
  effects
- CLI compatibility stays reasonable because the original summary line remains
  the first line of output
- this is still not deterministic tool re-execution; it is a richer trace view
  built from persisted events
