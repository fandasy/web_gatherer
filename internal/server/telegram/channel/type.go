package channel

import (
	"context"
	"io"
	"log/slog"
	"project/internal/clients/tg_bot"
	"project/internal/files"
	"project/internal/models"
	"time"
)

type Handler struct {
	tg  *tg_bot.Client
	db  Storage
	cdb Cache
	ac  AppCache
	fdb Files
	log *slog.Logger
}

type Storage interface {
	GetUserRole(ctx context.Context, userID int64) (string, error)
	CreateTgChannel(ctx context.Context, group models.TgChannel) error
	DeleteTgChannel(ctx context.Context, channelID int64) error
	TgChannelIsExists(ctx context.Context, channelID int64) (bool, error)
	InsertTgChannelMessages(ctx context.Context, msgs []models.TgChMessage) error
}

type Cache interface {
	Set(ctx context.Context, key string, value string) error
	Get(ctx context.Context, key string) (string, error)
	SAdd(ctx context.Context, k, v string) error
	SRem(ctx context.Context, k, v string) error
	SIsMember(ctx context.Context, k, v string) (bool, error)
}

type AppCache interface {
	GetFromMap(name string, key string) (any, bool)
	SetToMap(name, key string, value any, TTL time.Duration) bool
	SetToMapWithFunc(name, key string, value any, TTL time.Duration, fn func()) bool
	Mutex(name string, fn func())
}

type Files interface {
	SaveFile(ctx context.Context, bucketName, fileName string, reader io.Reader, options files.PutObjectOptions) (string, error)
}

func NewHandler(tg *tg_bot.Client, db Storage, cdb Cache, ac AppCache, fdb Files, log *slog.Logger) *Handler {
	return &Handler{
		tg:  tg,
		db:  db,
		cdb: cdb,
		ac:  ac,
		fdb: fdb,
		log: log,
	}
}
