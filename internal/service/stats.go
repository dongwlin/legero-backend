package service

import (
	"context"
	"time"

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/google/uuid"
)

// Stats provides aggregated order statistics.
type Stats interface {
	Daily(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]domain.DailyRow, error)
}
