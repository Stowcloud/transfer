# Contributing to transfer

Keep this module neutral. Do not add application authorization, persistence,
HTTP framework, filesystem, cloud SDK, or provider-specific dependencies to the
core package. Preserve explicit `NotPublished`, `Published`, and
`PublicationUncertain` outcomes and immutable commit identity.

Before opening a pull request, run `gofmt`, `go test ./...`, and `go vet ./...`.
Add focused behavioral tests for range normalization, state transitions,
commit-point outcomes, recovery, and whole-file verification. Do not add a
multipart helper that reports `Published` without verifying the assembled
content.

## Release checklist

1. Review the complete diff and update `RELEASE_NOTES.md`.
2. Confirm there are no local `replace` directives or generated artifacts.
3. Run formatting, tests, vet, and the consumer verification.
4. Commit the release and create an immutable annotated `vX.Y.Z` tag.
5. Push the commit and tag, then verify public module resolution.

Never move, delete, or reuse an existing release tag.
