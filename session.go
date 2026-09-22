package transfer

import "fmt"

// SessionState marks a transfer session's stage of life. Numeric values are
// append-only because applications may persist them.
type SessionState int64

const (
	StateReceiving SessionState = iota
	StateFinalizing
	StateDone
	StateAborted
	StateExpired
)

func (s SessionState) StateName() string {
	switch s {
	case StateReceiving:
		return "receiving"
	case StateFinalizing:
		return "finalizing"
	case StateDone:
		return "done"
	case StateAborted:
		return "aborted"
	case StateExpired:
		return "expired"
	default:
		return "unknown"
	}
}

func (s SessionState) Terminal() bool { return s != StateReceiving && s != StateFinalizing }

func (s SessionState) live() bool { return !s.Terminal() }

func StateNames() map[string]bool {
	return map[string]bool{
		"receiving": false,
		"finalizing": false,
		"done": true,
		"aborted": true,
		"expired": true,
	}
}

// CanTransition reports whether a durable session may make a direct state
// transition. Recovery may use RecoverSession for a finalizing session instead
// of pretending publication completed.
func CanTransition(from, to SessionState) bool {
	switch from {
	case StateReceiving:
		return to == StateReceiving || to == StateFinalizing || to == StateAborted || to == StateExpired
	case StateFinalizing:
		return to == StateFinalizing || to == StateDone || to == StateReceiving || to == StateAborted
	case StateDone, StateAborted, StateExpired:
		return to == from
	default:
		return false
	}
}

// Transition validates and returns a new state without mutating caller-owned
// state.
func Transition(from, to SessionState) (SessionState, error) {
	if !CanTransition(from, to) {
		return from, fmt.Errorf("transfer: invalid session transition %s -> %s", from.StateName(), to.StateName())
	}
	return to, nil
}

// Session describes the state-machine portion of one transfer independently of
// a persistence schema.
type Session struct {
	Identity CommitIdentity
	State    SessionState
}

func (s Session) Transition(to SessionState) (Session, error) {
	state, err := Transition(s.State, to)
	if err != nil {
		return s, err
	}
	s.State = state
	return s, nil
}
