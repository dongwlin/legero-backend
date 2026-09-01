package httpcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Validator identifies the representation validator selected by a handler.
// Concrete implementations are Strong (byte-level SHA-256) and Weak
// (resource-version based).
type Validator interface {
	ETag() string
}

// Strong creates a Strong ETag validator. The handler must pass the exact
// response body bytes that will be sent on the wire; the middleware computes
// SHA-256 at render time.
type strongValidator struct{}

// Strong returns a Strong ETag validator. The actual hash is computed by the
// middleware from the final representation bytes — this constructor merely
// signals the intent.
func Strong() Validator {
	return strongValidator{}
}

func (strongValidator) ETag() string {
	// The real ETag is computed by the middleware from the final body bytes.
	// Returning "" is how the middleware recognizes a Strong validator and
	// knows to compute the hash from the response body.
	return ""
}

// ETag materializes the validator for the supplied representation bytes.
// Strong validators are derived from the final bytes; Weak validators retain
// the resource-version value supplied at construction time. Keeping this
// dispatch in httpcache leaves the response renderer unaware of validator
// behavior.
func ETag(validator Validator, body []byte) string {
	if validator == nil {
		return ""
	}
	if _, ok := validator.(strongValidator); ok {
		return StrongETag(body)
	}
	return validator.ETag()
}

// Weak creates a Weak ETag validator from a resource identifier and version.
// Format: W/"{resource}-{id}-{version}"
//
// resource must be a server-controlled token (e.g. "order").
// id must be a safe token (e.g. a UUID string).
// version must be a monotonic business-resource version.
func Weak(resource, id string, version int64) Validator {
	value := fmt.Sprintf("%s-%s-%d", resource, id, version)
	if !safeOpaque(value) {
		value = hex.EncodeToString([]byte(value))
	}
	return weakValidator{etag: `W/"` + value + `"`}
}

type weakValidator struct {
	etag string
}

func (v weakValidator) ETag() string {
	return v.etag
}

// StrongETag computes the strong validator for the given body bytes.
// The input must be the exact bytes that will be written to the HTTP response.
func StrongETag(body []byte) string {
	digest := sha256.Sum256(body)
	return `"` + hex.EncodeToString(digest[:]) + `"`
}

// safeOpaque reports whether value can be embedded directly in an entity-tag
// opaque value. Invalid bytes are encoded by Weak before the value is exposed
// as a response header, preventing control characters or quotes from escaping
// the header field.
func safeOpaque(value string) bool {
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character == 0x21 || character >= 0x23 && character <= 0x7e || character >= 0x80 {
			continue
		}
		return false
	}
	return true
}
