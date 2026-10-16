package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
	"github.com/dongwlin/legero-backend/internal/handler/middleware"
	"github.com/dongwlin/legero-backend/internal/infra/config"
)

func TestRouterWiresOptInTimingAndKeepsHealthBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(nil, nil, nil, nil, nil, time.UTC, &config.Config{}, zerolog.New(io.Discard), time.Now)
	for _, id := range []string{"", "local-routercheck:1"} {
		request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		if id != "" {
			request.Header.Set(middleware.PerformanceRequestIDHeader, id)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code)
		require.JSONEq(t, `{"status":"ok"}`, response.Body.String())
		require.Equal(t, id, response.Header().Get(middleware.PerformanceRequestIDHeader))
		if id == "" {
			require.Empty(t, response.Header().Get("Server-Timing"))
		} else {
			require.Regexp(t, `^app;dur=[0-9.]+, db;dur=0.000;desc="queries:0"$`, response.Header().Get("Server-Timing"))
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	require.Equal(t, http.StatusNotFound, response.Code)
}

func newResponseContractTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	errorWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = io.Discard
	t.Cleanup(func() { gin.DefaultErrorWriter = errorWriter })
	return NewRouter(nil, nil, nil, nil, nil, time.UTC, &config.Config{}, zerolog.New(io.Discard), time.Now)
}

func TestRouterNotFoundResponseContract(t *testing.T) {
	router := newResponseContractTestRouter(t)
	for _, test := range []struct {
		method string
		path   string
		v2     bool
	}{
		{http.MethodGet, "/api/v2", true},
		{http.MethodGet, "/api/v2/missing", true},
		{http.MethodGet, "/api/v2/auth/login", true},
		{http.MethodPatch, "/api/v2/orders/7039d698-c186-49ce-8501-394b828acf90", true},
		{http.MethodGet, "/api/missing", false},
		{http.MethodPatch, "/api/orders/7039d698-c186-49ce-8501-394b828acf90", false},
		{http.MethodGet, "/api/v20/missing", false},
		{http.MethodGet, "/healthz/missing", false},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set(middleware.PerformanceRequestIDHeader, "local-routecheck:1")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, http.StatusNotFound, response.Code)
			if test.v2 {
				require.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
				require.JSONEq(t, `{"code":"route_not_found","message":"Not Found","data":null}`, response.Body.String())
			} else {
				require.Equal(t, "text/plain", response.Header().Get("Content-Type"))
				require.Equal(t, "404 page not found", response.Body.String())
			}
			require.Equal(t, "local-routecheck:1", response.Header().Get(middleware.PerformanceRequestIDHeader))
			require.NotEmpty(t, response.Header().Get("Server-Timing"))
		})
	}
}

func TestRouterRecoveryResponseContract(t *testing.T) {
	router := newResponseContractTestRouter(t)
	for _, path := range []string{"/api/v2/panic", "/api/panic", "/api/v20/panic"} {
		router.GET(path, func(*gin.Context) { panic("private service failure") })
	}
	for _, path := range []string{"/api/v2/panic", "/api/panic", "/api/v20/panic"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set(middleware.PerformanceRequestIDHeader, "local-routecheck:2")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, http.StatusInternalServerError, response.Code)
			if path == "/api/v2/panic" {
				require.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
				require.JSONEq(t, `{"code":"internal_error","message":"Internal Server Error","data":null}`, response.Body.String())
			} else {
				require.Empty(t, response.Body.String())
				require.Empty(t, response.Header().Get("Content-Type"))
			}
			require.Equal(t, "local-routecheck:2", response.Header().Get(middleware.PerformanceRequestIDHeader))
			require.NotEmpty(t, response.Header().Get("Server-Timing"))
		})
	}
}

func TestRouterRecoveryDoesNotAppendToWrittenResponse(t *testing.T) {
	router := newResponseContractTestRouter(t)
	router.GET("/api/v2/panic-after-write", func(c *gin.Context) {
		httpresp.JSON(c, http.StatusOK, httpresp.Success(nil))
		panic("private service failure")
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v2/panic-after-write", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"code":"0","message":"success","data":null}`, response.Body.String())
}
