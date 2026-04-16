# Customer Review

## Reviewer

Maya Chen

## Change Under Review

`network-disabled subprocess contract`

## Trial Verdict

`approve`

## Procurement Verdict

`conditional approve`

## Why This Matters

- `permissions.network=false` was previously declarative only
- subprocess launch was the clearest remaining false-security claim in the runtime
- this slice makes the contract auditable without overclaiming OS-level sandboxing

## Acceptance Criteria

- `network=false` has one runtime-managed subprocess rule shared by tools and verifiers
- unsupported subprocess executables fail closed as `PolicyViolation`
- verifier reruns use the same rule as initial runs
- negative tests and CI cover denied shell and verifier network attempts
- docs state exact guarantees and do not claim OS-level sandboxing

## Blockers

- broader launcher support is still missing
- procurement still needs stronger cross-platform conformance evidence and real adapter boundaries

## Do Not Build Next

- public registry
- UI dashboard
- remote executor
- marketplace work

## Buyer Summary

`This is the right P0 slice because it turns network denial from a false claim into a bounded, testable runtime contract.`
