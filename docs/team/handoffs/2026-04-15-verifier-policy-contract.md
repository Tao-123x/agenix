# Agent Handoff

## Role

`docs/team/roles/runtime-lead.md`

## Task

`Implement the minimal verifier policy contract for run command verifiers`

## File Ownership

- Read:
  - `docs/superpowers/specs/2026-04-15-verifier-policy-contract-design.md`
  - `docs/superpowers/plans/2026-04-15-verifier-policy-contract.md`
  - `internal/agenix/*.go`
  - `specs/*.md`
- Write:
  - `internal/agenix/manifest.go`
  - `internal/agenix/schema.go`
  - `internal/agenix/verifier.go`
  - `internal/agenix/trace.go`
  - `internal/agenix/*_test.go`
  - `examples/*/manifest.yaml`
  - `specs/skill-manifest.md`
  - `specs/policy.md`
  - `specs/tool-contract.md`
  - `docs/decisions/0003-verifier-policy-contract.md`
  - `docs/team/handoffs/2026-04-15-verifier-policy-contract.md`
- Do not touch:
  - `examples/repo.fix_test_failure/fixture/mathlib.py` in the original main
    workspace; it was already dirty from a prior demo run

## Context Loaded

- Team charter:
  - `docs/team/persistent-agent-collaboration.md`
- Role card:
  - `docs/team/roles/runtime-lead.md`
- Customer file:
  - `docs/customers/2026-04-14-first-customer-maya-chen.md`
- Plan or roadmap:
  - `docs/superpowers/plans/2026-04-15-verifier-policy-contract.md`
- Prior handoff:
  - `docs/team/handoffs/2026-04-15-structured-command-verifier.md`

## Work Completed

- Added `VerifierPolicy` parsing and validation for `run` command verifiers.
- Enforced verifier executable, cwd, and timeout at runtime before execution.
- Recorded verifier request context in trace: `cmd`, `resolved_cmd`, `cwd`,
  `timeout_ms`.
- Migrated canonical `run` verifiers to declare policy blocks.
- Added regression coverage for policy validation, policy enforcement, timeout,
  and trace request shape.

## Verification

```bash
go test ./internal/agenix -count=1
go test ./cmd/agenix -count=1
go test ./... -count=1
```

Result:

- pending final full-suite run in the isolated worktree before push

## Risks

- Legacy `cmd` verifiers still exist as a compatibility layer and do not meet
  the stricter verifier policy contract.
- Verifier env, network denial, and output redaction remain follow-up items.

## Customer Alignment

Maya verdict:

`P0 trial blocker reduced, procurement blocker still open`

Reason:

The runtime now enforces and traces a minimal verifier policy contract, but the
broader sandbox/redaction boundaries Maya asked for are still incomplete.

## Next Handoff

The next agent should:

- run the full suite from the isolated worktree
- inspect the final diff for doc/code consistency
- push `codex/verifier-policy-contract-exec` or fast-forward the public branch
  after review
