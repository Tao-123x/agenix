# Agenix Agent Runtime Team

## Mission

Build Agenix into a portable, verifiable runtime for reusable agent capabilities.

The team optimizes for the same invariants as the runtime:

- portable across Linux, macOS, and Windows
- constrained by explicit policy
- verified by evidence, not model claims
- replayable and auditable through trace
- packaged as movable artifacts

## Operating Model

The main agent acts as runtime lead and integration owner. Specialist agents take
bounded tasks with explicit file ownership. No agent should revert work it did
not make. If an implementation requires touching a file outside the assigned
scope, the agent must report that before expanding the change.

Persistent collaboration is repository-backed, not chat-backed. Role cards,
handoff templates, customer review loops, and decision records live in
[`persistent-agent-collaboration.md`](persistent-agent-collaboration.md).

All production changes need:

- a failing test first when behavior changes
- fresh verification before claiming completion
- trace or verifier evidence for runtime behavior
- a clear commit pushed to `main`

## Current Team

### Runtime Lead

Owner: main agent

Role card:

- [`roles/runtime-lead.md`](roles/runtime-lead.md)

Responsibilities:

- architecture and sequencing
- task decomposition
- final integration review
- release notes and GitHub push
- keeping Phase 1 focused on runtime contracts

### Artifact Integrity Engineer

Agent: `Helmholtz`

Role card:

- [`roles/artifact-integrity-engineer.md`](roles/artifact-integrity-engineer.md)

Initial ownership:

- `internal/agenix/artifact.go`
- `internal/agenix/artifact_test.go`
- artifact integrity notes in `specs/agenix-spec-v0.1.md`

First task:

- reject tampered `.agenix` artifacts during inspect/materialize
- keep build/inspect/run smoke behavior intact
- avoid registry, signing, OCI, and provenance expansion for now

### Schema/Trace Contract Engineer

Agent: `Zeno`

Role card:

- [`roles/schema-trace-contract-engineer.md`](roles/schema-trace-contract-engineer.md)

Initial ownership:

- `internal/agenix/schema.go`
- `internal/agenix/schema_test.go`
- validation hooks in `internal/agenix/manifest.go`
- validation hooks in `internal/agenix/trace.go`
- implemented-minimum notes in `specs/skill-manifest.md` and `specs/trace.md`

First task:

- reject malformed manifests and traces with stable `InvalidInput` errors
- keep validation lightweight and dependency-free
- avoid full JSON Schema implementation in this sprint

### Cross-Platform Conformance Explorer

Agent: `Socrates`

Role card:

- [`roles/cross-platform-conformance-explorer.md`](roles/cross-platform-conformance-explorer.md)

Initial ownership:

- read-only review
- Linux/macOS/Windows test matrix proposal

First task:

- identify path, shell, executable lookup, artifact materialization, and verifier
  conformance gaps
- propose concrete test names and assertions

### Adapter Boundary Explorer

Agent: `Mill`

Role card:

- [`roles/adapter-boundary-explorer.md`](roles/adapter-boundary-explorer.md)

Initial ownership:

- read-only review
- adapter/runtime boundary proposal

First task:

- define the smallest next adapter contract after the fake scripted adapter
- propose capability negotiation states and tests
- identify YAGNI boundaries before real model APIs

### First Customer

Agent: `Maya Chen`

Role card:

- [`roles/first-customer-maya-chen.md`](roles/first-customer-maya-chen.md)

Role:

- customer representative for an Internal Developer Platform / AI Enablement
  buyer
- decides whether the current runtime is trial-worthy
- blocks roadmap items that do not improve adoption, auditability, or control

Customer file:

- [`docs/customers/2026-04-14-first-customer-maya-chen.md`](../customers/2026-04-14-first-customer-maya-chen.md)

## First Sprint Scope

In scope:

- artifact integrity checks
- minimum manifest and trace validation
- cross-platform conformance test plan
- adapter boundary design notes
- customer acceptance criteria and procurement blockers

Out of scope:

- public registry
- daemon or remote executor
- real model API integration
- UI
- marketplace
- strong OS sandboxing

## Definition of Done

A sprint item is complete only when:

1. tests cover the behavior or the research output names concrete tests
2. relevant verification commands pass
3. no unrelated files are modified
4. runtime contract implications are documented
5. the integration owner has reviewed and pushed the final result
