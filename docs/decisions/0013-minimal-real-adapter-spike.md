# Decision 0013: Minimal Real-Adapter Spike

## Status

Accepted

## Context

V0 previously relied on a single fake scripted adapter for all canonical skills.
That was enough to prove the runtime loop, but not enough to support the v0
claim that model integrations can sit behind an explicit adapter boundary.

The smallest missing proof was not provider integration. It was showing that a
second adapter can run through the same runtime policy, trace, verifier, replay,
and artifact contracts without bypassing them.

## Decision

For v0, Agenix adds one builtin non-fake adapter:

- `heuristic-analyze`

This adapter:

- supports only `repo.analyze_test_failures`
- remains read-only
- uses the same runtime tool drivers as every other adapter
- passes through the same capability preflight boundary
- records explicit `adapter.execute` success/failure events in trace

The runtime also formalizes three adapter lifecycle points in trace:

1. `adapter.selection`
2. `adapter.capability_check`
3. `adapter.execute`

## Consequences

Positive:

- the runtime now runs against more than the fake scripted adapter
- adapter execution failure is distinct from verifier failure in trace
- the read-only canonical skill becomes the first safe place to exercise a
  non-fake adapter path

Tradeoffs:

- this remains a local builtin adapter, not provider integration
- degraded capability execution remains out of scope for v0
- unsupported adapter and invalid input still share the existing CLI error class
  surface

## Follow-up

- run the final v0 acceptance sweep across all canonical skills and artifact
  flows
- decide after v0 whether the next adapter step should be a subprocess adapter
  or a provider-backed adapter
