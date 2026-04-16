# Agenix 规范 v0.1（草案）

[English](agenix-spec-v0.1.md) | [简体中文](agenix-spec-v0.1.zh-CN.md)

## 1. 介绍

*目标：* 让 agent skill 具备可移植、可验证、可回放能力。

*范围：* skill 打包、runtime 行为、tool contract、capability 协商、trace 格式。

## 2. 术语

- **Agent** / **Skill** / **Tool** / **Tool Driver** / **Capability** / **Runtime** / **Trace** / **Verifier** / **Policy** / **Checkpoint**

## 3. 设计目标（不可妥协）

* 跨模型 / OS / 宿主环境可移植
* 用声明式契约替代隐式行为
* 把验证视为一等交付物
* 每次运行都可回放、可审计

## 4. 分层系统模型

- **Model Layer**
- **Tool Layer**
- **Skill Layer**
- **Runtime Layer**
- **Artifact Layer**

## 5. Capabilities 与协商

- Capability Manifest
- runtime 与 model profile 之间的协商协议
- 失败模式：`unsupported`、`degraded`、`adapter-required`

## 6. Tool Contracts

- 命名空间
- 请求 / 响应 schema
- 错误类
- replay 的确定性规则
- driver 要求（Linux/macOS/Windows）

## 7. Skill Manifest

- 必填字段
- schema 定义（已发布 JSON Schema）
- I/O 类型规则
- side-effect 声明
- verifier 要求

## 8. Agentfile

- model profile 要求
- tool 挂载
- 约束（network / filesystem / write scope）
- 入口工作流

## 9. Runtime 行为

- 加载与执行生命周期
- 上下文构造
- checkpoint 与恢复
- 失败报告

## 10. Trace 规范

- 必需元素
- redaction 规则
- replay 要求
- 已发布 JSON Schema

## 11. Verifier 规范

- command-based / schema-based / custom hook
- 输出 schema
- 成功标准

## 12. 安全与 Policy

- 最小权限
- policy 类型
- 审批门
- violation 处理

## 13. 打包与分发

- artifact 格式
- registry 命名与完整性
- provenance

### 参考 artifact 布局（v0.1）

reference runtime 可以构建一个本地 capsule，布局如下：

```text
manifest.yaml
files/...
agenix.lock.json
```

这个 capsule 是一个 gzip 压缩的 tar 文件。`agenix.lock.json` 是 v0.1 的最小
provenance 记录：artifact 版本、skill 标识、manifest digest、payload 文件 digest、
创建时间、构建者身份、可选的 source commit，以及 artifact digest。

v0.1 的 artifact 完整性是本地的、基于 lockfile 的。lock 会记录 `manifest.yaml`
以及每个 materialized `files/...` payload 的 sha256 digest。在 inspect 和
materialize 时，runtime 会基于 capsule 内容重新计算这些 digest，在有记录时校验
payload size，并拒绝那些缺失、重复、被修改或未在 lock 中声明的 materialized
payload entry。v0.1 还没有定义签名、registry trust 或 OCI 分发语义。

reference runtime 现在还包含一个本地 filesystem registry，默认根目录是
`~/.agenix/registry`。已发布的 capsule 会被复制进这个 registry，并通过精确的
`skill@version` 或完整 digest 做显式索引查找。v0.1 的 registry 语义刻意保持收敛：

- `publish` 是显式动作；`build` 不意味着 publish。
- `pull` 也是显式动作，适用于调用方需要拿到一个复制出的本地 capsule。
- `registry list`、`registry show <skill>` 和 `registry resolve <ref>` 提供显式 discovery，
  但不改变 retrieval 语义。
- 在同一个 skill 内，registry discovery 会按语义升序排列合法 semver 版本；
  非 semver 字符串仍然允许存在，但会排在合法 semver 之后。
- `run` 和 `inspect` 可以直接解析精确的 registry reference。
- 如果某个已发布的 `skill@version` 试图对应不同 digest，registry 会拒绝。
- registry entry 会记录发布时间、发布者身份，以及来自 artifact 的可选 source commit 元数据。
- 这个 registry 仅限本地，不定义签名、trust policy 或远程分发语义。

运行 capsule 时，runtime 会把它 materialize 到 run workspace 中：
`manifest.yaml` 保持在 workspace 根目录，`files/...` 条目则会在恢复时去掉 `files/`
前缀。该 workspace 会保留下来，以支持 trace verification。

## 14. 参考 Demo

- `repo.fix_test_failure`

## 15. 未来方向

- Agent compose
- Memory federation
- Benchmark suite

---

## 不变量（v0.1）

* 每一次 tool call 都必须可追踪。
* 每一次 run 都必须可验证，或者显式标记为 `unverified` 并说明原因。
* Skill 的 side-effect 必须声明；越界 side-effect 属于 policy violation。
