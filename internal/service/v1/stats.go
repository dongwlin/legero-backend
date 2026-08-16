package v1

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/repo"
	"github.com/dongwlin/legero-backend/internal/service"
)

// stats implements service.Stats.
type stats struct {
	db       *bun.DB
	timezone string
}

// NewStats creates a new Stats service.
func NewStats(db *bun.DB, timezone string) service.Stats {
	return &stats{
		db:       db,
		timezone: timezone,
	}
}

// Daily returns per-day order counts and revenue for a workspace within a date range.
func (s *stats) Daily(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]model.DailyRow, error) {
	if to.Before(from) {
		return nil, httpresp.ValidationError("to must be greater than or equal to from")
	}

	statsRepo := repo.NewStats(s.db)
	rows, err := statsRepo.Daily(ctx, workspaceID, s.timezone, from, to)
	if err != nil {
		return nil, httpresp.InternalError("failed to load daily stats", err)
	}
	return rows, nil
}
