# transfer

`github.com/stowcloud/transfer` contains neutral primitives and contracts for
resumable byte ranges, staged content, publication outcomes, immutable commit
identity, cancellation, recovery, and uncertain-completion reconciliation.

The package does not authorize callers, access an application database, open a
filesystem path, or select an S3 provider. Strategies receive caller-resolved
intents and must preserve publication outcomes when an operation returns an
error. Multipart conformance helpers always require whole-file verification
before returning `Published`; they do not authorize unsafe native multipart
completion.

## Requirements

- Go 1.27.1 or newer.
- No Stowcloud internal, Hanami, Fx, or Gin dependencies.

## Verification

```sh
go test ./...
go vet ./...
```

## Compatibility and releases

The package is pre-1.0. Consumers should pin an explicit release and run their
integration tests before updating. Release tags are immutable: publish a new
version to correct a release rather than moving an existing tag. See
`CONTRIBUTING.md` for release procedure and `SECURITY.md` for private reports.
