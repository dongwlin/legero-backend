package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/legero-backend/internal/infra/config"
)

func TestHealthzExcludesCacheAndConditionalHeaders(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, nil, time.UTC, &config.Config{}, zerolog.Nop(), time.Now)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotEmpty(t, recorder.Body.Bytes())
	require.Empty(t, recorder.Header().Get("ETag"))
	require.Empty(t, recorder.Header().Get("Cache-Control"))
	require.Empty(t, recorder.Header().Get("Vary"))

	recorder = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("If-None-Match", `"any-etag-value"`)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Header().Get("ETag"))
}

func TestHealthzDoesNotServeHead(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, nil, time.UTC, &config.Config{}, zerolog.Nop(), time.Now)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/healthz", nil))

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Empty(t, recorder.Header().Get("ETag"))
}
