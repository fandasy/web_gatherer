package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log/slog"
	"project/internal/server/telegram/events"
	"sync/atomic"
)

type Server struct {
	tg           *tgbotapi.BotAPI
	p            *events.Processor
	activeEvents atomic.Uint32
	eventsCount  atomic.Uint64
	log          *slog.Logger
}

func NewServer(tg *tgbotapi.BotAPI, p *events.Processor, log *slog.Logger) *Server {
	return &Server{
		tg,
		p,
		atomic.Uint32{},
		atomic.Uint64{},
		log,
	}
}
