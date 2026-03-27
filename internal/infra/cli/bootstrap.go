package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/logger"
)

// Runner executes a CLI entrypoint with initialized process dependencies.
type Runner func(context.Context, *config.Config, zerolog.Logger, []string) error

// Run bootstraps a CLI entrypoint with signal handling, logging, and config.
func Run(run Runner) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	appLogger := logger.New()

	cfg, err := config.Load()
	if err != nil {
		appLogger.Fatal().Err(fmt.Errorf("load config: %w", err)).Msg("application exited")
	}

	if err := run(ctx, cfg, appLogger, os.Args[1:]); err != nil {
		appLogger.Fatal().Err(err).Msg("application exited")
	}
}
