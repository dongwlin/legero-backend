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

func TestJSONGETAddsStableStrongETagFromRenderedBytes(t *testing.T) {
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

func TestJSONGETIfNoneMatchUsesWeakComparisonAcrossListsAndOpaqueCommas(t *testing.T) {
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

func TestJSONGETIfNoneMatchIgnoresMismatchedAndMalformedTags(t *testing.T) {
	for _, header := range []string{`"other"`, `W/not-an-etag`, `"unterminated`} {
		response := serveJSON(t, http.MethodGet, header, http.StatusOK, gin.H{"message": "hello"})
		require.Equal(t, http.StatusOK, response.Code, header)
		require.Equal(t, `{"message":"hello"}`, response.Body.String(), header)
		require.NotEmpty(t, response.Header().Get("ETag"), header)
	}
}

func TestJSONETagAppliesToHEADWithoutWritingBody(t *testing.T) {
	response := serveJSON(t, http.MethodHead, "", http.StatusOK, gin.H{"message": "hello"})

	require.Equal(t, http.StatusOK, response.Code)
	require.Empty(t, response.Body.Bytes())
	require.NotEmpty(t, response.Header().Get("ETag"))
	require.Equal(t, "19", response.Header().Get("Content-Length"))
}

func TestJSONETagDoesNotApplyToPostOrErrors(t *testing.T) {
	post := serveJSON(t, http.MethodPost, "", http.StatusOK, gin.H{"message": "hello"})
	require.Equal(t, http.StatusOK, post.Code)
	require.Empty(t, post.Header().Get("ETag"))
	require.Empty(t, post.Header().Get("Cache-Control"))

	errResponse := serveJSON(t, http.MethodGet, "", http.StatusBadRequest, gin.H{"error": "bad"})
	require.Equal(t, http.StatusBadRequest, errResponse.Code)
	require.Empty(t, errResponse.Header().Get("ETag"))
	require.Empty(t, errResponse.Header().Get("Cache-Control"))
}

func TestJSONETagAppendsVaryWithoutOverwritingExistingValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Header("Vary", "Origin")
		c.Next()
	})
	router.GET("/resource", func(c *gin.Context) {
		JSON(c, http.StatusOK, gin.H{"message": "hello"})
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/resource", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "Origin, Authorization", recorder.Header().Get("Vary"))
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
		JSON(c, status, payload)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, "/resource", nil)
	if ifNoneMatch != "" {
		request.Header.Set("If-None-Match", ifNoneMatch)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}
