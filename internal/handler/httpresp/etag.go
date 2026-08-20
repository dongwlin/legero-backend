package httpresp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/textproto"
	"strings"
)

func etagMethod(request *http.Request) bool {
	return request != nil && (request.Method == http.MethodGet || request.Method == http.MethodHead)
}

func etagStatus(status int) bool {
	// Validators in this package are intentionally scoped to complete 200
	// representations. In particular, a 201/202 response may have different
	// semantics and a 206 response is only a partial representation.
	return status == http.StatusOK
}

func strongETag(body []byte) string {
	digest := sha256.Sum256(body)
	return `"` + hex.EncodeToString(digest[:]) + `"`
}

// StrongETag returns the strong validator for a response representation. The
// input must be the exact bytes that will be sent in the HTTP response.
func StrongETag(body []byte) string {
	return strongETag(body)
}

// VersionETag returns the weak validator used for a single versioned
// resource. Version validators intentionally use weak comparison because a
// resource version guarantees semantic equivalence, not byte-for-byte JSON
// equality.
func VersionETag(resource, id string, version int64) string {
	return fmt.Sprintf(`W/"%s-%s-%d"`, resource, id, version)
}

func appendVary(header http.Header, value string) {
	existingValues := header.Values("Vary")
	for _, existing := range existingValues {
		for _, token := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(token), value) {
				return
			}
		}
	}
	if len(existingValues) == 0 {
		header.Set("Vary", value)
		return
	}
	header.Set("Vary", strings.Join(append(existingValues, value), ", "))
}

// matchesIfNoneMatch implements the weak comparison required by
// If-None-Match. It parses the complete field-value instead of splitting on
// commas because commas are valid inside an opaque tag. A malformed complete
// field-value is ignored, as required for an invalid precondition header.
func matchesIfNoneMatch(value, current string) bool {
	value = textproto.TrimString(value)
	if value == "*" {
		return true
	}

	etags, ok := parseEntityTagList(value)
	if !ok {
		return false
	}
	for _, etag := range etags {
		if weakETagMatch(etag, current) {
			return true
		}
	}
	return false
}

// parseEntityTagList parses an If-None-Match field-value. The wildcard is
// handled by matchesIfNoneMatch before this function and is valid only when
// it is the complete field-value. HTTP #list syntax permits empty elements,
// so leading, consecutive, and trailing commas are skipped. Any bytes
// outside entity-tag syntax make the whole field-value invalid.
func parseEntityTagList(value string) ([]string, bool) {
	value = textproto.TrimString(value)
	if value == "" {
		return nil, false
	}

	etags := make([]string, 0, 1)
	for {
		value = textproto.TrimString(value)
		if value == "" {
			return etags, true
		}
		if value[0] == ',' {
			value = value[1:]
			continue
		}
		etag, remain, ok := scanETag(value)
		if !ok {
			return nil, false
		}
		etags = append(etags, etag)

		remain = textproto.TrimString(remain)
		if remain == "" {
			return etags, true
		}
		if remain[0] != ',' {
			return nil, false
		}
		value = remain[1:]
	}
}

// scanETag returns one syntactically valid entity-tag, the text after it, and
// whether parsing succeeded. Commas inside the quoted opaque value are
// intentionally accepted.
func scanETag(value string) (etag, remain string, ok bool) {
	value = textproto.TrimString(value)
	start := 0
	if strings.HasPrefix(value, "W/") {
		start = 2
	}
	if len(value[start:]) < 2 || value[start] != '"' {
		return "", "", false
	}
	for index := start + 1; index < len(value); index++ {
		character := value[index]
		switch {
		case character == 0x21 || character >= 0x23 && character <= 0x7e || character >= 0x80:
			continue
		case character == '"':
			return value[:index+1], value[index+1:], true
		default:
			return "", "", false
		}
	}
	return "", "", false
}

func weakETagMatch(left, right string) bool {
	return strings.TrimPrefix(left, "W/") == strings.TrimPrefix(right, "W/")
}
