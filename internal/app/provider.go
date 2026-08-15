package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/handler"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/crypto"
	"github.com/dongwlin/legero-backend/internal/infra/database"
	"github.com/dongwlin/legero-backend/internal/infra/logger"
	"github.com/dongwlin/legero-backend/internal/infra/realtime"
	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/service"
	servicev1 "github.com/dongwlin/legero-backend/internal/service/v1"
	"github.com/dongwlin/legero-backend/migrations"
)

// ServerProviderSet assembles the full HTTP application.
var ServerProviderSet = wire.NewSet(
	config.Load,
	ProvideContext,
	ProvideLogger,
	ProvideLocation,
	ProvideDatabase,
	ProvideBroker,
	ProvideSessionManager,
	ProvideNow,
	ProvidePasswordHasher,
	ProvideTimezone,
	ProvideTokenTTL,
	ProvidePasetoKey,
	servicev1.NewOrder,
	ProvideAuth,
	servicev1.NewStats,
	wire.Bind(new(service.ActiveOrderLoader), new(service.Order)),
	wire.Bind(new(model.Publisher), new(*realtime.Broker)),
	handler.NewRouter,
	ProvideHTTPServer,
	NewApplication,
)

// UserCreatorProviderSet assembles the create-user CLI bootstrap.
var UserCreatorProviderSet = wire.NewSet(
	config.Load,
	ProvideContext,
	ProvideDatabase,
	ProvidePasswordHasher,
	servicev1.NewUser,
	NewUserCreator,
)

// ProvideLogger initializes the global zerolog logger.
func ProvideLogger() zerolog.Logger {
	return logger.New()
}

// ProvideLocation loads the business timezone.
func ProvideLocation(cfg *config.Config) (*time.Location, error) {
	location, err := time.LoadLocation(cfg.BizTimezone)
	if err != nil {
		return nil, fmt.Errorf("load biz timezone: %w", err)
	}
	return location, nil
}

// ProvideDatabase runs migrations and opens the database connection pool.
func ProvideDatabase(ctx context.Context, cfg *config.Config) (*bun.DB, func(), error) {
	if err := migrations.Migrate(cfg.DatabaseURL); err != nil {
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	db, err := database.New(ctx, database.Options{DSN: cfg.DatabaseURL})
	if err != nil {
		return nil, nil, err
	}
	return db, func() { _ = db.Close() }, nil
}

// ProvideBroker creates the realtime event broker.
func ProvideBroker() *realtime.Broker {
	return realtime.NewBroker()
}

// ProvideSessionManager creates the WebSocket session manager.
func ProvideSessionManager(cfg *config.Config) *realtime.SessionManager {
	return realtime.NewSessionManager(cfg.RealtimeSessionTTL, time.Now)
}

// ProvideNow exposes the current-time function for handlers.
func ProvideNow() func() time.Time {
	return time.Now
}

// ProvidePasswordHasher creates the Argon2 password hasher.
func ProvidePasswordHasher(cfg *config.Config) *crypto.PasswordHasher {
	return crypto.NewPasswordHasher(cfg.Argon2)
}

// ProvideContext returns the process-wide background context.
func ProvideContext() context.Context {
	return context.Background()
}

// TokenTTL carries the access/refresh token lifetimes.
type TokenTTL struct {
	Access  time.Duration
	Refresh time.Duration
}

// ProvideTokenTTL reads the token lifetimes from config.
func ProvideTokenTTL(cfg *config.Config) TokenTTL {
	return TokenTTL{
		Access:  cfg.AccessTokenTTL,
		Refresh: cfg.RefreshTokenTTL,
	}
}

// ProvidePasetoKey exposes the PASETO symmetric key bytes.
func ProvidePasetoKey(cfg *config.Config) []byte {
	return cfg.PasetoSymmetricKey
}

// ProvideAuth adapts the token TTL struct to the auth service constructor.
func ProvideAuth(
	db *bun.DB,
	orders service.ActiveOrderLoader,
	hasher *crypto.PasswordHasher,
	location *time.Location,
	ttl TokenTTL,
	keyBytes []byte,
) (service.Auth, error) {
	return servicev1.NewAuth(db, orders, hasher, location, ttl.Access, ttl.Refresh, keyBytes)
}

// ProvideTimezone exposes the business timezone name for services.
func ProvideTimezone(cfg *config.Config) string {
	return cfg.BizTimezone
}

// ProvideHTTPServer builds the HTTP server around the gin engine.
func ProvideHTTPServer(cfg *config.Config, engine *gin.Engine) *http.Server {
	return &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
