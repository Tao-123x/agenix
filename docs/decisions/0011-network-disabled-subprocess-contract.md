# Decision Record: Network-Disabled Subprocess Contract

## Status

accepted

## Context

`permissions.network: false` exists in the manifest contract, but v0 does not
provide a general OS-level network sandbox. Without a narrower runtime rule,
the field would imply stronger subprocess isolation than the platform actually
guarantees.

## Decision

Document `permissions.network: false` as a runtime-managed subprocess contract,
not an OS sandbox claim:

- v0 does not claim OS-level network sandboxing
- runtime-managed subprocess launch is allowed only for launcher types with
  explicit local-only or network-denied handling
- Python subprocesses run under a runtime-injected network-denied launcher
- offline-safe local git subcommands remain allowed
- unsupported executables fail closed as `PolicyViolation`
- verifier reruns use the same rule

## Alternatives Rejected

- Claim general OS-level network isolation in v0. The runtime does not provide
  that guarantee yet.
- Allow all local executables when `network: false` and rely on operator
  judgment. That leaves the contract too loose to audit.
- Disable every subprocess under `network: false`, including offline-safe git.
  That would block useful local-only workflows without improving the stated v0
  guarantee.

## Customer Impact

This makes the `network: false` contract auditable and honest for procurement
review: local-only execution is supported only where the runtime has an
explicit denial path, and unsupported cases fail predictably.

## Runtime Impact

Policy and tool-contract docs now describe `network: false` as a fail-closed
subprocess rule. Tool execution and verifier reruns share the same boundary,
which keeps traceable policy behavior aligned across runtime-managed launches.

## Verification

```bash
git diff -- specs/policy.md specs/tool-contract.md specs/policy.zh-CN.md specs/tool-contract.zh-CN.md docs/decisions/0011-network-disabled-subprocess-contract.md
```

Expected result: only the new `network: false` contract wording appears in the
target docs, plus the new decision record.

## Follow-Up

- Keep launcher-specific behavior listed explicitly until broader sandboxing
  exists.
- Add future decision records if non-Python launchers gain explicit
  network-denied handling.
