# Agenix Roadmap

## Purpose

Agenix is a portable runtime for reusable, verifiable agent skills. The current repository proves the first loop:

- declare a skill in a manifest
- execute it through constrained runtime tools
- verify the result with explicit verifiers
- record a trace for replay and audit
- package the skill as a movable artifact

The next work should harden that loop before adding ecosystem surface area.

## Current State

Reference runtime status:

- The Agenix reference runtime v0 loop is complete and acceptance-tested.
- The top-level acceptance command is:
  `go test ./internal/agenix -run TestV0AcceptanceSweepForCanonicalSkills -count=1`

Working runtime surface:

- CLI entrypoint: [`cmd/agenix/main.go`](../../cmd/agenix/main.go)
- Runtime core: [`internal/agenix/runtime.go`](../../internal/agenix/runtime.go)
- Artifact packaging: [`internal/agenix/artifact.go`](../../internal/agenix/artifact.go)
- Canonical demo: [`examples/repo.fix_test_failure`](../../examples/repo.fix_test_failure)
- Draft contracts: [`specs`](../../specs)

Known gaps:

- Post-v0 work should focus on provider-backed adapters and stronger
  provenance/registry guarantees.
- Provider-backed adapter work now has an explicit read-only spike path behind
  the remote policy boundary, and it remains outside the v0 acceptance sweep.
- Manifest and trace schemas are enforced at the current reference-runtime
  minimum, but remain intentionally narrow compared with a future stable spec.
- Local registry remains local-only; signatures and remote trust policy stay
  out of scope for v0.

## Guiding Principles

Every roadmap item should strengthen at least one claim:

1. A skill is portable across supported host environments.
2. A run is verified by evidence, not by model output alone.
3. A packaged skill can be inspected, moved, replayed, and reused safely.

If a feature does not deepen portability, verification, or replayability, defer it.

## Milestone 1: Cross-Platform Runtime Hardening

Goal: make current runtime behavior predictable across Windows, macOS, and Linux.

Success criteria:

- The canonical skill runs on Windows, macOS, and Linux with the same manifest.
- Shell allowlists use a shared platform compatibility layer.
- Runtime behavior is tested for executable discovery, shell invocation, path normalization, artifact materialization, and trace verification.
- README quickstart works on a fresh machine with platform notes only where unavoidable.

Key deliverables:

- Platform helpers for executable aliases, shell invocation, and path rules.
- Tests covering the helpers without depending on the host OS.
- Documentation of allowed platform-specific behavior.

## Milestone 2: Canonical Skill Expansion

Goal: prove the runtime is reusable beyond one repair demo.

Recommended examples:

- `repo.fix_test_failure`: constrained mutation with test verifier.
- `repo.analyze_test_failures`: read-only failure triage with structured output.
- `repo.apply_small_refactor`: small scoped rewrite with diff and verifier checks.

Success criteria:

- Every canonical skill can be built, run, inspected, replayed, and verified.
- The example suite covers read-only analysis, constrained mutation, and multi-verifier execution.
- Examples serve as regression fixtures, not just documentation.

## Milestone 3: Contract Stabilization

Goal: turn draft specs into runtime-enforced contracts.

Success criteria:

- Manifest schema is versioned and validated.
- Capability requirements produce explicit supported, degraded, and unsupported outcomes.
- Trace format has a stable minimum schema with compatibility expectations.
- Tool contracts define deterministic replay behavior and nondeterministic exceptions.

## Milestone 4: Local Distribution Loop

Goal: make skill artifacts usable without source checkout assumptions.

Success criteria:

- `build`, `inspect`, `run`, `verify`, and `replay` work on moved artifacts.
- A local filesystem registry can store and retrieve artifacts by exact
  `skill@version` and digest.
- Artifact metadata is sufficient for integrity and provenance inspection.

## Milestone 5: Adapter Realism

Goal: keep model integrations behind explicit adapter and capability contracts.

Success criteria:

- The runtime can run against more than the fake scripted adapter.
- Capability negotiation failures are explicit and diagnosable.
- The runtime distinguishes invalid skill, unsupported adapter, driver failure, policy violation, and verification failure.

Current status:

- The reference runtime now exposes `UnsupportedAdapter` separately from
  `InvalidInput`, `DriverError`, `PolicyViolation`, and `VerificationFailed`.
- Remaining work in this milestone is about adapter realism beyond builtin
  adapters, not about the basic error taxonomy.

## Defer

Do not prioritize these until the runtime loop is harder:

- marketplace or public registry
- cloud orchestration
- memory federation
- complex multi-agent composition
- UI polish before CLI and artifact workflows stabilize

## Near-Term Recommendation

1. Finish cross-platform runtime hardening.
2. Add two canonical skills with different side-effect profiles.
3. Stabilize manifest, trace, and tool contracts from those examples.
4. Only then invest in local registry and provenance.
