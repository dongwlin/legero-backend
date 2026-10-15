package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/dongwlin/legero-backend/internal/infra/database"
)

// The container is isolated from the configured development/store database.
func TestPerformanceTimingPostgresHTTPRoundTrip(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	container, err := postgres.Run(ctx, "postgres:18",
		postgres.WithDatabase("performance_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(15*time.Second)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable", "timezone=UTC")
	require.NoError(t, err)
	db, err := database.New(ctx, database.Options{DSN: dsn})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS(), PerformanceTiming())
	router.GET("/controlled-query", func(c *gin.Context) {
		// Only synthetic SQL; no schema, seed data, or production mutation.
		_, err := db.ExecContext(c.Request.Context(), "SELECT pg_sleep(0.025)")
		if err != nil {
			t.Errorf("controlled query: %v", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := &http.Client{Timeout: 5 * time.Second}
	for _, id := range []string{"", testPerformanceID, "local-controlsession:2"} {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/controlled-query", nil)
		require.NoError(t, err)
		request.Header.Set("Origin", "https://frontend.example")
		if id != "" {
			request.Header.Set(PerformanceRequestIDHeader, id)
		}
		started := time.Now()
		response, err := client.Do(request)
		require.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		clientRoundTripMS := float64(time.Since(started)) / float64(time.Millisecond)
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.JSONEq(t, `{"status":"ok"}`, string(body))
		header := response.Header.Get("Server-Timing")
		if id == "" {
			require.Empty(t, header)
			require.Empty(t, response.Header.Get(PerformanceRequestIDHeader))
			continue
		}
		require.Equal(t, id, response.Header.Get(PerformanceRequestIDHeader))
		parts := regexp.MustCompile(`^app;dur=([0-9.]+), db;dur=([0-9.]+);desc="queries:1"$`).FindStringSubmatch(header)
		require.Len(t, parts, 3, header)
		appMS, err := strconv.ParseFloat(parts[1], 64)
		require.NoError(t, err)
		dbMS, err := strconv.ParseFloat(parts[2], 64)
		require.NoError(t, err)
		require.GreaterOrEqual(t, dbMS, 20.0, "the controlled 25 ms query must be visible")
		require.GreaterOrEqual(t, appMS, dbMS)
		require.GreaterOrEqual(t, clientRoundTripMS, appMS)
		t.Logf("request=%s client_round_trip=%.3f ms app=%.3f ms db=%.3f ms completed_queries=1", id, clientRoundTripMS, appMS, dbMS)
	}
}
