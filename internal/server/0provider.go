package server

import (
	"github.com/dongwlin/legero-backend/internal/server/httpserver"
	"github.com/google/wire"
)

var Provider = wire.NewSet(
	httpserver.Provider,
)
