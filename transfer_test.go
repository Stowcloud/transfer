package transfer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func TestIntervalSetNormalizesRanges(t *testing.T) {
	s := NewIntervalSet(WithMaxRuns(2))
	for _, r := range []Range{{25, 30}, {10, 20}, {0, 10}, {28, 40}, {5, 15}} {
		if err := s.Insert(r.Lo, r.Hi); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Insert(20, 25); err != nil {
		t.Fatal(err)
	}
	if got := s.Runs(); len(got) != 1 || got[0] != (Range{0, 40}) {
		t.Fatalf("runs = %v", got)
	}
}

func TestPublicationIdentityAndUncertainReconciliation(t *testing.T) {
	intent := Intent{Operation: "op", Destination: "dst", Content: "content", Policy: struct{ Allowed bool }{true}}
	if intent.Identity() != (CommitIdentity{Operation: "op", Destination: "dst", Content: "content"}) {
		t.Fatal("policy changed commit identity")
	}
	reconciler := reconcilerFunc(func(context.Context, Intent) (Result, Receipt, error) {
		return Result{Outcome: Published, Commit: true}, Receipt{Operation: "op"}, nil
	})
	got, _, err := ReconcilePublication(context.Background(), reconciler, intent, Result{Outcome: PublicationUncertain, Commit: true}, Receipt{})
	if err != nil || got.Outcome != Published {
		t.Fatalf("reconciliation = %#v, %v", got, err)
	}
}

type reconcilerFunc func(context.Context, Intent) (Result, Receipt, error)

func (f reconcilerFunc) Reconcile(ctx context.Context, i Intent) (Result, Receipt, error) {
	return f(ctx, i)
}

func TestSessionTransitionsRejectTerminalResurrection(t *testing.T) {
	if _, err := Transition(StateDone, StateReceiving); err == nil {
		t.Fatal("terminal session was resurrected")
	}
	if got, err := Transition(StateReceiving, StateFinalizing); err != nil || got != StateFinalizing {
		t.Fatalf("transition = %v, %v", got, err)
	}
}

func TestS3FakeRequiresWholeFileVerification(t *testing.T) {
	fake := NewS3Fake()
	intent := Intent{Operation: "op", Destination: "bucket/key", Content: "file"}
	id, err := fake.BeginMultipart(context.Background(), intent, 5, "")
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("hello")
	sum := sha256.Sum256(data)
	part, err := fake.UploadPart(context.Background(), intent, id, 1, bytes.NewReader(data), uint64(len(data)), hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := fake.CompleteMultipart(context.Background(), intent, id, []S3Part{part}, "bad"); err == nil {
		t.Fatal("completion without matching whole-file digest succeeded")
	}
	want := hex.EncodeToString(sum[:])
	result, receipt, err := fake.CompleteMultipart(context.Background(), intent, id, []S3Part{part}, want)
	if err != nil || result.Outcome != Published || receipt.Digest != want {
		t.Fatalf("completion = %#v, %#v, %v", result, receipt, err)
	}
}

func TestPublicationErrorPreservesEvidence(t *testing.T) {
	want := errors.New("directory sync failed")
	e := &PublicationError{Result: Result{Outcome: PublicationUncertain, Commit: true}, Receipt: Receipt{Operation: "op"}, Err: want}
	if !errors.Is(e, want) || e.Result.Outcome != PublicationUncertain || e.Receipt.Operation != "op" {
		t.Fatal("publication evidence was not retained")
	}
}

func TestIntervalSetPastLengthIsComplete(t *testing.T) {
	set := NewIntervalSet()
	if err := set.Insert(0, 11); err != nil {
		t.Fatal(err)
	}
	if !set.IsComplete(10) {
		t.Fatal("coverage past the requested length is incomplete")
	}
}
