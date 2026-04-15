# Agent Handoff

## Role

`docs/team/roles/runtime-lead.md`

## Task

`Add structured command verifier support on top of latest origin/main and keep Windows conformance green`

## File Ownership

- Read:
  - `docs/team`
  - `docs/customers`
  - `docs/roadmap`
  - `docs/plans`
  - `examples`
  - `internal/agenix`
  - `specs`
- Write:
  - `examples/*/manifest.yaml`
  - `internal/agenix`
  - `specs/skill-manifest.md`
  - `docs/team/handoffs`
  - `docs/customers`
  - `docs/decisions`
- Do not touch:
  - unrelated docs outside runtime/verifier/customer-handoff scope
  - generated traces except for local verification

## Context Loaded

- Team charter: `docs/team/2026-04-14-agent-runtime-team.md`
- Role card: `docs/team/roles/runtime-lead.md`
- Customer file: `docs/customers/2026-04-14-first-customer-maya-chen.md`
- Plan or roadmap: `docs/plans/2026-04-14-phase1-hardening-plan.md`
- Prior handoff: `docs/team/handoffs/2026-04-14-persistent-agent-collaboration.md`

## Work Completed

- Added structured command verifier support by introducing `verifiers[].run`
  alongside the existing string `cmd` form.
- Updated manifest parsing in `internal/agenix/manifest.go` to load
  `run: ["..."]` arrays and expand `${repo_path}` substitutions inside each argv
  element.
- Updated manifest validation in `internal/agenix/schema.go` so command
  verifiers must declare either `cmd` or `run`.
- Updated verifier execution in `internal/agenix/verifier.go` so runtime prefers
  structured argv execution when `run` is present and falls back to shell string
  parsing only for legacy `cmd`.
- Switched the three canonical skill manifests to `run: [...]`:
  - `examples/repo.fix_test_failure/manifest.yaml`
  - `examples/repo.analyze_test_failures/manifest.yaml`
  - `examples/repo.apply_small_refactor/manifest.yaml`
- Updated `specs/skill-manifest.md` to recommend `run` over `cmd` for
  deterministic cross-platform verifier execution.
- Added customer review record:
  `docs/customers/2026-04-15-structured-command-verifier-review.md`
- Added decision record:
  `docs/decisions/0002-structured-command-verifiers.md`
- Added tests for:
  - structured verifier manifest parsing
  - rejection of command verifiers with neither `cmd` nor `run`
  - structured verifier execution without shell parsing
- Fixed a Windows regression already present on `origin/main`:
  `repo.apply_small_refactor` failed on CRLF input because the fake adapter used
  an LF-only multiline replacement and reported `changed_files` even when no
  write occurred.
- Added a regression test proving the constrained refactor skill succeeds when
  the source file uses CRLF line endings.

## Verification

```bash
go test ./internal/agenix -run "TestLoadManifestParsesStructuredCommandVerifierRun|TestLoadManifestRejectsCommandVerifierWithoutCmdOrRun|TestVerifierRunsStructuredCommandWithoutShellParsing|TestRuntimeRunsSmallRefactorSkillWithConstrainedWrite|TestRuntimeRunsSmallRefactorSkillWithCRLFSource|TestRuntimeRunsReadOnlyAnalyzeTestFailuresSkill|TestRuntimeRunsCanonicalFixTestFailureSkill" -count=1
go test ./cmd/agenix -run "TestCLIRunAcceptsArtifact|TestCLIRunReadOnlyAnalyzeArtifact|TestCLIRunSmallRefactorArtifact" -count=1
go test ./... -count=1
```

Result:

- All commands passed locally on Windows in this session.

## Risks

- `cmd` still exists as a legacy verifier form, so command verification policy is
  not fully normalized yet.
- Verifier execution still does not have a separate explicit policy contract for
  env, timeout, network, and cwd constraints beyond the current runtime shape.
- Push to GitHub is currently blocked from this machine by intermittent TLS /
  `schannel` handshake failures in Git for Windows, so commit `084775c` exists
  only locally on branch `codex/structured-command-verifier`.
- One subagent handoff attempt failed because that worker's local shell access
  was broken; do not treat subagent output as authoritative without local
  verification.

## Customer Alignment

Maya verdict:

conditional approve

Reason:

This change moves verifier execution toward Maya's requested verifier policy
contract by reducing shell-string dependence and making canonical skills more
deterministic across platforms. It improves trial readiness, but it does not yet
satisfy the full verifier boundary she wants around executable, cwd, timeout,
env, network, and capture behavior.

## Next Handoff

The next agent should:

- push branch `codex/structured-command-verifier` once GitHub TLS access is
  healthy again
- open a PR against `main`
- consider the next runtime-contract slice from Maya's P0 list:
  verifier policy boundary, adapter capability negotiation, or minimum trace
  redaction
