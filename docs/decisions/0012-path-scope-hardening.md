# Decision 0012: Path-Scope Hardening

## Status

Accepted

## Context

Phase 1 hardening already made filesystem scope checks absolute, but two gaps
remained:

- repo-relative paths could still resolve against the verifier process cwd
  instead of the manifest/workspace root
- lexical path validation could still be bypassed through preexisting symlinked
  path segments

Those gaps weakened two core v0 claims:

- artifact runs and verifies should be portable across cwd changes
- declared filesystem scope should constrain actual side effects, not just
  lexical paths

## Decision

For v0, Agenix will enforce these path rules:

1. When the runtime knows a manifest or materialized workspace root,
   repo-relative paths resolve against that root.
2. Filesystem scope checks resolve existing symlinked path segments before
   allow/deny comparison.
3. Artifact materialization rejects payload paths that escape the workspace
   through preexisting symlinked directories.

## Consequences

Positive:

- `verify` no longer depends on the current process cwd for repo-relative
  `changed_files`
- adapters can use repo-relative filesystem paths without silently inheriting
  shell cwd semantics
- artifact materialization is harder to steer outside the workspace

Tradeoffs:

- path resolution logic is more explicit and slightly more complex
- symlink-based tests may need to skip on hosts where symlink creation is not
  available

## Follow-up

- continue cross-platform conformance work on path behavior that still depends
  on host-specific filesystem semantics
- use the next v0 slice to harden the real-adapter boundary without weakening
  this path contract
