package news_gatherer

import (
	"context"
	"fmt"
	"github.com/lib/pq"
	"log/slog"
	"project/internal/models"
	"project/internal/pkg/logger/sl"
	"project/internal/server/web/handlers/news-gatherer/clients"
	"project/pkg/e"
	"sync"
)

const (
	tgChannelMsgNotify = "insert_tg_channel_message"
	tgGroupMsgNotify   = "insert_tg_group_message"
	vkGroupMsgNotify   = "insert_vk_message"
)

func NewsReader(db Storage, clients *clients.Clients, log *slog.Logger) func() {
	return func() {
		const fn = "[HTTP SERVER] web-socket.Reader"

		notifyCh, err := mergeNotify(db, 32, tgChannelMsgNotify, tgGroupMsgNotify, vkGroupMsgNotify)
		if err != nil {
			log.Error(fn, sl.Err(err))
		}

		for n := range notifyCh {
			webMsg, err := toWebMsg(n)
			if err != nil {
				log.Error(fn, sl.Err(err))
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

				for _, c := range clients.GetAll() {
					c.AddOffset(1)
					if err := c.SendMsg(reqWebMsg); err != nil {
						log.Error(fn, sl.Err(err))
					}
				}
			}()

			if err := db.InsertWebMessages(context.TODO(), []models.WebMessage{webMsg}); err != nil {
				log.Error(fn, sl.Err(err))
			}
		}
	}
}

func mergeNotify(db Storage, buf uint, notifyNames ...string) (<-chan *pq.Notification, error) {
	var in []<-chan *pq.Notification

	for _, name := range notifyNames {
		notifyCh, err := db.AddNotifier(context.TODO(), name, buf)
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
