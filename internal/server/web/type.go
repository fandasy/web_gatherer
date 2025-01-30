package web

import (
	"log/slog"
	"net/http"
	"project/internal/config"
	"project/internal/server/web/handlers"
)

type Server struct {
	handler *handlers.Handler
	web     *http.Server
	log     *slog.Logger
}

func New(handler *handlers.Handler, cfg *config.WebServer, log *slog.Logger) *Server {
	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      nil,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return &Server{
		handler: handler,
		web:     srv,
		log:     log,
	}
}
