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

func TestHealthzHasETagWithoutPrivateCachePolicy(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, nil, time.UTC, &config.Config{}, zerolog.Nop(), time.Now)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(method, "/healthz", nil))

			require.Equal(t, http.StatusOK, recorder.Code)
			if method == http.MethodHead {
				require.Empty(t, recorder.Body.Bytes())
			}
			require.NotEmpty(t, recorder.Header().Get("ETag"))
			require.Empty(t, recorder.Header().Get("Cache-Control"))
			require.Empty(t, recorder.Header().Get("Vary"))
		})
	}
}
