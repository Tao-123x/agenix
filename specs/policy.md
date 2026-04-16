# Policy Model (v0.1 Draft)

[English](policy.md) | [简体中文](policy.zh-CN.md)

## Policy domains

- **Filesystem:** read/write scopes
- **Network:** allow/deny
- **Shell execution:** command allowlist
- **Verifier execution:** executable/cwd/timeout contract for `run` verifiers
- **Browser actions:** allowed/denied patterns
- **Credentials:** no exfil
- **Cost/time constraints**

## Enforcement

- Runtime must deny tool calls outside policy.
- Violations become traceable events.
- When the runtime knows the manifest or workspace root, repo-relative
  filesystem paths must resolve against that root, not the verifier process
  cwd.
- Filesystem scope decisions must resolve existing symlinked path segments
  before permit/deny comparison.
- v0 does not claim OS-level network sandboxing.
- When `permissions.network` is `false`, runtime-managed subprocess launch is
  supported only for launcher types with explicit local-only or network-denied
  handling.
- Today this means Python subprocesses run under a runtime-injected
  network-denied launcher, offline-safe local git subcommands remain allowed,
  and unsupported executables fail closed as `PolicyViolation`.
- Shell allowlists are exact against the command requested by the adapter.
- Platform executable resolution happens only after shell policy succeeds.
- If executable resolution changes a command, tool traces must include both the
  requested command and the resolved command.
- `run` command verifiers have their own minimal policy contract:
  `policy.executable`, `policy.cwd`, and `policy.timeout_ms`.
- Verifier policy comparison uses the verifier-requested executable before
  platform alias resolution.
- Verifier reruns use the same `permissions.network=false` subprocess rule.
- Legacy `cmd` verifiers remain executable for backward compatibility, but they
  do not satisfy the stricter verifier policy contract.

## Approvals (optional)

- Some actions require explicit approval steps in the workflow, for example
  writing to remote systems or other high-impact operations.
