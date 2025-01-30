package rds

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log/slog"
	"project/internal/config"
	"project/pkg/e"
)

type Cache struct {
	rdb *redis.Client
	log *slog.Logger
}

func New(ctx context.Context, cfg *config.Redis, log *slog.Logger) (*Cache, error) {
	const fn = "cache.redis.New"

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	log.Info("[OK] redis successfully connected")

	return &Cache{
		rdb: rdb,
		log: log,
	}, nil
}
