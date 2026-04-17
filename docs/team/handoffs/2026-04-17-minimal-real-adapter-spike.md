# Agent Handoff

## Role

`docs/team/roles/runtime-lead.md`

## Task

`Add the smallest non-fake adapter path and make adapter execution observable in trace.`

## File Ownership

- Read:
  - `docs/roadmap/2026-04-14-agenix-roadmap.md`
  - `docs/team/customer-reviews/2026-04-16-path-scope-hardening.md`
  - `specs/capability.md`
  - `specs/trace.md`
  - `internal/agenix/runtime.go`
- Write:
  - `internal/agenix/adapter_builtin.go`
  - `internal/agenix/runtime.go`
  - `internal/agenix/runtime_integration_test.go`
  - `cmd/agenix/main.go`
  - `cmd/agenix/main_test.go`
  - `README.md`
  - `specs/capability.md`
  - `specs/trace.md`

## Work Completed

- Added builtin adapter selection through `agenix run ... --adapter <name>`.
- Added `heuristic-analyze`, a read-only non-fake adapter for
  `repo.analyze_test_failures`.
- Added `adapter.execute` trace events with explicit `ok` / `failed` states.
- Added tests proving:
  - the heuristic adapter runs end-to-end through runtime policy and verifiers
  - unsupported builtin adapter use fails in preflight before execute
  - adapter execute failure is distinct from verifier failure
  - CLI can select the non-fake adapter
- Updated README and spec docs so implemented capability and trace behavior no
  longer overclaim degraded negotiation or omit adapter events.

## Verification

```bash
git diff --check
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
```

## Risks

- the non-fake adapter is still builtin and local, not subprocess- or provider-backed
- unknown adapter, unsupported skill, and adapter preflight mismatch now use
  `UnsupportedAdapter`, but provider-backed adapter semantics are still absent
- bilingual spec mirrors are now slightly behind the English source again

## Customer Alignment

Maya impact:

- this removes the strongest remaining “fake-only” objection without adding a
  provider integration surface too early

## Next Handoff

The next agent should:

- run the final v0 acceptance sweep across all canonical skills and artifact
  flows
- produce a buyer-facing v0 readiness summary with explicit remaining risk
  notes instead of adding more runtime surface area
