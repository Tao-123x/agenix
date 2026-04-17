# Agent Handoff

## Role

`docs/team/roles/runtime-lead.md`

## Task

`Split unsupported adapter failures from InvalidInput without widening the adapter surface.`

## File Ownership

- Read:
  - `internal/agenix/errors.go`
  - `internal/agenix/adapter_builtin.go`
  - `internal/agenix/runtime.go`
  - `internal/agenix/runtime_integration_test.go`
  - `cmd/agenix/main_test.go`
- Write:
  - `internal/agenix/errors.go`
  - `internal/agenix/adapter_builtin.go`
  - `internal/agenix/runtime.go`
  - `internal/agenix/runtime_integration_test.go`
  - `cmd/agenix/main_test.go`
  - `specs/tool-contract.md`
  - `specs/tool-contract.zh-CN.md`
  - `specs/capability.md`
  - `specs/capability.zh-CN.md`
  - `docs/roadmap/2026-04-14-agenix-roadmap.md`
  - `docs/decisions/0015-adapter-failure-taxonomy.md`

## Work Completed

- Added the stable runtime error class `UnsupportedAdapter`.
- Moved adapter resolution failures to `UnsupportedAdapter`.
- Moved adapter preflight failures to `UnsupportedAdapter`.
- Kept adapter execution failures as `DriverError`.
- Kept verifier failures as `VerificationFailed`.
- Added runtime and CLI tests for:
  - unknown adapter name
  - adapter does not support the requested skill
  - adapter capability mismatch before execution
- Updated the published tool and capability contracts to document the sharper
  taxonomy.

## Verification

```bash
git diff --check
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
```

## Risks

- provider-backed adapters still do not exist, so this taxonomy has only been
  exercised against builtin adapters
- bilingual mirrors were updated only for contract docs touched by this change

## Customer Alignment

Maya impact:

- this removes the last remaining CLI-level ambiguity between malformed input
  and unsupported adapter selection

## Next Handoff

The next agent should:

- keep post-v0 work focused on provider-backed adapter realism or stronger
  provenance, not on inventing more top-level error classes
