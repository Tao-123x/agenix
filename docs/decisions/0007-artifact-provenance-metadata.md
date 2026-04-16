# 0007: Artifact Provenance Metadata

## Status

Accepted

## Context

Agenix could already build, inspect, publish, and pull portable capsules, but
the artifact and registry surfaces did not say who built a capsule, where it
was built, or which source commit it came from. That made local distribution
auditable only at the digest layer.

## Decision

Add minimal provenance metadata in two places:

- `agenix.lock.json` now records:
  - `created_at`
  - `provenance.built_by`
  - `provenance.build_host`
  - `provenance.source_commit`
- local registry `index.json` entries now record:
  - `published_at`
  - `published_by`
  - `source_commit`

Behavior for v0.1:

- provenance fields are additive and optional
- `source_commit` is best-effort and may be empty outside a git worktree
- artifact integrity remains lockfile-based; provenance is descriptive metadata,
  not a trust signature
- publish remains idempotent for the same digest; republishing does not refresh
  metadata for an already indexed identical capsule

## Consequences

- `agenix inspect` exposes builder provenance without unpacking the capsule
- registry consumers can see who published a capsule and when
- provenance improves auditability without introducing signing or remote trust
  semantics
- provenance fields in the lockfile are part of the artifact bytes, so they are
  part of the resulting digest by design
