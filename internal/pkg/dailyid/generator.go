package dailyid

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type DailyIDGenerator struct {
	rdb     *redis.Client
	keyBase string
}

func New(rdb *redis.Client) *DailyIDGenerator {
	return &DailyIDGenerator{
		rdb:     rdb,
		keyBase: "dailyid:",
	}
}

func (g *DailyIDGenerator) NextID(ctx context.Context) (int64, error) {

	today := time.Now().Format("20060102")
	key := fmt.Sprintf("%s%s", g.keyBase, today)

	id, err := g.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	if id == 1 {
		expiredAt := time.Now().Add(24 * time.Hour)
		g.rdb.ExpireAt(ctx, key, expiredAt)
	}

	return id, nil
}
