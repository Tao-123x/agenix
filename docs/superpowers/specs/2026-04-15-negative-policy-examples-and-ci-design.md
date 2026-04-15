# Negative Policy Examples And CI Design

## Status

Approved approach: add test-only negative policy examples plus minimum CI
coverage.

## Problem

Agenix already has some negative policy coverage, but it is scattered across
unit and integration tests:

- low-level policy tests cover shell allowlist mismatches and write-scope escape
- runtime tests cover policy violation trace handling and `verify` rejection
  paths

That is useful for local correctness, but it does not yet provide the
customer-facing shape Maya Chen asked for: explicit negative policy examples
that prove the runtime rejects unsafe behavior, and CI coverage that keeps those
rejections from regressing.

The current public `examples/` directory is intentionally positive. It shows
skills that should succeed. Adding intentionally failing skills there would make
the demo story noisier and blur the difference between canonical workflows and
test fixtures.

## Goals

- Add explicit negative policy examples without polluting the public positive
  demo flow.
- Turn those examples into stable runtime-level tests instead of only low-level
  unit assertions.
- Add minimum GitHub Actions CI so the policy regression path runs on GitHub,
  not only on a developer machine.
- Keep this slice local, small, and aligned with Maya's P0 priorities.

## Non-Goals

- No new public CLI command.
- No policy DSL redesign.
- No redaction work in this slice.
- No adapter capability work in this slice.
- No expansion of public `examples/` with intentionally failing demo skills.
- No OS-specific sandboxing or network denial changes in this slice.

## Approaches Considered

### Option 1: Test-only negative policy examples plus minimum CI

Store intentionally failing skill fixtures under `internal/agenix/testdata`,
exercise them through runtime integration tests, and run those tests in GitHub
Actions.

Pros:

- preserves a clean public `examples/` story
- gives Maya the explicit negative cases she asked for
- keeps the scope tightly on policy regressions
- fits the current test architecture

Cons:

- less visible to casual repo readers than public examples

### Option 2: Public negative examples under `examples/`

Add intentionally failing policy examples alongside the canonical skills.

Pros:

- more visible as product examples

Cons:

- weakens the positive demo path
- makes it less obvious which examples are expected to pass
- increases future doc and smoke-test complexity

### Option 3: CI-only changes with no new examples

Add workflow coverage but rely only on existing tests.

Pros:

- smallest code delta

Cons:

- does not satisfy the "negative policy examples" part of Maya's P0 ask
- leaves coverage implicit rather than explicit

## Decision

Choose Option 1.

Negative policy examples will live in `internal/agenix/testdata`, remain
test-only, and be exercised through runtime or verify integration tests. A small
GitHub Actions workflow will run the relevant Go tests on both Windows and
Linux.

## Testdata Layout

Add a new root:

```text
internal/agenix/testdata/policy_negative/
```

Within it, add one directory per scenario:

```text
internal/agenix/testdata/policy_negative/write_scope_escape/
internal/agenix/testdata/policy_negative/shell_allowlist_mismatch/
internal/agenix/testdata/policy_negative/verifier_policy_reject/
```

Each scenario should be self-contained and minimal:

- `manifest.yaml`
- any fixture files needed by that skill
- optional `README.md` only if the scenario is not obvious from filenames

These are examples in the sense that they model policy misuse explicitly, but
they are not user-facing canonical examples.

## Scenario Set

### 1. `write_scope_escape`

Purpose:

- prove that a write outside declared write scope fails as `PolicyViolation`
- prove the failure is recorded in trace

Shape:

- manifest read scope points at fixture repo
- manifest write scope points only at fixture repo
- test uses `EscapeAdapter` or equivalent constrained adapter path
- attempted target is outside the allowed repo directory

Expected result:

- `Run(...)` returns `PolicyViolation`
- trace file exists
- trace includes `fs.write` with error class `PolicyViolation`

### 2. `shell_allowlist_mismatch`

Purpose:

- prove an undeclared shell command is denied even if it looks related to an
  allowed command

Shape:

- manifest shell allowlist contains only `["python3", "-m", "pytest", "-q"]`
- test requests `["python3", "-m", "pip", "install", "pytest"]`

Expected result:

- shell policy check fails as `PolicyViolation`
- trace records the attempted command in the request payload

### 3. `verifier_policy_reject`

Purpose:

- prove verifier policy mismatches fail before verifier execution succeeds
- prove verifier policy failure is traceable

Shape:

- `run` verifier declares a `policy` block that mismatches requested
  executable, cwd, or timeout
- runtime reaches verifier stage and fails there

Expected result:

- `Run(...)` returns `PolicyViolation`
- verifier trace event is present with `status=failed`
- verifier trace request includes `cmd`, `resolved_cmd`, `cwd`, `timeout_ms`

## Runtime Test Strategy

Add new integration-style tests rather than burying all new coverage inside
low-level policy unit tests.

Suggested placement:

- `internal/agenix/runtime_integration_test.go` for end-to-end runtime cases
- `internal/agenix/verifier_test.go` only when the negative case is verifier
  internals without runtime orchestration

The runtime-level negative tests should assert:

- stable error class
- trace file creation when applicable
- trace event shape
- no false-positive success status

## CI Design

Add a minimal GitHub Actions workflow:

```text
.github/workflows/policy-negative.yml
```

The workflow should run:

- Windows
- Linux

Each job should execute the existing main test command:

```bash
go test ./... -count=1
```

This slice should not create a second special-case test command if the new
negative coverage can live inside the normal suite. The point is to make policy
regressions part of the default CI path.

## Files Expected To Change

- `internal/agenix/runtime_integration_test.go`
- `internal/agenix/policy_test.go`
- `internal/agenix/testdata/policy_negative/...`
- `.github/workflows/policy-negative.yml`
- one design spec
- one implementation plan

Depending on the exact helper shape, this slice may also touch:

- `internal/agenix/manifest_test.go`
- `internal/agenix/trace_test.go`

## Testing Plan

Before merge, run:

```bash
go test ./internal/agenix -count=1
go test ./cmd/agenix -count=1
go test ./... -count=1
```

Expected result:

- all commands pass

## Risks

- If the testdata scenarios depend too heavily on bespoke adapters, the
  examples may feel synthetic instead of proving realistic runtime behavior.
- If CI only runs the whole suite but does not surface which negative policy
  scenario failed, debugging may be slower.
- If public docs start linking to testdata scenarios as if they are canonical
  skills, the positive demo story could become confusing.

## Follow-Up After This Slice

- minimum trace redaction
- adapter boundary and capability negotiation
- broader cross-platform conformance coverage
