package cache

import (
	"context"
	"errors"
)

type Cache interface {
	MSet(ctx context.Context, args ...interface{}) error
	Set(ctx context.Context, key string, value string) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	Ping(ctx context.Context) error
}

var (
	ErrKeyNotFound = errors.New("key not found")
)
