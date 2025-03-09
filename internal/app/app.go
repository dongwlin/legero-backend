package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dongwlin/legero-backend/internal/config"
	"github.com/dongwlin/legero-backend/internal/ent"
	"github.com/dongwlin/legero-backend/internal/server/httpserver"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type App struct {
	config *config.Config
	logger *zerolog.Logger
	db     *ent.Client
	rdb    *redis.Client

	httpsrv *httpserver.HttpServer
}

func New(
	conf *config.Config,
	logger *zerolog.Logger,
	db *ent.Client,
	rdb *redis.Client,

	httpsrv *httpserver.HttpServer,
) *App {

	return &App{
		config:  conf,
		logger:  logger,
		db:      db,
		rdb:     rdb,
		httpsrv: httpsrv,
	}
}

func (a *App) Run() {
	errChan := make(chan error, 1)
	go func() {
		if err := a.httpsrv.Run(); err != nil {
			errChan <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		a.logger.Error().Err(err).Msg("http server failed")
	case sig := <-sigChan:
		a.logger.Info().Str("signal", sig.String()).Msg("received signal")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	a.Shutdown(ctx)
}

func (a *App) Shutdown(ctx context.Context) {
	if a.httpsrv != nil {
		if err := a.httpsrv.ShutdownWithContext(ctx); err != nil {
			a.logger.Error().Err(err).Msg("failed to shutdown http server")
		}
	}

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error().Err(err).Msg("failed to close database connection")
		}
	}

	if a.rdb != nil {
		if err := a.rdb.Close(); err != nil {
			a.logger.Error().Err(err).Msg("failed to close redis connection")
		}
	}

	a.logger.Info().Msg("application shutdown completed")
}
