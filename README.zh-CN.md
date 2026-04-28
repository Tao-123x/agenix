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

核心规范：

- [README](README.md) / [README.zh-CN](README.zh-CN.md)
- [Agenix Spec](specs/agenix-spec-v0.1.md) / [Agenix Spec.zh-CN](specs/agenix-spec-v0.1.zh-CN.md)
- [Skill Manifest](specs/skill-manifest.md) / [Skill Manifest.zh-CN](specs/skill-manifest.zh-CN.md)
- [Agentfile](specs/agentfile.md) / [Agentfile.zh-CN](specs/agentfile.zh-CN.md)
- [Tool Contracts](specs/tool-contract.md) / [Tool Contracts.zh-CN](specs/tool-contract.zh-CN.md)
- [Capabilities](specs/capability.md) / [Capabilities.zh-CN](specs/capability.zh-CN.md)
- [Trace](specs/trace.md) / [Trace.zh-CN](specs/trace.zh-CN.md)
- [Policy](specs/policy.md) / [Policy.zh-CN](specs/policy.zh-CN.md)
- [v0.1.0 Release Notes](docs/releases/v0.1.0.md) / [中文](docs/releases/v0.1.0.zh-CN.md)
- [v0.2.0 Plan](docs/releases/v0.2.0-plan.md) / [中文](docs/releases/v0.2.0-plan.zh-CN.md)

教程：

- [Write your first skill](docs/tutorials/write-your-first-skill.md) / [中文](docs/tutorials/write-your-first-skill.zh-CN.md)

示例文档：

- [repo.fix_test_failure README](examples/repo.fix_test_failure/README.md) / [中文](examples/repo.fix_test_failure/README.zh-CN.md)
- [repo.fix_test_failure verifier](examples/repo.fix_test_failure/verifier.md) / [中文](examples/repo.fix_test_failure/verifier.zh-CN.md)
- [repo.analyze_test_failures README](examples/repo.analyze_test_failures/README.md) / [中文](examples/repo.analyze_test_failures/README.zh-CN.md)
- [repo.analyze_test_failures verifier](examples/repo.analyze_test_failures/verifier.md) / [中文](examples/repo.analyze_test_failures/verifier.zh-CN.md)
- [repo.apply_small_refactor README](examples/repo.apply_small_refactor/README.md) / [中文](examples/repo.apply_small_refactor/README.zh-CN.md)
- [repo.apply_small_refactor verifier](examples/repo.apply_small_refactor/verifier.md) / [中文](examples/repo.apply_small_refactor/verifier.zh-CN.md)

## Runtime v0 快速开始

前置条件：

- Go 1.22+
- Python 3，并安装 `pytest`

在全新的 Ubuntu 主机上，先安装 runtime 依赖：

```bash
sudo apt-get update
sudo apt-get install -y golang-go python3 python3-pytest
```

如果发行版软件源里的 Go 版本低于 1.22，请先安装更新的 Go toolchain。

从 V0.2 authoring 模板创建一个可运行的 skill skeleton：

```bash
go run ./cmd/agenix init templates
go run ./cmd/agenix init templates --json
go run ./cmd/agenix init skill repo.demo_skill --template python-pytest -o /tmp/repo.demo_skill
go run ./cmd/agenix validate /tmp/repo.demo_skill/manifest.yaml
go run ./cmd/agenix build /tmp/repo.demo_skill -o /tmp/repo.demo_skill.agenix
go run ./cmd/agenix run /tmp/repo.demo_skill.agenix --adapter python-pytest-template
go run ./cmd/agenix check /tmp/repo.demo_skill --adapter python-pytest-template
go run ./cmd/agenix check /tmp/repo.demo_skill --adapter python-pytest-template --json > /tmp/report.json
go run ./cmd/agenix validate /tmp/report.json
```

生成的 skill 会包含一个最小 pytest fixture、受 policy 约束的 manifest、command
和 schema verifier，以及一个本地确定性的模板 adapter。这个 adapter 不修改文件；它通过
`fs.list` 读取 fixture 列表，返回结构化输出，并把最终成功与否交给 verifier 判断。
`agenix check` 是一条命令的 authoring gate：它会校验 manifest、构建临时 artifact、
运行 artifact、校验 trace、重跑 verifier，并回放 trace summary。CI 或其他 agent
需要稳定的机器可读报告时，可以传 `--json`；报告使用 `kind: check_report`，
并且可以用 `agenix validate` 校验。gate 失败时，`--json` 仍会把失败 report
写到 stdout，包含 `error_class`、`error_message` 和可用的 `trace_path`，同时
CLI 保持非零退出码。

