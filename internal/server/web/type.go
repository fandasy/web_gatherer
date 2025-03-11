package web

import (
	"log/slog"
	"net/http"
	"project/internal/config"
	news_gatherer "project/internal/server/web/handlers/news-gatherer"
	clients "project/internal/server/web/handlers/news-gatherer/clients"
)

type Server struct {
	srv *http.Server
	log *slog.Logger
}

type Handlers struct {
	newsSender func(w http.ResponseWriter, r *http.Request)
	newsReader func()
}

func NewServer(cfg *config.WebServer, log *slog.Logger) *Server {
	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      nil,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return &Server{
		srv: srv,
		log: log,
	}
}

func NewHandler(db news_gatherer.Storage, log *slog.Logger) Handlers {
	wsConnClients := clients.New()

	return Handlers{
		newsSender: news_gatherer.NewsSender(db, wsConnClients, log),
		newsReader: news_gatherer.NewsReader(db, wsConnClients, log),
	}
}
