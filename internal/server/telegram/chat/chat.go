package chat

import (
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log/slog"
	"project/internal/clients/vk"
	"project/internal/models"
	"project/internal/pkg/logger/sl"
	"project/internal/storage"
	"project/pkg/e"
	"regexp"
	"strconv"
	"strings"
)

const (
	startChatCmd = "/start"
	helpChatCmd  = "/help"

	getSourcesPrefix = "/get "
	getAll           = "/get sources"
	getTg            = "/get tg"
	getTgChannel     = "/get tg ch"
	getTgGroup       = "/get tg group"
	getVkGroup       = "/get vk"

	addVkGroupCmd     = "/add vk "
	deleteVkGroupCmd  = "/delete vk "
	addUserChatCmd    = "/add user "
	deleteUserChatCmd = "/delete user "
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrNotEnoughArgs = errors.New("not enough arguments")
	ErrIncorrectArgs = errors.New("incorrect arguments")
)

func (h *Handler) ChatCmd(ctx context.Context, update *tgbotapi.Update) error {

	if update.Message != nil {
		role, err := h.getRole(ctx, update.Message.From.ID)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrNoRecordsFound):
				return models.ErrUnknownUser
			default:
				return err
			}
		}

		h.log.Debug("user role obtained",
			slog.String("ID", ctx.Value("ID").(string)),
			slog.String("role", role),
		)

		ctx = context.WithValue(ctx, "Role", role)

		var text string

		if update.Message.Text != "" {
			text = update.Message.Text

		} else if update.Message.Caption != "" {
			text = update.Message.Caption
		}

		switch {
		case text == startChatCmd:
			return h.startCmd(ctx, update.Message)

		case text == helpChatCmd:
			return h.helpCmd(ctx, update.Message)

		case strings.HasPrefix(text, addVkGroupCmd):
			return h.addVkGroup(ctx, update.Message)

		case strings.HasPrefix(text, deleteVkGroupCmd):
			return h.deleteNewsVkGroup(ctx, update.Message)

		case strings.HasPrefix(text, addUserChatCmd):
			return h.addUser(ctx, update.Message)

		case strings.HasPrefix(text, deleteUserChatCmd):
			return h.deleteUser(ctx, update.Message)

		case strings.HasPrefix(text, getSourcesPrefix):
			return h.getNewsSources(ctx, update.Message)

		default:
			return models.ErrSkipEvent
		}
	}

	return models.ErrSkipEvent
}

