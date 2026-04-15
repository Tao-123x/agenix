# Decision Record: Verifier Policy Contract

## Status

accepted

## Context

Verifier commands are part of the Agenix trust boundary. Before this change,
command verifiers could run from a shell string without an explicit verifier
policy contract, which made verifier behavior less auditable than tool calls.

## Decision

Keep legacy `cmd` verifiers for backward compatibility, but define a stricter
contract for structured `run` verifiers:

- `run` carries the exact argv
- `policy.executable` declares the requested executable
- `policy.cwd` declares the allowed working directory
- `policy.timeout_ms` declares the allowed timeout

The runtime enforces these fields before verifier execution and records the
verifier request in trace.

## Alternatives Rejected

- Keep shell-string-only verifiers and trust manifest text without an explicit
  policy block.
- Reuse shell tool allowlists for verifiers. Verifiers need their own contract
  because they are runtime-owned execution, not agent-owned tool calls.
- Break all legacy `cmd` verifiers immediately. That would create unnecessary
  migration pain before the runtime loop is harder.

## Customer Impact

This directly addresses Maya Chen's request for verifier execution to have an
auditable, explicit contract instead of an implicit shell escape hatch.

## Runtime Impact

Manifest parsing, validation, verifier execution, and trace shape now all carry
the verifier policy contract. This strengthens verification without weakening
the existing adapter capability negotiation boundary.

## Verification

```bash
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
```

Expected result: all commands pass.

## Follow-Up

- Convert canonical and future manifests to `run + policy`.
- Add negative verifier policy examples to CI.
- Keep trace redaction compatible with verifier request and output fields.
