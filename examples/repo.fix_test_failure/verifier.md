# Verifier for repo.fix_test_failure

[English](verifier.md) | [简体中文](verifier.zh-CN.md)

Success criteria:

1. `pytest -q` exits 0
2. Output matches output schema (patch summary + changed files)
3. No writes outside `${repo_path}`
