# Agentfile 规范（v0.1 草案）

[English](agentfile.md) | [简体中文](agentfile.zh-CN.md)

Agentfile 用来描述如何构建和运行一个 agent package。它类似于 `Dockerfile`，但面向的是 skills 和 agents。

## 关键部分

- `from`：基础 runtime image 或 profile
- `model`：首选 model profile（关注 capability 要求，而不是厂商名）
- `tools`：挂载进 runtime 环境的工具
- `skills`：引用的 skill package
- `memory`：是否挂载持久化卷
- `constraints`：runtime 限制（最大运行时间、成本上限、network policy、write scope）
- `entry`：入口 skill 或工作流

## 示例

```yaml
from: agent-base:ubuntu-24.04

model:
  provider: openai
  profile: reasoning-medium

tools:
  - shell
  - filesystem
  - git

skills:
  - skills/code/edit-repo@1.0.0
  - skills/test/run-pytest@1.1.0

memory:
  mounted: false

constraints:
  network: off
  max_runtime_minutes: 20
  max_cost_usd: 1.50

entry:
  skill: workflow.fix_and_validate
```

## Runtime v0.1 capsule 布局

`agenix build <skill-dir> -o <artifact>` 会生成一个 gzip 压缩的 tar artifact，
布局如下：

```text
manifest.yaml
files/...
agenix.lock.json
```

规则：

- `manifest.yaml` 从 skill 目录根部复制。
- `files/...` 保存 skill 目录中的其他文件，并保留相对路径。
- `.DS_Store`、`.agenix`、`.pytest_cache`、`__pycache__` 和 `*.pyc` 文件会被排除。
- `agenix.lock.json` 记录 artifact version、skill name/version、manifest digest、
  source file digests、创建时间和 artifact digest。
- `agenix inspect <artifact>` 只读取 capsule 本身，并打印 skill identity、file count、
  digest 和 artifact path。
- `agenix run <artifact>` 会把 `manifest.yaml` materialize 到 workspace 根目录，并在
 运行 manifest 之前把 `files/...` 恢复为不带 `files/` 前缀的文件。workspace 会保留在
 run 目录下，以便 trace verification 能针对 materialized manifest 做 replay。
