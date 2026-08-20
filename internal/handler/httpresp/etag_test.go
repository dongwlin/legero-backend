package httpresp

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPrivateJSONWithETagGETAddsStableStrongETagFromRenderedBytes(t *testing.T) {
	first := serveJSON(t, http.MethodGet, "", http.StatusOK, gin.H{"message": "hello"})
	second := serveJSON(t, http.MethodGet, "", http.StatusOK, gin.H{"message": "hello"})
	changed := serveJSON(t, http.MethodGet, "", http.StatusOK, gin.H{"message": "goodbye"})

	require.Equal(t, first.Body.String(), `{"message":"hello"}`)
	require.Equal(t, first.Header().Get("ETag"), second.Header().Get("ETag"))
	require.NotEqual(t, first.Header().Get("ETag"), changed.Header().Get("ETag"))
	require.Equal(t, StrongETag(first.Body.Bytes()), first.Header().Get("ETag"))
	require.Regexp(t, `^"[0-9a-f]{64}"$`, first.Header().Get("ETag"))
	_, err := hex.DecodeString(strings.Trim(first.Header().Get("ETag"), `"`))
	require.NoError(t, err)
	require.Equal(t, "private, no-cache", first.Header().Get("Cache-Control"))
	require.Equal(t, "Authorization", first.Header().Get("Vary"))
}

func TestPrivateJSONWithETagGETIfNoneMatchUsesWeakComparisonAcrossListsAndOpaqueCommas(t *testing.T) {
	initial := serveJSON(t, http.MethodGet, "", http.StatusOK, gin.H{"message": "hello"})
	etag := initial.Header().Get("ETag")

	for _, test := range []struct {
		name   string
		header string
	}{
		{name: "weak tag in list", header: `"other", W/` + etag},
		{name: "opaque tag containing commas", header: `"opaque,tag", ` + etag},
		{name: "wildcard", header: `*`},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := serveJSON(t, http.MethodGet, test.header, http.StatusOK, gin.H{"message": "hello"})
			require.Equal(t, http.StatusNotModified, response.Code)
			require.Empty(t, response.Body.Bytes())
			require.Equal(t, etag, response.Header().Get("ETag"))
			require.Empty(t, response.Header().Get("Content-Length"))
			require.Empty(t, response.Header().Get("Content-Type"))
			require.Equal(t, "private, no-cache", response.Header().Get("Cache-Control"))
			require.Equal(t, "Authorization", response.Header().Get("Vary"))
		})
	}
}

func TestPrivateJSONWithETagGETIfNoneMatchIgnoresMismatchedAndMalformedTags(t *testing.T) {
	for _, header := range []string{`"other"`, `W/not-an-etag`, `"unterminated`} {
		response := serveJSON(t, http.MethodGet, header, http.StatusOK, gin.H{"message": "hello"})
		require.Equal(t, http.StatusOK, response.Code, header)
		require.Equal(t, `{"message":"hello"}`, response.Body.String(), header)
		require.NotEmpty(t, response.Header().Get("ETag"), header)
	}
}

func TestStrongETagHashesExactResponseBytes(t *testing.T) {
	body := []byte(`{"message":"hello"}`)
	digest := sha256.Sum256(body)
	want := `"` + hex.EncodeToString(digest[:]) + `"`

	require.Equal(t, want, StrongETag(body))
	require.Equal(t, want, strongETag(body))
}

func TestVersionETagUsesWeakResourceVersionFormat(t *testing.T) {
	require.Equal(t, `W/"order-0198cabc-42"`, VersionETag("order", "0198cabc", 42))
	require.NotEqual(t, VersionETag("order", "0198cabc", 42), VersionETag("order", "0198cabc", 43))
	require.NotEqual(t, VersionETag("order", "0198cabc", 42), VersionETag("order", "0198cabd", 42))
}

func TestPrivateJSONWithETagOnlyAppliesTo200Responses(t *testing.T) {
	for _, status := range []int{
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusNoContent,
		http.StatusPartialContent,
		http.StatusMultipleChoices,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			response := serveJSON(t, http.MethodGet, "", status, gin.H{"message": "hello"})

			require.Equal(t, status, response.Code)
			require.Empty(t, response.Header().Get("ETag"))
			require.Empty(t, response.Header().Get("Cache-Control"))
			require.Empty(t, response.Header().Get("Vary"))
		})
	}
}

