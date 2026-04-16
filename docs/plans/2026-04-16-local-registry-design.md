# Local Registry Design

## Goal

Add a minimal local filesystem registry so Agenix artifacts can be explicitly
published and later retrieved without relying on the source checkout.

## Scope

This slice adds two explicit CLI commands:

- `agenix publish <artifact> [--registry <dir>]`
- `agenix pull <skill@version|sha256:digest> -o <artifact> [--registry <dir>]`

It also lets `agenix run` and `agenix inspect` resolve exact registry
references directly. Filesystem paths still keep priority, so existing path
behavior does not regress.

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
- `run` and `inspect` may resolve those same exact registry references in
  place.

## Why This Approach

Three options were on the table:

1. Add explicit `publish`/`pull` commands backed by a local index.
2. Make `run` and `inspect` accept registry references directly.
3. Skip an index and scan the filesystem on every lookup.

The implemented shape combines Option 1 with a narrow version of Option 2. That
gives us a real local distribution loop plus direct `run`/`inspect` ergonomics,
without introducing fuzzy implicit lookup. Option 3 keeps implementation
shorter but weakens determinism and makes error handling fuzzy.

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
- CLI tests for `publish`, `pull`, direct `inspect`, and direct `run`
- Full `go test -count=1 ./...`, `go vet ./...`, and `go build ./cmd/agenix`
