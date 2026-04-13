# Agenix Specification v0.1 (Draft)

## 1. Introduction

*Purpose:* make agent skills portable, verifiable, replayable.

*Scope:* skill packaging, runtime behavior, tool contracts, capability negotiation, trace format.

## 2. Terminology

- **Agent** / **Skill** / **Tool** / **Tool Driver** / **Capability** / **Runtime** / **Trace** / **Verifier** / **Policy** / **Checkpoint**

## 3. Design Goals (non‑negotiable)

* Portability across model/OS/host
* Declarative contracts over implicit behavior
* Verification as first‑class deliverable
* Replay & auditability for every run

## 4. Layered System Model

- **Model Layer**
- **Tool Layer**
- **Skill Layer**
- **Runtime Layer**
- **Artifact Layer**

## 5. Capabilities & Negotiation

- Capability Manifest
- Negotiation protocol between runtime and model profile
- Failure modes: `unsupported`, `degraded`, `adapter‑required`

## 6. Tool Contracts

- Namespaces
- Request/response schema
- Error classes
- Replay determinism rules
- Driver requirements (Linux/macOS/Windows)

## 7. Skill Manifest

- Required fields
- Schema definition (JSON Schema)
- I/O typing rules
- Side‑effect declarations
- Verifier requirements

## 8. Agentfile

- Model profile requirements
- Tool mounts
- Constraints (network/filesystem/write scope)
- Entry workflow

## 9. Runtime Behavior

- Loading & execution lifecycle
- Context construction
- Checkpoints & resume
- Failure reporting

## 10. Trace Specification

- Required elements
- Redaction rules
- Replay requirements

## 11. Verifier Specification

- Command‑based / schema‑based / custom hooks
- Output schema
- Success criteria

## 12. Security & Policy

- Minimal privilege
- Policy types
- Approval gates
- Violation handling

## 13. Packaging & Distribution

- Artifact format
- Registry naming & integrity
- Provenance

## 14. Reference Demo

- `repo.fix_test_failure`

## 15. Future Directions

- Agent compose
- Memory federation
- Benchmark suite

---

## Invariants (v0.1)

* Every tool call must be traceable.
* Every run must be verifiable or explicitly `unverified` with reason.
* Skill side‑effects must be declared; out‑of‑scope side‑effects are policy violations.
