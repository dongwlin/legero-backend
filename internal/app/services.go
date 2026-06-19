package app

import (
	"time"

	"github.com/dongwlin/legero-backend/internal/infra/crypto"
	"github.com/dongwlin/legero-backend/internal/realtime"
	"github.com/dongwlin/legero-backend/internal/service"
)

// Services holds all wired business Services.
// Internal realtime components (broker, session manager) are unexported —
// they are consumed by handlers in app.New().
type Services struct {
	Auth  *service.Auth
	Order *service.Order
	Stats *service.Stats

	broker     *realtime.Broker
	sessionMgr *realtime.SessionManager
}

func newServices(infra *Infra) (*Services, error) {
	broker := realtime.NewBroker()
	sessionMgr := realtime.NewSessionManager(infra.Config.RealtimeSessionTTL, time.Now)

	orderSvc := service.NewOrder(infra.DB, infra.Location, broker)

	authSvc, err := service.NewAuth(
		infra.DB,
		orderSvc,
		crypto.NewPasswordHasher(infra.Config.Argon2),
		infra.Location,
		infra.Config.AccessTokenTTL,
		infra.Config.RefreshTokenTTL,
		infra.Config.PasetoSymmetricKey,
	)
	if err != nil {
		return nil, err
	}

	statsSvc := service.NewStats(infra.DB, infra.Config.BizTimezone)

	return &Services{
		Auth:       authSvc,
		Order:      orderSvc,
		Stats:      statsSvc,
		broker:     broker,
		sessionMgr: sessionMgr,
	}, nil
}
