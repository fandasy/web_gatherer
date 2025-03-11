package tg_bot

import (
	"log/slog"
	"project/internal/clients/tg_bot/custom_tg_bot"
	"project/pkg/e"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client struct {
	*tgbotapi.BotAPI
	*custom_tg_bot.Client
}

func New(host, token string, log *slog.Logger) (*Client, error) {
	const fn = "clients.tg_bot.New"

	customClient := custom_tg_bot.New(host, token)

	botApiClient, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	log.Info("[OK] tg_bot bot successfully connected")

	return &Client{
		botApiClient,
		customClient,
	}, nil
}
