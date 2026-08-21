package httpcache

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
)

// The tests in this file run against a real net/http server (httptest.Server)
// rather than httptest.ResponseRecorder: net/http snapshots the header set at
// the moment the header is committed, so a middleware that sets ETag and cache
// headers after the handler has written the body or committed headers would
// lose them on the wire. The recorder masks that failure; the real server does
// not.

func newTestServer(t *testing.T, handler gin.HandlerFunc) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(Middleware())
	r.GET("/thing", handler)
	r.HEAD("/thing", handler)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func doAndRead(t *testing.T, srv *httptest.Server, method string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+"/thing", nil)
	if err != nil {
		t.Fatalf("building %s request: %v", method, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s request failed: %v", method, err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading %s body: %v", method, err)
	}
	resp.Body.Close()
	return resp, body
}

func mustHave(t *testing.T, resp *http.Response, header, want string) {
	t.Helper()
	if got := resp.Header.Get(header); got != want {
		t.Errorf("%s = %q, want %q", header, got, want)
	}
}

// Weak ETag: GET and HEAD must carry identical headers (body excluded) and the
// same ETag, per the HEAD semantics of conditional-request.md §7.
func TestWeakETagGetAndHead(t *testing.T) {
	srv := newTestServer(t, func(c *gin.Context) {
		httpresp.JSON(c, http.StatusOK, map[string]any{"a": 1}, WithValidator(Weak("thing", "abc-123", 42)))
	})

	getResp, getBody := doAndRead(t, srv, http.MethodGet, nil)
	headResp, headBody := doAndRead(t, srv, http.MethodHead, nil)

	if getResp.StatusCode != http.StatusOK || headResp.StatusCode != http.StatusOK {
		t.Fatalf("status: GET=%d HEAD=%d, want 200/200", getResp.StatusCode, headResp.StatusCode)
	}
	if string(getBody) != `{"a":1}` {
		t.Errorf("GET body = %q, want %q", getBody, `{"a":1}`)
	}
	if len(headBody) != 0 {
		t.Errorf("HEAD body = %q, want empty", headBody)
	}

	const etag = `W/"thing-abc-123-42"`
	mustHave(t, getResp, "ETag", etag)
	mustHave(t, headResp, "ETag", etag)
	mustHave(t, getResp, "Cache-Control", "private, no-cache")
	mustHave(t, headResp, "Cache-Control", "private, no-cache")
	mustHave(t, getResp, "Vary", "Authorization")
	mustHave(t, headResp, "Vary", "Authorization")
	mustHave(t, getResp, "Content-Type", "application/json; charset=utf-8")
	mustHave(t, headResp, "Content-Type", "application/json; charset=utf-8")

	if getResp.Header.Get("Content-Length") != headResp.Header.Get("Content-Length") {
		t.Errorf("Content-Length differs: GET=%q HEAD=%q, want identical",
			getResp.Header.Get("Content-Length"), headResp.Header.Get("Content-Length"))
	}
}

// Strong ETag: the hash must be computed from the exact bytes sent on the wire,
// and HEAD must yield the same ETag as GET.
func TestStrongETagGetAndHead(t *testing.T) {
	srv := newTestServer(t, func(c *gin.Context) {
		httpresp.JSON(c, http.StatusOK, map[string]any{"a": 1}, WithValidator(Strong()))
	})

	getResp, getBody := doAndRead(t, srv, http.MethodGet, nil)
	headResp, headBody := doAndRead(t, srv, http.MethodHead, nil)

	want := StrongETag(getBody)
	mustHave(t, getResp, "ETag", want)
	mustHave(t, headResp, "ETag", want)
	if string(getBody) != `{"a":1}` {
		t.Errorf("GET body = %q, want %q", getBody, `{"a":1}`)
	}
	if len(headBody) != 0 {
		t.Errorf("HEAD body = %q, want empty", headBody)
	}
}

