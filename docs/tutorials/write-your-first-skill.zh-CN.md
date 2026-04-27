# 写你的第一个 Agenix Skill

这篇教程用 V0.2 模板 `repo-fix-test-failure` 写一个 beginner skill：它从一个失败的
pytest fixture 开始，让 adapter 修复代码，再由 Agenix 验证测试通过。模板 adapter
`repo-fix-test-failure-template` 是确定性的、本地运行的 adapter，不调用真实模型、不需要网络。
真实模型 adapter 可以之后再接入；先把 skill contract 写清楚。

## 心智模型

一个 Agenix skill 不是一段提示词，而是“任务定义 + 安全边界 + 可执行验收”：
- `manifest`：声明 `name`、`version`、`inputs`、`permissions`、`tools`、`outputs`、`verifiers`。
- `fixture/source files`：skill 操作的最小项目，这里是故意失败的 pytest 代码。
- `policy`：限制读写路径、网络和 shell 命令。
- `verifier`：adapter 做完后，Agenix 自己执行的验收规则。
- `adapter contract`：adapter 必须在 manifest 允许范围内工作，并返回结构化输出。
- `check report`：`agenix check` 生成的 JSON 证据，包含 artifact、trace、改动和 verifier 结果。

## 1. 前置条件

所有命令从 Agenix 仓库根目录执行：
```sh
go version
python3 --version
python3 -m pytest --version
```

如果 pytest 不存在：
```sh
python3 -m pip install pytest
```

## 2. 列出模板

```sh
go run ./cmd/agenix init templates
```

你应该看到：
```text
template=python-pytest adapter=python-pytest-template writes=false description=Minimal read-only pytest skill skeleton.
template=repo-fix-test-failure adapter=repo-fix-test-failure-template writes=true description=Writable failing-test repair skill skeleton.
```

这里选择 `repo-fix-test-failure`，因为它覆盖完整路径：初始失败、adapter 写入修复、
verifier 重新运行测试。

## 3. 生成 skill

生成到 `/tmp`，避免改动仓库示例：
```sh
rm -rf /tmp/repo.demo_fix
go run ./cmd/agenix init skill repo.demo_fix --template repo-fix-test-failure -o /tmp/repo.demo_fix
find /tmp/repo.demo_fix -maxdepth 3 -type f | sort
```

典型输出：
```text
/tmp/repo.demo_fix/README.md
/tmp/repo.demo_fix/fixture/mathlib.py
/tmp/repo.demo_fix/fixture/test_mathlib.py
/tmp/repo.demo_fix/manifest.yaml
```

`manifest.yaml` 是运行合同，`fixture/` 是待修复小项目，`README.md` 只是人类说明。

## 4. 确认 fixture 初始失败

```sh
python3 -m pytest -q /tmp/repo.demo_fix/fixture
```

预期失败类似：
```text
FAILED test_mathlib.py::test_adds_numbers - assert -1 == 5
```

查看原因：
```sh
sed -n '1,80p' /tmp/repo.demo_fix/fixture/mathlib.py
sed -n '1,80p' /tmp/repo.demo_fix/fixture/test_mathlib.py
```

`add(a, b)` 当前做减法，测试期望 `add(2, 3) == 5`。这个失败是模板的起点。

## 5. 读生成布局

生成目录只有 `manifest.yaml`、`README.md` 和 `fixture/`。写 skill 时先让 fixture 足够小：
一个明确失败、一个明确期望、一个明确修复目标。这样 verifier 的完成标准也清楚。

## 6. 理解 manifest 字段

```sh
sed -n '1,220p' /tmp/repo.demo_fix/manifest.yaml
```

关键片段：
```yaml
name: repo.demo_fix
version: 0.1.0
inputs:
  repo_path: fixture
permissions:
  network: false
  filesystem:
    read: [${repo_path}]
    write: [${repo_path}]
tools:
  - fs
  - shell
outputs:
  required:
    - patch_summary
    - changed_files
```

字段怎么读：`name` 是稳定标识；`version` 是 contract 版本；`inputs` 让 `${repo_path}`
指向 `fixture`；`permissions` 限制网络、文件和 shell；`tools` 声明 adapter 可用工具；
`outputs` 声明必需结构化字段；`verifiers` 定义 Agenix 自己执行的验收。

