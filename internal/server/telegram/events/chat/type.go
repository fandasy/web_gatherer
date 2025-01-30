package chat

import (
	"github.com/SevereCloud/vksdk/v3/api"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/zelenin/go-tdlib/client"
	"log/slog"
	"project/internal/clients/custom_tg_bot"
	"project/internal/storer"
)

type Handler struct {
	tg    *tgbotapi.BotAPI
	cTg   *custom_tg_bot.Client
	tdlib *client.Client
	vk    *api.VK
	s     *storer.Storer
	log   *slog.Logger
}

func NewHandler(tg *tgbotapi.BotAPI, customTg *custom_tg_bot.Client, tdlib *client.Client, vk *api.VK, s *storer.Storer, log *slog.Logger) *Handler {
	return &Handler{
		tg:    tg,
		cTg:   customTg,
		tdlib: tdlib,
		vk:    vk,
		s:     s,
		log:   log,
	}
}
