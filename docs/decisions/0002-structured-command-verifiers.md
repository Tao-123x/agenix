# Decision Record: Structured Command Verifiers

## Status

`accepted`

## Context

Canonical Agenix skills used shell-string verifier commands such as
`cmd: "python3 -m pytest -q"`. That worked, but it coupled verifier execution to
 shell parsing and made cross-platform behavior harder to reason about. Maya
 Chen's trial criteria explicitly call out verifier policy and cross-platform
 consistency as P0 concerns. At the same time, the runtime already moved shell
 tool execution toward explicit requested-versus-resolved argv tracing on
 Windows.

The runtime needed a verifier form that:

- is more deterministic than shell strings
- keeps argv explicit in the manifest
- preserves backward compatibility for existing `cmd` manifests
- lets canonical skills demonstrate the preferred contract immediately

## Decision

Add a structured verifier form:

- command verifiers may now declare `run: ["executable", "arg1", ...]`
- runtime prefers `run` when present
- legacy `cmd` remains supported for backward compatibility
- command verifiers must declare at least one of `cmd` or `run`
- canonical example manifests should prefer `run`

The runtime continues to normalize only the executable token through the
existing platform compatibility layer, so Windows `python3` fallback behavior
still applies to structured verifier argv.

## Alternatives Rejected

- Keep only `cmd` and rely on shell parsing forever
- Remove `cmd` immediately and require a breaking migration for all manifests
- Introduce a much larger verifier policy DSL before shipping the minimum argv
  form

## Customer Impact

This improves Maya Chen's trial path because verifier execution becomes more
portable and less dependent on host shell quirks. It also makes canonical skill
artifacts easier to explain in a platform review. It does not yet satisfy her
full verifier policy boundary requirement, so procurement blockers remain.

## Runtime Impact

This change affects the verifier and manifest contracts:

- manifests can now express command verifiers as explicit argv
- validator enforces that command verifiers are not empty
- canonical examples demonstrate the preferred verifier form
- runtime still supports legacy shell-string verifiers for compatibility
- Windows platform alias handling now applies to structured verifier execution

## Verification

```bash
go test ./internal/agenix -run "TestLoadManifestParsesStructuredCommandVerifierRun|TestLoadManifestRejectsCommandVerifierWithoutCmdOrRun|TestVerifierRunsStructuredCommandWithoutShellParsing|TestRuntimeRunsSmallRefactorSkillWithConstrainedWrite|TestRuntimeRunsSmallRefactorSkillWithCRLFSource|TestRuntimeRunsReadOnlyAnalyzeTestFailuresSkill|TestRuntimeRunsCanonicalFixTestFailureSkill" -count=1
go test ./cmd/agenix -run "TestCLIRunAcceptsArtifact|TestCLIRunReadOnlyAnalyzeArtifact|TestCLIRunSmallRefactorArtifact" -count=1
go test ./... -count=1
```

Expected result:

- all commands pass

## Follow-Up

- define the explicit verifier policy contract Maya asked for: executable, cwd,
  timeout, env, network, stdout, and stderr boundaries
- migrate remaining verifier docs and future examples to `run` unless shell
  syntax is genuinely required
- decide later whether `cmd` stays as a long-term compatibility layer or is
  deprecated in a future manifest version
