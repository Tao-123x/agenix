# Verifier for repo.analyze_test_failures

[English](verifier.md) | [简体中文](verifier.zh-CN.md)

The verifier has two checks:

1. `fixture_still_fails` runs `python3 verify_failing.py` inside the fixture.
   That script runs pytest and exits 0 only when the fixture still fails.
2. `output_schema_check` requires:
   - `analysis_summary`
   - `failing_tests`
   - `likely_root_cause`
   - `changed_files`

The skill is read-only. A successful run should not emit any `fs.write` event and
should return an empty `changed_files` list.
