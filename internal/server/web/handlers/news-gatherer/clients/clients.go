package clients

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

type Clients struct {
	m  map[string]*Client
	mu sync.RWMutex
}

type Client struct {
	wsConn *websocket.Conn
	offset int
	mu     sync.RWMutex
}

func New() *Clients {
	return &Clients{
		m: make(map[string]*Client),
	}
}

var clientCounter atomic.Uint64

func (cs *Clients) Add(ws *websocket.Conn) string {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c := &Client{
		wsConn: ws,
	}

	id := fmt.Sprintf("%s:%d", c.wsConn.LocalAddr().String(), clientCounter.Add(1))
	cs.m[id] = c

	return id
}

func (cs *Clients) Get(id string) (*Client, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	v, ok := cs.m[id]

	return v, ok
}

func (cs *Clients) GetAll() []*Client {
	var c []*Client
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	for _, v := range cs.m {
		c = append(c, v)
	}

	return c
}

func (cs *Clients) Remove(id string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.m, id)
}

func (c *Client) SendMsg(msg any) error {
	if err := c.wsConn.WriteJSON(msg); err != nil {
		return err
	}

	return nil
}

func (c *Client) AddOffset(v int) {
	c.mu.Lock()
	c.offset += v
	c.mu.Unlock()
}

func (c *Client) GetOffset() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.offset
}
