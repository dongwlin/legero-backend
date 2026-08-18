package v1

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/repo"
	"github.com/dongwlin/legero-backend/internal/service"
)

// stats implements service.Stats.
type stats struct {
	db          *bun.DB
	timezone    string
	location    *time.Location
	locationErr error
}

// NewStats creates a new Stats service.
func NewStats(db *bun.DB, timezone string) service.Stats {
	location, err := time.LoadLocation(timezone)
	return &stats{
		db:          db,
		timezone:    timezone,
		location:    location,
		locationErr: err,
	}
}

// Daily returns per-day completed-order counts and revenue for a workspace
// within an inclusive business-date range.
func (s *stats) Daily(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]domain.DailyRow, error) {
	if s.locationErr != nil {
		return nil, apperr.InternalError("failed to load business timezone", s.locationErr)
	}
	window, err := domain.NewDailyReportWindow(from, to, s.location)
	if err != nil {
		return nil, apperr.ValidationError(err.Error())
	}

	statsRepo := repo.NewStats(s.db)
	rows, err := statsRepo.DailyWindow(ctx, workspaceID, s.timezone, window)
	if err != nil {
		return nil, apperr.InternalError("failed to load daily stats", err)
	}
	return rows, nil
}

// Report returns the currently supported period report. The query shape is
// intentionally period-based even though M1 only enables a business day, so
// future week/month implementations can reuse the same service and API
// contract.
func (s *stats) Report(ctx context.Context, workspaceID uuid.UUID, query domain.ReportQuery) (*domain.Report, error) {
	if s.locationErr != nil {
		return nil, apperr.InternalError("failed to load business timezone", s.locationErr)
	}
	if !query.Period.Valid() {
		return nil, apperr.ValidationError("period must be one of day, week, month")
	}
	if query.Period != domain.ReportPeriodDay {
		return nil, apperr.Wrap(
			apperr.KindInvalidArgument,
			"report_period_unsupported",
			"the requested report period is not supported",
			domain.ErrUnsupportedReportPeriod,
		)
	}
	if query.Date.IsZero() {
		return nil, apperr.ValidationError("date must use YYYY-MM-DD")
	}

	window := domain.NewDayReportWindow(query.Date, s.location)
	orders, err := repo.NewStats(s.db).CompletedOrders(ctx, workspaceID, window.StartAt, window.EndAt)
	if err != nil {
		return nil, apperr.InternalError("failed to load report orders", err)
	}
	report := domain.AggregateReport(window, orders, s.location)
	return &report, nil
}