func TestIfNoneMatchRequiresAValidCompleteTagList(t *testing.T) {
	current := `W/"current"`
	for _, test := range []struct {
		name  string
		value string
		want  bool
	}{
		{name: "matching tag then malformed tag", value: `"current", malformed`, want: false},
		{name: "leading empty element", value: `, "current"`, want: true},
		{name: "consecutive empty elements", value: `"other",, "current"`, want: true},
		{name: "trailing empty element", value: `"current",`, want: true},
		{name: "wildcard mixed with tags", value: `*, "current"`, want: false},
		{name: "wildcard with trailing comma", value: `*,`, want: false},
		{name: "matching weak tag", value: `"current"`, want: true},
		{name: "opaque comma", value: `"opaque,tag", "current"`, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, matchesIfNoneMatch(test.value, current))
		})
	}
}

func TestMatchesIfNoneMatchAllocationsDoNotScaleWithTagCount(t *testing.T) {
	const current = `"current"`
	oneTag := `"current"`
	manyTags := strings.Repeat(`"",`, 8_192) + current

	require.True(t, matchesIfNoneMatch(oneTag, current))
	require.True(t, matchesIfNoneMatch(manyTags, current))

	oneTagAllocs := testing.AllocsPerRun(100, func() {
		if !matchesIfNoneMatch(oneTag, current) {
			panic("single tag should match")
		}
	})
	manyTagAllocs := testing.AllocsPerRun(100, func() {
		if !matchesIfNoneMatch(manyTags, current) {
			panic("large tag list should match")
		}
	})

	require.LessOrEqual(t, manyTagAllocs, oneTagAllocs+1)
}

func TestPrivateJSONWithETagCombinesRepeatedIfNoneMatchHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/resource", func(c *gin.Context) {
		PrivateJSONWithETag(c, http.StatusOK, gin.H{"message": "hello"})
	})

	initial := httptest.NewRecorder()
	router.ServeHTTP(initial, httptest.NewRequest(http.MethodGet, "/resource", nil))
	etag := initial.Header().Get("ETag")

	request := httptest.NewRequest(http.MethodGet, "/resource", nil)
	request.Header.Add("If-None-Match", `"other"`)
	request.Header.Add("If-None-Match", `W/`+etag)
	revalidated := httptest.NewRecorder()
	router.ServeHTTP(revalidated, request)

	require.Equal(t, http.StatusNotModified, revalidated.Code)
	require.Empty(t, revalidated.Body.Bytes())
	require.Equal(t, etag, revalidated.Header().Get("ETag"))
}

func TestPrivateJSONWithETagIgnoresRepeatedHeaderWhenLaterValueIsMalformed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/resource", func(c *gin.Context) {
		PrivateJSONWithETag(c, http.StatusOK, gin.H{"message": "hello"})
	})

	initial := httptest.NewRecorder()
	router.ServeHTTP(initial, httptest.NewRequest(http.MethodGet, "/resource", nil))
	etag := initial.Header().Get("ETag")

	request := httptest.NewRequest(http.MethodGet, "/resource", nil)
	request.Header.Add("If-None-Match", etag)
	request.Header.Add("If-None-Match", `"unterminated`)
	revalidated := httptest.NewRecorder()
	router.ServeHTTP(revalidated, request)

	require.Equal(t, http.StatusOK, revalidated.Code)
	require.Equal(t, `{"message":"hello"}`, revalidated.Body.String())
	require.Equal(t, etag, revalidated.Header().Get("ETag"))
}

