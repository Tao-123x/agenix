# Local Registry Implementation Plan

**Goal:** Add a minimal local filesystem registry with explicit `publish` and
`pull` commands for Agenix artifacts.

**Architecture:** Keep the registry out of the runtime execution path for this
slice. Add a focused registry module under `internal/agenix` for storage,
indexing, and lookup, then wire thin CLI commands on top.

**Tech Stack:** Go standard library, existing artifact integrity checks, CLI
tests via `go run`.

## Files

- Create: `internal/agenix/registry.go`
- Create: `internal/agenix/registry_test.go`
- Modify: `cmd/agenix/main.go`
- Modify: `cmd/agenix/main_test.go`
- Modify: `README.md`
- Modify: `specs/agenix-spec-v0.1.md`

## Tasks

### Task 1: Registry contract tests

- [ ] Add unit tests for publish, idempotent republish, version conflict, pull
      by digest, and pull by `skill@version`.
- [ ] Add CLI tests for `agenix publish` and `agenix pull`.
- [ ] Run the focused tests and confirm they fail for missing registry support.

### Task 2: Registry implementation

- [ ] Implement registry root resolution, artifact copy, index load/store, and
      reference lookup in `internal/agenix/registry.go`.
- [ ] Reuse `InspectArtifact` for integrity-backed publish metadata.
- [ ] Keep `skill@version` deterministic by rejecting conflicting digests.

### Task 3: CLI and docs

- [ ] Add `publish` and `pull` subcommands to `cmd/agenix/main.go`.
- [ ] Document the local registry loop in `README.md` and `specs/agenix-spec-v0.1.md`.
- [ ] Run `go test -count=1 ./...`, `go vet ./...`, and `go build ./cmd/agenix`.
