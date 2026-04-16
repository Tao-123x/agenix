# Agenix

[English](README.md) | [简体中文](README.zh-CN.md)

**面向可复用、可验证、跨模型 agent skill 的可移植 runtime。**

> Docker 让软件具备可移植性。  
> **Agenix 让能力具备可移植性。**

## Agenix 是什么？

Agenix 是一个面向 agent 的开放 runtime 与打包系统。

它的目标是让 agent 能力具备：

- 跨模型可移植
- 跨 Linux/macOS/Windows 可移植
- 跨宿主环境可移植（本地 / 容器 / 远程执行器）
- 可在不同 agent / 团队之间复用
- 不再依赖口头信任，而是可验证

在 Agenix 里，**skill** 是一等 artifact，而不是零散的 prompt。

## 问题是什么？

容器解决了工程里这样一种痛点：

> “在我机器上是好的。”

而 agent 带来了另一种可移植性问题：

> “这个 agent 只在*我的*栈、*我的*模型、*我的*机器上能跑。”

一个真正可复用的 skill，必须做到：

- 一次打包
- 明确声明
- 在约束下执行
- 自动验证
- 可回放 / 可审计

## 核心设计（v0.1）

Agenix 定义了五层：

1. **Model Layer**：模型可替换，关注 capability 要求，而不是厂商绑定
2. **Tool Layer**：稳定的工具契约（`fs.*`、`shell.*`、`git.*`、`browser.*` 等）
3. **Skill Layer**：声明式 manifest，描述目的、I/O schema、权限、verifier、恢复策略
4. **Runtime Layer**：负责策略执行、工具挂载、checkpoint、trace、replay
5. **Artifact Layer**：把 skill / package / trace 变成可分发 artifact

## 当前交付物（bootstrap）

- `README.md`
- `cmd/agenix/`：v0 reference runtime CLI
- `internal/agenix/`：manifest、policy、tool、trace、verifier、fake adapter 核心实现
- `specs/`
  - `agenix-spec-v0.1.md`：目录、术语表、核心不变量
  - `skill-manifest.md`：最关键的接口
  - `agentfile.md`：打包 / build 契约
  - `tool-contract.md`：工具 schema、错误语义、replay 规则
  - `capability.md`：capability 要求与协商
  - `trace.md`：可回放 trace schema
  - `policy.md`：安全 / policy 模型
- `examples/`
  - `repo.fix_test_failure/`：canonical demo，打补丁并验证
  - `repo.analyze_test_failures/`：canonical demo，只读分析失败测试
  - `repo.apply_small_refactor/`：canonical demo，受限写入的小型重构

## 双语文档入口

以下核心入口都已提供英文版和简体中文版：

- [README](README.md) / [README.zh-CN](README.zh-CN.md)
- [Agenix Spec](specs/agenix-spec-v0.1.md) / [Agenix Spec.zh-CN](specs/agenix-spec-v0.1.zh-CN.md)
- [Skill Manifest](specs/skill-manifest.md) / [Skill Manifest.zh-CN](specs/skill-manifest.zh-CN.md)
- [Agentfile](specs/agentfile.md) / [Agentfile.zh-CN](specs/agentfile.zh-CN.md)
- [Tool Contracts](specs/tool-contract.md) / [Tool Contracts.zh-CN](specs/tool-contract.zh-CN.md)
- [Capabilities](specs/capability.md) / [Capabilities.zh-CN](specs/capability.zh-CN.md)
- [Trace](specs/trace.md) / [Trace.zh-CN](specs/trace.zh-CN.md)
- [Policy](specs/policy.md) / [Policy.zh-CN](specs/policy.zh-CN.md)

## Runtime v0 快速开始

前置条件：

- Go 1.22+
- Python 3，并安装 `pytest`

在仓库根目录运行 canonical demo：

```bash
python3 -m pytest -q examples/repo.fix_test_failure/fixture
go run ./cmd/agenix run examples/repo.fix_test_failure/manifest.yaml
go run ./cmd/agenix replay .agenix/runs/<run_id>/trace.json
go run ./cmd/agenix verify .agenix/runs/<run_id>/trace.json
go run ./cmd/agenix validate examples/repo.fix_test_failure/manifest.yaml
```

第一条命令应该失败，因为 fixture 的初始状态就是坏的。`agenix run`
会通过 runtime 的 `fs.write` 工具完成修复，把每一次 tool 和 verifier
事件写进 JSON trace，并且只有在 verifier 通过后才报告成功。

构建并检查一个可移植 capsule：

```bash
go run ./cmd/agenix build examples/repo.fix_test_failure -o repo.fix_test_failure.agenix
go run ./cmd/agenix inspect repo.fix_test_failure.agenix
go run ./cmd/agenix run repo.fix_test_failure.agenix
```

