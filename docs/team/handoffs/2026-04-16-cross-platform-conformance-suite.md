# Agent Handoff

## Role

`docs/team/roles/runtime-lead.md`

## Task

`Expand the Milestone 1 cross-platform conformance suite without changing runtime semantics.`

## File Ownership

- Read:
  - `docs/roadmap/2026-04-14-agenix-roadmap.md`
  - `docs/plans/2026-04-14-phase1-hardening-plan.md`
  - `internal/agenix/platform.go`
  - `internal/agenix/process_runner.go`
  - `internal/agenix/tools.go`
  - `internal/agenix/verifier.go`
- Write:
  - `internal/agenix/conformance_test.go`
  - `internal/agenix/platform.go`
  - `internal/agenix/tools.go`
  - `internal/agenix/verifier.go`
- Do not touch:
  - registry flow
  - artifact format
  - runtime policy semantics
  - examples and manifests

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
  - `docs/team/handoffs/2026-04-16-network-disabled-subprocess-contract.md`

## Work Completed

- Added `internal/agenix/conformance_test.go` with host-independent table tests for:
  - requested vs resolved command request payloads
  - platform shell wrapping contract
  - normalized launch behavior under `network=false`
  - offline-safe git subcommand classification
  - verifier requested vs resolved command payloads
  - verifier launch argv alias resolution on Windows
  - legacy command-verifier shell wrapper behavior
  - trace/replay preservation of distinct `cmd` and `resolved_cmd`
- Extracted tiny shared helpers so tools and verifiers use the same request and shell-args logic:
  - `commandRequestForOS(...)`
  - `shellArgsForOS(...)`
- Added verifier-side helpers so structured command verifiers now launch with the same resolved executable alias that trace records:
  - `verifierRequestForOS(...)`
  - `verifierLaunchArgvForOS(...)`
- Tightened runtime behavior where it was previously inconsistent: verifier trace and actual launch now agree on Windows `python3` shim fallback.

## Verification

```bash
git diff --check
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
```

Result:

- all commands passed

## Risks

- conformance coverage is better, but still not a full platform matrix for path normalization beyond current-host semantics
- the explorer feedback was integrated for verifier contract coverage, but path-scope and adapter-boundary conformance are still open

## Customer Alignment

Maya verdict:

- consistent with current priority

Reason:

- this deepens portability evidence and closes a real verifier portability gap without adding new ecosystem surface area

## Next Handoff

The next agent should:

- inspect whether `specs/agenix-spec-v0.1.md` and `README*.md` need a short note that the conformance suite now covers request/launch/shell wrapping contracts
- then choose either:
  - broaden host-independent path-scope conformance tests, or
  - start a bounded real-adapter interface spike without weakening the current fake-adapter contract
