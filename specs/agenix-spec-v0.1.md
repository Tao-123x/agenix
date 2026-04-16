# Agenix Specification v0.1 (Draft)

[English](agenix-spec-v0.1.md) | [简体中文](agenix-spec-v0.1.zh-CN.md)

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

Reference runtime note:

- v0.1 currently implements explicit preflight `ok` / reject behavior only.
- `degraded` remains a planned contract state, not an implemented runtime path.

## 6. Tool Contracts

- Namespaces
- Request/response schema
- Error classes
- Replay determinism rules
- Driver requirements (Linux/macOS/Windows)

## 7. Skill Manifest

- Required fields
- Schema definition (published JSON Schema)
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
- Published JSON Schema

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

### Reference artifact layout (v0.1)

The reference runtime can build a local capsule with:

```text
manifest.yaml
files/...
agenix.lock.json
```

The capsule is a gzip-compressed tar file. `agenix.lock.json` is the minimum
provenance record for v0.1: artifact version, skill identity, manifest digest,
payload file digests, creation timestamp, builder identity, optional source
commit, and artifact digest.

Artifact integrity in v0.1 is local and lockfile-based. The lock records a
sha256 digest for `manifest.yaml` and each materialized `files/...` payload. On
inspect and materialize, the runtime recomputes those digests from the capsule
contents, verifies payload sizes when recorded, and rejects capsules with
missing, duplicate, modified, or unlocked materialized payload entries. v0.1
does not define signatures, registry trust, or OCI distribution semantics.

The reference runtime now includes a local filesystem registry rooted at
`~/.agenix/registry` by default. Published capsules are copied into the registry
and indexed for explicit lookup by exact `skill@version` or full digest. v0.1
registry semantics are intentionally narrow:

- `publish` is explicit; `build` does not imply publish.
- `pull` is explicit when the caller wants a copied local capsule.
- `registry list`, `registry show <skill>`, and `registry resolve <ref>` expose
  explicit discovery without changing retrieval semantics
- registry discovery orders valid semver versions semantically within a skill;
  non-semver strings remain allowed but sort after valid semver values
- `run` and `inspect` may resolve exact registry references directly.
- the registry rejects publishing a different digest for an already published
  `skill@version`
- registry entries record publish time, publisher identity, and optional source
  commit metadata from the artifact
- the registry is local only and does not define signatures, trust policy, or
  remote distribution semantics

When running a capsule, the runtime materializes it into the run workspace:
`manifest.yaml` remains at the workspace root, and `files/...` entries are
restored without the `files/` prefix. The workspace remains available for trace
verification.

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
