package repo

import "github.com/google/wire"

var Provider = wire.NewSet(
	NewUser,
	NewToken,
	NewOrderItem,
)
