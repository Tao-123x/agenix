# Tool Contracts (v0.1 Draft)

## Global requirements

- Every tool call must produce trace entries.
- Tool responses must be JSON‑serializable.
- Errors must be stable and classify as:
  - `InvalidInput`
  - `PermissionDenied`
  - `NotFound`
  - `Timeout`
  - `DriverError`
  - `PolicyViolation`

## Namespaces

### fs

- `fs.read(path) -> {content, encoding}`
- `fs.write(path, content, overwrite=true)`
- `fs.list(path) -> [{name, type}]`

Constraints:

- Writes must be within declared write scope.

### shell

- `shell.exec(cmd, cwd, timeout) -> {stdout, stderr, exit_code}`

Constraints:

- Only allowed commands (by policy/tool whitelist) may run.
- Runtime may apply documented platform executable aliases after policy comparison and before execution.
- v0 only defines one alias: on Windows, `python3` may resolve to `python` when `python3` points at the Microsoft Store shim and `python` is available.
- Alias normalization must not alter arguments. It may only replace the executable token.
- Policy comparison uses the command requested by the adapter, before alias resolution.
- Trace entries must record both the requested command and the resolved command that was executed.

### git

- `git.status(repo_path)`
- `git.diff(repo_path) -> patch`
- `git.apply(repo_path, patch)`

### browser

- `browser.open(url)`
- `browser.act(actions)`

### http

- `http.fetch(url, method, headers, body) -> {status, headers, body}`

## Replay determinism

- Tool results must be recorded in trace.
- Runtime may choose replay from trace rather than re‑run.
- Non‑deterministic tools must be explicit (with `nondeterministic: true`) and flagged as not strictly replayable.
