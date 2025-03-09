//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/dongwlin/legero-backend/internal/app"
	"github.com/dongwlin/legero-backend/internal/config"
	"github.com/dongwlin/legero-backend/internal/infra"
	"github.com/dongwlin/legero-backend/internal/logic"
	"github.com/dongwlin/legero-backend/internal/pkg/logger"
	"github.com/dongwlin/legero-backend/internal/repo"
	"github.com/dongwlin/legero-backend/internal/server"
	"github.com/google/wire"
)

var GalbolSet = wire.NewSet(
	config.Provider,
	logger.Provider,
	infra.Provider,
	repo.Provider,
	logic.Provider,
)

func InitApp() (*app.App, error) {
	panic(wire.Build(
		GalbolSet,
		server.Provider,
		app.Provider,
	))
}
