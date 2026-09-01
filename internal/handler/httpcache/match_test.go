package httpcache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchIfNoneMatchUsesWeakComparison(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
		want  bool
	}{
		{name: "strong request tag matches weak current tag", value: `"current"`, want: true},
		{name: "weak request tag matches strong current tag", value: `W/"current"`, want: true},
		{name: "matching tag later in list", value: `"other", W/"current"`, want: true},
		{name: "opaque value may contain comma", value: `"opaque,tag", "current"`, want: true},
		{name: "wildcard", value: `*`, want: true},
		{name: "mismatch", value: `"other"`, want: false},
		{name: "malformed tag is ignored", value: `"unterminated`, want: false},
		{name: "wildcard must stand alone", value: `*, "current"`, want: false},
		{name: "wildcard after a tag is invalid", value: `"current", *`, want: false},
		{name: "wildcard after an empty list element is invalid", value: `, *`, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, MatchIfNoneMatch(test.value, `W/"current"`))
		})
	}
}

func TestMatchIfNoneMatchRejectsMalformedTrailingBytes(t *testing.T) {
	for _, value := range []string{`"current" extra`, `"current";`, `W/`, `"bad` + string(rune(1)) + `"`} {
		require.False(t, MatchIfNoneMatch(value, `"current"`), value)
	}
}

func TestETagMaterializesStrongAndWeakValidators(t *testing.T) {
	body := []byte(`{"code":"0"}`)
	require.Equal(t, StrongETag(body), ETag(Strong(), body))
	require.Equal(t, `W/"order-order-1-7"`, ETag(Weak("order", "order-1", 7), nil))
}

func TestWeakEscapesUnsafeOpaqueValue(t *testing.T) {
	etag := ETag(Weak("ord\"er", "id\r\nX-Test: injected", 42), nil)
	require.Regexp(t, `^W/"[0-9a-f]+"$`, etag)
	require.NotContains(t, etag, "\r")
	require.NotContains(t, etag, "\n")
}
