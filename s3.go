package transfer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
)

// S3Part is provider evidence for one multipart part. Number is one-based.
type S3Part struct {
	Number   int
	ETag     string
	Size     uint64
	Checksum string
}

// S3MultipartStrategy describes the safe multipart lifecycle without tying the
// transfer package to an SDK. Implementations must verify the complete object
// after assembly before returning Published.
type S3MultipartStrategy interface {
	BeginMultipart(context.Context, Intent, uint64, string) (uploadID string, err error)
	UploadPart(context.Context, Intent, string, int, io.Reader, uint64, string) (S3Part, error)
	ListParts(context.Context, Intent, string) ([]S3Part, error)
	CompleteMultipart(context.Context, Intent, string, []S3Part, string) (Result, Receipt, error)
	AbortMultipart(context.Context, Intent, string) error
}

// S3Strategy is a Strategy that exposes an explicit multipart contract. This
// interface does not authorize direct client uploads or permit completion
// without whole-file verification.
type S3Strategy interface {
	Strategy
	S3MultipartStrategy
}

// VerifyWholeFile checks the assembled object represented by content and
// expectedDigest. Native multipart implementations should use an equivalent
// provider-side check before reporting Published.
func VerifyWholeFile(content []byte, expectedDigest string) error {
	if expectedDigest == "" {
		return errors.New("transfer: whole-file digest required")
	}
	sum := sha256.Sum256(content)
	if !bytes.Equal([]byte(hex.EncodeToString(sum[:])), []byte(expectedDigest)) {
		return errors.New("transfer: whole-file verification failed")
	}
	return nil
}

// S3Fake is an in-memory conformance fake for strategy tests. It intentionally
// assembles parts and verifies the whole-file digest before publication.
type S3Fake struct {
	mu      sync.Mutex
	nextID  uint64
	uploads map[string]*s3FakeUpload
	objects map[Destination][]byte
}

type s3FakeUpload struct {
	intent Intent
	parts  map[int][]byte
}

func NewS3Fake() *S3Fake {
	return &S3Fake{uploads: make(map[string]*s3FakeUpload), objects: make(map[Destination][]byte)}
}

func (f *S3Fake) BeginMultipart(_ context.Context, intent Intent, _ uint64, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := fmt.Sprintf("fake-%d", f.nextID)
	f.uploads[id] = &s3FakeUpload{intent: intent, parts: make(map[int][]byte)}
	return id, nil
}

func (f *S3Fake) UploadPart(_ context.Context, _ Intent, uploadID string, number int, body io.Reader, size uint64, checksum string) (S3Part, error) {
	if number <= 0 {
		return S3Part{}, errors.New("transfer: multipart part number must be positive")
	}
	data, err := io.ReadAll(io.LimitReader(body, int64(size)+1))
	if err != nil {
		return S3Part{}, err
	}
	if uint64(len(data)) != size {
		return S3Part{}, errors.New("transfer: multipart part size mismatch")
	}
	if checksum != "" {
		sum := sha256.Sum256(data)
		if checksum != hex.EncodeToString(sum[:]) {
			return S3Part{}, errors.New("transfer: multipart part checksum mismatch")
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	u := f.uploads[uploadID]
	if u == nil {
		return S3Part{}, errors.New("transfer: unknown multipart upload")
	}
	u.parts[number] = append([]byte(nil), data...)
	sum := sha256.Sum256(data)
	return S3Part{Number: number, ETag: hex.EncodeToString(sum[:]), Size: size, Checksum: hex.EncodeToString(sum[:])}, nil
}

func (f *S3Fake) ListParts(_ context.Context, _ Intent, uploadID string) ([]S3Part, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := f.uploads[uploadID]
	if u == nil {
		return nil, errors.New("transfer: unknown multipart upload")
	}
	out := make([]S3Part, 0, len(u.parts))
	for n, b := range u.parts {
		sum := sha256.Sum256(b)
		out = append(out, S3Part{Number: n, ETag: hex.EncodeToString(sum[:]), Size: uint64(len(b)), Checksum: hex.EncodeToString(sum[:])})
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Number < out[j-1].Number; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out, nil
}

func (f *S3Fake) CompleteMultipart(_ context.Context, intent Intent, uploadID string, parts []S3Part, digest string) (Result, Receipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := f.uploads[uploadID]
	if u == nil {
		return Result{Outcome: NotPublished}, Receipt{}, errors.New("transfer: unknown multipart upload")
	}
	if u.intent.Identity() != intent.Identity() {
		return Result{Outcome: NotPublished}, Receipt{}, errors.New("transfer: multipart intent mismatch")
	}
	var content []byte
	for _, p := range parts {
		b, ok := u.parts[p.Number]
		if !ok || uint64(len(b)) != p.Size {
			return Result{Outcome: NotPublished}, Receipt{}, errors.New("transfer: multipart part evidence mismatch")
		}
		content = append(content, b...)
	}
	if err := VerifyWholeFile(content, digest); err != nil {
		return Result{Outcome: NotPublished}, Receipt{}, err
	}
	f.objects[intent.Destination] = append([]byte(nil), content...)
	delete(f.uploads, uploadID)
	sum := sha256.Sum256(content)
	r := Receipt{Operation: intent.Operation, Destination: intent.Destination, Content: intent.Content, Range: Range{Hi: uint64(len(content))}, Digest: hex.EncodeToString(sum[:]), Revision: hex.EncodeToString(sum[:])}
	return Result{Outcome: Published, Commit: true}, r, nil
}

func (f *S3Fake) AbortMultipart(_ context.Context, _ Intent, uploadID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.uploads[uploadID]; !ok {
		return errors.New("transfer: unknown multipart upload")
	}
	delete(f.uploads, uploadID)
	return nil
}

func (f *S3Fake) Object(destination Destination) ([]byte, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.objects[destination]
	return append([]byte(nil), b...), ok
}

var _ S3MultipartStrategy = (*S3Fake)(nil)
