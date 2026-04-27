# Agent Handoff

## Role

`docs/team/roles/runtime-lead.md`

## Task

`Codify the final v0 reference-runtime acceptance sweep and close the v0 loop.`

## File Ownership

- Read:
  - `docs/roadmap/2026-04-14-agenix-roadmap.md`
  - `docs/team/customer-reviews/2026-04-17-minimal-real-adapter-spike.md`
  - `internal/agenix/runtime_integration_test.go`
  - `internal/agenix/registry_test.go`
- Write:
  - `internal/agenix/acceptance_test.go`
  - `README.md`

## Work Completed

- Added `TestV0AcceptanceSweepForCanonicalSkills`.
- The sweep covers:
  - manifest validation
  - artifact build and inspect
  - artifact run
  - trace validation
  - verifier rerun
  - replay
  - local registry publish and pull
  - direct registry-reference run
- The read-only canonical skill runs through the builtin non-fake
  `heuristic-analyze` adapter inside the sweep.
- Added the acceptance command to `README.md`.

## Verification

```bash
go run ./cmd/agenix acceptance
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
```

## Risks

- the reference runtime is now closed as v0, but adapter failure taxonomy still
  has room to sharpen in post-v0 work
- bilingual docs for the newest English-only records are not mirrored yet

## Customer Alignment

Maya impact:

- this turns “v0 is probably ready” into one explicit engineering gate that can
  be re-run by a buyer or reviewer

## Next Handoff

The next agent should:

- treat v0 as complete
- open a post-v0 track for adapter taxonomy, provider-backed adapter work, or
  stronger registry/provenance only after checking whether it changes the
  accepted v0 claims
