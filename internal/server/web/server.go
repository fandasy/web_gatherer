package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"project/pkg/e"
)

func (s *Server) Listener(h Handlers) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "front.html")
	})

	http.HandleFunc("/ws", h.newsSender)

	go h.newsReader()

	s.log.Info("[HTTP SERVER] started", slog.String("addr", s.srv.Addr))

	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(e.Wrap("failed to start server", err))
	}
}

func (s *Server) Shutdown(ctx context.Context) {
	if err := s.srv.Shutdown(ctx); err != nil {
		s.log.Info("failed to shutdown server", slog.String("error", err.Error()))
	}
}
