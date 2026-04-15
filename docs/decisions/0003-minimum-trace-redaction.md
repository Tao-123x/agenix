# 0003: Minimum Trace Redaction

## Status

Accepted

## Context

Verifier policy and negative policy examples tightened runtime execution, but
trace persistence still accepted raw tool payloads, verifier stdout/stderr, and
final output without any masking. That left obvious secrets like bearer tokens,
API keys, and passwords visible in the persisted audit log. For the v0 runtime,
we need a minimal contract that preserves audit context while preventing common
secret leakage.

## Decision

The runtime now applies trace redaction before writing `trace.json`.

- Built-in redaction covers common sensitive keys such as `authorization`,
  `api_key`, `token`, `secret`, and `password`.
- Built-in text masking covers common patterns such as bearer tokens and
  `*_api_key=` assignments.
- Skills may append additional redaction rules through a top-level manifest
  `redaction` block with `keys` and `patterns`.
- Each custom pattern must declare `name`, `regex`, and `secret_group`.
- If redaction configuration is invalid, manifest load fails with
  `InvalidInput`.
- If trace sanitization fails, the runtime fails closed and refuses to persist
  the trace.

## Consequences

- Persisted traces are safer to archive and share for debugging.
- Runtime behavior stays auditable because field names, command vectors, file
  paths, and surrounding message structure are preserved where possible.
- This is still a minimal v0 contract: it does not yet cover provenance-grade
  classification, contextual PII detection, or per-sink redaction policies.
