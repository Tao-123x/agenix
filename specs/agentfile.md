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
