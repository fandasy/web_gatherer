package telegram

import (
	"context"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"project/internal/models"
)

const (
	OK         = "OK"
	SKIP       = "SKIP"
	ERROR      = "ERROR"
	RECOVER    = "RECOVER"
	UNKNOWN    = "UNKNOWN"
	TIMEOUT    = "TIMEOUT"
	CANCELED   = "CANCELED"
	BADREQUEST = "BAD REQUEST"
)

func (p Processor) Process(ctx context.Context, u *tgbotapi.Update) (status string, err error) {
	defer func() {
		if r := recover(); r != nil {
			status = RECOVER
			err = r.(error)
		}
	}()

	switch {
	case fromChat(u).IsPrivate():
		err = p.chat.ChatCmd(ctx, u)

	case fromChat(u).IsGroup():
		err = p.group.GroupCmd(ctx, u)

	case fromChat(u).IsSuperGroup():
		err = p.group.SupergroupCmd(ctx, u)

	case fromChat(u).IsChannel():
		err = p.channel.ChannelCmd(ctx, u)
	}

	if err != nil {
		switch {
		case errors.Is(err, models.ErrSkipEvent):
			return SKIP, nil

		case errors.Is(err, models.ErrBadRequest):
			return BADREQUEST, nil

		case errors.Is(err, models.ErrUnknownUser):
			return UNKNOWN, nil

		case errors.Is(err, context.DeadlineExceeded):
			return TIMEOUT, err

		case errors.Is(err, context.Canceled):
			return CANCELED, err

		default:
			return ERROR, err
		}
	}

	return OK, nil
}
