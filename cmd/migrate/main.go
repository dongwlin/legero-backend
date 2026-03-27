package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/dongwlin/legero-backend/internal/infra/cli"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	dbpkg "github.com/dongwlin/legero-backend/internal/infra/db"
)

func main() {
	cli.Run(run)
}

func run(ctx context.Context, cfg *config.Config, _ zerolog.Logger, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing migration subcommand: use up, down, status")
	}

	database, err := dbpkg.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		_ = database.Close()
	}()

	switch args[0] {
	case "up":
		return dbpkg.MigrateUp(ctx, database)
	case "down":
		return dbpkg.MigrateDown(ctx, database)
	case "status":
		rows, err := dbpkg.MigrationStatus(ctx, database)
		if err != nil {
			return err
		}
		for _, row := range rows {
			fmt.Printf("%s migrated=%t group=%d\n", row.Name, row.Migrated, row.GroupID)
		}
		return nil
	default:
		return fmt.Errorf("unknown migration subcommand: %s", args[0])
	}
}
