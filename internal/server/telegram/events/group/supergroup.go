package group

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"project/internal/models"
)

func (h *Handler) SupergroupCmd(ctx context.Context, update *tgbotapi.Update) error {

	if update.Message != nil {
		var text string

		if update.Message.Text != "" {
			text = update.Message.Text

		} else if update.Message.Caption != "" {
			text = update.Message.Caption
		}

		switch {
		case update.Message.SuperGroupChatCreated:
			return h.initNewGroup(ctx, update.Message)

		case update.Message.NewChatMembers != nil:
			return h.handlerNewChatMembers(ctx, update.Message)

		case update.Message.LeftChatMember != nil:
			return h.leftChatMember(ctx, update.Message)

		case update.Message.NewChatTitle != "":
			return h.newGroupTitle(ctx, update.Message)

		case text != "" || update.Message.MediaGroupID != "":
			return h.saveMsg(ctx, update.Message)

		default:
			return models.ErrSkipEvent
		}
	}

	switch {
	case update.MyChatMember != nil && update.MyChatMember.NewChatMember.Status == "left":
		return h.leaveChat(ctx, update.MyChatMember)

	default:
		return models.ErrSkipEvent
	}
}
