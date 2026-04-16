# Trace 规范（v0.1 草案）

[English](trace.md) | [简体中文](trace.zh-CN.md)

## 必填字段

- `run_id`（uuid）
- `skill`
- `model_profile`
- `timestamp`
- `policy`（实际应用的权限）
- `events[]`

## 事件类型

- `tool_call`：name、request、result、error、duration
- `checkpoint`：用于恢复的标记
- `verifier`：name、result、output
- `final`：status

## Redaction

- 持久化 trace 文件必须经过 runtime redaction 后再写盘。
- Runtime 会在写入 trace JSON 之前，对常见携带 secret 的键和值模式应用内建 redaction 规则。
- Skill 可以通过顶层 manifest 块 `redaction.keys` 和 `redaction.patterns` 追加规则。
- Redaction 应尽量保留周围的审计上下文，并在可能时只把 secret value 替换为 `[REDACTED]`。
- 如果 trace redaction 失败，runtime 必须 fail closed，并拒绝写入 trace。

## Replay

replay runner 可以：

- 重新执行
- 或者在支持时，根据 trace 中的结果做确定性回放

## 当前已实现的最小校验

reference runtime 现在发布了一个 schema 文件：`specs/trace.schema.json`。
当前实现仍然把 `ReadTrace` 视为权威的 runtime parser 和最小语义校验器，而
`agenix validate` 会在它成功后再做基于已发布 schema 的文档校验。
当以下字段缺失时，`ReadTrace` 会返回 `InvalidInput`：

- `run_id`
- `skill`
- `model_profile`
- `final.status`
- 每个 event 的 `type`
- 每个 event 的 `name`

这样可以防止 `verify` 和 `replay` 接受那些明显损坏的 trace。当前 validator
还不会校验 timestamp 是否存在或格式是否正确、policy 结构、允许的 event type 取值、
request/result/error payload schema、status 枚举值，或确定性 replay 的完整性。
