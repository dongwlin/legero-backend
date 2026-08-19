package v1

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/legero-backend/internal/infra/config"
)

func TestRegisterRoutesDoesNotRegisterBootstrapHEAD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, nil, nil, nil, nil, nil, time.UTC, &config.Config{}, time.Now)

	methodsByPath := make(map[string]map[string]struct{})
	for _, route := range router.Routes() {
		if methodsByPath[route.Path] == nil {
			methodsByPath[route.Path] = make(map[string]struct{})
		}
		methodsByPath[route.Path][route.Method] = struct{}{}
	}

	_, hasBootstrapHEAD := methodsByPath["/bootstrap"][http.MethodHead]
	require.False(t, hasBootstrapHEAD)
	_, hasBootstrapGET := methodsByPath["/bootstrap"][http.MethodGet]
	require.True(t, hasBootstrapGET)

	for _, path := range []string{"/orders", "/stats/daily", "/stats/report"} {
		_, hasHEAD := methodsByPath[path][http.MethodHead]
		require.True(t, hasHEAD, path)
	}
}
