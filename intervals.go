package transfer

import "errors"

// ErrFragmented reports an insert that would exceed the configured run bound.
var ErrFragmented = errors.New("too many disjoint received ranges")

// Range is one received span, half-open: [Lo, Hi).
type Range struct {
	Lo uint64
	Hi uint64
}

type options struct{ maxRuns int }

// Option configures an IntervalSet.
type Option func(*options)

// WithMaxRuns bounds the number of disjoint runs. A non-positive value means
// unlimited; callers that enforce a product limit should inject it explicitly.
func WithMaxRuns(maxRuns int) Option { return func(o *options) { o.maxRuns = maxRuns } }

// IntervalSet is a received set in one normal form: sorted, disjoint, and
// coalesced, however the ranges arrived.
type IntervalSet struct {
	runs    []Range
	maxRuns int
}

// NewIntervalSet returns an empty set.
func NewIntervalSet(opts ...Option) *IntervalSet {
	o := options{}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return &IntervalSet{maxRuns: o.maxRuns}
}

// FullIntervalSet is the set of a file wholly received.
func FullIntervalSet(length uint64, opts ...Option) *IntervalSet {
	s := NewIntervalSet(opts...)
	if length != 0 {
		s.runs = []Range{{Lo: 0, Hi: length}}
	}
	return s
}

// LoadIntervalSet rebuilds a set from stored rows. Rows are inserted rather
// than adopted, so unsorted or overlapping rows are normalized identically to
// live inserts. Empty and inverted rows are corruption and are refused.
func LoadIntervalSet(rows []Range, opts ...Option) (*IntervalSet, error) {
	s := NewIntervalSet(opts...)
	for _, r := range rows {
		if r.Lo >= r.Hi {
			return nil, errors.New("invalid received range")
		}
		if err := s.Insert(r.Lo, r.Hi); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Insert records a received range, merging every run it overlaps or touches.
// Empty ranges change nothing. A refused insert leaves the set unchanged.
func (s *IntervalSet) Insert(lo, hi uint64) error {
	if lo >= hi {
		return nil
	}
	if len(s.runs) == 0 {
		if s.maxRuns > 0 && s.maxRuns < 1 {
			return ErrFragmented
		}
		s.runs = []Range{{Lo: lo, Hi: hi}}
		return nil
	}

	first := 0
	for first < len(s.runs) && s.runs[first].Hi < lo {
		first++
	}
	last := first
	for last < len(s.runs) && s.runs[last].Lo <= hi {
		if s.runs[last].Lo < lo {
			lo = s.runs[last].Lo
		}
		if s.runs[last].Hi > hi {
			hi = s.runs[last].Hi
		}
		last++
	}
	newCount := len(s.runs) - (last - first) + 1
	if s.maxRuns > 0 && newCount > s.maxRuns {
		return ErrFragmented
	}
	merged := make([]Range, 0, newCount)
	merged = append(merged, s.runs[:first]...)
	merged = append(merged, Range{Lo: lo, Hi: hi})
	merged = append(merged, s.runs[last:]...)
	s.runs = merged
	return nil
}

func (s *IntervalSet) touchesExisting(lo, hi uint64) bool {
	if lo >= hi {
		return false
	}
	for _, r := range s.runs {
		if r.Hi < lo {
			continue
		}
		if r.Lo > hi {
			return false
		}
		return true
	}
	return false
}

// ContiguousPrefix is the resumable offset: the end of the run starting at 0,
// or zero when the set does not start there.
func (s *IntervalSet) ContiguousPrefix() uint64 {
	if len(s.runs) == 0 || s.runs[0].Lo != 0 {
		return 0
	}
	return s.runs[0].Hi
}

// IsComplete reports whether the set covers the whole file.
func (s *IntervalSet) IsComplete(length uint64) bool {
	if length == 0 {
		return len(s.runs) == 0
	}
	return len(s.runs) == 1 && s.runs[0].Lo == 0 && s.runs[0].Hi >= length
}

// Missing returns the uncovered ranges below length.
func (s *IntervalSet) Missing(length uint64) []Range {
	if length == 0 {
		return nil
	}
	missing := make([]Range, 0)
	at := uint64(0)
	for _, r := range s.runs {
		if at >= length {
			break
		}
		if r.Lo > at {
			hi := r.Lo
			if hi > length {
				hi = length
			}
			if at < hi {
				missing = append(missing, Range{Lo: at, Hi: hi})
			}
		}
		if r.Hi > at {
			at = r.Hi
		}
	}
	if at < length {
		missing = append(missing, Range{Lo: at, Hi: length})
	}
	return missing
}

// Received returns the bytes covered by the set.
func (s *IntervalSet) Received() uint64 {
	var n uint64
	for _, r := range s.runs {
		n += r.Hi - r.Lo
	}
	return n
}

// Count reports how many disjoint runs the set contains.
func (s *IntervalSet) Count() int { return len(s.runs) }

// Runs returns a copy of the normalized set.
func (s *IntervalSet) Runs() []Range { return append([]Range(nil), s.runs...) }
