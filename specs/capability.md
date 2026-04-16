# Capabilities (v0.1 Draft)

[English](capability.md) | [简体中文](capability.zh-CN.md)

## Model capability requirements

- `tool_calling` support
- `structured_output` mode
- token budget (context window)
- latency preference (optional)
- reasoning level (heuristic)

## Negotiation

- Skill declares `requires`.
- Runtime checks model profile.
- Outcome:
  - **ok:** proceed
  - **degraded:** proceed with warnings
  - **fail:** runtime reports `unsupported`

## Implemented minimum

The current runtime implements a local preflight check before any tool call:

- manifest may declare `capabilities.requires`
- the adapter reports `name`, `model_profile`, `supported_skills`, and a minimum
  capability set
- runtime rejects unsupported skills before adapter execution
- runtime rejects missing `tool_calling`, `structured_output`, insufficient
  `max_context_tokens`, or insufficient `reasoning_level`
- trace records `adapter` events for selection and capability check results

The current runtime does not yet implement a degraded execution path or
vendor-specific capability discovery.

## Failure reporting

- Must include: which requirement failed, what capability was missing, and suggestions.
