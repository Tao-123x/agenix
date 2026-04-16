# 0004: Local Filesystem Registry

## Status

Accepted

## Context

Artifact capsules were portable once built, but there was no explicit local
distribution loop after `build`. That meant skill reuse still depended on
sharing raw paths or rebuilding from source.

## Decision

Add a minimal local filesystem registry with two explicit commands:

- `agenix publish <artifact> [--registry <dir>]`
- `agenix pull <skill@version|sha256:digest> -o <artifact> [--registry <dir>]`

Registry behavior for v0:

- default root is `~/.agenix/registry`
- capsules are copied into the registry and indexed in `index.json`
- `skill@version` must resolve deterministically to a single digest
- publishing a different digest for the same `skill@version` is rejected
- `run` and `inspect` do not yet resolve registry references directly

## Consequences

- Agenix now has an explicit local artifact distribution loop.
- Retrieval stays easy to reason about because lookup is explicit and exact.
- Registry semantics remain intentionally narrow; provenance, signatures, and
  remote transport remain future work.
