# repo.apply_small_refactor 示例

[English](README.md) | [简体中文](README.zh-CN.md)

目标：

- 演示受限写入执行。
- 只重构一个已声明的文件，并且不改变行为。
- 同时验证测试行为和预期重构结构。

运行方式：

```bash
go run ./cmd/agenix run examples/repo.apply_small_refactor/manifest.yaml
```

manifest 对 fixture 授予了读取权限，但只对 `greeter.py` 授予写权限。
fake adapter 会读取 `greeter.py`，把重复的姓名格式化逻辑提取到 `full_name`，
通过 `fs.write` 写回同一个文件，并报告这一个变更文件。

verifier 会运行 pytest，并做一个结构检查，确认 helper 已存在，且
`greeting` 会委托给它。
