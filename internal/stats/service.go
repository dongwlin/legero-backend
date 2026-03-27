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
	repo     Repository
	timezone string
}

func NewService(database *bun.DB, repo Repository, timezone string) *Service {
	return &Service{
		db:       database,
		repo:     repo,
		timezone: timezone,
	}
}

func (s *Service) Daily(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]DailyRow, error) {
	if to.Before(from) {
		return nil, httpx.ValidationError("to must be greater than or equal to from")
	}
	rows, err := s.repo.Daily(ctx, s.db, workspaceID, s.timezone, from, to)
	if err != nil {
		return nil, httpx.InternalError("failed to load daily stats", err)
	}
	return rows, nil
}
