# Policy Model (v0.1 Draft)

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
- Shell allowlists are exact against the command requested by the adapter.
- Platform executable resolution happens only after shell policy succeeds.
- If executable resolution changes a command, tool traces must include both the
  requested command and the resolved command.
- `run` command verifiers have their own minimal policy contract:
  `policy.executable`, `policy.cwd`, and `policy.timeout_ms`.
- Verifier policy comparison uses the verifier-requested executable before
  platform alias resolution.
- Legacy `cmd` verifiers remain executable for backward compatibility, but they
  do not satisfy the stricter verifier policy contract.

## Approvals (optional)

- Some actions require explicit approval step in workflow (for example, writing
  to remote systems or high-impact operations).
