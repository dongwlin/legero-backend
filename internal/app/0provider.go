package app

import "github.com/google/wire"

var Provider = wire.NewSet(
	New,
)
