package httpresp

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/textproto"
	"strings"
)

func etagMethod(request *http.Request) bool {
	return request != nil && (request.Method == http.MethodGet || request.Method == http.MethodHead)
}

func etagStatus(status int) bool {
	return status >= http.StatusOK && status < http.StatusMultipleChoices &&
		status != http.StatusNoContent && status != http.StatusNotModified
}

func strongETag(body []byte) string {
	digest := sha256.Sum256(body)
	return `"` + hex.EncodeToString(digest[:]) + `"`
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
// If-None-Match. It scans entity-tags instead of splitting on commas because
// commas are valid inside an opaque tag.
func matchesIfNoneMatch(value, current string) bool {
	for {
		value = textproto.TrimString(value)
		if value == "" {
			return false
		}
		if value[0] == ',' {
			value = value[1:]
			continue
		}
		if value[0] == '*' {
			remainder := textproto.TrimString(value[1:])
			return remainder == "" || strings.HasPrefix(remainder, ",")
		}

		etag, remain := scanETag(value)
		if etag == "" {
			// Invalid precondition syntax is ignored.
			return false
		}
		if weakETagMatch(etag, current) {
			return true
		}
		value = remain
	}
}

// scanETag returns one syntactically valid entity-tag and the text after it.
// This follows the parser used by net/http's conditional file responses.
func scanETag(value string) (etag, remain string) {
	value = textproto.TrimString(value)
	start := 0
	if strings.HasPrefix(value, "W/") {
		start = 2
	}
	if len(value[start:]) < 2 || value[start] != '"' {
		return "", ""
	}
	for index := start + 1; index < len(value); index++ {
		character := value[index]
		switch {
		case character == 0x21 || character >= 0x23 && character <= 0x7e || character >= 0x80:
		case character == '"':
			return value[:index+1], value[index+1:]
		default:
			return "", ""
		}
	}
	return "", ""
}

func weakETagMatch(left, right string) bool {
	return strings.TrimPrefix(left, "W/") == strings.TrimPrefix(right, "W/")
}
