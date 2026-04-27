# V0 Release Checklist

## Scope

This checklist is for the Agenix V0 reference runtime. It is a local acceptance
gate for the repository implementation, not a claim about a strong sandbox,
remote executor, hosted registry, or provider-backed remote adapter.

## Required Gate

Run from the repository root:

```bash
go run ./cmd/agenix acceptance
```

The command is expected to pass after running the canonical V0 acceptance sweep
across the three canonical skills.

## Full Local Verification

Run these commands before cutting or reviewing a V0 release:

```bash
go run ./cmd/agenix acceptance
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
```

## Acceptance Coverage

The V0 acceptance command covers:

- manifest validation
- portable capsule build and inspect
- artifact execution
- trace validation
- verifier rerun
- trace replay
- local registry publish and pull
- direct registry-reference execution
- the builtin read-only `heuristic-analyze` adapter path for the analysis skill

## Intentional Exclusions

V0 intentionally excludes:

- strong sandbox guarantees
- remote executor semantics
- provider-backed remote adapter coverage in the default acceptance sweep
- registry trust policy
- artifact signatures
- OCI distribution semantics
- hosted or shared registry behavior
- provider/runtime compatibility matrices beyond the canonical local sweep

The opt-in `openai-analyze` smoke path remains outside the default V0 release
gate.
