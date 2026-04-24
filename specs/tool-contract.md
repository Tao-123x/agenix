# Tool Contracts (v0.1 Draft)

[English](tool-contract.md) | [简体中文](tool-contract.zh-CN.md)

## Global requirements

- Every tool call must produce trace entries.
- Tool responses must be JSON-serializable.
- Errors must be stable and classify as:
  - `InvalidInput`
  - `UnsupportedAdapter`
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
- When the runtime knows the manifest or workspace root, repo-relative paths
  are resolved against that root before scope checks.
- Scope checks must resolve existing symlinked path segments before comparing a
  target path against declared read/write scope.

### shell

- `shell.exec(cmd, cwd, timeout) -> {stdout, stderr, exit_code}`

Constraints:

- Only allowed commands (by policy/tool whitelist) may run.
- v0 does not claim OS-level network sandboxing.
- When `permissions.network` is `false`, runtime-managed subprocess launch is
  supported only for launcher types with explicit local-only or network-denied
  handling.
- Today this means Python subprocesses run under a runtime-injected
  network-denied launcher, offline-safe local git subcommands remain allowed,
  and unsupported executables fail closed as `PolicyViolation`.
- Runtime may apply documented platform executable aliases after policy
  comparison and before execution.
- v0 only defines one alias: on Windows, `python3` may resolve to `python`
  when `python3` points at the Microsoft Store shim and `python` is available.
- Alias normalization must not alter arguments. It may only replace the
  executable token.
- Policy comparison uses the command requested by the adapter, before alias
  resolution.
- Trace entries must record both the requested command and the resolved command
  that was executed.

### git

- `git.status(repo_path)`
- `git.diff(repo_path) -> patch`
- `git.apply(repo_path, patch)`

### browser

- `browser.open(url)`
- `browser.act(actions)`

### http

- `http.fetch(url, method, headers, body) -> {status, headers, body}`

## Verifier contract

- `run` command verifiers must declare `policy.executable`, `policy.cwd`, and
  `policy.timeout_ms`.
- Verifier subprocess launch uses the same `permissions.network=false` rule as
  runtime-managed tool execution.
- Verifier policy comparison uses the requested executable before platform alias
  resolution.
- Command verifier trace entries record `cmd`, `resolved_cmd`, `cwd`, and
  `timeout_ms`.
- Legacy `cmd` verifiers remain backward compatible but do not satisfy the
  procurement-grade verifier policy contract.

## Replay determinism

- Tool results must be recorded in trace.
- Runtime may choose replay from trace rather than re-run.
- Non-deterministic tools must be explicit (with `nondeterministic: true`) and
  flagged as not strictly replayable.

## Adapter failure taxonomy

- `InvalidInput` is reserved for malformed manifests, malformed traces, and CLI
  usage errors.
- `UnsupportedAdapter` is reserved for adapter selection and preflight
  mismatches, including:
  - unknown adapter name
  - adapter does not support the requested skill
  - adapter capability set does not satisfy `capabilities.requires`
- `DriverError` covers adapter execution failures after selection and preflight
  have already succeeded.
- Provider-backed adapter failures stay `DriverError`; when the provider
  response includes a status/message, the runtime should surface those details,
  and 429 responses may also include retry-after guidance.
- Provider-backed HTTP timeouts are classified as `Timeout`, not
  `DriverError`.
- `VerificationFailed` remains the verifier surface after adapter execution has
  completed.
