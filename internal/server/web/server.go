package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"project/pkg/e"
)

func (s *Server) Listener() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "front.html")
	})

	http.HandleFunc("/ws", s.handler.WS)

	go s.handler.NewsReader()

	s.log.Info("[WEB SERVER] started", slog.String("addr", s.web.Addr))

	if err := s.web.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(e.Wrap("failed to start server", err))
	}
}

func (s *Server) Shutdown(ctx context.Context) {
	if err := s.web.Shutdown(ctx); err != nil {
		s.log.Info("failed to shutdown server", slog.String("error", err.Error()))
	}
}
