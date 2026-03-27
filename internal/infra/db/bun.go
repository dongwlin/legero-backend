package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func Open(ctx context.Context, databaseURL string) (*bun.DB, error) {
	connector := pgdriver.NewConnector(
		pgdriver.WithDSN(databaseURL),
		pgdriver.WithTimeout(5*time.Second),
	)

	sqldb := sql.OpenDB(connector)
	sqldb.SetMaxOpenConns(10)
	sqldb.SetMaxIdleConns(5)
	sqldb.SetConnMaxLifetime(30 * time.Minute)

	db := bun.NewDB(sqldb, pgdialect.New())
	if err := db.PingContext(ctx); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}