func (h *Handler) startCmd(ctx context.Context, msgInfo *tgbotapi.Message) error {
	const fn = "events.startCmd"

	h.log.Info("[CHAT]",
		slog.String("fn", fn),
		slog.Any("ID", ctx.Value("ID")),
	)

	var text string

	switch ctx.Value("Role").(string) {
	case models.SubUserRole:
		text = msgSubUserHelp

	case models.AdminRole:
		text = msgAdminHelp
	}

	text = msgStart + text

	var msg tgbotapi.MessageConfig

	msg = tgbotapi.NewMessage(msgInfo.Chat.ID, text)

	_, err := h.tg.Send(msg)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (h *Handler) helpCmd(ctx context.Context, msgInfo *tgbotapi.Message) error {
	const fn = "events.helpCmd"

	h.log.Info("[CHAT]",
		slog.String("fn", fn),
		slog.Any("ID", ctx.Value("ID")),
	)

	var text string

	switch ctx.Value("Role").(string) {
	case models.SubUserRole:
		text = msgSubUserHelp

	case models.AdminRole:
		text = msgAdminHelp
	}

	var msg tgbotapi.MessageConfig

	msg = tgbotapi.NewMessage(msgInfo.Chat.ID, text)

	_, err := h.tg.Send(msg)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (h *Handler) getNewsSources(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "events.getNewsSources"

	h.log.Info("[CHAT]",
		slog.String("fn", fn),
		slog.Any("ID", ctx.Value("ID")),
		slog.String("username", msg.From.UserName),
		slog.String("cmd", msg.Text),
	)

	if !defineRole(ctx.Value("Role").(string), models.SubUserRole, models.AdminRole) {
		return models.ErrSkipEvent
	}

	log := h.log.With(slog.Any("ID", ctx.Value("ID")))

	getTgGroups := func() (string, error) {
		tgGroups, err := h.db.GetTgGroups(ctx)
		if err != nil {
			if errors.Is(err, storage.ErrNoRecordsFound) {
				return "", nil
			}

			return "", err
		} else {
			text := "TG Группы:\n"
			var groupsInfo []string

			for _, tgGroup := range tgGroups {
				groupsInfo = append(groupsInfo,
					fmt.Sprintf("%s %s\n", tgGroup.Name, tgGroup.Description),
				)
			}
			text += strings.Join(groupsInfo, "")
			return text, nil
		}
	}

	getTgChannels := func() (string, error) {
		tgChannels, err := h.db.GetTgChannels(ctx)
		if err != nil {
			if errors.Is(err, storage.ErrNoRecordsFound) {
				return "", nil
			}

			return "", err
		} else {
			text := "TG Каналы:\n"
			var channelInfo []string

			for _, tgChannel := range tgChannels {
				channelInfo = append(channelInfo,
					fmt.Sprintf("%s %s\n", tgChannel.Name, tgChannel.Description),
				)
			}
			text += strings.Join(channelInfo, "")
			return text, nil
		}
	}

	getVkGroups := func() (string, error) {
		vkGroups, err := h.db.GetVkGroups(ctx)
		if err != nil {
			if errors.Is(err, storage.ErrNoRecordsFound) {
				return "", nil
			}

			return "", err
		} else {
			text := "VK:\n"
			var groupsInfo []string
			for _, vkGroup := range vkGroups {
				groupsInfo = append(groupsInfo,
					fmt.Sprintf("%s %s\n", vkGroup.Name, vkGroup.Domain),
				)
			}
			text += strings.Join(groupsInfo, "")
			return text, nil
		}
	}

	funcArr := make([]func() (string, error), 0)

	switch msg.Text {
	case getAll:
		funcArr = append(funcArr, getTgGroups, getTgChannels, getVkGroups)

	case getTg:
		funcArr = append(funcArr, getTgGroups, getTgChannels)

	case getTgChannel:
		funcArr = append(funcArr, getTgChannels)

	case getTgGroup:
		funcArr = append(funcArr, getTgGroups)

	case getVkGroup:
		funcArr = append(funcArr, getVkGroups)

	default:
	}

	var reqText string

	if len(funcArr) == 0 {
		reqText = msgIncorrectArgs
	}

	for _, getFunc := range funcArr {
		getText, err := getFunc()
		if err != nil {
			return e.Wrap(fn, err)
		}

		reqText += getText
	}

	if reqText == "" {
		reqText = msgNewsSourcesNotFound
	}

	req := tgbotapi.NewMessage(msg.Chat.ID, reqText)

	resp, err := h.tg.Send(req)
	if err != nil {
		return e.Wrap(fn, err)
	}

	log.Debug("send message", slog.Any("resp", resp))

	return nil
}

func (h *Handler) addVkGroup(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "events.addVkGroup"

	h.log.Info("[CHAT]",
		slog.String("fn", fn),
		slog.Any("ID", ctx.Value("ID")),
		slog.String("username", msg.From.UserName),
		slog.String("cmd", msg.Text),
	)

	if !defineRole(ctx.Value("Role").(string), models.SubUserRole, models.AdminRole) {
		return models.ErrSkipEvent
	}

	log := h.log.With(slog.Any("ID", ctx.Value("ID")))

	vkDomain := strings.TrimPrefix(msg.Text, addVkGroupCmd)

	if err := h.vk.ListenStart(ctx, vkDomain); err != nil {
		switch {
		case errors.Is(err, vk.ErrVkGroupIsExists):
			if err := h.sendReplyTgMsg(msg, msgVkGroupIsExists); err != nil {
				log.Error(fn, err)
			}

			log.Error("Bad request", sl.Err(err))

			return e.Wrap(fn, models.ErrBadRequest)

		case errors.Is(err, vk.ErrVkGroupNotFound):
			if err := h.sendReplyTgMsg(msg, msgVkGroupNotFound); err != nil {
				log.Error(fn, err)
			}

			log.Error("Bad request", sl.Err(err))

			return e.Wrap(fn, models.ErrBadRequest)

		case errors.Is(err, vk.ErrVkGroupIsPrivate):
			if err := h.sendReplyTgMsg(msg, msgVkGroupIsPrivate); err != nil {
				log.Error(fn, err)
			}

			log.Error("Bad request", sl.Err(err))

			return e.Wrap(fn, models.ErrBadRequest)

		default:
			return e.Wrap(fn, err)
		}
	}

	if err := h.sendReplyTgMsg(msg, msgSuccessfullyAddVKNewsGroup); err != nil {
		log.Error(fn, err)
	}

	return nil
}

func (h *Handler) deleteNewsVkGroup(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "events.deleteNewsVkGroup"

	h.log.Info("[CHAT]",
		slog.String("fn", fn),
		slog.Any("ID", ctx.Value("ID")),
		slog.String("username", msg.From.UserName),
		slog.String("cmd", msg.Text),
	)

	if !defineRole(ctx.Value("Role").(string), models.SubUserRole, models.AdminRole) {
		return models.ErrSkipEvent
	}

	log := h.log.With(slog.Any("ID", ctx.Value("ID")))

	vkDomain := strings.TrimPrefix(msg.Text, deleteVkGroupCmd)

	if err := h.vk.Shutdown(ctx, vkDomain); err != nil {
		if errors.Is(err, vk.ErrVkGroupNotFound) {
			if err := h.sendReplyTgMsg(msg, msgVkGroupNotFound); err != nil {
				log.Error(fn, err)
			}

			log.Error("Bad Request", sl.Err(err))

			return e.Wrap(fn, models.ErrBadRequest)
		}

		return e.Wrap(fn, err)
	}

	if err := h.sendReplyTgMsg(msg, msgSuccessfullyDeleteVkNewsGroup); err != nil {
		log.Error(fn, err)
	}

	return nil
}

func (h *Handler) addUser(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "events.addUser"

	h.log.Info("[CHAT]",
		slog.String("fn", fn),
		slog.Any("ID", ctx.Value("ID")),
		slog.String("username", msg.From.UserName),
		slog.String("cmd", msg.Text),
	)

	if !defineRole(ctx.Value("Role").(string), models.AdminRole) {
		return models.ErrSkipEvent
	}

	log := h.log.With(slog.Any("ID", ctx.Value("ID")))

	input := strings.TrimPrefix(msg.Text, addUserChatCmd)
	re := regexp.MustCompile(`@(\w+)`)
	args := re.FindStringSubmatch(input)

	if len(args) != 2 {
		if err := h.sendReplyTgMsg(msg, msgIncorrectArgs); err != nil {
			log.Error(fn, err)
		}

		log.Error("Bad Request", sl.Err(ErrIncorrectArgs))

		return e.Wrap(fn, models.ErrBadRequest)
	}

	username := args[1]

	chat, err := h.tg.GetChat(tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{
			SuperGroupUsername: username,
		},
	})
	if err != nil {
		return e.Wrap(fn, err)
	}

	log.Debug("GetChat result",
		slog.Int64("user id", chat.ID),
	)

	roleID, _ := h.ac.GetFromMap(models.RoleIDsMapName, models.SubUserRole)

	users := []models.User{
		{
			UserID:    chat.ID,
			Username:  chat.UserName,
			FirstName: chat.FirstName,
			LastName:  chat.LastName,
			RoleID:    roleID.(int64),
		},
	}

	log.Debug("Get TgUsers Info",
		slog.Any("user", users[0]),
	)

	if err := h.db.InsertUsers(ctx, users); err != nil {
		return e.Wrap(fn, err)
	}

	if err := h.sendReplyTgMsg(msg, msgSuccessfullyAddUser); err != nil {
		log.Error(fn, err)
	}

	if err := h.cdb.Set(ctx, strconv.FormatInt(chat.ID, 10), models.SubUserRole); err != nil {
		log.Warn("Cache error",
			slog.String("fn", fn),
			sl.Err(err),
		)
	}

	return nil
}

func (h *Handler) deleteUser(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "events.deleteUser"

	h.log.Info("[CHAT]",
		slog.String("fn", fn),
		slog.Any("ID", ctx.Value("ID")),
		slog.String("username", msg.From.UserName),
		slog.String("cmd", msg.Text),
	)

	if !defineRole(ctx.Value("Role").(string), models.AdminRole) {
		return models.ErrSkipEvent
	}

	log := h.log.With(slog.Any("ID", ctx.Value("ID")))

	input := strings.TrimPrefix(msg.Text, deleteUserChatCmd)
	re := regexp.MustCompile(`@(\w+)`)
	args := re.FindStringSubmatch(input)

	if len(args) != 2 {
		if err := h.sendReplyTgMsg(msg, msgNotEnoughArgs); err != nil {
			log.Error(fn, err)
		}

		log.Error("Bad Request", sl.Err(ErrNotEnoughArgs))

		return e.Wrap(fn, models.ErrBadRequest)
	}

	username := args[1]

	chat, err := h.tg.GetChat(tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{
			SuperGroupUsername: username,
		},
	})
	if err != nil {
		return e.Wrap(fn, err)
	}

	log.Debug("GetChat result",
		slog.Int64("user id", chat.ID),
	)

	userRole, err := h.getRole(ctx, chat.ID)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecordsFound) {
			if err := h.sendReplyTgMsg(msg, msgUserNotFound); err != nil {
				log.Error(fn, err)
			}

			log.Error("Bad Request", sl.Err(ErrUserNotFound))

			return e.Wrap(fn, models.ErrBadRequest)
		}

		return e.Wrap(fn, err)
	}

	if defineRole(userRole, models.AdminRole) {
		if err := h.sendReplyTgMsg(msg, msgIncorrectArgs); err != nil {
			log.Error(fn, err)
		}

		log.Error("Bad Request", sl.Err(ErrIncorrectArgs))

		return e.Wrap(fn, models.ErrBadRequest)
	}

	if err := h.db.DeleteUser(ctx, chat.ID); err != nil {
		return e.Wrap(fn, err)
	}

	if err := h.sendReplyTgMsg(msg, msgSuccessfullyDeleteUser); err != nil {
		log.Error(fn, err)
	}

	if err := h.cdb.Del(ctx, strconv.FormatInt(chat.ID, 10)); err != nil {
		log.Warn("Cache error",
			slog.String("fn", fn),
			sl.Err(err),
		)
	}

	return nil
}

func (h *Handler) sendReplyTgMsg(msgInfo *tgbotapi.Message, text string) error {
	const fn = "events.sendReplyTgMsg"

	var msg tgbotapi.MessageConfig

	msg = tgbotapi.NewMessage(msgInfo.Chat.ID, text)

	msg.ReplyToMessageID = msgInfo.MessageID

	_, err := h.tg.Send(msg)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func defineRole(role string, roles ...string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}

	return false
}

func (h *Handler) getRole(ctx context.Context, userID int64) (role string, err error) {
	const fn = "chat.getRole"

	if userID == h.tg.Self.ID {
		return models.AdminRole, nil
	}

	userIdStr := strconv.FormatInt(userID, 10)

	role, err = h.cdb.Get(ctx, userIdStr)
	if err != nil {
		h.log.Warn(fn, sl.Err(err))

		goto getFromDb
	}

	return role, nil

getFromDb:
	role, err = h.db.GetUserRole(ctx, userID)
	if err != nil {
		return "", err
	}

	if err := h.cdb.Set(ctx, userIdStr, role); err != nil {
		h.log.Warn("Cache error",
			slog.String("fn", fn),
			sl.Err(err))
	}

	return role, nil
}
