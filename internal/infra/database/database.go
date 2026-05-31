package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// Options holds configuration for database connection.
type Options struct {
	DSN string `mapstructure:"dsn"`
}

// New creates a new bun.DB instance using pgx driver.
func New(ctx context.Context, opts Options) (*bun.DB, error) {
	poolConfig, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	sqldb := stdlib.OpenDBFromPool(pool)
	db := bun.NewDB(sqldb, pgdialect.New())

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		_ = sqldb.Close()
		return nil, err
	}

	return db, nil
}