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