func TestPrivateJSONWithVersionETagSkipsMarshalOnHit(t *testing.T) {
	var factoryCalls, marshalCalls int
	payload := func() any {
		factoryCalls++
		return countingJSONMarshaler{calls: &marshalCalls}
	}

	first := serveVersionJSON(t, http.MethodGet, "", payload)
	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, 1, factoryCalls)
	require.Equal(t, 1, marshalCalls)
	etag := first.Header().Get("ETag")
	require.Equal(t, `W/"order-order-1-7"`, etag)

	factoryCalls = 0
	marshalCalls = 0
	second := serveVersionJSON(t, http.MethodGet, etag, payload)
	require.Equal(t, http.StatusNotModified, second.Code)
	require.Empty(t, second.Body.Bytes())
	require.Equal(t, 0, factoryCalls)
	require.Equal(t, 0, marshalCalls)
	require.Equal(t, etag, second.Header().Get("ETag"))
	require.Equal(t, "private, no-cache", second.Header().Get("Cache-Control"))
	require.Equal(t, "Authorization", second.Header().Get("Vary"))
}

func TestPrivateJSONWithVersionETagUsesWeakComparisonForHead(t *testing.T) {
	get := serveVersionJSON(t, http.MethodGet, "", gin.H{"message": "hello"})
	head := serveVersionJSON(t, http.MethodHead, `"order-order-1-7"`, gin.H{"message": "hello"})

	require.Equal(t, http.StatusOK, get.Code)
	require.Equal(t, http.StatusNotModified, head.Code)
	require.Empty(t, head.Body.Bytes())
	require.Equal(t, get.Header().Get("ETag"), head.Header().Get("ETag"))
}

func TestPrivateJSONWithVersionETagHeadOmitsBodyAndMarshalWhenUnconditional(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var factoryCalls, marshalCalls int
	payload := func() any {
		factoryCalls++
		return countingJSONMarshaler{calls: &marshalCalls}
	}
	router := gin.New()
	router.HEAD("/resource", func(c *gin.Context) {
		PrivateJSONWithVersionETag(c, http.StatusOK, payload, "order", "order-1", 7)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/resource", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Body.Bytes())
	require.Equal(t, `W/"order-order-1-7"`, recorder.Header().Get("ETag"))
	require.Equal(t, jsonContentType, recorder.Header().Get("Content-Type"))
	require.Empty(t, recorder.Header().Get("Content-Length"))
	require.Equal(t, 0, factoryCalls)
	require.Equal(t, 0, marshalCalls)
}

func TestPrivateJSONWithVersionETagPanicDoesNotLeakETag(t *testing.T) {
	tests := []struct {
		name    string
		payload func() any
	}{
		{
			name: "payload factory",
			payload: func() any {
				panic("payload panic")
			},
		},
		{
			name: "JSON marshaler",
			payload: func() any {
				return panickingJSONMarshaler{}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(gin.RecoveryWithWriter(io.Discard))
			router.GET("/resource", func(c *gin.Context) {
				PrivateJSONWithVersionETag(c, http.StatusOK, test.payload, "order", "order-1", 7)
			})

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/resource", nil))

			require.Equal(t, http.StatusInternalServerError, recorder.Code)
			require.Empty(t, recorder.Header().Get("ETag"))
		})
	}
}

func TestResponseHashETagTracksFinalBytesAndIgnoresQueryWhenBytesMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/varying", func(c *gin.Context) {
		JSON(c, http.StatusOK, gin.H{"value": c.Query("value")})
	})
	router.GET("/stable", func(c *gin.Context) {
		JSON(c, http.StatusOK, gin.H{"value": "same"})
	})

	firstVarying := httptest.NewRecorder()
	router.ServeHTTP(firstVarying, httptest.NewRequest(http.MethodGet, "/varying?value=one", nil))
	secondVarying := httptest.NewRecorder()
	router.ServeHTTP(secondVarying, httptest.NewRequest(http.MethodGet, "/varying?value=two", nil))
	require.NotEqual(t, firstVarying.Body.Bytes(), secondVarying.Body.Bytes())
	require.NotEqual(t, firstVarying.Header().Get("ETag"), secondVarying.Header().Get("ETag"))

	firstStable := httptest.NewRecorder()
	router.ServeHTTP(firstStable, httptest.NewRequest(http.MethodGet, "/stable?value=one", nil))
	secondStable := httptest.NewRecorder()
	router.ServeHTTP(secondStable, httptest.NewRequest(http.MethodGet, "/stable?value=two", nil))
	require.Equal(t, firstStable.Body.Bytes(), secondStable.Body.Bytes())
	require.Equal(t, firstStable.Header().Get("ETag"), secondStable.Header().Get("ETag"))
}