从失败测试模板创建一个可写修复 skill：

```bash
go run ./cmd/agenix init skill repo.demo_fix --template repo-fix-test-failure -o /tmp/repo.demo_fix
python3 -m pytest -q /tmp/repo.demo_fix/fixture
go run ./cmd/agenix check /tmp/repo.demo_fix --adapter repo-fix-test-failure-template --json > /tmp/fix-report.json
go run ./cmd/agenix validate /tmp/fix-report.json
```

`pytest` 命令在 `check` 前应该失败。模板 adapter 随后会通过 runtime 的 `fs.write`
工具修复 `fixture/mathlib.py`，并在 verifier 通过后把 changed file 写进 check report。

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

canonical demo 会有意修改
`examples/repo.fix_test_failure/fixture/mathlib.py`。如果你还想把当前源码目录继续
当作初始失败 fixture 使用，请先恢复它：

```bash
git restore examples/repo.fix_test_failure/fixture/mathlib.py
```

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
- `specs/check-report.schema.json`

可以使用 `agenix validate <path>` 对 manifest、trace 或 check report 做基于已发布 schema 的契约检查。

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
go run ./cmd/agenix run examples/repo.analyze_test_failures/manifest.yaml --adapter heuristic-analyze
go run ./cmd/agenix build examples/repo.analyze_test_failures -o repo.analyze_test_failures.agenix
go run ./cmd/agenix run repo.analyze_test_failures.agenix
```

这个 skill 会分析一个已知失败的 pytest fixture，并且不声明任何写权限。成功运行时，
它会报告空的 `changed_files` 列表，并且 trace 中不会出现 `fs.write` 事件。可选的
`--adapter heuristic-analyze` 路径使用单独的只读 builtin adapter，而不是默认的 fake scripted
adapter，但仍然走同一套 runtime policy、trace、verifier、replay 和 artifact 流程。

如果你运行的是可选的 provider-backed `--adapter openai-analyze` 路径，失败时仍然会
报告 `DriverError`。Provider-backed OpenAI 请求默认 30 秒超时，响应体上限默认 1 MiB；
本地 smoke 运行时可以通过 `AGENIX_OPENAI_TIMEOUT_MS` 或
`AGENIX_OPENAI_MAX_RESPONSE_BYTES` 覆盖。当上游响应里带有状态码和消息时，Agenix
会保留这些信息；对于 429 响应，还可能附带 retry-after 提示。超出响应体上限的 provider
响应会报告为 `DriverError`；Provider HTTP 超时会单独报告为 `Timeout`。

运行受限重构 demo：

```bash
go run ./cmd/agenix run examples/repo.apply_small_refactor/manifest.yaml
go run ./cmd/agenix build examples/repo.apply_small_refactor -o repo.apply_small_refactor.agenix
go run ./cmd/agenix run repo.apply_small_refactor.agenix
```

这个 skill 只允许写 `greeter.py`。成功运行时，它会报告这一个文件、执行测试，并运行一个
verifier 来检查重构后的结构。

运行 V0 release gate：

```bash
go run ./cmd/agenix acceptance
```

`agenix acceptance` 是 reference runtime 的 canonical V0 acceptance 命令。它会在本地
对三个 canonical skill 运行 acceptance sweep：manifest 校验、可移植 capsule 的 build
与 inspect、artifact 执行、trace 校验、verifier 重新运行、trace replay、本地 registry
publish / pull，以及直接使用 registry reference 执行。

在 cut 或 review V0 release 前，本地完整验证命令是：

```bash
go run ./cmd/agenix acceptance
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
```

V0 acceptance 有意限定为本地 reference-runtime gate。它不声称提供强 sandbox、远程执行器
语义、registry trust、签名、OCI 分发或 provider-backed 远程 adapter 覆盖。可选的
`openai-analyze` smoke 路径仍然不属于默认 V0 acceptance sweep。

参见 [V0 release checklist](docs/v0-release-checklist.zh-CN.md)。

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
- `agenix build/run/verify/replay/validate/publish/pull/acceptance`
- skill package 的 registry push/pull 闭环（至少本地 filesystem registry）
- 用 acceptance gate 在 canonical skills 上验证 portability invariants

## 参与贡献

Agenix 想做的是面向 agent 的 “OCI thinking”。

PR 应该优先围绕这些方向：

- 更强的契约，更少魔法
- 更严的 policy 执行，更少 agent 混乱
- 更完整的验证与回放，更少口头信任
- 更好的跨 OS 可移植性，更少平台假设

---

**一句话总结：**让能力可移植。
