package httpserver

import (
	"github.com/dongwlin/legero-backend/internal/handler"
	"github.com/google/wire"
)

var Provider = wire.NewSet(
	handler.Provider,
	NewHttpServer,
)