func TestPrivateJSONWithVersionETagDoesNotTreatWeakTagAsIfMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/resource", func(c *gin.Context) {
		PrivateJSONWithVersionETag(c, http.StatusOK, func() any {
			return gin.H{"message": "hello"}
		}, "order", "order-1", 7)
	})

	initial := httptest.NewRecorder()
	router.ServeHTTP(initial, httptest.NewRequest(http.MethodGet, "/resource", nil))
	etag := initial.Header().Get("ETag")

	request := httptest.NewRequest(http.MethodGet, "/resource", nil)
	request.Header.Set("If-Match", etag)
	revalidated := httptest.NewRecorder()
	router.ServeHTTP(revalidated, request)

	require.Equal(t, http.StatusOK, revalidated.Code)
	require.NotEmpty(t, revalidated.Body.Bytes())
	require.Equal(t, etag, revalidated.Header().Get("ETag"))
}

type countingJSONMarshaler struct {
	calls *int
}

func (m countingJSONMarshaler) MarshalJSON() ([]byte, error) {
	*m.calls = *m.calls + 1
	return []byte(`{"message":"hello"}`), nil
}

type panickingJSONMarshaler struct{}

func (panickingJSONMarshaler) MarshalJSON() ([]byte, error) {
	panic("marshal panic")
}

func TestMatchesIfNoneMatchRequiresWildcardToStandAlone(t *testing.T) {
	for _, test := range []struct {
		name    string
		value   string
		current string
		want    bool
	}{
		{name: "bare wildcard", value: `*`, current: `"current"`, want: true},
		{name: "wildcard with optional whitespace", value: " \t* \t", current: `"current"`, want: true},
		{name: "wildcard followed by tag", value: `*, "foo"`, current: `"foo"`, want: false},
		{name: "tag followed by wildcard", value: `"foo", *`, current: `"foo"`, want: false},
		{name: "wildcard followed by comma", value: `*,`, current: `"current"`, want: false},
		{name: "wildcard after one leading comma", value: `, *`, current: `"current"`, want: false},
		{name: "wildcard after multiple leading commas", value: `,, *`, current: `"current"`, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, matchesIfNoneMatch(test.value, test.current))
		})
	}
}

func TestPrivateJSONWithETagAppliesToHEADWithoutWritingBody(t *testing.T) {
	get := serveJSON(t, http.MethodGet, "", http.StatusOK, gin.H{"message": "hello"})
	response := serveJSON(t, http.MethodHead, "", http.StatusOK, gin.H{"message": "hello"})

	require.Equal(t, http.StatusOK, response.Code)
	require.Empty(t, response.Body.Bytes())
	require.Equal(t, get.Header().Get("ETag"), response.Header().Get("ETag"))
	require.Equal(t, "19", response.Header().Get("Content-Length"))
}

func TestPrivateJSONWithETagDoesNotApplyToPostOrErrors(t *testing.T) {
	post := serveJSON(t, http.MethodPost, "", http.StatusOK, gin.H{"message": "hello"})
	require.Equal(t, http.StatusOK, post.Code)
	require.Empty(t, post.Header().Get("ETag"))
	require.Empty(t, post.Header().Get("Cache-Control"))

	errResponse := serveJSON(t, http.MethodGet, "", http.StatusBadRequest, gin.H{"error": "bad"})
	require.Equal(t, http.StatusBadRequest, errResponse.Code)
	require.Empty(t, errResponse.Header().Get("ETag"))
	require.Empty(t, errResponse.Header().Get("Cache-Control"))
}

