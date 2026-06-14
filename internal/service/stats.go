package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/repo"
)

// Stats provides aggregated order statistics.
type Stats struct {
	db       *bun.DB
	timezone string
}

// NewStats creates a new StatsService.
func NewStats(db *bun.DB, timezone string) *Stats {
	return &Stats{
		db:       db,
		timezone: timezone,
	}
}

// Daily returns per-day order counts and revenue for a workspace within a date range.
func (s *Stats) Daily(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]model.DailyRow, error) {
	if to.Before(from) {
		return nil, httpx.ValidationError("to must be greater than or equal to from")
	}

	statsRepo := repo.NewStats(s.db)
	rows, err := statsRepo.Daily(ctx, workspaceID, s.timezone, from, to)
	if err != nil {
		return nil, httpx.InternalError("failed to load daily stats", err)
	}
	return rows, nil
}