verifier 片段：
```yaml
verifiers:
  - type: command
    name: run_tests
    run: ["python3", "-m", "pytest", "-q"]
    cwd: ${repo_path}
    success:
      exit_code: 0
  - type: schema
    name: output_schema_check
    schemaRef: outputs
```

含义：Agenix 会进入 `${repo_path}` 跑 pytest，并检查输出包含 `patch_summary` 和
`changed_files`。

## 7. 运行 check

```sh
go run ./cmd/agenix check /tmp/repo.demo_fix --adapter repo-fix-test-failure-template --json > /tmp/fix-report.json
sed -n '1,220p' /tmp/fix-report.json
```

`agenix check` 会校验 manifest、构建临时 artifact、运行 adapter、记录 trace、执行 verifier，
并输出 JSON report。成功时重点看：
```json
{
  "kind": "check_report",
  "status": "passed",
  "verifier_summary": ["run_tests:passed", "output_schema_check:passed"]
}
```

## 8. 验证 report

```sh
go run ./cmd/agenix validate /tmp/fix-report.json
```

成功输出类似：
```text
status=valid kind=check_report schema=.../specs/check-report.schema.json path=/tmp/fix-report.json
```

这让 CI 或其他 agent 可以读取稳定 JSON，而不是解析人类日志。

## 9. 检查 changed_files 和 trace

```sh
python3 - <<'PY'
import json
report = json.load(open("/tmp/fix-report.json"))
print("status:", report["status"])
print("trace_path:", report["trace_path"])
print("changed_files:")
for path in report["changed_files"]:
    print(" -", path)
PY
```

注意：`changed_files` 指向 `.agenix/runs/.../workspace/...` 的临时运行工作区，
不是直接改 `/tmp/repo.demo_fix/fixture/mathlib.py`。这能保护原始模板目录。

replay trace：
```sh
TRACE_PATH=$(python3 - <<'PY'
import json
print(json.load(open("/tmp/fix-report.json"))["trace_path"])
PY
)
go run ./cmd/agenix replay "$TRACE_PATH"
```

你应该看到 adapter 选择、能力检查、policy 检查、`fs.read`、`fs.write`、adapter execute、
`run_tests:passed`、`output_schema_check:passed`。trace 的价值是说明 adapter 做了什么，
以及 verifier 为什么通过。

## 10. 安全定制

推荐顺序：先改 `fixture/` 的 source/test，让失败场景表达真实问题；再改 `manifest.yaml`
的 `inputs`、`permissions`、`tools`、`outputs`、`verifiers`；最后运行
`agenix check ... --json`，检查 report 和 trace。

原则：先改源代码和测试，再扩大 manifest 权限或 verifier。只有新测试需要新命令时，
才加入 `permissions.shell.allow`；只有 adapter 需要新目录时，才扩大
`permissions.filesystem`。不要一开始开放整个仓库写权限或网络权限。

## 11. 常见失败

- pytest 不存在：运行 `python3 -m pip install pytest`。
- adapter 名字写错：模板名是 `repo-fix-test-failure`，adapter 名是 `repo-fix-test-failure-template`。
- `output_schema_check` 失败：检查输出是否包含 `patch_summary` 和 `changed_files`。
- `run_tests` 失败：运行 `go run ./cmd/agenix replay "$TRACE_PATH"`，确认是否有 `fs.write`，
  以及 verifier 是否在正确目录执行。
- 权限失败：检查 `permissions.filesystem` 和 `permissions.shell.allow` 是否覆盖真实需要；
  先保持最小权限，确认需要时再扩大。

## 12. 下一步：构建、分享、运行

`agenix check` 通过后，report 里的 `artifact_path` 指向临时 `.agenix` artifact：
```sh
python3 - <<'PY'
import json
print(json.load(open("/tmp/fix-report.json"))["artifact_path"])
PY
```

你可以把 artifact 交给其他运行环境，或之后接入真实模型 adapter。接入真实 adapter 时，
尽量保持同一套 manifest、policy、verifier 和 report 检查不变；这样替换的是执行者，
不是验收标准。

写第二个 skill 时重复这个循环：用最小 fixture 表达问题，用 manifest 限定工具和权限，
用 verifier 定义完成标准，用 `agenix check --json` 生成证据，用 trace 检查 adapter 行为。
