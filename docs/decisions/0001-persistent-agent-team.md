# Decision Record: Persistent Agent Team

## Status

accepted

## Context

Agenix needs specialist and customer agents to collaborate over time, but the
current execution environment provides task-scoped subagents rather than
permanent workers. Treating chat memory as the source of truth would make the
team impossible to audit or restart.

## Decision

Represent persistent agents as repository-backed role cards, handoff templates,
customer review loops, and decision records. A live subagent can stop at any
time; another agent can resume the same role by loading the durable files.

## Alternatives Rejected

- Keep roles only in conversation history. This fails when context is compacted
  or another worker starts from the repository.
- Build a daemonized multi-agent scheduler now. This adds product surface before
  the local runtime, policy, trace, artifact, and verifier loop is stable.
- Put every agent note into one long team document. This makes role revival
  noisy and harder to assign with clear file ownership.

## Customer Impact

Maya Chen gets a more auditable project process. She can see why a role exists,
what it blocks, what evidence it requires, and how decisions affected trial or
procurement readiness.

## Runtime Impact

The decision does not change runtime behavior. It creates the collaboration
contract needed to keep future runtime work aligned with portability,
verification, replayability, and policy enforcement.

## Verification

```bash
git diff --check
```

Expected result: no whitespace or patch formatting errors.

## Follow-Up

- Use role cards when creating future specialist agents.
- Use the customer review template before calling a milestone trial-ready.
- Add new decision records for adapter policy, verifier policy, redaction, and
  local registry choices.
