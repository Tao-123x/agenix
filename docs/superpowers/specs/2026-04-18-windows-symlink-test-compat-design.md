# Windows Symlink Test Compatibility Design

## Goal

Keep symlink-dependent tests meaningful on hosts that support symlink creation,
while skipping them on Windows hosts where the current session lacks the
privilege required to create symlinks.

## Problem

The current test helper `isSymlinkUnsupported(err)` only treats
`os.IsPermission(err)` as an unsupported-host signal. On this Windows machine,
`os.Symlink` fails with the message `A required privilege is not held by the
client.`, but that error is not currently classified as unsupported. As a
result, tests that should skip instead fail during setup.

## Design

Keep the change test-scoped.

- Add focused tests for the helper behavior in `internal/agenix/policy_test.go`.
- Expand `isSymlinkUnsupported(err)` so it still accepts `os.IsPermission(err)`,
  and additionally recognizes the Windows symlink privilege error.
- Do not loosen production policy or runtime behavior.
- Do not skip other filesystem failures that are unrelated to symlink privilege.

## Success Criteria

- The helper returns true for `os.ErrPermission`.
- The helper returns true for the Windows symlink privilege error.
- The helper returns false for unrelated errors.
- The three currently failing symlink tests skip rather than fail on this host.
