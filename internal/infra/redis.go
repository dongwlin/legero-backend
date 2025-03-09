package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/dongwlin/legero-backend/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func NewRedis(conf *config.Config) (*redis.Client, error) {

	addr := fmt.Sprintf("%s:%d",
		conf.Redis.Host,
		conf.Redis.Port,
	)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       conf.Redis.DB,
		Username: conf.Redis.Username,
		Password: conf.Redis.Password,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	ping := rdb.Ping(ctx)
	if ping.Err() != nil {
		log.Error().
			Err(ping.Err()).
			Msg("redis connection failed")
		return nil, ping.Err()
	}

	return rdb, nil
}
