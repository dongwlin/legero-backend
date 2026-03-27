package db

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

func NewMigrator(database *bun.DB) *migrate.Migrator {
	return migrate.NewMigrator(database, Migrations)
}

func MigrateUp(ctx context.Context, database *bun.DB) error {
	migrator := NewMigrator(database)
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	group, err := migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	if group.IsZero() {
		return nil
	}
	return nil
}

func MigrateDown(ctx context.Context, database *bun.DB) error {
	migrator := NewMigrator(database)
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	if _, err := migrator.Rollback(ctx); err != nil {
		return fmt.Errorf("rollback migrations: %w", err)
	}
	return nil
}

func MigrationStatus(ctx context.Context, database *bun.DB) ([]MigrationStatusRow, error) {
	migrator := NewMigrator(database)
	if err := migrator.Init(ctx); err != nil {
		return nil, fmt.Errorf("init migrator: %w", err)
	}

	ms, err := migrator.MigrationsWithStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}

	rows := make([]MigrationStatusRow, 0, len(ms))
	for _, item := range ms {
		rows = append(rows, MigrationStatusRow{
			Name:     item.Name,
			GroupID:  item.GroupID,
			Migrated: item.IsApplied(),
		})
	}
	return rows, nil
}

type MigrationStatusRow struct {
	Name     string
	GroupID  int64
	Migrated bool
}