artifact 是一个 gzip 压缩的 tar capsule，内部包含 `manifest.yaml`、
`files/...` 和 `agenix.lock.json`。lockfile 记录了 skill 标识、源文件 digest、
创建时间、构建 provenance 和 artifact digest，因此 capsule 可以在不信任原始目录
布局的前提下被移动和检查。`inspect` 现在会在可用时输出 `created_at`、
`built_by`、`build_host` 和 `source_commit`。运行 capsule 时，runtime 会把它
materialize 到 run 目录下的工作区里，以保留 verifier replay 所需的 manifest 相对路径。

已发布的 schema 文件位于：

- `specs/manifest.schema.json`
- `specs/trace.schema.json`

可以使用 `agenix validate <path>` 对 manifest 或 trace 做基于已发布 schema 的契约检查。

把 capsule 发布到本地 registry，再拉回本地：

```bash
go run ./cmd/agenix publish repo.fix_test_failure.agenix
go run ./cmd/agenix pull repo.fix_test_failure@0.1.0 -o pulled.fix_test_failure.agenix
```

默认 registry 根目录是 `~/.agenix/registry`。`publish` 是显式动作；对于相同 digest，
它是幂等的。如果你尝试为同一个 `skill@version` 发布不同 digest，Agenix 会拒绝，
强制你提升版本号，从而保证 `skill@version` 的确定性。`pull` 当前接受两种引用：
`skill@version` 或完整的 `sha256:...` digest。registry index entry 还会在可用时记录
`published_at`、`published_by` 和 artifact 的 `source_commit`。

registry discovery 也是显式的：

```bash
go run ./cmd/agenix registry list
go run ./cmd/agenix registry show repo.fix_test_failure
go run ./cmd/agenix registry resolve repo.fix_test_failure@0.1.0
```

`registry list` 会打印所有已索引条目，`registry show` 会按精确 skill 名过滤，
`registry resolve` 会打印 `skill@version` 或 `sha256:...` 对应的精确 index entry。
当同一个 skill 有多个 registry entry 时，`list` 和 `show` 会对合法 semver 版本做
语义升序排序。暂时仍接受非 semver 字符串，但它们会排在合法 semver 之后。

`inspect` 和 `run` 也可以直接使用 registry 引用：

```bash
go run ./cmd/agenix inspect repo.fix_test_failure@0.1.0
go run ./cmd/agenix run repo.fix_test_failure@0.1.0
```

如果你需要使用非默认 registry 根目录，可以给 `publish`、`pull`、`inspect`、
`run` 或 `registry` 传 `--registry <dir>`。

运行只读分析 demo：

```bash
go run ./cmd/agenix run examples/repo.analyze_test_failures/manifest.yaml
go run ./cmd/agenix build examples/repo.analyze_test_failures -o repo.analyze_test_failures.agenix
go run ./cmd/agenix run repo.analyze_test_failures.agenix
```

这个 skill 会分析一个已知失败的 pytest fixture，并且不声明任何写权限。成功运行时，
它会报告空的 `changed_files` 列表，并且 trace 中不会出现 `fs.write` 事件。

运行受限重构 demo：

```bash
go run ./cmd/agenix run examples/repo.apply_small_refactor/manifest.yaml
go run ./cmd/agenix build examples/repo.apply_small_refactor -o repo.apply_small_refactor.agenix
go run ./cmd/agenix run repo.apply_small_refactor.agenix
```

这个 skill 只允许写 `greeter.py`。成功运行时，它会报告这一个文件、执行测试，并运行一个
verifier 来检查重构后的结构。

## 路线图与完成定义（DoD）

### Phase 0：规范（DoD）
- 术语冻结（skill / runtime / capability / trace / verifier）
- Skill manifest 草案 + JSON Schema 版本
- Tool contract、错误类、replay 规则草案
- Trace schema 草案（run id / tool calls / verifier output / redaction rule）

### Phase 1：Runtime 原型（DoD）
- 使用 model adapter（tool calling）端到端执行一个 `Skill Manifest`
- 为每次 tool call 产出 trace
- 运行 verifier（command-based + schema-based）
- 至少对 `fs.*` / `shell.*` / `git.*` 做跨 OS 检查

### Phase 2：CLI 与 Registry（DoD）
- `agenix build/run/verify/replay/publish/pull`
- skill package 的 registry push/pull 闭环（至少本地 filesystem registry）
- 用 benchmark 套件验证 portability invariants

## 参与贡献

Agenix 想做的是面向 agent 的 “OCI thinking”。

PR 应该优先围绕这些方向：

- 更强的契约，更少魔法
- 更严的 policy 执行，更少 agent 混乱
- 更完整的验证与回放，更少口头信任
- 更好的跨 OS 可移植性，更少平台假设

---

**一句话总结：**让能力可移植。
