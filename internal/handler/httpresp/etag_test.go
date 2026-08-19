package httpresp

import (
	"encoding/hex"
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
	response := serveJSON(t, http.MethodHead, "", http.StatusOK, gin.H{"message": "hello"})

	require.Equal(t, http.StatusOK, response.Code)
	require.Empty(t, response.Body.Bytes())
	require.NotEmpty(t, response.Header().Get("ETag"))
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
