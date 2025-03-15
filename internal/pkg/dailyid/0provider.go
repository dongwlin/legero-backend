package dailyid

import "github.com/google/wire"

var Provider = wire.NewSet(
	New,
)
