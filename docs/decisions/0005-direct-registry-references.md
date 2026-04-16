# 0005: Direct Registry References for Run and Inspect

## Status

Accepted

## Context

The local filesystem registry added explicit `publish` and `pull`, but the user
experience still forced an unnecessary copy step before inspecting or running a
known published capsule.

## Decision

`agenix inspect` and `agenix run` now accept exact local registry references in
addition to normal filesystem paths.

- Existing filesystem paths keep priority.
- Exact registry references are `skill@version` and `sha256:digest`.
- Registry resolution can use the default root or an explicit
  `--registry <dir>` override.
- Invalid reference syntax returns `InvalidInput`.
- Missing registry entries return `NotFound`.

## Consequences

- The explicit `pull` command still matters when a caller wants a copied local
  capsule.
- The CLI is easier to use for common registry-backed flows.
- Path semantics stay stable because the resolver only switches to registry mode
  for exact registry reference grammar.
