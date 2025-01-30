package storage

import (
	"context"
	"errors"
	"github.com/lib/pq"
	"project/internal/models"
)

type Storage interface {
	GetUsersRole() (map[string]string, error)
	GetRoleIDs() ([]models.Role, error)
	GetUserRole(ctx context.Context, userID int64) (string, error)
	InsertUsers(ctx context.Context, users []models.User) error
	CreateTgChannel(ctx context.Context, group *models.TgChannel) error
	CreateTgGroup(ctx context.Context, group *models.TgGroup) error
	UpdateTgGroup(ctx context.Context, groupID int64, group *models.TgGroup) error
	TgGroupIsExists(ctx context.Context, groupID int64) (bool, error)
	TgChannelIsExists(ctx context.Context, channelID int64) (bool, error)
	DeleteUser(ctx context.Context, userID int64) error
	InsertVkGroup(ctx context.Context, vkGroup *models.VkGroup) error
	UpdateTgGroupInfo(ctx context.Context, group *models.TgGroup) error
	DeleteTgChannel(ctx context.Context, channelID int64) error
	DeleteTgGroup(ctx context.Context, groupID int64) error
	InsertTgGroupMessages(ctx context.Context, msgs []models.TgGroupMessage) error
	InsertTgChannelMessages(ctx context.Context, msgs []models.TgChMessage) error
	InsertVkMessages(ctx context.Context, msgs []models.VkMessage) error
	GetVkGroups(ctx context.Context) ([]models.VkGroup, error)
	DeleteVkGroup(ctx context.Context, vkDomain string) error
	GetTgGroups(ctx context.Context) ([]models.TgGroup, error)
	GetTgChannels(ctx context.Context) ([]models.TgChannel, error)
	GetWebMessages(ctx context.Context, limit int, offset int) ([]models.WebMessage, error)
	InsertWebMessages(ctx context.Context, msgs []models.WebMessage) error
	AddNotifier(ctx context.Context, name string, buf uint) (<-chan *pq.Notification, error)
	GetNotifier(name string) (<-chan *pq.Notification, error)
	RemoveNotifier(ctx context.Context, name string) error
}

var (
	ErrNoRecordsFound     = errors.New("no records found")
	ErrChannelAlreadyOpen = errors.New("channel already open")
	ErrChannelNotFound    = errors.New("channel not found")
)