// If-None-Match match: 304 with the ETag preserved, no body, and no
// representation headers (Content-Type, Content-Length), for both GET and HEAD.
func TestIfNoneMatchMatch304(t *testing.T) {
	srv := newTestServer(t, func(c *gin.Context) {
		httpresp.JSON(c, http.StatusOK, map[string]any{"a": 1}, WithValidator(Weak("thing", "abc-123", 42)))
	})
	const etag = `W/"thing-abc-123-42"`

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		resp, body := doAndRead(t, srv, method, map[string]string{"If-None-Match": etag})
		if resp.StatusCode != http.StatusNotModified {
			t.Fatalf("%s status = %d, want 304", method, resp.StatusCode)
		}
		if len(body) != 0 {
			t.Errorf("%s body = %q, want empty", method, body)
		}
		mustHave(t, resp, "ETag", etag)
		mustHave(t, resp, "Cache-Control", "private, no-cache")
		if resp.Header.Get("Content-Type") != "" {
			t.Errorf("%s Content-Type = %q, want removed on 304", method, resp.Header.Get("Content-Type"))
		}
		if resp.Header.Get("Content-Length") != "" {
			t.Errorf("%s Content-Length = %q, want removed on 304", method, resp.Header.Get("Content-Length"))
		}
	}
}

// If-None-Match miss: a normal 200 with the full body and ETag.
func TestIfNoneMatchMiss200(t *testing.T) {
	srv := newTestServer(t, func(c *gin.Context) {
		httpresp.JSON(c, http.StatusOK, map[string]any{"a": 1}, WithValidator(Weak("thing", "abc-123", 42)))
	})

	resp, body := doAndRead(t, srv, http.MethodGet, map[string]string{"If-None-Match": `W/"thing-abc-123-99"`})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if string(body) != `{"a":1}` {
		t.Errorf("body = %q, want %q", body, `{"a":1}`)
	}
	mustHave(t, resp, "ETag", `W/"thing-abc-123-42"`)
}

// Non-cacheable method: POST must not produce ETag or cache headers.
func TestPostNoETag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(Middleware())
	r.POST("/thing", func(c *gin.Context) {
		httpresp.JSON(c, http.StatusOK, map[string]any{"a": 1}, WithValidator(Weak("thing", "abc-123", 42)))
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/thing", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if resp.Header.Get("ETag") != "" {
		t.Errorf("ETag = %q, want empty for POST", resp.Header.Get("ETag"))
	}
	if resp.Header.Get("Cache-Control") != "" {
		t.Errorf("Cache-Control = %q, want empty for POST", resp.Header.Get("Cache-Control"))
	}
}

// Non-200 status: no ETag even with a validator declared.
func TestNon200NoETag(t *testing.T) {
	srv := newTestServer(t, func(c *gin.Context) {
		httpresp.JSON(c, http.StatusCreated, map[string]any{"a": 1}, WithValidator(Weak("thing", "abc-123", 42)))
	})

	resp, body := doAndRead(t, srv, http.MethodGet, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if string(body) != `{"a":1}` {
		t.Errorf("body = %q, want %q", body, `{"a":1}`)
	}
	if resp.Header.Get("ETag") != "" {
		t.Errorf("ETag = %q, want empty for non-200", resp.Header.Get("ETag"))
	}
	if resp.Header.Get("Cache-Control") != "" {
		t.Errorf("Cache-Control = %q, want empty for non-200", resp.Header.Get("Cache-Control"))
	}
}

// No validator declared: 200 response passes through without ETag or cache
// headers.
func TestNoValidatorNoETag(t *testing.T) {
	srv := newTestServer(t, func(c *gin.Context) {
		httpresp.JSON(c, http.StatusOK, map[string]any{"a": 1})
	})

	resp, body := doAndRead(t, srv, http.MethodGet, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if string(body) != `{"a":1}` {
		t.Errorf("body = %q, want %q", body, `{"a":1}`)
	}
	if resp.Header.Get("ETag") != "" {
		t.Errorf("ETag = %q, want empty", resp.Header.Get("ETag"))
	}
	if resp.Header.Get("Cache-Control") != "" {
		t.Errorf("Cache-Control = %q, want empty", resp.Header.Get("Cache-Control"))
	}
}

// A panicking handler must not leak the buffered partial response: the body
// written before the panic is discarded and the Recovery middleware's bare 500
// goes straight to the client.
func TestPanicRecovery(t *testing.T) {
	srv := newTestServer(t, func(c *gin.Context) {
		httpresp.JSON(c, http.StatusOK, map[string]any{"a": 1})
		panic("boom")
	})

	resp, body := doAndRead(t, srv, http.MethodGet, nil)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	if len(body) != 0 {
		t.Errorf("body = %q, want empty (partial response must be discarded)", body)
	}
}
