package v1

import (
	"context"
	"testing"
	"time"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStatsServiceDailyDateRangeLimit(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	require.Equal(t, 366, service.MaxDailyStatsDays)

	from := time.Date(2026, 1, 1, 12, 0, 0, 0, location)

	t.Run("max allowed inclusive business dates succeeds", func(t *testing.T) {
		svc := NewStats(testDB, location.String())
		to := from.AddDate(0, 0, service.MaxDailyStatsDays-1)

		rows, err := svc.Daily(context.Background(), uuid.New(), from, to)

		require.NoError(t, err)
		require.Len(t, rows, service.MaxDailyStatsDays)
	})

	t.Run("max plus one is rejected before repository access", func(t *testing.T) {
		// A nil DB makes an accidental repository call panic, so this verifies
		// that validation happens before repo.NewStats(...).DailyWindow(...).
		svc := NewStats(nil, location.String())
		to := from.AddDate(0, 0, service.MaxDailyStatsDays)

		_, err := svc.Daily(context.Background(), uuid.New(), from, to)

		assertDailyStatsValidationError(t, err, maxDailyStatsRangeMessage)
	})

	t.Run("from after to is rejected before repository access", func(t *testing.T) {
		// Keep this path database-free for the same no-repository-call guarantee.
		svc := NewStats(nil, location.String())
		to := from.AddDate(0, 0, -1)

		_, err := svc.Daily(context.Background(), uuid.New(), from, to)

		assertDailyStatsValidationError(t, err, "to must be greater than or equal to from")
	})

	t.Run("same business date succeeds", func(t *testing.T) {
		svc := NewStats(testDB, location.String())

		rows, err := svc.Daily(context.Background(), uuid.New(), from, from)

		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, from.Format("2006-01-02"), rows[0].Date.In(location).Format("2006-01-02"))
	})
}

func assertDailyStatsValidationError(t *testing.T, err error, message string) {
	t.Helper()

	var appErr *apperr.AppError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.KindInvalidArgument, appErr.Kind)
	require.Equal(t, "validation_failed", appErr.Code)
	require.Equal(t, message, appErr.Message)
}
