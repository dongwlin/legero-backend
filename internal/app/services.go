package app

import (
	"github.com/dongwlin/legero-backend/internal/infra/crypto"
	"github.com/dongwlin/legero-backend/internal/service"
	servicev1 "github.com/dongwlin/legero-backend/internal/service/v1"
)

// Services holds all wired business services.
type Services struct {
	Auth  service.Auth
	Order service.Order
	Stats service.Stats
}

func newServices(infra *Infra) (*Services, error) {
	orderSvc := servicev1.NewOrder(infra.DB, infra.Location, infra.Broker)

	authSvc, err := servicev1.NewAuth(
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

	statsSvc := servicev1.NewStats(infra.DB, infra.Config.BizTimezone)

	return &Services{
		Auth:  authSvc,
		Order: orderSvc,
		Stats: statsSvc,
	}, nil
}
