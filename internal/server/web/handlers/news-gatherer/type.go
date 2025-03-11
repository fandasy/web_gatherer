package news_gatherer

import (
	"context"
	"github.com/lib/pq"
	"project/internal/models"
	"time"
)

type Storage interface {
	InsertWebMessages(ctx context.Context, msgs []models.WebMessage) error
	GetWebMessages(ctx context.Context, limit int, offset int) ([]models.WebMessage, error)
	AddNotifier(ctx context.Context, name string, buf uint) (<-chan *pq.Notification, error)
}

type webMessageReq struct {
	GroupName string            `json:"group_name"`
	Text      string            `json:"text"`
	Metadata  []models.MetaPair `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	Type      string            `json:"type"`
	New       bool              `json:"new"`
}
