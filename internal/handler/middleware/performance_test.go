package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/legero-backend/internal/infra/performance"
)

const testPerformanceID = "a13af206-010e-4141-9ae5-235ff0adeac0:17"

func TestPerformanceTimingWritesAtHeaderBoundary(t *testing.T) {
	for _, mode := range []string{"json", "string", "bytes", "headers", "flush", "empty", "error"} {
		t.Run(mode, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			now := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
			router := gin.New()
			router.Use(performanceTiming(func() time.Time { return now }))
			router.GET("/test", func(c *gin.Context) {
				trace := performance.FromContext(c.Request.Context())
				require.NotNil(t, trace)
				trace.RecordQuery(3 * time.Millisecond)
				trace.RecordQuery(4 * time.Millisecond)
				now = now.Add(12 * time.Millisecond)
				switch mode {
				case "json":
					c.JSON(http.StatusOK, gin.H{"status": "ok"})
				case "string":
					_, _ = c.Writer.WriteString("same body")
				case "bytes":
					_, _ = c.Writer.Write([]byte("same body"))
				case "headers":
					c.Writer.WriteHeader(http.StatusNoContent)
					c.Writer.WriteHeaderNow()
				case "flush":
					c.Writer.Flush()
				case "empty":
					c.Status(http.StatusNoContent)
				case "error":
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "same error"})
				}
				if mode != "empty" {
					now = now.Add(time.Second)
					trace.RecordQuery(time.Second)
				}
			})
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			request.Header.Set(PerformanceRequestIDHeader, testPerformanceID)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, testPerformanceID, response.Result().Header.Get(PerformanceRequestIDHeader))
			require.Equal(t, `app;dur=12.000, db;dur=7.000;desc="queries:2"`, response.Result().Header.Get("Server-Timing"))
			if mode == "json" {
				assertJSON(t, response.Body.String(), `{"status":"ok"}`)
			}
			if mode == "error" {
				require.Equal(t, http.StatusBadRequest, response.Code)
				assertJSON(t, response.Body.String(), `{"error":"same error"}`)
			}
		})
	}
}

func assertJSON(t *testing.T, actual, expected string) {
	t.Helper()
	require.JSONEq(t, expected, actual)
}

func TestPerformanceTimingSkipsMissingInvalidDuplicateAndUpgradeIDs(t *testing.T) {
	for _, id := range []string{"", "13800000001:1", "local-short:1", "phone:13800000001", "secret credentials", "abcd1234:0", "abcd1234:01", "abcd1234:1000000000", strings.Repeat("x", 65) + ":1", "abcd1234:1,abcd1234:2", "abcd1234:1;token=secret"} {
		t.Run(id, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(PerformanceTiming())
			router.GET("/test", func(c *gin.Context) {
				require.Nil(t, performance.FromContext(c.Request.Context()))
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			if id != "" {
				request.Header.Set(PerformanceRequestIDHeader, id)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Empty(t, response.Header().Get(PerformanceRequestIDHeader))
			require.Empty(t, response.Header().Get("Server-Timing"))
		})
	}
	for _, variant := range []string{"duplicate", "upgrade"} {
		t.Run(variant, func(t *testing.T) {
			router := gin.New()
			router.Use(PerformanceTiming())
			router.GET("/test", func(c *gin.Context) {
				require.Nil(t, performance.FromContext(c.Request.Context()))
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			request.Header.Set(PerformanceRequestIDHeader, testPerformanceID)
			if variant == "duplicate" {
				request.Header.Add(PerformanceRequestIDHeader, "another-session:2")
			} else {
				request.Header.Set("Upgrade", "websocket")
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Empty(t, response.Header().Get("Server-Timing"))
		})
	}
}

func TestPerformanceTimingAcceptsGeneratedFallbackSessionID(t *testing.T) {
	require.True(t, performanceRequestIDPattern.MatchString("local-a012d51254abcdef:1"))
}

func TestPerformanceTimingSurvivesRecoveryAndDoesNotExposePprof(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(PerformanceTiming(), gin.CustomRecovery(func(c *gin.Context, _ any) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}))
	router.GET("/panic", func(_ *gin.Context) { panic("synthetic failure") })
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	request.Header.Set(PerformanceRequestIDHeader, testPerformanceID)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Contains(t, response.Header().Get("Server-Timing"), "app;dur=")
	require.NotContains(t, response.Header().Get("Server-Timing"), "synthetic")
	pprofResponse := httptest.NewRecorder()
	router.ServeHTTP(pprofResponse, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	require.Equal(t, http.StatusNotFound, pprofResponse.Code)
}

func TestCORSAllowsAndExposesPerformanceHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS(), PerformanceTiming())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	preflight := httptest.NewRequest(http.MethodOptions, "/test", nil)
	preflight.Header.Set("Origin", "https://frontend.example")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflight.Header.Set("Access-Control-Request-Headers", PerformanceRequestIDHeader)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, preflight)
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Contains(t, response.Header().Get("Access-Control-Allow-Headers"), "X-Performance-Request-Id")
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set("Origin", "https://frontend.example")
	request.Header.Set(PerformanceRequestIDHeader, testPerformanceID)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Contains(t, response.Header().Get("Access-Control-Expose-Headers"), "X-Performance-Request-Id")
	require.Contains(t, response.Header().Get("Access-Control-Expose-Headers"), "Server-Timing")
}
