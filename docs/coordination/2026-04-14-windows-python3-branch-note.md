# Coordination Note: Windows `python3` Compatibility Branch

This note is for the developer who worked on `codex/windows-python3-alias`.

You found the right portability problem. The canonical `repo.fix_test_failure` manifest uses:

```json
["python3", "-m", "pytest", "-q"]
```

That is fine on macOS and most Linux machines, but it is fragile on Windows. Windows often has either `python` without `python3`, or a Microsoft Store shim at `Microsoft/WindowsApps/python3.exe` that appears in executable lookup but does not behave like a real Python runtime. If Agenix wants one skill artifact to run across Linux, macOS, and Windows, the runtime has to own this compatibility layer.

## What We Absorbed

The main branch now includes a platform compatibility layer in `internal/agenix/platform.go`.

The runtime can resolve `python3` to `python` only under these conditions:

- host OS is Windows
- requested executable is exactly `python3`
- `python3` resolves to a Microsoft Store Python shim
- `python` is available

The helper tests are host-independent, so a macOS or Linux developer can still test the Windows path semantics with injected `GOOS` and lookup behavior.

## What We Changed From The Branch

We did not directly merge the branch implementation because it weakened the runtime policy boundary.

The important contract decision is:

- policy checks the command requested by the adapter
- executable alias resolution happens only after policy succeeds
- execution uses the resolved command
- trace records both the requested command and the resolved command

That means a manifest that allowlists `python3 -m pytest -q` does not allow an agent to directly request `python -m pytest -q`. The runtime may resolve `python3` to `python` as a platform execution detail, but the agent does not get extra authority by guessing local aliases.

This preserves the main Agenix rule: policy is not a convenience layer for the model. Policy is the runtime boundary.

## Current State On `main`

The integrated commit is:

- `af84b98bba71b3c3cf2dca2fb9dfb16fcb0eb424` - `Harden platform command compatibility`

It includes:

- platform executable alias helpers
- Windows Store shim detection that works independent of host path semantics
- exact policy tests that reject direct alias borrowing
- trace tests for requested and resolved command recording
- absolute manifest paths in traces, so `verify` can rerun from a different working directory
- roadmap and Phase 1 hardening docs

Verification used:

```bash
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
./agenix build examples/repo.fix_test_failure -o /tmp/repo.fix_test_failure.agenix
./agenix inspect /tmp/repo.fix_test_failure.agenix
./agenix run /tmp/repo.fix_test_failure.agenix
./agenix verify <trace.json>
./agenix replay <trace.json>
```

## Good Next Tasks

The best next task is not another Windows patch in isolation. The next useful step is to keep hardening the contract:

1. Add a structured command verifier form so verifier commands do not depend on shell string parsing.
2. Add `repo.analyze_test_failures` as a read-only canonical skill.
3. Add `repo.apply_small_refactor` as a constrained write canonical skill.
4. Add trace schema validation for `verify` and `replay`.
5. Add artifact integrity verification before `inspect` and `run`.

The branch caught a real issue. The integration keeps the idea, but moves it under the runtime contract instead of making it an implicit policy shortcut.
