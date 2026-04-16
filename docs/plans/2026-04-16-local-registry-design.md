# Local Registry Design

## Goal

Add a minimal local filesystem registry so Agenix artifacts can be explicitly
published and later retrieved without relying on the source checkout.

## Scope

This slice adds two explicit CLI commands:

- `agenix publish <artifact> [--registry <dir>]`
- `agenix pull <skill@version|sha256:digest> -o <artifact> [--registry <dir>]`

It does not change `agenix run` or `agenix inspect` to resolve registry
references implicitly. That keeps the first registry loop explicit and avoids
mixing path parsing, registry resolution, and runtime execution semantics in one
change.

## Chosen Approach

Use a local filesystem registry rooted at `~/.agenix/registry` by default.

- Artifacts are copied into
  `artifacts/<skill>/<version>/<digest>.agenix` inside the registry.
- The registry stores a lightweight `index.json` with one entry per published
  digest.
- `publish` is idempotent for the same digest.
- `publish` rejects publishing a different digest for the same `skill@version`.
  That keeps `pull skill@version` deterministic and forces explicit version
  bumps.
- `pull` resolves either by full digest (`sha256:...`) or exact
  `skill@version`, then copies the stored capsule to the requested output path.

## Why This Approach

Three options were on the table:

1. Add explicit `publish`/`pull` commands backed by a local index.
2. Make `run` and `inspect` accept registry references directly.
3. Skip an index and scan the filesystem on every lookup.

Option 1 is the right first cut. It gives us a real local distribution loop,
stable lookup semantics, and focused tests without expanding runtime path
resolution. Option 2 is a valid later enhancement once registry references are
proven. Option 3 keeps implementation shorter but weakens determinism and makes
error handling fuzzy.

## Data Model

Registry root:

- `index.json`
- `artifacts/<skill>/<version>/<digest>.agenix`

Index entry fields:

- `skill`
- `version`
- `digest`
- `artifact_path` (relative to registry root)
- `published_at`

## Error Model

- Missing registry or missing entry: `NotFound`
- Invalid reference syntax: `InvalidInput`
- Same `skill@version` published with a different digest: `InvalidInput`
- Copy/index write failures: `DriverError`

## Verification

- Unit tests for publish, idempotent republish, conflict rejection, and pull by
  digest / by `skill@version`
- CLI tests for `publish` and `pull`
- Full `go test -count=1 ./...`, `go vet ./...`, and `go build ./cmd/agenix`
