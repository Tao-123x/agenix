# Verifier Policy Contract Design

## Status

Approved approach: minimal P0 scope for command verifier policy hardening.

## Problem

Agenix currently treats command verifiers as executable runtime steps, but not as
first-class policy subjects. The runtime records verifier success and failure,
yet it does not enforce an explicit verifier contract for which executable may
run, which working directory it may use, or which timeout applies.

That leaves a gap in Maya Chen's P0 trial criteria. `shell.exec` already has an
exact policy boundary against adapter-requested argv, but verifier execution
still bypasses an equivalent contract and records less context in trace.

## Goals

- Add an explicit verifier policy contract for `command` verifiers.
- Enforce only the minimal P0 boundary in this slice:
  - executable
  - cwd
  - timeout
- Preserve current cross-platform executable resolution behavior:
  - policy compares the requested executable
  - platform alias resolution happens only after policy succeeds
- Make verifier policy failures traceable and stable.
- Keep the current `run: [...]` verifier form as the preferred contract.

## Non-Goals

- No verifier `env` contract in this change.
- No verifier network sandbox or denial in this change.
- No stdout or stderr redaction in this change.
- No manifest-wide policy DSL redesign.
- No removal of legacy `cmd` verifiers in this change.

## Chosen Approach

Add a `policy` block under `command` verifiers and enforce it before execution.

Example:

```yaml
verifiers:
  - type: command
    name: run_tests
    run: ["python3", "-m", "pytest", "-q"]
    cwd: ${repo_path}
    policy:
      executable: python3
      cwd: ${repo_path}
      timeout_ms: 120000
    success:
      exit_code: 0
```

This policy block is intentionally narrow. It duplicates a small amount of
execution data so the manifest states both:

- what the verifier intends to run
- what the runtime is allowed to execute for verifier purposes

The duplication is deliberate because the P0 requirement is a contract boundary,
not only a convenience field.

## Contract Rules

### Supported scope

This P0 contract applies only to `type: command` verifiers that use `run`.

Rationale:

- `run` keeps argv explicit and deterministic.
- `cmd` still depends on shell parsing and does not provide a stable basis for
  exact executable policy comparison.

### Backward compatibility

- Legacy `cmd` command verifiers remain loadable and executable.
- Legacy `cmd` verifiers do not satisfy the new procurement-grade verifier
  policy contract.
- Canonical skills and new examples must use `run` plus `policy`.

### Required policy fields

For a `command` verifier with `run`, the manifest must declare:

- `policy`
- `policy.executable`
- `policy.cwd`
- `policy.timeout_ms`

### Validation rules

Manifest validation must reject a `command` verifier when:

- it declares `run` but does not declare `policy`
- it declares `policy` but does not declare `run`
- `policy.executable` does not exactly match `run[0]`
- `policy.cwd` does not exactly match verifier `cwd`
- `policy.timeout_ms` is zero or negative

The runtime should continue to reject empty `run` argv.

## Runtime Flow

### Execution path

For each `command` verifier with `run`:

1. Build the requested verifier argv from `run`.
2. Build a verifier execution request with:
   - requested argv
   - cwd
   - timeout
3. Enforce verifier policy against the requested executable, cwd, and timeout.
4. If verifier policy passes, apply existing platform executable normalization.
5. Execute the resolved argv.
6. Record verifier result in trace, including requested and resolved command
   context.

### Policy semantics

- Executable comparison is exact against the requested executable token.
- CWD comparison is exact after manifest substitution and path normalization.
- Timeout comparison is exact in milliseconds.
- Platform alias resolution may only replace the executable token, and only
  after verifier policy succeeds.

This keeps verifier policy semantics aligned with the existing shell policy
contract.

## Trace Design

Verifier trace events should grow from status-only summaries into a small,
stable execution record.

Each command verifier event should include a request object with:

- `type`
- `cmd`
- `resolved_cmd`
- `cwd`
- `timeout_ms`

A verifier policy failure should still emit a verifier trace event with:

- `status=failed`
- the same request object shape
- stable error class information in the event error payload

This keeps verifier policy failures visible to `verify`, review, and audit
flows instead of hiding them in CLI output alone.

## Error Handling

- Verifier policy mismatches return `PolicyViolation`.
- Verifier runtime timeout returns `Timeout`.
- Verifier success mismatch still returns `VerificationFailed`.

This preserves the existing distinction between "you were not allowed to run
that verifier request" and "the allowed verifier ran but failed its success
criteria."

## Testing Strategy

Add failing tests first for these cases:

1. command verifier policy rejects mismatched executable
2. command verifier policy rejects mismatched cwd
3. command verifier policy rejects non-positive timeout
4. command verifier policy failure is recorded in trace
5. command verifier timeout uses `policy.timeout_ms`
6. existing Windows executable alias behavior still records requested versus
   resolved command after policy success

Regression coverage must keep passing for:

- canonical verifier execution via `run`
- canonical artifact loop
- current shell policy exact-match rules
- CRLF-safe constrained refactor example

## Files Expected To Change

- `internal/agenix/manifest.go`
- `internal/agenix/schema.go`
- `internal/agenix/verifier.go`
- `internal/agenix/trace.go`
- `internal/agenix/manifest_test.go`
- `internal/agenix/schema_test.go`
- `internal/agenix/verifier_test.go`
- `internal/agenix/runtime_integration_test.go`
- `examples/*/manifest.yaml`
- `specs/skill-manifest.md`
- `specs/policy.md`
- `specs/tool-contract.md`
- one decision record and one handoff note

## Rollout

1. Add validation and trace tests for the new contract.
2. Implement minimal verifier policy enforcement.
3. Migrate canonical command verifiers to declare `policy`.
4. Update docs and decision records.

## Follow-Up After This Slice

- add `env` boundary
- define stdout and stderr handling contract
- design verifier network denial
- add redaction rules for verifier output
