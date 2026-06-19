package app

import (
	"github.com/dongwlin/legero-backend/internal/infra/crypto"
	"github.com/dongwlin/legero-backend/internal/service"
)

// Services holds all wired business services.
type Services struct {
	Auth  *service.Auth
	Order *service.Order
	Stats *service.Stats
}

func newServices(infra *Infra) (*Services, error) {
	orderSvc := service.NewOrder(infra.DB, infra.Location, infra.Broker)

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
		Auth:  authSvc,
		Order: orderSvc,
		Stats: statsSvc,
	}, nil
}
