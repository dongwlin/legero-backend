package logic

import "github.com/google/wire"

var Provider = wire.NewSet(
	NewAuth,
	NewUser,
	NewOrderItem,
)
