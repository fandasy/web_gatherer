package vk

import (
	"context"
	"errors"
	"github.com/SevereCloud/vksdk/v3/api"
	"log/slog"
	"project/internal/models"
	"sync"
)

type Handler struct {
	vk  *api.VK
	db  Storage
	ls  *listeners
	log *slog.Logger
}

type listeners struct {
	m  map[string]*Listener
	mu sync.RWMutex
}

type Listener struct {
	stop chan struct{}
}

type Storage interface {
	GetVkGroups(ctx context.Context) ([]models.VkGroup, error)
	InsertVkGroup(ctx context.Context, vkGroup models.VkGroup) error
	DeleteVkGroup(ctx context.Context, vkDomain string) error
	InsertVkMessages(ctx context.Context, msgs []models.VkMessage) error
}

var (
	ErrVkGroupIsPrivate = errors.New("vk group is private")
	ErrVkGroupNotFound  = errors.New("vk group not found")
	ErrVkGroupIsExists  = errors.New("vk group is exists")
)

func New(api *api.VK, db Storage, log *slog.Logger) *Handler {
	return &Handler{
		vk:  api,
		db:  db,
		log: log,
	}
}
