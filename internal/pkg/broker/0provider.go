package broker

import "github.com/google/wire"

var Provider = wire.NewSet(
	NewBroker,
)
