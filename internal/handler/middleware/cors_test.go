package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS())
	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "http://example.com:9999")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin to allow any origin, got %q", got)
	}
}

func TestCORSHandlesPreflightForAnyOrigin(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS())
	router.OPTIONS("/api/auth/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/auth/login", nil)
	req.Header.Set("Origin", "https://frontend.example.com:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodHead)
	req.Header.Set("Access-Control-Request-Headers", "Authorization, If-None-Match")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if got := recorder.Code; got != http.StatusNoContent {
		t.Fatalf("expected preflight request to return %d, got %d", http.StatusNoContent, got)
	}

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin to allow any origin, got %q", got)
	}
	if got := strings.ToLower(recorder.Header().Get("Access-Control-Allow-Headers")); !strings.Contains(got, "if-none-match") {
		t.Fatalf("expected preflight to allow If-None-Match, got %q", got)
	}
	if got := strings.ToLower(recorder.Header().Get("Access-Control-Allow-Headers")); !strings.Contains(got, "authorization") {
		t.Fatalf("expected preflight to allow Authorization, got %q", got)
	}
	if got := strings.ToLower(recorder.Header().Get("Access-Control-Allow-Methods")); !strings.Contains(got, strings.ToLower(http.MethodHead)) {
		t.Fatalf("expected preflight to allow HEAD, got %q", got)
	}
}

func TestCORSExposesETag(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS())
	router.GET("/resource", func(c *gin.Context) {
		c.Header("ETag", `"tag"`)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "https://frontend.example.com:3000")
	router.ServeHTTP(recorder, req)

	if got := strings.ToLower(recorder.Header().Get("Access-Control-Expose-Headers")); !strings.Contains(got, "etag") {
		t.Fatalf("expected ETag to be exposed, got %q", got)
	}
}
