# repo.apply_small_refactor demo

[English](README.md) | [简体中文](README.zh-CN.md)

Goal:

- Demonstrate constrained write execution.
- Refactor one declared file without changing behavior.
- Verify both test behavior and expected refactor shape.

Run with:

```bash
go run ./cmd/agenix run examples/repo.apply_small_refactor/manifest.yaml
```

The manifest grants read access to the fixture but write access only to
`greeter.py`. The fake adapter reads `greeter.py`, extracts repeated name
formatting into `full_name`, writes the same file through `fs.write`, and reports
that single changed file.

The verifier runs pytest and a shape check that confirms the helper exists and
`greeting` delegates to it.
