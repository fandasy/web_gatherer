package handlers

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/atomic"
	"log/slog"
	"project/internal/storer"
	"sync"
)

type Handler struct {
	store   *storer.Storer
	tgApi   *tgbotapi.BotAPI
	clients *clients
	log     *slog.Logger
}

type clients struct {
	m  map[string]*client
	mu sync.RWMutex
}

type client struct {
	wsConn *websocket.Conn
	offset int
	mu     sync.RWMutex
}

func New(s *storer.Storer, log *slog.Logger) *Handler {
	return &Handler{
		store: s,
		clients: &clients{
			m: make(map[string]*client),
		},
		log: log,
	}
}

var clientCounter atomic.Uint64

func (cs *clients) add(c *client) string {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	id := fmt.Sprintf("%s:%d", c.wsConn.LocalAddr().String(), clientCounter.Inc())
	cs.m[id] = c

	return id
}

func (cs *clients) get(id string) (*client, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	v, ok := cs.m[id]

	return v, ok
}

func (cs *clients) getAll() []*client {
	var c []*client
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	for _, v := range cs.m {
		c = append(c, v)
	}

	return c
}

func (cs *clients) remove(id string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.m, id)
}
