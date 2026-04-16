# repo.fix_test_failure 的 Verifier

[English](verifier.md) | [简体中文](verifier.zh-CN.md)

成功标准：

1. `pytest -q` 退出码为 0
2. 输出符合 output schema（patch summary + changed files）
3. 不允许写出 `${repo_path}` 之外
