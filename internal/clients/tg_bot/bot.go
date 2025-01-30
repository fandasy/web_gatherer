package tg_bot

import (
	"log/slog"
	"project/pkg/e"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func New(token string, log *slog.Logger) (*tgbotapi.BotAPI, error) {
	const fn = "clients.tg_bot.New"

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	log.Info("[OK] tg_bot bot successfully connected")

	return bot, nil
}
