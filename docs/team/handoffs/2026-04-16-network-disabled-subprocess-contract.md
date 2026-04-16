# Agent Handoff

## Role

`docs/team/roles/runtime-lead.md`

## Task

`Make permissions.network=false enforceable for runtime-managed subprocess launch without claiming OS-level sandboxing.`

## File Ownership

- Read:
  - `docs/roadmap/2026-04-14-agenix-roadmap.md`
  - `docs/customers/2026-04-14-first-customer-maya-chen.md`
  - `internal/agenix/*.go`
- Write:
  - `internal/agenix/process_runner.go`
  - `internal/agenix/*_test.go`
  - `internal/agenix/testdata/policy_negative/*`
  - `specs/policy*.md`
  - `specs/tool-contract*.md`
  - `docs/decisions/0011-network-disabled-subprocess-contract.md`
- Do not touch:
  - registry flow
  - artifact format
  - UI / marketplace / daemon work

## Context Loaded

- Team charter:
  - `docs/team/2026-04-14-agent-runtime-team.md`
- Role card:
  - `docs/team/roles/runtime-lead.md`
- Customer file:
  - `docs/customers/2026-04-14-first-customer-maya-chen.md`
- Plan or roadmap:
  - `docs/roadmap/2026-04-14-agenix-roadmap.md`
  - `docs/plans/2026-04-14-phase1-hardening-plan.md`
- Prior handoff:
  - `docs/team/handoffs/2026-04-14-persistent-agent-collaboration.md`

## Work Completed

- Added a shared subprocess launch gate for `permissions.network=false`.
- Supported Python subprocesses through a runtime-injected network-denied launcher.
- Kept offline-safe local git subcommands allowed under the same contract.
- Made unsupported subprocess executables fail closed as `PolicyViolation`.
- Applied the same rule to command verifiers and verifier reruns.
- Added negative tests and fixtures for shell and verifier network attempts.
- Documented the contract in specs and ADR `0011`.

## Verification

```bash
git diff --check
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
go run ./cmd/agenix run examples/repo.fix_test_failure/manifest.yaml
go test ./internal/agenix -run 'TestPolicyNegativeNetworkFalseRejectsShellExec|TestPolicyNegativeNetworkFalseRejectsCommandVerifier' -count=1
```

Result:

- all commands passed

## Risks

- v0 still does not provide OS-level sandboxing; the contract is runtime-managed subprocess denial, not host kernel isolation
- only Python launchers and offline-safe local git subcommands are explicitly supported under `network=false`
- legacy `cmd` verifiers remain backward compatible, but they are denied when `network=false`

## Customer Alignment

Maya verdict:

- trial: approve
- procurement: conditional approve

Reason:

- this removes the largest false-security claim still left in the runtime contract without pretending v0 has a full subprocess sandbox

## Next Handoff

The next agent should:

- inspect the GitHub Actions run for this slice and confirm the matrix stays green
- decide whether to tighten README / spec-v0.1 wording for the new network contract
- then return to Milestone 1 by expanding the cross-platform conformance suite rather than adding new ecosystem surface area
