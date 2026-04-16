# Decision 0014: V0 Acceptance Sweep

## Status

Accepted

## Context

After cross-platform hardening, path-scope hardening, and the minimal
real-adapter spike, the remaining v0 gap was not a new runtime behavior. It was
the absence of one codified acceptance line that exercised the full claim
surface together.

Without that sweep, v0 readiness still depended on many separate tests and
human interpretation.

## Decision

The Agenix reference runtime v0 is accepted through one dedicated integration
test:

- `TestV0AcceptanceSweepForCanonicalSkills`

That sweep covers all three canonical skills and exercises:

- manifest validation
- artifact build and inspect
- artifact run
- trace validation
- verifier rerun via `Verify`
- trace-driven replay
- publish to local registry
- pull from local registry
- direct registry-reference run

The read-only canonical skill uses the builtin non-fake `heuristic-analyze`
adapter inside the sweep so the final acceptance line also covers the minimal
real-adapter boundary.

## Consequences

Positive:

- v0 readiness is now expressed as one repeatable command instead of many
  scattered checks
- regressions in portability, replay, verification, artifact, or registry flow
  now break a single obvious gate
- the reference runtime can be described as complete with explicit evidence

Tradeoffs:

- the sweep is slower than narrow unit tests
- it is intentionally broad, so failures require follow-up diagnosis rather than
  pointing to one tiny unit immediately

## Follow-up

- keep post-v0 work out of the acceptance sweep unless it changes the reference
  runtime claims
- use this sweep as the top-level gate while evolving adapter taxonomy and later
  provider-backed integrations
