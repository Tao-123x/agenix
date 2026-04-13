# Policy Model (v0.1 Draft)

## Policy domains

- **Filesystem:** read/write scopes
- **Network:** allow/deny
- **Shell execution:** command allowlist
- **Browser actions:** allowed/denied patterns
- **Credentials:** no exfil
- **Cost/time constraints**

## Enforcement

- Runtime must deny tool calls outside policy.
- Violations become traceable events.

## Approvals (optional)

- Some actions require explicit approval step in workflow (e.g., write to remote, high impact operations).
