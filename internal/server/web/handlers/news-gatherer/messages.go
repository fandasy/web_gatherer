package news_gatherer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"project/internal/models"
	"project/pkg/e"
	"time"
)

type notifyTgChannelMessage struct {
	ChanName  string            `json:"channel_name"`
	Text      string            `json:"text"`
	Metadata  []models.MetaPair `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
}

type notifyTgGroupMessage struct {
	GroupName string            `json:"group_name"`
	Username  string            `json:"username"`
	Text      string            `json:"text"`
	Metadata  []models.MetaPair `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
}

type notifyVkMessage struct {
	GroupName string            `json:"group_name"`
	Text      string            `json:"text"`
	Metadata  []models.MetaPair `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
}

const (
	tgWebType = "tg"
	vkWebType = "vk"
)

var (
	ErrUnknownChannel = errors.New("error unknown notify channel")
)

func (msg notifyTgChannelMessage) ToWebMsg() models.WebMessage {
	return models.WebMessage{
		GroupName: fmt.Sprintf("Канал: %s", msg.ChanName),
		Text:      msg.Text,
		Metadata:  msg.Metadata,
		CreatedAt: msg.CreatedAt,
		Type:      tgWebType,
	}
}

func (msg notifyTgGroupMessage) ToWebMsg() models.WebMessage {
	return models.WebMessage{
		GroupName: fmt.Sprintf("Группа: %s", msg.GroupName),
		Text:      fmt.Sprintf("От: %s\n\n%s", msg.Username, msg.Text),
		Metadata:  msg.Metadata,
		CreatedAt: msg.CreatedAt,
		Type:      tgWebType,
	}
}

func (msg notifyVkMessage) ToWebMsg() models.WebMessage {
	return models.WebMessage{
		GroupName: msg.GroupName,
		Text:      msg.Text,
		Metadata:  msg.Metadata,
		CreatedAt: msg.CreatedAt,
		Type:      vkWebType,
	}
}

func toWebMsg(n *pq.Notification) (models.WebMessage, error) {
	switch n.Channel {
	case tgChannelMsgNotify:
		var sqlMsg notifyTgChannelMessage
		if err := json.Unmarshal([]byte(n.Extra), &sqlMsg); err != nil {
			return models.WebMessage{}, e.Wrap(fmt.Sprintf("data: %s", n.Extra), err)
		}

		return sqlMsg.ToWebMsg(), nil

	case tgGroupMsgNotify:
		var sqlMsg notifyTgGroupMessage
		if err := json.Unmarshal([]byte(n.Extra), &sqlMsg); err != nil {
			return models.WebMessage{}, e.Wrap(fmt.Sprintf("data: %s", n.Extra), err)
		}

		return sqlMsg.ToWebMsg(), nil

	case vkGroupMsgNotify:
		var sqlMsg notifyVkMessage
		if err := json.Unmarshal([]byte(n.Extra), &sqlMsg); err != nil {
			return models.WebMessage{}, e.Wrap(fmt.Sprintf("data: %s", n.Extra), err)
		}

		return sqlMsg.ToWebMsg(), nil

	default:
		return models.WebMessage{}, ErrUnknownChannel
	}
}
