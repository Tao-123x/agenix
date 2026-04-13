# repo.fix_test_failure demo

Goal:

- Demonstrate portable skill execution, trace emission, and verification.
- Run with `go run ./cmd/agenix run examples/repo.fix_test_failure/manifest.yaml`.

This demo is also an **invariance test**:

- Same manifest
- Same tool contract
- Different model/OS
- Consistent outcomes

The fixture intentionally starts broken: `mathlib.add(2, 3)` returns `-1`.
The v0 fake adapter fixes `mathlib.py` through the runtime `fs.write` tool,
then the verifier runs `python3 -m pytest -q`.
