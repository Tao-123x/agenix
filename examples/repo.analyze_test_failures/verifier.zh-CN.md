# repo.analyze_test_failures 的 Verifier

[English](verifier.md) | [简体中文](verifier.zh-CN.md)

这个 verifier 有两项检查：

1. `fixture_still_fails` 会在 fixture 内执行 `python3 verify_failing.py`。  
   这个脚本会运行 pytest，并且只有当 fixture 仍然失败时才以 0 退出。
2. `output_schema_check` 要求输出包含：
   - `analysis_summary`
   - `failing_tests`
   - `likely_root_cause`
   - `changed_files`

这个 skill 是只读的。一次成功运行不应该产生任何 `fs.write` 事件，并且
应该返回空的 `changed_files` 列表。
