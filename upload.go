package transfer

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// SessionID is an opaque resumable-transfer handle. Its binary form is safe to
// persist directly; its string form is the unpadded URL spelling.
type SessionID [16]byte

// ErrInvalidSessionID reports a malformed wire or persisted session handle.
var ErrInvalidSessionID = errors.New("transfer: invalid session id")

// NewSessionID mints a cryptographically random transfer session handle.
func NewSessionID() (SessionID, error) {
	var id SessionID
	if _, err := rand.Read(id[:]); err != nil {
		return SessionID{}, fmt.Errorf("transfer: naming session: %w", err)
	}
	return id, nil
}

// SessionIDFromBytes parses the fixed-width persisted representation.
func SessionIDFromBytes(b []byte) (SessionID, error) {
	if len(b) != len(SessionID{}) {
		return SessionID{}, fmt.Errorf("%w: got %d bytes, want %d", ErrInvalidSessionID, len(b), len(SessionID{}))
	}
	var id SessionID
	copy(id[:], b)
	return id, nil
}

// ParseSessionID parses the strict unpadded base64url wire representation.
func ParseSessionID(s string) (SessionID, error) {
	b, err := base64.RawURLEncoding.Strict().DecodeString(s)
	if err != nil || len(b) != len(SessionID{}) {
		return SessionID{}, fmt.Errorf("%w: expected %d encoded bytes", ErrInvalidSessionID, len(SessionID{}))
	}
	return SessionIDFromBytes(b)
}

func (id SessionID) String() string { return base64.RawURLEncoding.EncodeToString(id[:]) }

// Bytes returns the persisted binary representation.
func (id SessionID) Bytes() []byte { return id[:] }

// SpoolMode determines how chunks map onto staged content.
type SpoolMode int64

const (
	SpoolOffsetAddressed SpoolMode = iota
	SpoolNameOrdered
)

func (m SpoolMode) ModeName() string {
	switch m {
	case SpoolOffsetAddressed:
		return "offset"
	case SpoolNameOrdered:
		return "named"
	default:
		return "unknown"
	}
}

// Algo names a checksum algorithm supported by the resumable content
// contracts. Algorithms are deliberately explicit: an unknown value is never
// silently replaced with a default.
type Algo int64

const (
	AlgoCRC32C Algo = iota
	AlgoBLAKE3
)

func (a Algo) String() string {
	switch a {
	case AlgoCRC32C:
		return "crc32c"
	case AlgoBLAKE3:
		return "blake3"
	default:
		return "unknown"
	}
}

var (
	ErrUnknownAlgorithm = errors.New("transfer: unknown checksum algorithm")
	ErrInvalidChecksum  = errors.New("transfer: invalid checksum")
)

func ParseAlgo(s string) (Algo, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "crc32c":
		return AlgoCRC32C, nil
	case "blake3":
		return AlgoBLAKE3, nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrUnknownAlgorithm, s)
	}
}

func Algorithms() []Algo { return []Algo{AlgoCRC32C, AlgoBLAKE3} }

// DigestLen reports the exact output length required for an algorithm.
func DigestLen(a Algo) int {
	switch a {
	case AlgoCRC32C:
		return 4
	case AlgoBLAKE3:
		return 32
	default:
		return 0
	}
}

// ValidateDigest rejects a digest that could not have been produced by a.
func ValidateDigest(a Algo, n int) error {
	if want := DigestLen(a); want == 0 || n != want {
		return fmt.Errorf("%w: a %s digest has length %d, want %d", ErrInvalidChecksum, a, n, want)
	}
	return nil
}

// Checksum carries an algorithm and its expected digest together.
type Checksum struct {
	Algo   Algo
	Digest []byte
}

func ParseChecksum(s string) (Checksum, error) {
	name, digest, ok := strings.Cut(strings.TrimSpace(s), " ")
	if !ok {
		return Checksum{}, fmt.Errorf("%w: expected an algorithm and digest", ErrInvalidChecksum)
	}
	a, err := ParseAlgo(name)
	if err != nil {
		return Checksum{}, err
	}
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(digest))
	if err != nil {
		return Checksum{}, fmt.Errorf("%w: digest is not base64", ErrInvalidChecksum)
	}
	if err := ValidateDigest(a, len(b)); err != nil {
		return Checksum{}, err
	}
	return Checksum{Algo: a, Digest: b}, nil
}

func (c Checksum) String() string {
	return c.Algo.String() + " " + base64.StdEncoding.EncodeToString(c.Digest)
}

// Verify is the optional whole-file integrity requirement for a session.
type Verify struct {
	Algo   Algo
	Digest []byte
}

// Meta is caller-provided content metadata; it does not select a destination.
type Meta struct {
	Filename     string
	MtimeNs      *int64
	Mime         string
	RelativePath string
	Verify       *Verify
}

// SessionSpec describes the neutral lifecycle portion of a resumable session.
type SessionSpec struct {
	TotalLen    *uint64
	ChunkSize   *uint64
	RandomAccess bool
	IfMatch     string
	Mode        SpoolMode
	Meta        Meta
}
