package service

import (
	"context"
	"time"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/google/uuid"
)

// MaxDailyStatsDays is the maximum number of inclusive business dates that a
// daily statistics query may cover.
const MaxDailyStatsDays = 366

// Stats provides aggregated order statistics.
type Stats interface {
	Daily(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]domain.DailyRow, error)
	Report(ctx context.Context, workspaceID uuid.UUID, query domain.ReportQuery) (*domain.Report, error)
}
