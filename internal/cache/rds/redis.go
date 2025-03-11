package rds

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"project/internal/cache"
	"project/pkg/e"
)

func (c *Cache) MSet(ctx context.Context, args ...interface{}) error {
	const fn = "cache.redis.MSet"

	if err := c.rdb.MSet(ctx, args...).Err(); err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (c *Cache) Set(ctx context.Context, key string, value string) error {
	const fn = "cache.redis.Set"

	err := c.rdb.Set(ctx, key, value, 0).Err()
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	const fn = "cache.redis.Get"

	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", e.Wrap(fn, cache.ErrKeyNotFound)
		}

		return "", e.Wrap(fn, err)
	}

	return val, nil
}

func (c *Cache) Del(ctx context.Context, key string) error {
	const fn = "cache.redis.Del"

	res, err := c.rdb.Del(ctx, key).Result()
	if err != nil {
		return e.Wrap(fn, err)
	}

	if res == 0 {
		return cache.ErrKeyNotFound
	}

	return nil
}

func (c *Cache) SAdd(ctx context.Context, k, v string) error {
	const fn = "cache.redis.SAdd"

	err := c.rdb.SAdd(ctx, k, v).Err()
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (c *Cache) SRem(ctx context.Context, k, v string) error {
	const fn = "cache.redis.SRem"

	err := c.rdb.SRem(ctx, k, v).Err()
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (c *Cache) SIsMember(ctx context.Context, k, v string) (bool, error) {
	const fn = "cache.redis.SIsMember"

	exists, err := c.rdb.SIsMember(ctx, k, v).Result()
	if err != nil {
		return false, e.Wrap(fn, err)
	}

	return exists, nil
}

func (c *Cache) Ping(ctx context.Context) error {
	const fn = "cache.redis.Ping"

	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}
