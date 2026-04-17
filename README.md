# Agenix

[English](README.md) | [简体中文](README.zh-CN.md)

**Portable runtime for reusable, verifiable, cross-model agent skills.**

> Docker made software portable.
> **Agenix makes capabilities portable.**

## What is Agenix?

Agenix is an open runtime + packaging system for agents.

It is designed to make agent capabilities:

- portable across models
- portable across Linux/macOS/Windows
- portable across hosts (local / container / remote executor)
- reusable across agents/teams
- verifiable instead of trust-based

Agenix treats **skills** as first‑class artifacts, not informal prompts.

## The problem

Containers solved the engineering pain of:

> “It works on my machine.”

Agents introduced a new portability pain:

> “This agent only works in *my* stack, with *my* model, on *my* machine.”

A skill that is truly reusable must be:

- packaged once
- declared clearly
- executed under constraints
- verified automatically
- replayable/auditable

## Core design (v0.1)

Agenix defines five layers:

1. **Model Layer** — model is replaceable (capability requirements, not vendor lock)
2. **Tool Layer** — stable tool contracts (`fs.*`, `shell.*`, `git.*`, `browser.*`, ...)
3. **Skill Layer** — declarative manifest: purpose, I/O schema, permissions, verifier, recovery
4. **Runtime Layer** — enforces policy, mounts tools, checkpoints, traces, replays
5. **Artifact Layer** — skills/packages/traces as distributable artifacts

## Deliverables (bootstrap)

- `README.md`
- `cmd/agenix/` — reference runtime v0 CLI
- `internal/agenix/` — manifest, policy, tool, trace, verifier, and builtin adapter core
- `specs/`
  - `agenix-spec-v0.1.md` — TOC + glossary + invariants
  - `skill-manifest.md` — the most important interface
  - `agentfile.md` — packaging/build contract
  - `tool-contract.md` — tool schema + error semantics + replay rules
  - `capability.md` — capability requirements + negotiation
  - `trace.md` — replayable trace schema
  - `policy.md` — security/policy model
- `examples/`
  - `repo.fix_test_failure/` — canonical demo: patch + verify
  - `repo.analyze_test_failures/` — canonical demo: read-only failure analysis
  - `repo.apply_small_refactor/` — canonical demo: constrained write refactor

## Bilingual Docs

Core specs:

- [README](README.md) / [README.zh-CN](README.zh-CN.md)
- [Agenix Spec](specs/agenix-spec-v0.1.md) / [Agenix Spec.zh-CN](specs/agenix-spec-v0.1.zh-CN.md)
- [Skill Manifest](specs/skill-manifest.md) / [Skill Manifest.zh-CN](specs/skill-manifest.zh-CN.md)
- [Agentfile](specs/agentfile.md) / [Agentfile.zh-CN](specs/agentfile.zh-CN.md)
- [Tool Contracts](specs/tool-contract.md) / [Tool Contracts.zh-CN](specs/tool-contract.zh-CN.md)
- [Capabilities](specs/capability.md) / [Capabilities.zh-CN](specs/capability.zh-CN.md)
- [Trace](specs/trace.md) / [Trace.zh-CN](specs/trace.zh-CN.md)
- [Policy](specs/policy.md) / [Policy.zh-CN](specs/policy.zh-CN.md)

Example docs:

- [repo.fix_test_failure README](examples/repo.fix_test_failure/README.md) / [中文](examples/repo.fix_test_failure/README.zh-CN.md)
- [repo.fix_test_failure verifier](examples/repo.fix_test_failure/verifier.md) / [中文](examples/repo.fix_test_failure/verifier.zh-CN.md)
- [repo.analyze_test_failures README](examples/repo.analyze_test_failures/README.md) / [中文](examples/repo.analyze_test_failures/README.zh-CN.md)
- [repo.analyze_test_failures verifier](examples/repo.analyze_test_failures/verifier.md) / [中文](examples/repo.analyze_test_failures/verifier.zh-CN.md)
- [repo.apply_small_refactor README](examples/repo.apply_small_refactor/README.md) / [中文](examples/repo.apply_small_refactor/README.zh-CN.md)
- [repo.apply_small_refactor verifier](examples/repo.apply_small_refactor/verifier.md) / [中文](examples/repo.apply_small_refactor/verifier.zh-CN.md)

## Runtime v0 quickstart

Prerequisites:

- Go 1.22+
- Python 3 with `pytest`

Run the canonical demo from the repository root:

```bash
python3 -m pytest -q examples/repo.fix_test_failure/fixture
go run ./cmd/agenix run examples/repo.fix_test_failure/manifest.yaml
go run ./cmd/agenix replay .agenix/runs/<run_id>/trace.json
go run ./cmd/agenix verify .agenix/runs/<run_id>/trace.json
go run ./cmd/agenix validate examples/repo.fix_test_failure/manifest.yaml
```

The first command should fail because the fixture starts broken. The `agenix run`
command fixes it through the runtime `fs.write` tool, records every tool and
verifier event in a JSON trace, and only reports success after verifier pass.
`agenix replay` then reads that trace and prints the recorded event sequence plus
the final output without re-executing the tool loop.

Build and inspect a portable capsule:

```bash
go run ./cmd/agenix build examples/repo.fix_test_failure -o repo.fix_test_failure.agenix
go run ./cmd/agenix inspect repo.fix_test_failure.agenix
go run ./cmd/agenix run repo.fix_test_failure.agenix
```

