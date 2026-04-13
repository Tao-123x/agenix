# Agentfile Specification (v0.1 Draft)

The Agentfile describes how to build and run an agent package. It is analogous to a `Dockerfile` but for skills and agents.

## Key sections

- `from`: base runtime image or profile
- `model`: preferred model profile (capability requirements, not vendor name)
- `tools`: tools to mount into the runtime environment
- `skills`: referenced skill packages
- `memory`: whether to mount persistent volumes
- `constraints`: limits on runtime (max runtime, cost cap, network policy, write scope)
- `entry`: entry skill or workflow

## Example

```yaml
from: agent-base:ubuntu-24.04

model:
  provider: openai
  profile: reasoning-medium

tools:
  - shell
  - filesystem
  - git

skills:
  - skills/code/edit-repo@1.0.0
  - skills/test/run-pytest@1.1.0

memory:
  mounted: false

constraints:
  network: off
  max_runtime_minutes: 20
  max_cost_usd: 1.50

entry:
  skill: workflow.fix_and_validate
```

## Runtime v0.1 capsule layout

`agenix build <skill-dir> -o <artifact>` produces a gzip-compressed tar artifact
with this layout:

```text
manifest.yaml
files/...
agenix.lock.json
```

Rules:

- `manifest.yaml` is copied from the skill directory root.
- `files/...` contains other skill directory files with their relative paths
  preserved.
- `.DS_Store`, `.agenix`, `.pytest_cache`, `__pycache__`, and `*.pyc` files are
  excluded.
- `agenix.lock.json` records artifact version, skill name/version, manifest
  digest, source file digests, creation time, and artifact digest.
- `agenix inspect <artifact>` reads only the capsule and prints skill identity,
  file count, digest, and artifact path.
- `agenix run <artifact>` materializes `manifest.yaml` at the workspace root and
  restores `files/...` without the `files/` prefix before running the manifest.
  The workspace is kept under the run directory so trace verification can replay
  against the materialized manifest.
