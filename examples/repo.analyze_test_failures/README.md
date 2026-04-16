# repo.analyze_test_failures demo

[English](README.md) | [简体中文](README.zh-CN.md)

Goal:

- Demonstrate a read-only Agenix skill.
- Analyze a known failing pytest fixture without writing files.
- Prove the result through a verifier and structured output.

Run with:

```bash
go run ./cmd/agenix run examples/repo.analyze_test_failures/manifest.yaml
```

This demo intentionally keeps the fixture broken. The v0 fake adapter reads the
source and test files through the runtime `fs.read` tool, reports the likely root
cause, and returns an empty `changed_files` list. The verifier confirms the
fixture still fails in the expected way, then the schema verifier checks the
analysis output.
