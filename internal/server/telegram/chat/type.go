package chat

import (
	"context"
	"log/slog"
	"project/internal/clients/tg_bot"
	"project/internal/clients/vk"
	"project/internal/models"
	"time"
)

const (
	secretCodeMap = models.GetPermissionSecretCodeMapName
)

type Handler struct {
	tg  *tg_bot.Client
	vk  *vk.Handler
	db  Storage
	cdb Cache
	ac  AppCache
	log *slog.Logger
}

type Storage interface {
	GetTgGroups(ctx context.Context) ([]models.TgGroup, error)
	GetTgChannels(ctx context.Context) ([]models.TgChannel, error)
	GetVkGroups(ctx context.Context) ([]models.VkGroup, error)
	InsertUsers(ctx context.Context, users []models.User) error
	DeleteUser(ctx context.Context, userID int64) error
	GetUserRole(ctx context.Context, userID int64) (string, error)
	GetUserWithUsername(ctx context.Context, username string) (models.User, error)
}

type Cache interface {
	Set(ctx context.Context, key string, value string) error
	Del(ctx context.Context, key string) error
	Get(ctx context.Context, key string) (string, error)
}

type AppCache interface {
	SetToMap(name, key string, value any, TTL time.Duration) bool
	GetFromMap(name string, key string) (any, bool)
	DeleteFromMap(name, key string)
}

func NewHandler(tg *tg_bot.Client, vk *vk.Handler, db Storage, cdb Cache, ac AppCache, log *slog.Logger) *Handler {
	return &Handler{
		tg:  tg,
		vk:  vk,
		db:  db,
		cdb: cdb,
		ac:  ac,
		log: log,
	}
}
