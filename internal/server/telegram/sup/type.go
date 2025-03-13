package sup

import (
	"context"
	"project/internal/models"
)

type Handler struct {
	db  Storage
	cdb Cache
	ac  AppCache
}

type Storage interface {
	InsertUsers(ctx context.Context, users []models.User) error
}

type Cache interface {
	Set(ctx context.Context, key string, value string) error
}

type AppCache interface {
	GetFromMap(name string, key string) (any, bool)
}

func NewHandler(db Storage, cdb Cache, ac AppCache) *Handler {
	return &Handler{
		db:  db,
		cdb: cdb,
		ac:  ac,
	}
}
