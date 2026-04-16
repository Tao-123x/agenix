# repo.apply_small_refactor 的 Verifier

[English](verifier.md) | [简体中文](verifier.zh-CN.md)

这个 verifier 有三项检查：

1. `run_tests` 会在 fixture 内执行 `python3 -m pytest -q`。
2. `refactor_shape` 会运行 `python3 verify_refactor.py`，确认 `full_name`
   存在，且 `greeting` 会委托给它。
3. `output_schema_check` 要求输出包含：
   - `patch_summary`
   - `refactor_summary`
   - `changed_files`

这个 skill 只允许写 `greeter.py`。`verify` 还会检查上报的 changed files
是否始终位于 manifest 声明的 write scope 之内。
