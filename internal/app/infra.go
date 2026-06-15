package app

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/database"
	"github.com/dongwlin/legero-backend/internal/infra/logger"
	"github.com/dongwlin/legero-backend/migrations"
)

// Infra holds the common application infrastructure shared by all CLI commands.
type Infra struct {
	Config   *config.Config
	DB       *bun.DB
	Location *time.Location
	Logger   zerolog.Logger
}

// NewInfra loads configuration, runs migrations, and connects to the database.
// The returned Infra must be closed with Close() when no longer needed.
func NewInfra(ctx context.Context) (*Infra, error) {
	// Initialize global logger first, so all subsequent components use formatted output.
	appLogger := logger.New()

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	location, err := time.LoadLocation(cfg.BizTimezone)
	if err != nil {
		return nil, fmt.Errorf("load biz timezone: %w", err)
	}

	if err := migrations.Migrate(cfg.DatabaseURL); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	db, err := database.New(ctx, database.Options{DSN: cfg.DatabaseURL})
	if err != nil {
		return nil, err
	}

	return &Infra{
		Config:   cfg,
		DB:       db,
		Location: location,
		Logger:   appLogger,
	}, nil
}

// Close releases resources held by Infra (e.g., database connection pool).
func (i *Infra) Close() error {
	return i.DB.Close()
}
