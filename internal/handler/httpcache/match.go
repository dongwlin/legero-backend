package httpcache

import (
	"net/textproto"
	"strings"
)

// MatchIfNoneMatch implements the weak comparison required by If-None-Match.
// It parses the complete field-value instead of splitting on commas because
// commas are valid inside an opaque tag. A malformed field-value is ignored
// (returns false) as required for an invalid precondition header.
func MatchIfNoneMatch(value, current string) bool {
	value = textproto.TrimString(value)
	if value == "*" {
		return true
	}

	matched := false
	for {
		value = textproto.TrimString(value)
		if value == "" {
			return matched
		}
		if value[0] == ',' {
			value = value[1:]
			continue
		}
		etag, remain, ok := scanETag(value)
		if !ok {
			return false
		}
		if weakMatch(etag, current) {
			matched = true
		}

		remain = textproto.TrimString(remain)
		if remain == "" {
			return matched
		}
		if remain[0] != ',' {
			return false
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

// weakMatch compares two ETags ignoring the W/ prefix, implementing the weak
// comparison semantics required by If-None-Match.
func weakMatch(left, right string) bool {
	return strings.TrimPrefix(left, "W/") == strings.TrimPrefix(right, "W/")
}
