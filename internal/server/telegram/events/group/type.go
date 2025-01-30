package group

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log/slog"
	"project/internal/clients/custom_tg_bot"
	"project/internal/storer"
)

type Handler struct {
	tg  *tgbotapi.BotAPI
	cTg *custom_tg_bot.Client
	s   *storer.Storer
	log *slog.Logger
}

func NewHandler(tg *tgbotapi.BotAPI, customTg *custom_tg_bot.Client, s *storer.Storer, log *slog.Logger) *Handler {
	return &Handler{
		tg:  tg,
		cTg: customTg,
		s:   s,
		log: log,
	}
}
