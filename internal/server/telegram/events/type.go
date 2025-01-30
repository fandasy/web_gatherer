package events

import (
	"github.com/SevereCloud/vksdk/v3/api"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/zelenin/go-tdlib/client"
	"log/slog"
	"project/internal/clients/custom_tg_bot"
	"project/internal/server/telegram/events/channel"
	"project/internal/server/telegram/events/chat"
	"project/internal/server/telegram/events/group"
	"project/internal/storer"
)

type Processor struct {
	tg  *tgbotapi.BotAPI
	h   Handlers
	s   *storer.Storer
	log *slog.Logger
}

type Handlers struct {
	group   *group.Handler
	chat    *chat.Handler
	channel *channel.Handler
}

func NewProcessor(tgClient *tgbotapi.BotAPI, customTg *custom_tg_bot.Client, tdlib *client.Client, vk *api.VK, s *storer.Storer, log *slog.Logger) *Processor {
	return &Processor{
		tg: tgClient,
		h: Handlers{
			group:   group.NewHandler(tgClient, customTg, s, log),
			chat:    chat.NewHandler(tgClient, customTg, tdlib, vk, s, log),
			channel: channel.NewHandler(tgClient, customTg, s, log),
		},
		s:   s,
		log: log,
	}
}
