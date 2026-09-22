package transfer

import (
	"context"
	"errors"
)

// Capability names an operation a strategy can support.
type Capability uint8

const (
	CapabilityPublish Capability = 1 << iota
	CapabilityReconcile
	CapabilityCancel
	CapabilityRecover
	CapabilityStage
)

type Capabilities uint8

func (c Capabilities) Has(capability Capability) bool { return uint8(c)&uint8(capability) != 0 }
func (c Capabilities) With(capability Capability) Capabilities {
	return Capabilities(uint8(c) | uint8(capability))
}
func (c Capabilities) Without(capability Capability) Capabilities {
	return Capabilities(uint8(c) &^ uint8(capability))
}

// Accepted describes the portion of an intent accepted during preparation.
type Accepted struct {
	Identity CommitIdentity
	Range    Range
}

// Preparation is the backend-neutral handoff between Strategy.Prepare and a
// caller's operations.
type Preparation struct {
	Accepted  Accepted
	Publish   Publisher
	Reconcile Reconciler
	Cancel    Canceler
	Recover   Recoverer
	Stage     Stager
}

// Strategy discovers capabilities and validates one intent. Prepare never
// performs publication and never owns authorization.
type Strategy interface {
	Capabilities() Capabilities
	Prepare(context.Context, Intent) (Preparation, error)
}

// StagedContent is a local staged object. A stage is private until its
// publisher crosses the publication commit point.
type StagedContent struct {
	Identity CommitIdentity
	Range    Range
	Digest   string
	Revision string
}

// Stager prepares content without publishing it. The returned StagedContent is
// evidence only; callers must still authorize and explicitly Publish it.
type Stager interface {
	Stage(context.Context, Intent) (StagedContent, error)
}

// StagedStrategy is a Strategy whose preparation includes a local staging
// contract. It is deliberately separate from Publisher so staging cannot be
// mistaken for publication.
type StagedStrategy interface {
	Strategy
	Stage(context.Context, Intent) (StagedContent, error)
}

// ErrUnsupported reports an operation not offered by a strategy.
var ErrUnsupported = errors.New("transfer operation unsupported")
