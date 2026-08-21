package httpcache

import (
	"net/http"
	"strings"
)

// SetPrivateCachePolicy sets Cache-Control and Vary headers for an
// authenticated representation that should be privately cached with
// revalidation.
//
//	Cache-Control: private, no-cache
//	Vary: Authorization
func SetPrivateCachePolicy(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-cache")
	AppendVary(w.Header(), "Authorization")
}

// AppendVary appends value to the Vary header, deduplicating
// case-insensitively and preserving existing values.
func AppendVary(header http.Header, value string) {
	existing := header.Values("Vary")
	for _, raw := range existing {
		for _, token := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' }) {
			if strings.EqualFold(strings.TrimSpace(token), value) {
				return
			}
		}
	}
	if len(existing) == 0 {
		header.Set("Vary", value)
		return
	}
	header.Set("Vary", strings.Join(append(existing, value), ", "))
}

// SetETag sets the ETag response header.
func SetETag(w http.ResponseWriter, etag string) {
	w.Header().Set("ETag", etag)
}

// IsCacheableStatus returns true for status codes eligible for ETag generation.
// Only 200 OK representations receive validators.
func IsCacheableStatus(status int) bool {
	return status == http.StatusOK
}

// IsCacheableMethod returns true for methods eligible for ETag generation.
func IsCacheableMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

// WriteNotModified sends a 304 Not Modified, preserving validator and cache
// headers while removing representation metadata (Content-Type,
// Content-Length) that would describe a body which is not present.
func WriteNotModified(w http.ResponseWriter) {
	w.Header().Del("Content-Length")
	w.Header().Del("Content-Type")
	w.WriteHeader(http.StatusNotModified)
}
