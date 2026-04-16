# Capabilities（v0.1 草案）

[English](capability.md) | [简体中文](capability.zh-CN.md)

## 模型 capability 要求

- 支持 `tool_calling`
- 支持 `structured_output` 模式
- token 预算（上下文窗口）
- 延迟偏好（可选）
- 推理等级（启发式）

## 协商

- Skill 声明 `requires`。
- Runtime 检查 model profile。
- 结果：
  - **ok：** 继续执行
  - **degraded：** 带告警继续执行
  - **fail：** runtime 报告 `unsupported`

## 当前已实现的最小能力

当前 runtime 会在任何 tool call 之前执行本地 preflight 检查：

- manifest 可以声明 `capabilities.requires`
- adapter 会报告 `name`、`model_profile`、`supported_skills`，以及最小 capability 集合
- runtime 会在 adapter 执行前拒绝不支持的 skill
- runtime 会拒绝缺失 `tool_calling`、`structured_output`、`max_context_tokens`
  不足或 `reasoning_level` 不足的情况
- trace 会记录 `adapter` 事件，用于表示 selection 和 capability check 结果

当前 runtime 还没有实现 degraded execution path，也没有实现 vendor-specific
capability discovery。

## 失败报告

- 必须包含：哪一条 requirement 失败、缺失了什么 capability，以及建议动作。
