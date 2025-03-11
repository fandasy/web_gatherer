package telegram

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"project/internal/pkg/logger/sl"
	"strconv"
	"syscall"
	"time"
)

func (s *Server) Listener(timeout int) {
	cfg := tgbotapi.NewUpdate(0)
	cfg.Timeout = timeout

	updates := s.tg.GetUpdatesChan(cfg)

	s.log.Info("Telegram server started")

	for update := range updates {
		u := update

		if s.middleware(&u) {
			go s.processing(&u)
		}
	}
}

func (s *Server) middleware(update *tgbotapi.Update) bool {

	if update.Message != nil {

		if update.Message.From.IsBot && !(update.Message.From.ID == s.tg.Self.ID) {
			return false
		}
	}

	return true
}

func (s *Server) Shutdown(ctx context.Context) {
	s.tg.StopReceivingUpdates()

	const shutdownPollIntervalMax = 500 * time.Millisecond

	s.log.Info("server stopping")
	defer s.log.Info("server shutdown")

	if s.activeEvents.Load() > 0 {
		s.log.Info("waiting unfinished events", slog.Any("events", s.activeEvents.Load()))
		s.log.Info("You can forcefully terminate the server with a repeated signal")

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		pollIntervalBase := time.Millisecond
		nextPollInterval := func() time.Duration {
			// Add 10% jitter.
			interval := pollIntervalBase + time.Duration(rand.Intn(int(pollIntervalBase/10)))
			// Double and clamp for next time.
			pollIntervalBase *= 2
			if pollIntervalBase > shutdownPollIntervalMax {
				pollIntervalBase = shutdownPollIntervalMax
			}
			return interval
		}

		timer := time.NewTimer(nextPollInterval())
		defer timer.Stop()

		for {
			if s.activeEvents.Load() == 0 {
				s.log.Info("all events are complete")
				return
			}

			select {
			case <-stop:
				s.log.Info("got a repeat signal")
				return

			case <-ctx.Done():
				s.log.Info("ctx done")
				return

			case <-timer.C:
				timer.Reset(nextPollInterval())
			}
		}
	}
}

func (s *Server) processing(update *tgbotapi.Update) {
	s.activeEvents.Add(1)
	defer s.activeEvents.Add(^uint32(0))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := strconv.FormatUint(s.eventsCount.Add(1), 10)
	ctx = context.WithValue(ctx, "ID", id)

	start := time.Now()

	status, err := s.p.Process(ctx, update)
	if err != nil {

		switch status {
		case SKIP:
			goto logging

		case BADREQUEST:
			s.log.Warn("[BAD REQUEST]", slog.String("ID", id), sl.Err(err))
			goto logging

		case UNKNOWN:
			goto logging

		case TIMEOUT:
			s.log.Warn("[TIME OUT]", slog.String("ID", id), sl.Err(err))
			s.sendReplyMsg(update, "timed out")

		case RECOVER:
			s.log.Error("[EVENT RECOVERED]", slog.String("ID", id), sl.Err(err))
			return

		default:
			s.log.Error("[EVENT FAILED]", slog.String("ID", id), sl.Err(err))
			s.sendReplyMsg(update, "internal server error")
		}
	}

logging:

	latency := time.Since(start)

	username := "Unknown"
	if fromUser(update) != nil {
		username = fromUser(update).UserName
	}

	from := "Unknown"
	if fromChat(update) != nil {
		from = fromChat(update).Type
	}

	s.log.Info("[ENDED EVENT]",
		slog.String("ID", id),
		slog.String("Status", status),
		slog.String("User", username),
		slog.String("From", from),
		slog.String("Duration", latency.String()),
	)
}

func (s *Server) sendReplyMsg(u *tgbotapi.Update, text string) {
	const fn = "server.sendReplyMsg"

	var (
		chatID int64
		msgID  int
	)

	switch {
	case u.Message != nil:
		chatID = u.Message.Chat.ID
		msgID = u.Message.MessageID

	case u.CallbackQuery != nil:
		chatID = u.CallbackQuery.From.ID
		msgID = u.CallbackQuery.Message.MessageID

	case u.ChannelPost != nil:
		chatID = u.ChannelPost.Chat.ID
		msgID = u.ChannelPost.MessageID

	default:
		return
	}

	var msg tgbotapi.MessageConfig

	msg = tgbotapi.NewMessage(chatID, text)

	msg.ReplyToMessageID = msgID

	_, err := s.tg.Send(msg)
	if err != nil {
		s.log.Error(fn, sl.Err(err))
	}
}

func fromUser(u *tgbotapi.Update) *tgbotapi.User {
	switch {
	case u.Message != nil:
		return u.Message.From
	case u.MyChatMember != nil:
		return &u.MyChatMember.From
	default:
		return nil
	}
}

func fromChat(u *tgbotapi.Update) *tgbotapi.Chat {
	switch {
	case u.Message != nil:
		return u.Message.Chat
	case u.EditedMessage != nil:
		return u.EditedMessage.Chat
	case u.ChannelPost != nil:
		return u.ChannelPost.Chat
	case u.EditedChannelPost != nil:
		return u.EditedChannelPost.Chat
	case u.CallbackQuery != nil:
		return u.CallbackQuery.Message.Chat
	case u.MyChatMember != nil:
		return &u.MyChatMember.Chat
	default:
		return nil
	}
}
