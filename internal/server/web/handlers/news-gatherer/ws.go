package news_gatherer

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"log/slog"
	"net/http"
	"project/internal/pkg/logger/sl"
	"project/internal/server/web/handlers/news-gatherer/clients"
	"project/internal/storage"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewsSender(db Storage, clients *clients.Clients, log *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "[HTTP SERVER] news-gatherer.New"

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error(fn, sl.Err(err))

			return
		}

		connID := clients.Add(conn)

		log.Info("[HTTP SERVER] new client", slog.String("connID", connID))

		log := log.With(slog.String("connID", connID))

		defer func() {
			log.Info("[HTTP SERVER] client disconnected")

			clients.Remove(connID)

			if err := conn.Close(); err != nil {
				if !errors.Is(err, websocket.ErrCloseSent) {
					log.Error(fn, sl.Err(err))
				}
			}
		}()

		prepareMsg, err := db.GetWebMessages(context.TODO(), 10, 0)
		if err != nil {
			if !errors.Is(err, storage.ErrNoRecordsFound) {
				log.Error(fn, sl.Err(err))
				return
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

		c, _ := clients.Get(connID)
		c.AddOffset(len(prepareMsgReq))

		err = conn.WriteJSON(prepareMsgReq)
		if err != nil {
			log.Error(fn, sl.Err(err))
			return
		}

		const pongWait = 30 * time.Minute

		// Ping
		err = conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(pongWait))
		if err != nil {
			log.Error(fn, sl.Err(err))
			return
		}
		log.Debug("[HTTP SERVER] Ping")

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
					log.Debug(fn, sl.Err(err))
					return
				}

				log.Error(fn, sl.Err(err))
				return
			}

			// Pong
			if messageType == websocket.PongMessage {
				log.Debug("[WEB SERVER] Pong")

				timer.Reset(pongWait)

				err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
				if err != nil {
					log.Error(fn, sl.Err(err))
					return
				}
				log.Debug("[HTTP SERVER] Ping")

				continue
			}

			var req map[string]any

			err = json.Unmarshal(p, &req)
			if err != nil {
				log.Error(fn, sl.Err(err))
				return
			}

			log.Debug("read json", slog.Any("req", req))

			if action, ok := req["action"].(string); ok && action == "getMsg" {

				offset := c.GetOffset()
				oldMsg, err := db.GetWebMessages(context.TODO(), 10, offset)
				if err != nil {
					if errors.Is(err, storage.ErrNoRecordsFound) {
						continue
					}

					log.Error(fn, sl.Err(err))
					return
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

				c.AddOffset(len(oldMsgReq))

				err = conn.WriteJSON(oldMsgReq)
				if err != nil {
					log.Error(fn, sl.Err(err))
					return
				}
			}
		}
	}
}
