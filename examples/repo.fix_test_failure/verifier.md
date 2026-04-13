# Verifier for repo.fix_test_failure

Success criteria:

1. `pytest -q` exits 0
2. Output matches output schema (patch summary + changed files)
3. No writes outside `${repo_path}`
