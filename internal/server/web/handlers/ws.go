package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"log/slog"
	"net/http"
	"project/internal/pkg/logger/sl"
	"project/internal/storer/storage"
	"project/pkg/e"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Handler) WS(w http.ResponseWriter, r *http.Request) {
	const fn = "[WEB SERVER] handlers.WS"

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error(fn, sl.Err(err))

		return
	}

	c := &client{
		wsConn: conn,
	}

	connID := h.clients.add(c)

	h.log.Info("[WEB SERVER] new client", slog.String("connID", connID))

	if err := h.handleConnection(conn, connID); err != nil {
		h.log.Error(fn, sl.Err(err))
	}
}

func (h *Handler) handleConnection(conn *websocket.Conn, connID string) error {
	const fn = "handlers.handleConnection"

	log := h.log.With(slog.String("connID", connID))

	prepareMsg, err := h.store.DB.GetWebMessages(context.TODO(), 10, 0)
	if err != nil {
		if !errors.Is(err, storage.ErrNoRecordsFound) {
			return e.Wrap(fn, err)
		}
	}

	var prepareMsgReq []webMessageReq

	for _, msg := range prepareMsg {
		prepareMsgReq = append(prepareMsgReq, webMessageReq{
			GroupName: msg.GroupName,
			Text:      msg.Text,
			Metadata:  msg.Metadata,
			CreatedAt: msg.CreatedAt,
			Type:      msg.Type,
			New:       false,
		})
	}

	c, _ := h.clients.get(connID)
	c.addOffset(len(prepareMsgReq))

	err = conn.WriteJSON(prepareMsgReq)
	if err != nil {
		return e.Wrap(fn, err)
	}

	const pongWait = 30 * time.Minute

	defer func() {
		log.Info("[WEB SERVER] client disconnected")

		h.clients.remove(connID)

		if err := conn.Close(); err != nil {
			if !errors.Is(err, websocket.ErrCloseSent) {
				log.Error(fn, sl.Err(err))
			}
		}
	}()

	// Ping
	err = conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(pongWait))
	if err != nil {
		return e.Wrap(fn, err)
	}
	log.Debug("[WEB SERVER] Ping")

	timer := time.AfterFunc(pongWait, func() {
		if err := conn.Close(); err != nil {
			log.Error(fn, sl.Err(err))
		}
	})

	defer timer.Stop()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				return nil
			}

			return e.Wrap(fn, err)
		}

		// Pong
		if messageType == websocket.PongMessage {
			log.Debug("[WEB SERVER] Pong")

			timer.Reset(pongWait)

			err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
			if err != nil {
				return e.Wrap(fn, err)
			}
			log.Debug("[WEB SERVER] Ping")

			continue
		}

		var req map[string]any

		err = json.Unmarshal(p, &req)
		if err != nil {
			return e.Wrap(fn, err)
		}

		log.Debug("read json", slog.Any("req", req))

		if action, ok := req["action"].(string); ok && action == "getMsg" {

			offset := c.getOffset()
			oldMsg, err := h.store.DB.GetWebMessages(context.TODO(), 10, offset)
			if err != nil {
				if errors.Is(err, storage.ErrNoRecordsFound) {
					continue
				}

				return e.Wrap(fn, err)
			}

			var oldMsgReq []webMessageReq

			for _, msg := range oldMsg {
				oldMsgReq = append(oldMsgReq, webMessageReq{
					GroupName: msg.GroupName,
					Text:      msg.Text,
					Metadata:  msg.Metadata,
					CreatedAt: msg.CreatedAt,
					Type:      msg.Type,
					New:       false,
				})
			}

			c.addOffset(len(oldMsgReq))

			err = conn.WriteJSON(oldMsgReq)
			if err != nil {
				return e.Wrap(fn, err)
			}
		}
	}
}