The artifact is a gzip-compressed tar capsule with `manifest.yaml`,
`files/...`, and `agenix.lock.json`. The lockfile records the skill identity,
source file digests, creation timestamp, builder provenance, and artifact
digest so the capsule can be moved and inspected without trusting the original
directory layout. `inspect` now reports `created_at`, `built_by`,
`build_host`, and `source_commit` when available. Running a capsule
materializes it under the run directory as a workspace, preserving
manifest-relative paths for verifier replay.

Published schema files live in:

- `specs/manifest.schema.json`
- `specs/trace.schema.json`

Use `agenix validate <path>` to check a manifest or trace against the published
schema-backed contract.

Publish a capsule into the local registry and pull it back out:

```bash
go run ./cmd/agenix publish repo.fix_test_failure.agenix
go run ./cmd/agenix pull repo.fix_test_failure@0.1.0 -o pulled.fix_test_failure.agenix
```

The default registry root is `~/.agenix/registry`. `publish` is explicit and
idempotent for the same digest. If you try to publish a different digest for the
same `skill@version`, Agenix rejects it and forces a version bump so
`skill@version` remains deterministic. `pull` currently accepts either
`skill@version` or a full `sha256:...` digest reference. Registry index entries
also record `published_at`, `published_by`, and the artifact `source_commit`
when available.

Registry discovery stays explicit:

```bash
go run ./cmd/agenix registry list
go run ./cmd/agenix registry show repo.fix_test_failure
go run ./cmd/agenix registry resolve repo.fix_test_failure@0.1.0
```

`registry list` prints every indexed entry, `registry show` filters by exact
skill name, and `registry resolve` prints the exact indexed entry for
`skill@version` or `sha256:...`. When registry entries share the same skill,
`list` and `show` order valid semver versions semantically in ascending order.
Non-semver strings remain accepted for now, but sort after valid semver values.

Direct registry references also work for `inspect` and `run`:

```bash
go run ./cmd/agenix inspect repo.fix_test_failure@0.1.0
go run ./cmd/agenix run repo.fix_test_failure@0.1.0
```

If you need a non-default registry root, pass `--registry <dir>` to
`publish`, `pull`, `inspect`, `run`, or `registry`.

Run the read-only analysis demo:

```bash
go run ./cmd/agenix run examples/repo.analyze_test_failures/manifest.yaml
go run ./cmd/agenix run examples/repo.analyze_test_failures/manifest.yaml --adapter heuristic-analyze
go run ./cmd/agenix build examples/repo.analyze_test_failures -o repo.analyze_test_failures.agenix
go run ./cmd/agenix run repo.analyze_test_failures.agenix
```

This skill analyzes a known failing pytest fixture without any declared write
scope. A passing run reports an empty `changed_files` list and records no
`fs.write` event. The optional `--adapter heuristic-analyze` path uses a
separate read-only builtin adapter instead of the default fake scripted one,
while still going through the same runtime policy, trace, verifier, replay, and
artifact loop.

Run the opt-in remote smoke path:

```bash
OPENAI_API_KEY="$OPENAI_API_KEY" go run ./cmd/agenix run examples/repo.analyze_test_failures.remote/manifest.yaml --adapter openai-analyze
```

This path is opt-in, requires `permissions.network=true`, and is outside the
default offline CI sweep. It exercises the provider-backed remote adapter path
without changing the manifest contract or widening the default runtime surface.

Run the constrained refactor demo:

```bash
go run ./cmd/agenix run examples/repo.apply_small_refactor/manifest.yaml
go run ./cmd/agenix build examples/repo.apply_small_refactor -o repo.apply_small_refactor.agenix
go run ./cmd/agenix run repo.apply_small_refactor.agenix
```

This skill may write only `greeter.py`. A passing run reports that single file,
runs the tests, and runs a verifier that checks the refactor shape.

Run the reference v0 acceptance sweep:

```bash
go test ./internal/agenix -run TestV0AcceptanceSweepForCanonicalSkills -count=1
```

This sweep codifies the reference-runtime claim across all three canonical
skills. It validates manifest schema, builds portable capsules, inspects them,
executes artifact and registry-reference runs, reruns verifiers, replays traces,
and checks local registry publish/pull flows.

## Roadmap & Definition of Done (DoD)

### Phase 0: Specs (DoD)
- Vocabulary frozen (skill / runtime / capability / trace / verifier)
- Skill manifest schema draft + JSON Schema version
- Tool contract + error classes + replay rule draft
- Trace schema draft (run id / tool calls / verifier output / redaction rule)

### Phase 1: Runtime prototype (DoD)
- Run a `Skill Manifest` end‑to‑end using a model adapter (tool calling)
- Produce trace for every tool call
- Run verifier (command-based + schema-based)
- Cross‑OS check for at least `fs.*` / `shell.*` / `git.*`

### Phase 2: CLI & Registry (DoD)
- `agenix build/run/verify/replay/publish/pull`
- Registry push/pull story for skill packages (at least local filesystem registry)
- Benchmark suite verifying portability invariants

## Contributing

Agenix is “OCI‑thinking for agents”.

PRs should be oriented around:

- stronger contracts (less magic)
- enforcement of policy (less agent chaos)
- verification and replay (less trust)
- cross‑OS portability (less platform assumptions)

---

**One‑line summary:** portable capability.