func TestPrivateJSONWithETagAppendsVaryWithoutOverwritingExistingValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Header("Vary", "Origin")
		c.Next()
	})
	router.GET("/resource", func(c *gin.Context) {
		PrivateJSONWithETag(c, http.StatusOK, gin.H{"message": "hello"})
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/resource", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "Origin, Authorization", recorder.Header().Get("Vary"))
}

func TestJSONAddsETagWithoutPrivateCachePolicy(t *testing.T) {
	response := servePlainJSON(t, http.MethodGet, http.StatusOK, gin.H{"message": "hello"})

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, `{"message":"hello"}`, response.Body.String())
	require.Regexp(t, `^"[0-9a-f]{64}"$`, response.Header().Get("ETag"))
	require.Empty(t, response.Header().Get("Cache-Control"))
	require.Empty(t, response.Header().Get("Vary"))
}

func TestJSONGETIfNoneMatchUsesETagWithoutPrivateCachePolicy(t *testing.T) {
	initial := servePlainJSON(t, http.MethodGet, http.StatusOK, gin.H{"message": "hello"})
	etag := initial.Header().Get("ETag")

	response := servePlainJSONWithIfNoneMatch(t, http.MethodGet, etag, http.StatusOK, gin.H{"message": "hello"})

	require.Equal(t, http.StatusNotModified, response.Code)
	require.Empty(t, response.Body.Bytes())
	require.Equal(t, etag, response.Header().Get("ETag"))
	require.Empty(t, response.Header().Get("Cache-Control"))
	require.Empty(t, response.Header().Get("Vary"))
}

func TestPrivateJSONAddsPolicyWithoutETag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/resource", func(c *gin.Context) {
		PrivateJSON(c, http.StatusOK, gin.H{"message": "hello"})
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/resource", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Header().Get("ETag"))
	require.Equal(t, "private, no-cache", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "Authorization", recorder.Header().Get("Vary"))
}

func TestPrivateJSONOnlyAppliesPolicyToSuccessfulGETAndHEAD(t *testing.T) {
	for _, test := range []struct {
		name   string
		method string
		status int
	}{
		{name: "post", method: http.MethodPost, status: http.StatusOK},
		{name: "error", method: http.MethodGet, status: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := servePrivateJSON(t, test.method, test.status, gin.H{"message": "hello"})

			require.Equal(t, test.status, response.Code)
			require.Empty(t, response.Header().Get("ETag"))
			require.Empty(t, response.Header().Get("Cache-Control"))
			require.Empty(t, response.Header().Get("Vary"))
		})
	}
}

func TestAppendVaryAvoidsCaseInsensitiveDuplicates(t *testing.T) {
	header := make(http.Header)
	header.Set("Vary", "Origin")
	appendVary(header, "Authorization")
	appendVary(header, "authorization")

	require.Equal(t, "Origin, Authorization", header.Get("Vary"))
}

func serveJSON(t *testing.T, method, ifNoneMatch string, status int, payload any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, "/resource", func(c *gin.Context) {
		PrivateJSONWithETag(c, status, payload)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, "/resource", nil)
	if ifNoneMatch != "" {
		request.Header.Set("If-None-Match", ifNoneMatch)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func servePlainJSON(t *testing.T, method string, status int, payload any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, "/resource", func(c *gin.Context) {
		JSON(c, status, payload)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(method, "/resource", nil))
	return recorder
}

func servePlainJSONWithIfNoneMatch(t *testing.T, method, ifNoneMatch string, status int, payload any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, "/resource", func(c *gin.Context) {
		JSON(c, status, payload)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, "/resource", nil)
	request.Header.Set("If-None-Match", ifNoneMatch)
	router.ServeHTTP(recorder, request)
	return recorder
}

func servePrivateJSON(t *testing.T, method string, status int, payload any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, "/resource", func(c *gin.Context) {
		PrivateJSON(c, status, payload)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(method, "/resource", nil))
	return recorder
}

func serveVersionJSON(t *testing.T, method, ifNoneMatch string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, "/resource", func(c *gin.Context) {
		payloadFactory := func() any { return payload }
		if lazyPayload, ok := payload.(func() any); ok {
			payloadFactory = lazyPayload
		}
		PrivateJSONWithVersionETag(c, http.StatusOK, payloadFactory, "order", "order-1", 7)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, "/resource", nil)
	if ifNoneMatch != "" {
		request.Header.Set("If-None-Match", ifNoneMatch)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}
