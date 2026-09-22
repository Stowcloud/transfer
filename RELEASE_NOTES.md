# transfer v0.1.0

Initial standalone release.

## Included

- Normalized half-open range arithmetic with bounded fragmentation.
- Immutable publication receipts, outcomes, operation identity, and uncertain
  completion reconciliation.
- Append-only session state machine with cancellation and recovery contracts.
- Local staged strategy contracts.
- SDK-neutral S3 multipart strategy interfaces and a conformance fake that
  requires whole-file verification before publication.
