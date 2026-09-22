package transfer

import "context"

// Outcome describes what is known about the publication commit point.
type Outcome uint8

const (
	// NotPublished means the commit point was not reached and the destination
	// is known not to contain this publication.
	NotPublished Outcome = iota
	// Published means the commit point was reached and the publication is known
	// to be visible.
	Published
	// PublicationUncertain means the commit point may have been reached, but the
	// caller cannot establish the resulting destination state.
	PublicationUncertain
)

func (o Outcome) String() string {
	switch o {
	case NotPublished:
		return "not published"
	case Published:
		return "published"
	case PublicationUncertain:
		return "publication uncertain"
	default:
		return "unknown"
	}
}

// Result is the publication state returned alongside an operation error.
type Result struct {
	Outcome Outcome
	Commit  bool
}

// OperationID identifies one logical publication attempt and is reused for an
// idempotent retry.
type OperationID string

// Destination identifies the caller-resolved publication target. Its meaning
// is opaque to this package.
type Destination string

// ContentIdentity identifies the content being published.
type ContentIdentity string

// Receipt is immutable-by-value evidence returned by a publication.
type Receipt struct {
	Operation   OperationID
	Destination Destination
	Content     ContentIdentity
	Range       Range
	Digest      string
	Revision    string
}

// Identity returns the stable identity portion of a receipt.
func (r Receipt) Identity() CommitIdentity {
	return CommitIdentity{Operation: r.Operation, Destination: r.Destination, Content: r.Content}
}

// CommitIdentity is the immutable identity shared by an intent and its
// receipts. It carries no authorization decision.
type CommitIdentity struct {
	Operation   OperationID
	Destination Destination
	Content     ContentIdentity
}

// ReceiptIdentity is retained as a descriptive alias for commit identity.
type ReceiptIdentity = CommitIdentity

// Intent is the caller-owned description of one logical publication. Policy is
// opaque and must be authorized before invoking a publisher or reconciler.
type Intent struct {
	Operation   OperationID
	Destination Destination
	Content     ContentIdentity
	Policy      any
}

func (i Intent) Identity() CommitIdentity {
	return CommitIdentity{Operation: i.Operation, Destination: i.Destination, Content: i.Content}
}

// Publisher attempts to publish one intent. Result and Receipt remain
// meaningful when err is non-nil.
type Publisher interface {
	Publish(context.Context, Intent) (Result, Receipt, error)
}

// Reconciler resolves an uncertain publication without guessing.
type Reconciler interface {
	Reconcile(context.Context, Intent) (Result, Receipt, error)
}

// PublicationError preserves an outcome and receipt with the operation error.
// It is useful when an implementation wants one error value while retaining
// commit evidence for callers that use errors.As.
type PublicationError struct {
	Result  Result
	Receipt Receipt
	Err     error
}

func (e *PublicationError) Error() string {
	if e.Err == nil {
		return e.Result.Outcome.String()
	}
	return e.Result.Outcome.String() + ": " + e.Err.Error()
}
func (e *PublicationError) Unwrap() error { return e.Err }
