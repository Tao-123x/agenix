# Phase 1 Hardening Plan

## Goal

Harden Agenix from a single-demo runtime into a portable reference runtime with broader examples and clearer contracts.

Phase 1 stays local. It does not include daemon work, remote execution, marketplace, or public registry flows.

## Task 1: Platform Compatibility

Files:

- `internal/agenix/platform.go`
- `internal/agenix/platform_test.go`
- `internal/agenix/tools.go`
- `internal/agenix/verifier.go`
- `specs/tool-contract.md`
- `specs/policy.md`

Steps:

- [x] Add host-independent tests for Windows Store Python shim detection.
- [x] Add executable alias tests that do not require the test host to be Windows.
- [x] Normalize only the executable token, never command arguments.
- [x] Make `shell.exec` traces record both requested and resolved commands.
- [x] Document normalization as runtime contract behavior.

Acceptance:

- `python3` may fall back to `python` only on Windows, only when `python3` resolves to the Microsoft Store shim, and only when `python` exists.
- Shell policy remains exact against the adapter-requested argv before fallback.
- Non-Windows hosts keep exact executable names.
- Arguments remain exact after normalization.

## Task 2: Read-Only Canonical Skill

Files:

- `examples/repo.analyze_test_failures/manifest.yaml`
- `examples/repo.analyze_test_failures/README.md`
- `examples/repo.analyze_test_failures/verifier.md`
- `examples/repo.analyze_test_failures/fixture/...`
- `cmd/agenix/main_test.go`

Steps:

- [x] Add a fixture with a known failing test.
- [x] Add a manifest with read scope but no write scope.
- [x] Make the fake adapter produce structured analysis without filesystem writes.
- [x] Verify no changed files are reported.
- [x] Exercise build, run, verify, and replay.

Acceptance:

- The skill proves useful agent work can be read-only.
- A write attempt from this skill produces `PolicyViolation`.

## Task 3: Constrained Refactor Canonical Skill

Files:

- `examples/repo.apply_small_refactor/manifest.yaml`
- `examples/repo.apply_small_refactor/README.md`
- `examples/repo.apply_small_refactor/verifier.md`
- `examples/repo.apply_small_refactor/fixture/...`
- `internal/agenix/runtime_integration_test.go`

Steps:

- [ ] Add a small fixture repo with a deterministic refactor target.
- [ ] Restrict write scope to the target file or directory.
- [ ] Verify expected content and changed files.
- [ ] Verify out-of-scope writes fail and are traced.

Acceptance:

- The skill demonstrates scoped mutation beyond bug repair.
- Verifier success requires both correct output and allowed side effects.

## Task 4: Manifest and Trace Contract Enforcement

Files:

- `internal/agenix/schema.go`
- `internal/agenix/schema_test.go`
- `internal/agenix/manifest.go`
- `internal/agenix/manifest_test.go`
- `internal/agenix/trace.go`
- `internal/agenix/trace_test.go`
- `specs/skill-manifest.md`
- `specs/trace.md`

Steps:

- [ ] Validate required manifest fields with stable error classes.
- [ ] Validate trace minimum fields for `verify` and `replay`.
- [ ] Add JSON schema files or equivalent runtime validators.
- [ ] Document implemented versus planned fields.

Acceptance:

- Invalid manifests fail before tool execution.
- Invalid traces fail before verifier reruns.
- Errors are stable enough for CLI and tests.

## Task 5: Artifact Integrity

Files:

- `internal/agenix/artifact.go`
- `internal/agenix/artifact_test.go`
- `cmd/agenix/main.go`
- `specs/agenix-spec-v0.1.md`

Steps:

- [ ] Define exactly which bytes are covered by an artifact digest.
- [ ] Verify digest during `inspect` and `run`.
- [ ] Fail moved or modified artifacts with `InvalidInput` or `VerificationFailed`.
- [ ] Preserve movable artifact behavior after integrity checks.

Acceptance:

- A modified `.agenix` artifact is detected before execution.
- Integrity checks do not depend on source checkout paths.

## Task 6: Adapter Contract Boundary

Files:

- `internal/agenix/runtime.go`
- `internal/agenix/manifest.go`
- `specs/capability.md`

Steps:

- [ ] Split fake adapter behavior by skill name instead of hard-coding one flow.
- [ ] Add adapter capability checks before execution.
- [ ] Trace adapter selection and capability failures.

Acceptance:

- Unsupported skills fail explicitly before partial execution.
- Future real model adapters can plug into the same runtime boundary.

## Verification Commands

Run before merging Phase 1 changes:

```bash
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
```

Run canonical smoke checks after artifact or runtime changes:

```bash
./agenix build examples/repo.fix_test_failure -o /tmp/repo.fix_test_failure.agenix
./agenix inspect /tmp/repo.fix_test_failure.agenix
./agenix run /tmp/repo.fix_test_failure.agenix
```
