# Write Your First Agenix Skill

This tutorial builds a first Agenix skill with the V0.2
`repo-fix-test-failure` template. The goal is to understand what you are
authoring, not only to run commands.

Mental model: a skill is a manifest plus files. The fixture/source files create
a concrete problem. The manifest declares identity, inputs, tools, permissions,
outputs, and verifiers. Policy limits what the adapter and verifier can read,
write, execute, or access over the network. The verifier proves the result. The
adapter contract says the adapter must use runtime tools and return the
structured outputs required by the manifest. The check report records the
artifact, trace, changed files, and verifier result for audit.

The adapter here is deterministic and local. It is a template adapter, not a
real model adapter. Real model-backed adapters come later, but they still have
to pass through the same policy, verifier, trace, and report contract.

## Prerequisites

Run every command from the repository root. You need Go, Python 3, and pytest:

```bash
go version
python3 --version
python3 -m pytest --version
```

If pytest is missing, install it in the Python environment used by `python3`.

## 1. List Templates

Ask Agenix what it can generate:

```bash
go run ./cmd/agenix init templates
```

Look for:

```text
template=repo-fix-test-failure adapter=repo-fix-test-failure-template writes=true description=Writable failing-test repair skill skeleton.
```

`repo-fix-test-failure` is the scaffold. `repo-fix-test-failure-template` is the
local deterministic adapter for this scaffold. `writes=true` means the skill is
expected to modify a file. For automation, use:

```bash
go run ./cmd/agenix init templates --json
```

## 2. Generate A Skill

Create a temporary skill:

```bash
rm -rf /tmp/repo.demo_fix
go run ./cmd/agenix init skill repo.demo_fix --template repo-fix-test-failure -o /tmp/repo.demo_fix
```

`repo.demo_fix` becomes the skill identity in `manifest.yaml`. The output
directory must be empty or missing, which protects existing work.

## 3. Confirm The Fixture Fails

Run the fixture test before running Agenix:

```bash
python3 -m pytest -q /tmp/repo.demo_fix/fixture
```

Expected result: the test fails because `mathlib.add(2, 3)` returns `-1`
instead of `5`. This matters because a repair skill needs a verifier that can
distinguish the broken state from the fixed state.

## 4. Read The File Layout

Inspect the generated files:

```bash
find /tmp/repo.demo_fix -maxdepth 3 -type f | sort
```

Expected files: `README.md`, `fixture/mathlib.py`,
`fixture/test_mathlib.py`, and `manifest.yaml`. Read
`fixture/test_mathlib.py` first because it defines the behavior users need,
then `fixture/mathlib.py`, `manifest.yaml`, and `README.md`.

## 5. Understand The Manifest

Open the manifest:

```bash
sed -n '1,220p' /tmp/repo.demo_fix/manifest.yaml
```

Key fields:

```yaml
name: repo.demo_fix
version: 0.1.0
inputs:
  repo_path: fixture
tools:
  - fs
  - shell
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
      - ${repo_path}
```

`name` and `version` identify the skill in reports, artifacts, registry entries,
and traces. `inputs` define reusable values; this template uses `${repo_path}`
to point policy and verifiers at the fixture directory. `tools` are runtime
capabilities. `permissions` are the safety boundary: the adapter can read and
write only in the fixture, and network access is disabled.

```yaml
outputs:
  required:
    - patch_summary
    - changed_files
```

`outputs` define the structured adapter result. `changed_files` tells users what
the run modified.

```yaml
verifiers:
  - type: command
    name: run_tests
    run: ["python3", "-m", "pytest", "-q"]
    cwd: ${repo_path}
    success:
      exit_code: 0
  - type: schema
    name: output_schema_check
    schemaRef: outputs
```

`run_tests` proves the repaired source passes pytest. `output_schema_check`
proves the adapter returned the required fields.

## 6. Run The Check

Run the full authoring gate:

```bash
go run ./cmd/agenix check /tmp/repo.demo_fix --adapter repo-fix-test-failure-template --json > /tmp/fix-report.json
```

`agenix check` validates the manifest, builds a temporary artifact, runs it,
validates the trace, reruns verification, replays the trace summary, and writes
a check report.

Validate the report:

```bash
go run ./cmd/agenix validate /tmp/fix-report.json
```

Expected result: `status=valid kind=check_report`.

## 7. Inspect The Report And Trace

Print the report:

```bash
cat /tmp/fix-report.json
```

Look for:

```json
{
  "kind": "check_report",
  "status": "passed",
  "skill": "repo.demo_fix",
  "changed_files": [".../workspace/fixture/mathlib.py"],
  "trace_path": ".agenix/runs/<run_id>/trace.json",
  "verifier_summary": ["run_tests:passed", "output_schema_check:passed"]
}
```

`changed_files` should include `fixture/mathlib.py`; that confirms the adapter
changed source in the materialized run workspace. Replay `trace_path`:

```bash
go run ./cmd/agenix replay "$(python3 -c 'import json; print(json.load(open("/tmp/fix-report.json"))["trace_path"])')"
```

Replay reads recorded events. It does not rerun the adapter.

## 8. Customize Safely

Change behavior before widening power:

1. Edit fixture source and tests first.
2. Run pytest directly and confirm the starting state is meaningful.
3. Update manifest `description`, `inputs`, `outputs`, and verifiers.
4. Widen permissions only when the fixture truly needs more scope.
5. Run `agenix check --json` and validate the report again.

Example: if your skill fixes `parser.py`, first write a failing
`test_parser.py`. Then update the verifier command and filesystem policy to
cover only the intended fixture directory. Do not add broad permissions "just
in case"; broad write scope makes the skill harder to trust.

## 9. Common Failure Modes

- `unsupported skill template`: run `go run ./cmd/agenix init templates` and
  copy the template name exactly.
- `output directory is not empty`: use a new directory or remove the temporary
  one:

```bash
rm -rf /tmp/repo.demo_fix
```

- Pytest passes before `agenix check`: the fixture no longer proves repair
  behavior. Reintroduce a broken source state or write a stricter test.
- Policy failure during check: something tried to read, write, execute, or
  access outside `permissions`. Narrow the fixture first; expand policy only
  when the user-visible task requires it.
- `output_schema_check` fails: the adapter did not return every required output
  field. Keep `outputs.required` aligned with the adapter contract.
- `run_tests` fails: the repair is wrong, the verifier command is wrong, or
  pytest is missing. Run pytest directly against the fixture to isolate it.
- When `agenix check --json` fails, stdout is still a valid check report. Save
  it and inspect `error_class`, `error_message`, and `trace_path` before
  changing the manifest.

## 10. Build And Run The Artifact

Build, inspect, and run a portable artifact:

```bash
go run ./cmd/agenix build /tmp/repo.demo_fix -o /tmp/repo.demo_fix.agenix
go run ./cmd/agenix inspect /tmp/repo.demo_fix.agenix
go run ./cmd/agenix run /tmp/repo.demo_fix.agenix --adapter repo-fix-test-failure-template
```

You now have the beginner loop: write fixture behavior, declare policy and
verifiers, run a deterministic adapter through runtime tools, validate the
check report, then build and share the artifact.
