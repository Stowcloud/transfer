package transfer

import "context"

// Cancellation records a cancellation request and its durable identity.
type Cancellation struct {
	Identity CommitIdentity
	Reason   string
}

// Canceler requests cancellation. Result and Receipt follow the same outcome
// rules as publication: an error does not erase a crossed commit point.
type Canceler interface {
	Cancel(context.Context, Intent, Cancellation) (Result, Receipt, error)
}

// RecoveryStatus describes whether an operation needs reconciliation.
type RecoveryStatus uint8

const (
	RecoveryNotRequired RecoveryStatus = iota
	RecoveryRequired
	RecoveryResolved
)

func (s RecoveryStatus) String() string {
	switch s {
	case RecoveryNotRequired:
		return "not required"
	case RecoveryRequired:
		return "required"
	case RecoveryResolved:
		return "resolved"
	default:
		return "unknown"
	}
}

// Recovery is durable evidence retained after an interrupted operation.
type Recovery struct {
	Identity CommitIdentity
	Status   RecoveryStatus
	Result   Result
	Receipt  Receipt
}

// Recoverer resolves durable recovery evidence. It must never infer a
// publication from an absent or incomplete receipt.
type Recoverer interface {
	Recover(context.Context, Intent, Recovery) (Result, Receipt, error)
}

// ReconcilePublication invokes a reconciler only for uncertain outcomes.
// Published and NotPublished results are already authoritative.
func ReconcilePublication(ctx context.Context, r Reconciler, intent Intent, result Result, receipt Receipt) (Result, Receipt, error) {
	if result.Outcome != PublicationUncertain {
		return result, receipt, nil
	}
	if r == nil {
		return result, receipt, ErrUnsupported
	}
	return r.Reconcile(ctx, intent)
}
