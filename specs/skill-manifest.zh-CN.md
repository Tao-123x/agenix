# Skill Manifest（v0.1 草案）

[English](skill-manifest.md) | [简体中文](skill-manifest.zh-CN.md)

## 目的

把一个可复用的 agent capability 描述为可移植、可验证的 package。

## 必填字段

- `name`（字符串）
- `version`（semver）
- `description`（字符串）
- `capabilities`（capability 要求）
- `tools`（所需 tool namespace）
- `permissions`（network / filesystem / tool scope）
- `inputs`（JSON Schema）
- `outputs`（JSON Schema）
- `verifiers`（列表）
- `recovery`（checkpoint 策略）

## 示例

```yaml
apiVersion: agenix/v0.1
kind: Skill

name: repo.fix_test_failure
version: 0.1.0
description: Locate failing tests, patch code, and verify via test runner.

capabilities:
  requires:
    tool_calling: true
    structured_output: true
    max_context_tokens: 32000
    reasoning_level: medium

tools:
  - fs
  - shell
  - git

permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
      - ${repo_path}
  shell:
    allow:
      - run: ["pytest", "-q"]
      - run: ["python", "-m", "pip", "--version"]

inputs:
  type: object
  required: [repo_path]
  properties:
    repo_path:
      type: string
      description: Absolute or repo‑relative path in the runtime workspace.

outputs:
  type: object
  required: [patch_summary, changed_files]
  properties:
    patch_summary:
      type: string
    changed_files:
      type: array
      items:
        type: string

verifiers:
  - type: command
    name: run_tests
    run: ["python3", "-m", "pytest", "-q"]
    cwd: ${repo_path}
    policy:
      executable: python3
      cwd: ${repo_path}
      timeout_ms: 120000
    success:
      exit_code: 0
    artifacts:
      logs: true

  - type: schema
    name: output_schema_check
    schemaRef: "outputs"

recovery:
  strategy: checkpoint
  intervals: 5
```

## 说明

- `${repo_path}` 是 runtime substitution。
- Verifier 不是可选项；“agent 说做完了”不算 verifier。
- 权限必须显式声明。
- Command verifier 可以使用 `cmd` 或 `run`，但更推荐 `run`，因为它避免 shell 字符串解析，
  在跨平台场景里更利于确定性参数处理。
- `run` 形式的 command verifier 必须声明 `policy.executable`、`policy.cwd` 和
  `policy.timeout_ms`。
- Verifier policy 比较会先看请求中的 executable，再做平台 alias 解析。
- Verifier trace entry 会记录 `cmd`、`resolved_cmd`、`cwd` 和 `timeout_ms`。
- Skill 可以声明顶层 `redaction` 块。
- `redaction.keys` 会把结构化敏感字段名追加到 runtime 默认集合里。
- `redaction.patterns` 会追加文本掩码规则，字段包括 `name`、`regex` 和 `secret_group`。
- `redaction.patterns[*].secret_group` 是从 1 开始的 regex 捕获组编号，不能超过
  `regex` 中实际捕获组数量。
- 非法 redaction pattern 必须在 manifest load 阶段以 `InvalidInput` 失败。

## 当前已实现的最小校验

reference runtime 现在发布了一个 schema 文件：
`specs/manifest.schema.json`。当前实现仍然把 `LoadManifest` 视为权威的 runtime parser 和
语义校验器，而 `agenix validate` 会在它成功后，再做基于已发布 schema 的文档校验。
当以下字段缺失时，`LoadManifest` 会返回 `InvalidInput`：

- `apiVersion`
- `kind`
- `name`
- `version`
- `description`
- `tools`
- `outputs.required`
- `verifiers`
- 每个 verifier 的 `type`
- 每个 verifier 的 `name`
- 每个 command verifier 的 `cmd` 或 `run`
- 每个 `run` verifier 的 `policy`
- 每个 `run` verifier 的 `policy.executable`
- 每个 `run` verifier 的 `policy.cwd`
- 每个 `run` verifier 的 `policy.timeout_ms`
- 每个 `redaction.patterns[*].name`
- 每个 `redaction.patterns[*].regex`
- 每个 `redaction.patterns[*].secret_group`

parser 目前能理解的 `capabilities.requires` 子集包括：

- `tool_calling`
- `structured_output`
- `max_context_tokens`
- `reasoning_level`

当前 validator 还**不会**校验 semver 格式、permission scope 完整性、input/output
property schema、超出当前最小实现范围的 verifier type-specific 字段，以及 recovery
设置。Registry discovery 仍然会在排序上特殊对待合法 semver 值；这是一个排序契约，
不是 manifest validation 保证。
