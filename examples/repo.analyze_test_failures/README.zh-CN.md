# repo.analyze_test_failures 示例

[English](README.md) | [简体中文](README.zh-CN.md)

目标：

- 演示一个只读的 Agenix skill。
- 在不写文件的前提下分析一个已知失败的 pytest fixture。
- 通过 verifier 和 structured output 证明结果。

运行方式：

```bash
go run ./cmd/agenix run examples/repo.analyze_test_failures/manifest.yaml
```

这个示例会故意保持 fixture 处于失败状态。v0 fake adapter 会通过 runtime 的
`fs.read` 工具读取源码和测试文件，报告最可能的根因，并返回空的
`changed_files` 列表。verifier 会确认 fixture 仍然以预期方式失败，
然后 schema verifier 再检查分析输出。
