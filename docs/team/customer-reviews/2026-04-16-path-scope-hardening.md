# Customer Review

## Reviewer

Maya Chen

## Change Under Review

`path-scope hardening`

## Trial Verdict

`approve`

## Procurement Verdict

`conditional approve`

## Why This Matters

- repo-relative paths should not silently depend on the shell cwd
- lexical scope checks are not enough if preexisting symlinks can redirect
  writes outside the declared workspace
- moved artifacts and verifier reruns are only believable if path semantics are
  stable across cwd changes

## Acceptance Criteria

- repo-relative filesystem paths resolve against the manifest/workspace root
- `verify` treats repo-relative `changed_files` consistently across cwd changes
- symlinked scope escapes fail closed as `PolicyViolation`
- artifact materialization rejects preexisting workspace symlink escapes
- docs explain the path contract without claiming an OS sandbox

## Blockers

- procurement still needs the smallest real-adapter spike
- adapter execution states and failure taxonomy still need tightening

## Do Not Build Next

- public registry
- UI dashboard
- remote executor
- provider-specific model integration

## Buyer Summary

`This makes the filesystem boundary more believable because declared scope now tracks real side effects across cwd changes and symlink tricks.`
