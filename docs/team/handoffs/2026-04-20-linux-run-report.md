# Agent Handoff

## Role

`docs/team/roles/cross-platform-conformance-explorer.md`

## Task

`Run the Agenix repository on Linux, record the exact dependency gaps and passing commands, and hand the result to the next writer.`

## File Ownership

- Read:
  - `README.md`
  - `cmd/agenix/main.go`
  - `examples/repo.fix_test_failure/manifest.yaml`
  - `docs/team/templates/handoff.md`
  - `docs/team/handoffs/2026-04-16-cross-platform-conformance-suite.md`
- Write:
  - `docs/team/handoffs/2026-04-20-linux-run-report.md`
- Do not touch:
  - runtime source files
  - manifests
  - specs

## Context Loaded

- Team charter:
  - `docs/team/2026-04-14-agent-runtime-team.md`
- Role card:
  - `docs/team/roles/cross-platform-conformance-explorer.md`
- Customer file:
  - not reloaded for this bounded runtime check
- Plan or roadmap:
  - `docs/roadmap/2026-04-14-agenix-roadmap.md`
- Prior handoff:
  - `docs/team/handoffs/2026-04-16-cross-platform-conformance-suite.md`

## Work Completed

- Verified the host was Ubuntu 24.04.4 LTS on `linux/amd64`.
- Confirmed the repo does not run from a bare default shell on this host because:
  - `go` was not installed
  - `python3` existed, but `pytest` was not installed
- Installed a local Go toolchain outside the repo at `~/.local/tools/go` and verified:
  - `go version go1.26.2 linux/amd64`
- Installed a local Python virtual environment outside the repo at `~/.local/venvs/agenix` and installed `pytest` there.
- Confirmed the canonical broken fixture was still broken before runtime execution:
  - `python3 -m pytest -q examples/repo.fix_test_failure/fixture`
  - result: `1 failed`
- Confirmed the CLI and schema path work on Linux:
  - `go run ./cmd/agenix validate examples/repo.fix_test_failure/manifest.yaml`
  - result: `status=valid`
- Ran the canonical demo successfully on Linux:
  - `go run ./cmd/agenix run examples/repo.fix_test_failure/manifest.yaml`
  - result: `status=passed`
  - trace: `.agenix/runs/0794634c68843482e97c3a95ca1fba57/trace.json`
- Replayed and re-verified that trace successfully:
  - `go run ./cmd/agenix replay .agenix/runs/0794634c68843482e97c3a95ca1fba57/trace.json`
  - `go run ./cmd/agenix verify .agenix/runs/0794634c68843482e97c3a95ca1fba57/trace.json`
  - result: both passed
- Re-ran the repo test suite after restoring the example fixture to its original broken state.
  - `go test ./...`
  - result: passed
- Restored the runtime-mutated example file after the demo run so the repo stayed clean.

## Verification

```bash
PATH=/home/taojiacheng/.local/tools/go/bin:/home/taojiacheng/.local/venvs/agenix/bin:$PATH \
  go test ./...

PATH=/home/taojiacheng/.local/tools/go/bin:/home/taojiacheng/.local/venvs/agenix/bin:$PATH \
  go run ./cmd/agenix validate examples/repo.fix_test_failure/manifest.yaml

PATH=/home/taojiacheng/.local/tools/go/bin:/home/taojiacheng/.local/venvs/agenix/bin:$PATH \
  go run ./cmd/agenix run examples/repo.fix_test_failure/manifest.yaml

PATH=/home/taojiacheng/.local/tools/go/bin:/home/taojiacheng/.local/venvs/agenix/bin:$PATH \
  go run ./cmd/agenix replay .agenix/runs/0794634c68843482e97c3a95ca1fba57/trace.json

PATH=/home/taojiacheng/.local/tools/go/bin:/home/taojiacheng/.local/venvs/agenix/bin:$PATH \
  go run ./cmd/agenix verify .agenix/runs/0794634c68843482e97c3a95ca1fba57/trace.json
```

Result:

- `go test ./...` passed
- `validate` passed
- `run` passed
- `replay` passed
- `verify` passed

## Risks

- The README prerequisites are accurate, but a fresh Ubuntu machine still needs explicit setup for both Go and `pytest` before the demo works.
- `agenix run` intentionally mutates the demo fixture during the canonical flow. Anyone reproducing the demo must either restore the example afterward or run tests before and after with that mutation in mind.
- The successful reproduction used local user-scoped tooling, not system package manager installs, so another writer should not assume `go` or `pytest` is globally available on this host.

## Customer Alignment

Maya verdict:

- consistent with current portability claims

Reason:

- the Linux runtime path now has direct evidence for the core v0 loop: validate, run, replay, verify, and full Go test coverage

## Next Handoff

The next agent should:

- decide whether to add a short Linux setup note to `README.md` that calls out `Go 1.22+` and `pytest` as hard prerequisites on a fresh host
- consider adding a tiny `make demo` or script-based bootstrap so future writers can reproduce the same Linux run without manually managing `PATH`
