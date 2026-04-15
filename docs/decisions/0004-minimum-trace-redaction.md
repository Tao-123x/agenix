# Decision Record: Minimum Trace Redaction

## Status

`accepted`

## Context

Maya Chen's P0 trial criteria require minimum trace redaction because secrets
can currently land in verifier stdout, stderr, tool payloads, and final output.

## Decision

Persist traces only after applying runtime redaction.

- Runtime applies built-in redaction rules for common secret-bearing keys and
  text patterns.
- Skills may append additional `redaction.keys` and `redaction.patterns`.
- Redaction runs in `WriteTrace` against a sanitized copy of the trace.
- If redaction fails, trace persistence fails closed.

## Consequences

- `verify` and `replay` consume already-redacted trace files.
- Audit context such as paths, commands, statuses, and timing remains visible
  unless the value itself is a secret.
- In-memory traces may still contain raw values until write time in this slice.
