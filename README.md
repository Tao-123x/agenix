# Agenix

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
- `internal/agenix/` — manifest, policy, tool, trace, verifier, and fake adapter core
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
```

The first command should fail because the fixture starts broken. The `agenix run`
command fixes it through the runtime `fs.write` tool, records every tool and
verifier event in a JSON trace, and only reports success after verifier pass.

Build and inspect a portable capsule:

```bash
go run ./cmd/agenix build examples/repo.fix_test_failure -o repo.fix_test_failure.agenix
go run ./cmd/agenix inspect repo.fix_test_failure.agenix
go run ./cmd/agenix run repo.fix_test_failure.agenix
```

The artifact is a gzip-compressed tar capsule with `manifest.yaml`,
`files/...`, and `agenix.lock.json`. The lockfile records the skill identity,
source file digests, and artifact digest so the capsule can be moved and
inspected without trusting the original directory layout. Running a capsule
materializes it under the run directory as a workspace, preserving
manifest-relative paths for verifier replay.

Publish a capsule into the local registry and pull it back out:

```bash
go run ./cmd/agenix publish repo.fix_test_failure.agenix
go run ./cmd/agenix pull repo.fix_test_failure@0.1.0 -o pulled.fix_test_failure.agenix
```

The default registry root is `~/.agenix/registry`. `publish` is explicit and
idempotent for the same digest. If you try to publish a different digest for the
same `skill@version`, Agenix rejects it and forces a version bump so
`skill@version` remains deterministic. `pull` currently accepts either
`skill@version` or a full `sha256:...` digest reference.

Direct registry references also work for `inspect` and `run`:

```bash
go run ./cmd/agenix inspect repo.fix_test_failure@0.1.0
go run ./cmd/agenix run repo.fix_test_failure@0.1.0
```

If you need a non-default registry root, pass `--registry <dir>` to
`publish`, `pull`, `inspect`, or `run`.

Run the read-only analysis demo:

```bash
go run ./cmd/agenix run examples/repo.analyze_test_failures/manifest.yaml
go run ./cmd/agenix build examples/repo.analyze_test_failures -o repo.analyze_test_failures.agenix
go run ./cmd/agenix run repo.analyze_test_failures.agenix
```

This skill analyzes a known failing pytest fixture without any declared write
scope. A passing run reports an empty `changed_files` list and records no
`fs.write` event.

Run the constrained refactor demo:

```bash
go run ./cmd/agenix run examples/repo.apply_small_refactor/manifest.yaml
go run ./cmd/agenix build examples/repo.apply_small_refactor -o repo.apply_small_refactor.agenix
go run ./cmd/agenix run repo.apply_small_refactor.agenix
```

This skill may write only `greeter.py`. A passing run reports that single file,
runs the tests, and runs a verifier that checks the refactor shape.

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
