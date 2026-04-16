# repo.fix_test_failure 示例

[English](README.md) | [简体中文](README.zh-CN.md)

目标：

- 演示可移植的 skill 执行、trace 产出和验证闭环。
- 运行方式：`go run ./cmd/agenix run examples/repo.fix_test_failure/manifest.yaml`。

这个示例同时也是一个**不变量测试**：

- 相同的 manifest
- 相同的 tool contract
- 不同的模型 / 操作系统
- 一致的结果

fixture 的初始状态是故意损坏的：`mathlib.add(2, 3)` 会返回 `-1`。
v0 fake adapter 会通过 runtime 的 `fs.write` 工具修复 `mathlib.py`，
然后 verifier 运行 `python3 -m pytest -q`。
