package httpresp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type failingJSON struct{}

func (failingJSON) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal failed")
}

func TestJSONClearsMetadataWhenASecondRenderFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/resource", func(c *gin.Context) {
		JSON(c, http.StatusOK, map[string]string{"value": "first"}, func(cfg *Config) {
			cfg.Metadata.Validator = struct{}{}
		})
		JSON(c, http.StatusOK, failingJSON{})

		_, exists := c.Get(ConfigKey())
		require.False(t, exists)
		require.Nil(t, BodyFromContext(c))
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/resource", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestJSONIgnoresNilOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/resource", func(c *gin.Context) {
		JSON(c, http.StatusOK, map[string]string{"value": "ok"}, nil)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/resource", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"value":"ok"}`, recorder.Body.String())
}
