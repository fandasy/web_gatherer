package telegram

import (
	"log/slog"
	"project/internal/clients/tg_bot"
	"project/internal/server/telegram/channel"
	"project/internal/server/telegram/chat"
	"project/internal/server/telegram/group"
	"sync/atomic"
)

type Server struct {
	tg           *tg_bot.Client
	p            Processor
	activeEvents atomic.Uint32
	eventsCount  atomic.Uint64
	log          *slog.Logger
}

type Processor struct {
	chat    *chat.Handler
	group   *group.Handler
	channel *channel.Handler
}

func NewServer(tg *tg_bot.Client, p Processor, log *slog.Logger) *Server {
	return &Server{
		tg,
		p,
		atomic.Uint32{},
		atomic.Uint64{},
		log,
	}
}

func NewProcessor(chat *chat.Handler, group *group.Handler, channel *channel.Handler) Processor {
	return Processor{
		chat:    chat,
		group:   group,
		channel: channel,
	}
}
