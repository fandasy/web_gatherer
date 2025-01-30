package handlers

import (
	"context"
	"fmt"
	"github.com/lib/pq"
	"project/internal/models"
	"project/internal/pkg/logger/sl"
	"project/pkg/e"
	"sync"
	"time"
)

type webMessageReq struct {
	GroupName string            `json:"group_name"`
	Text      string            `json:"text"`
	Metadata  []models.MetaPair `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	Type      string            `json:"type"`
	New       bool              `json:"new"`
}

const (
	tgChannelMsgNotify = "insert_tg_channel_message"
	tgGroupMsgNotify   = "insert_tg_group_message"
	vkGroupMsgNotify   = "insert_vk_message"
)

func (h *Handler) NewsReader() {
	const fn = "[WEB SERVER] handlers.NewsReader"

	notifyCh, err := h.mergeNotify(32, tgChannelMsgNotify, tgGroupMsgNotify, vkGroupMsgNotify)
	if err != nil {
		h.log.Error(fn, sl.Err(err))
	}

	for n := range notifyCh {
		webMsg, err := toWebMsg(n)
		if err != nil {
			h.log.Error(fn, sl.Err(err))
			continue
		}

		go func() {
			reqWebMsg := webMessageReq{
				GroupName: webMsg.GroupName,
				Text:      webMsg.Text,
				Metadata:  webMsg.Metadata,
				CreatedAt: webMsg.CreatedAt,
				Type:      webMsg.Type,
				New:       true,
			}

			for _, c := range h.clients.getAll() {
				c.addOffset(1)
				if err := c.sendMsg(reqWebMsg); err != nil {
					h.log.Error(fn, sl.Err(err))
				}
			}
		}()

		if err := h.store.DB.InsertWebMessages(context.TODO(), []models.WebMessage{webMsg}); err != nil {
			h.log.Error(fn, sl.Err(err))
		}
	}
}

func (c *client) sendMsg(msg any) error {
	if err := c.wsConn.WriteJSON(msg); err != nil {
		return err
	}

	return nil
}

func (c *client) addOffset(v int) {
	c.mu.Lock()
	c.offset += v
	c.mu.Unlock()
}

func (c *client) getOffset() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.offset
}

func (h *Handler) mergeNotify(buf uint, notifyNames ...string) (<-chan *pq.Notification, error) {
	var in []<-chan *pq.Notification

	for _, name := range notifyNames {
		notifyCh, err := h.store.DB.AddNotifier(context.TODO(), name, buf)
		if err != nil {
			return nil, e.Wrap(fmt.Sprintf("notifier: %s started error", name), err)
		}

		in = append(in, notifyCh)
	}

	var wg sync.WaitGroup

	out := make(chan *pq.Notification)

	output := func(c <-chan *pq.Notification) {
		defer wg.Done()
		for n := range c {
			out <- n
		}
	}

	wg.Add(len(in))

	for _, c := range in {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out, nil
}
