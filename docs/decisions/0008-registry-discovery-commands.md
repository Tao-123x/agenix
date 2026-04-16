# 0008: Registry Discovery Commands

## Status

Accepted

## Context

The local registry could already publish, pull, and resolve exact references
indirectly through `run` and `inspect`, but there was no explicit way to ask
the registry what it contained. That made local reuse scriptable only if the
caller already knew the exact `skill@version` or digest.

## Decision

Add three explicit read-only commands:

- `agenix registry list [--registry <dir>]`
- `agenix registry show <skill> [--registry <dir>]`
- `agenix registry resolve <skill@version|sha256:digest> [--registry <dir>]`

Behavior for v0.1:

- output reuses the existing single-line registry entry summary
- `registry list` succeeds on an empty registry and prints nothing
- `registry show` requires an exact skill name and returns `NotFound` when the
  skill is absent
- `registry resolve` accepts only exact `skill@version` or `sha256:digest`
  references and returns the indexed registry entry, not an artifact summary
- discovery commands read only `index.json`; they do not materialize or inspect
  capsule payloads

## Consequences

- scripts can enumerate and filter the local registry without shelling into the
  filesystem layout
- discovery stays explicit and exact; this does not introduce `latest` or any
  implicit version resolution
- output and error classes remain aligned with existing CLI and registry
  semantics
