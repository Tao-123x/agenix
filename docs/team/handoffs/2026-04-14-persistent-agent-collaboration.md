# Agent Handoff: Persistent Collaboration Protocol

## Role

`docs/team/roles/runtime-lead.md`

## Task

Turn task-scoped subagents into repository-backed, restartable team members.

## File Ownership

- Read: `docs/team`, `docs/customers`, `docs/roadmap`, `docs/plans`
- Write: `docs/team`, `docs/customers`, `docs/decisions`
- Do not touch: runtime code, examples, specs, generated traces

## Context Loaded

- Team charter: `docs/team/2026-04-14-agent-runtime-team.md`
- Role card: `docs/team/roles/runtime-lead.md`
- Customer file: `docs/customers/2026-04-14-first-customer-maya-chen.md`
- Plan or roadmap: `docs/roadmap/2026-04-14-agenix-roadmap.md`
- Prior handoff: none

## Work Completed

- Added the persistent collaboration protocol.
- Added role cards for the standing runtime team.
- Added reusable handoff, customer review, and decision record templates.
- Added the first decision record for repository-backed persistent agents.
- Linked the team charter and Maya customer file to the new protocol.

## Verification

```bash
git diff --check
```

Result: passed before commit.

## Risks

- This is a process contract, not a live scheduler.
- Future agents must actually load role cards and write handoffs for the model to
  hold.
- No GitHub issue automation has been added yet.

## Customer Alignment

Maya verdict: conditional approve

Reason: repository-backed roles improve auditability and continuity without
adding marketplace, UI, daemon, or scheduler surface area before the runtime
loop is ready.

## Next Handoff

The next agent should:

- use these role cards for all future specialist or customer subagents
- create a customer review record before calling the next milestone trial-ready
- add decision records for verifier policy and adapter capability negotiation
- avoid building a multi-agent scheduler until the local runtime contract is
  harder
