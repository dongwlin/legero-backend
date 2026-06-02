package stats

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
)

type DailyRow struct {
	Date            time.Time
	OrderCount      int
	TotalPriceCents int
}

type Service struct {
	db       *bun.DB
	timezone string
}

func NewService(db *bun.DB, timezone string) *Service {
	return &Service{
		db:       db,
		timezone: timezone,
	}
}

func (s *Service) Daily(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]DailyRow, error) {
	if to.Before(from) {
		return nil, httpx.ValidationError("to must be greater than or equal to from")
	}

	repo := NewStatsRepo(s.db)
	rows, err := repo.Daily(ctx, workspaceID, s.timezone, from, to)
	if err != nil {
		return nil, httpx.InternalError("failed to load daily stats", err)
	}
	return rows, nil
}
