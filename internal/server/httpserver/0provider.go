package httpserver

import (
	"github.com/dongwlin/legero-backend/internal/handler"
	"github.com/dongwlin/legero-backend/internal/pkg/broker"
	"github.com/google/wire"
)

var Provider = wire.NewSet(
	broker.Provider,
	handler.Provider,
	NewHttpServer,
)
