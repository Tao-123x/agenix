# Agent Handoff

## Role

`docs/team/roles/runtime-lead.md`

## Task

`Harden V0 path-scope semantics for repo-relative paths, symlinked scope checks, and artifact materialization.`

## File Ownership

- Read:
  - `docs/roadmap/2026-04-14-agenix-roadmap.md`
  - `docs/team/handoffs/2026-04-16-cross-platform-conformance-suite.md`
  - `internal/agenix/policy.go`
  - `internal/agenix/runtime.go`
  - `internal/agenix/artifact.go`
- Write:
  - `internal/agenix/policy.go`
  - `internal/agenix/policy_test.go`
  - `internal/agenix/runtime.go`
  - `internal/agenix/runtime_integration_test.go`
  - `internal/agenix/artifact.go`
  - `internal/agenix/artifact_test.go`
  - `specs/policy.md`
  - `specs/tool-contract.md`

## Work Completed

- Added symlink-aware filesystem scope tests for reads and writes.
- Added a policy base-dir contract so repo-relative paths can resolve against a
  manifest/workspace root instead of the process cwd.
- Updated `Run(...)` and `Verify(...)` to construct policy with the manifest
  directory as base context.
- Added cross-cwd verify coverage for repo-relative `changed_files`.
- Hardened artifact materialization so preexisting workspace symlinks cannot
  steer payload writes outside the workspace.
- Updated policy and tool-contract docs to reflect the new path rules.

## Verification

```bash
go test ./internal/agenix -run 'TestPolicyWithBaseResolvesRepoRelativePathAcrossProcessCWD|TestVerifyAcceptsRepoRelativeChangedFilesAgainstMaterializedWorkspaceAcrossCWD|TestVerifyRejectsRepoRelativeChangedFilesThatEscapeMaterializedWorkspace|TestPolicyRejectsReadAndWriteThroughScopedSymlink|TestToolsFSWriteRejectsScopedSymlinkEscapeWithoutWritingOutside|TestMaterializeArtifactRejectsPreexistingWorkspaceSymlinkEscape' -count=1
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
```

## Risks

- symlink tests may skip on hosts that do not permit symlink creation
- path behavior is stronger, but adapter failure taxonomy and the non-fake
  adapter spike are still open

## Customer Alignment

Maya impact:

- this directly improves procurement confidence because filesystem policy now
  constrains actual side effects instead of only lexical path strings

## Next Handoff

The next agent should:

- use the adapter-boundary audit to implement the smallest read-only non-fake
  adapter spike
- add explicit `adapter.execute` trace semantics and tighten unsupported vs
  execution-failure reporting without inventing new ecosystem surface area
