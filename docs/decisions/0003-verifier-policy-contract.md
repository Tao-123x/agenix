# Decision Record: Verifier Policy Contract

## Status

`accepted`

## Context

Structured command verifiers made canonical Agenix skills more portable, but
they still lacked their own enforceable runtime contract. Maya Chen's P0 trial
criteria explicitly called out verifier policy boundaries for executable, cwd,
timeout, env, network, stdout, and stderr. This slice intentionally narrows the
problem to the minimum contract that materially hardens local execution without
expanding into a larger sandbox redesign.

## Decision

Add a verifier policy block for `run` command verifiers:

- `policy.executable`
- `policy.cwd`
- `policy.timeout_ms`

The runtime validates these fields at manifest load time and re-checks them at
verifier execution time. Verifier policy comparison uses the requested
executable before platform alias resolution. Verifier trace entries now record
`cmd`, `resolved_cmd`, `cwd`, and `timeout_ms`.

Legacy `cmd` verifiers remain backward compatible, but they do not satisfy this
procurement-grade verifier policy contract.

## Alternatives Rejected

- Keep verifier execution unconstrained until a full sandbox exists.
- Introduce `env`, network denial, and redaction in the same slice.
- Break compatibility by removing `cmd` verifiers immediately.

## Customer Impact

This closes one of Maya Chen's explicit P0 blockers for a technical trial:
verifier execution is no longer a loosely implied runtime behavior. It still
does not satisfy her full procurement bar because env, network denial, and
output redaction are not implemented yet.

## Runtime Impact

- `run` command verifiers must declare verifier policy fields.
- Manifest validation rejects malformed verifier policy contracts before
  execution.
- Runtime enforces verifier executable, cwd, and timeout before verifier command
  execution.
- Trace now records verifier request context instead of status-only summaries.
- Existing `cmd` verifiers remain executable for backward compatibility.

## Verification

```bash
go test ./internal/agenix -count=1
go test ./cmd/agenix -count=1
go test ./... -count=1
```

Expected result:

- all commands pass

## Follow-Up

- add verifier env boundaries
- define verifier stdout and stderr handling/redaction rules
- add verifier network denial
- decide whether `cmd` verifier support is deprecated in a future manifest
  version
