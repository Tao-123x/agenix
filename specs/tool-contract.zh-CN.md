# Tool Contracts（v0.1 草案）

[English](tool-contract.md) | [简体中文](tool-contract.zh-CN.md)

## 全局要求

- 每一次 tool call 都必须产出 trace entry。
- Tool 响应必须可 JSON 序列化。
- 错误必须稳定，并归类到以下类型之一：
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

约束：

- 写入必须发生在已声明的 write scope 之内。

### shell

- `shell.exec(cmd, cwd, timeout) -> {stdout, stderr, exit_code}`

约束：

- 只能执行 policy / tool whitelist 允许的命令。
- v0 不宣称提供 OS 级别的网络沙箱。
- 当 `permissions.network` 为 `false` 时，runtime 管理的子进程启动只支持那些
  明确处理为 local-only 或 network-denied 的 launcher 类型。
- 目前这意味着：Python 子进程通过 runtime 注入的 network-denied launcher
  运行；离线安全的本地 git 子命令仍然允许；不受支持的 executable 一律以
  `PolicyViolation` 失败关闭。
- Runtime 可以在 policy 比较之后、执行之前，应用已记录的平台 executable alias。
- v0 目前只定义了一个 alias：在 Windows 上，如果 `python3` 指向 Microsoft Store
  shim，而 `python` 可用，则 `python3` 可以解析成 `python`。
- Alias normalization 不能改参数；它只能替换 executable token。
- Policy 比较使用 adapter 请求的命令，也就是 alias 解析前的命令。
- Trace entry 必须同时记录请求命令和最终执行的解析后命令。

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

- `run` 形式的 command verifier 必须声明 `policy.executable`、`policy.cwd` 和
  `policy.timeout_ms`。
- Verifier 子进程启动与 runtime 管理的 tool 执行一样，使用同一套
  `permissions.network=false` 规则。
- Verifier policy 比较使用请求的 executable，也就是平台 alias 解析之前的值。
- Command verifier trace entry 会记录 `cmd`、`resolved_cmd`、`cwd` 和 `timeout_ms`。
- 旧的 `cmd` verifier 保持向后兼容，但不满足采购级别的 verifier policy contract。

## Replay determinism

- Tool 结果必须记录在 trace 中。
- Runtime 可以选择直接从 trace replay，而不是重新执行。
- 非确定性工具必须显式声明（`nondeterministic: true`），并标记为不能严格回放。

## Adapter failure taxonomy

- `InvalidInput` 只用于 manifest / trace 格式错误，以及 CLI 用法错误。
- `UnsupportedAdapter` 只用于 adapter 选择和 preflight 不匹配，包括：
  - adapter 名称不存在
  - adapter 不支持请求的 skill
  - adapter capability 集合不满足 `capabilities.requires`
- `DriverError` 用于 adapter 在通过 selection 和 preflight 之后的执行失败。
- `VerificationFailed` 仍然表示 adapter 执行完成之后的 verifier 失败。
