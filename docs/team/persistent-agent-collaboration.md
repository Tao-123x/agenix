# Persistent Agent Collaboration

## Purpose

Agenix agents are task-scoped workers, not long-running people in a chat room.
This protocol makes them persistent in the way that matters for the project:
their role, context, constraints, and acceptance criteria can be restored on
every run.

The target behavior is image-like reproducibility for contributors:

1. load the team charter
2. load the role card
3. load the current handoff
4. produce evidence-backed output
5. write the next handoff before stopping

## Persistence Model

An agent is considered persistent only when all of these are true:

- its role card exists in `docs/team/roles`
- its current customer or runtime constraints are linked from that role card
- its task output is captured as a commit, issue, decision record, trace, test,
  or documented review
- it leaves a handoff that another agent can execute without private memory

Conversational memory is not part of the contract. Repository state is the
contract.

## Durable Files

- Team charter: [`2026-04-14-agent-runtime-team.md`](2026-04-14-agent-runtime-team.md)
- First customer:
  [`../customers/2026-04-14-first-customer-maya-chen.md`](../customers/2026-04-14-first-customer-maya-chen.md)
- Role cards: [`roles`](roles)
- Handoffs: [`handoffs`](handoffs)
- Handoff template: [`templates/handoff.md`](templates/handoff.md)
- Customer review template: [`templates/customer-review.md`](templates/customer-review.md)
- Decision record template: [`templates/decision-record.md`](templates/decision-record.md)

## Revival Protocol

When starting a specialist agent, give it this context in order:

1. the mission from the team charter
2. the role card for its specific role
3. the current roadmap or plan section relevant to the task
4. the last handoff for that role, if one exists
5. the exact file ownership and write boundaries for the task

The agent must return:

- files changed, or explicit `read-only`
- verification performed
- risks, blockers, and open questions
- next handoff text if work remains
- whether the work still matches Maya Chen's adoption criteria

## Agent Lifecycle

### Start

The runtime lead assigns:

- role card
- bounded task
- allowed files
- expected artifact type: patch, review, plan, test list, or customer verdict

### Work

The agent works inside its boundary. It must not revert unrelated changes. If it
needs to expand scope, it reports the required files before editing them.

### Review

The runtime lead checks:

- the output matches the role card
- tests or review evidence exist
- the result improves portability, verification, replayability, policy, or
  customer adoption
- out-of-scope ideas are captured but not implemented

### Stop

The agent is closed after its output is integrated or rejected. The persistent
state is the repository update, not the live worker.

## Standing Team

The default reusable roles are:

- [`runtime-lead.md`](roles/runtime-lead.md)
- [`artifact-integrity-engineer.md`](roles/artifact-integrity-engineer.md)
- [`schema-trace-contract-engineer.md`](roles/schema-trace-contract-engineer.md)
- [`cross-platform-conformance-explorer.md`](roles/cross-platform-conformance-explorer.md)
- [`adapter-boundary-explorer.md`](roles/adapter-boundary-explorer.md)
- [`first-customer-maya-chen.md`](roles/first-customer-maya-chen.md)

New roles are allowed only when an existing role cannot own the decision cleanly.

## Customer Review Loop

Maya Chen is the customer gate. Her review must happen at these points:

- before a new milestone enters active work
- before adding ecosystem surface area such as registry, UI, marketplace, or
  remote execution
- after any change to policy, trace, verifier, artifact, or adapter contracts
- before calling a release trial-ready

The lead should use [`templates/customer-review.md`](templates/customer-review.md)
for each review and commit the result when it changes direction or priority.

## Decision Log Rules

Use [`templates/decision-record.md`](templates/decision-record.md) when a choice
will affect future runtime compatibility, customer adoption, security posture,
or public contract shape.

A decision record must include:

- context
- decision
- alternatives rejected
- customer impact
- follow-up verification

Small implementation details can stay in commits or PR descriptions.

## Out of Scope

This protocol does not create daemonized agents, background watchers, autonomous
GitHub bots, or a multi-agent scheduler. Those are product features for a later
runtime. For now, persistence means repeatable restart from repository state.
