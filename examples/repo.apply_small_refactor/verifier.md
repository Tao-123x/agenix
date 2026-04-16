# Verifier for repo.apply_small_refactor

[English](verifier.md) | [简体中文](verifier.zh-CN.md)

The verifier has three checks:

1. `run_tests` runs `python3 -m pytest -q` inside the fixture.
2. `refactor_shape` runs `python3 verify_refactor.py` to confirm `full_name`
   exists and `greeting` delegates to it.
3. `output_schema_check` requires:
   - `patch_summary`
   - `refactor_summary`
   - `changed_files`

The skill is allowed to write only `greeter.py`. `verify` also checks that the
reported changed files remain inside the manifest write scope.
