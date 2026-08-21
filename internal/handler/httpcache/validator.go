package httpcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Validator produces an ETag string for a given HTTP representation.
// Concrete implementations are Strong (byte-level SHA-256) and Weak
// (resource-version based). The interface mirrors httpresp.Validator so that
// httpcache can provide implementations without importing httpresp.
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

// Weak creates a Weak ETag validator from a resource identifier and version.
// Format: W/"{resource}-{id}-{version}"
//
// resource must be a server-controlled token (e.g. "order").
// id must be a safe token (e.g. a UUID string).
// version must be a monotonic business-resource version.
func Weak(resource, id string, version int64) Validator {
	return weakValidator{etag: fmt.Sprintf(`W/"%s-%s-%d"`, resource, id, version)}
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
