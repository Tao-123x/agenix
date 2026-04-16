# Policy 模型（v0.1 草案）

[English](policy.md) | [简体中文](policy.zh-CN.md)

## Policy 域

- **Filesystem：** 读 / 写 scope
- **Network：** allow / deny
- **Shell execution：** 命令 allowlist
- **Verifier execution：** `run` verifier 的 executable / cwd / timeout contract
- **Browser actions：** 允许 / 拒绝的模式
- **Credentials：** 不允许外泄
- **Cost / time constraints**

## Enforcement

- Runtime 必须拒绝超出 policy 的 tool call。
- Violation 必须成为可追踪事件。
- Shell allowlist 必须精确匹配 adapter 请求的命令。
- 平台 executable resolution 只能在 shell policy 成功后发生。
- 如果 executable resolution 改写了命令，tool trace 必须同时记录请求命令和解析后的命令。
- `run` 形式的 command verifier 有自己最小的 policy contract：
  `policy.executable`、`policy.cwd` 和 `policy.timeout_ms`。
- Verifier policy 比较会先基于 verifier 请求的 executable，再做平台 alias 解析。
- 出于向后兼容，旧的 `cmd` verifier 仍然可以执行，但它们不满足更严格的 verifier policy contract。

## 审批（可选）

- 某些动作在工作流中需要显式审批步骤，例如写远程系统或其他高影响操作。
