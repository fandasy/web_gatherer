package chat

import (
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/zelenin/go-tdlib/client"
	"log/slog"
	"project/internal/models"
	"project/internal/pkg/logger/sl"
	"project/internal/storer/cache"
	"project/internal/storer/storage"
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
		tgGroups, err := h.s.DB.GetTgGroups(ctx)
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
		tgChannels, err := h.s.DB.GetTgChannels(ctx)
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
		vkGroups, err := h.s.DB.GetVkGroups(ctx)
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

	_, ok := h.s.MM.GetFromMap(models.VkNewsGroupMapName, vkDomain)
	if ok {
		if err := h.sendReplyTgMsg(msg, msgVkGroupIsExists); err != nil {
			log.Error(fn, err)
		}

		log.Error("Bad request", sl.Err(ErrVkGroupIsExists))

		return e.Wrap(fn, models.ErrBadRequest)
	}

	vkGroup, err := h.validateVkGroup(vkDomain)
	if err != nil {
		switch {
		case errors.Is(err, ErrVkGroupNotFound):
			if err := h.sendReplyTgMsg(msg, msgVkGroupNotFound); err != nil {
				log.Error(fn, err)
			}

			log.Error("Bad request", sl.Err(err))

			return e.Wrap(fn, models.ErrBadRequest)

		case errors.Is(err, ErrVkGroupIsPrivate):
			if err := h.sendReplyTgMsg(msg, msgVkGroupIsPrivate); err != nil {
				log.Error(fn, err)
			}

			log.Error("Bad request", sl.Err(err))

			return e.Wrap(fn, models.ErrBadRequest)

		default:
			return e.Wrap(fn, err)
		}
	}

	_, ok = h.s.MM.GetFromMap(models.VkNewsGroupMapName, vkDomain)
	if ok {
		if err := h.sendReplyTgMsg(msg, msgSuccessfullyAddVKNewsGroup); err != nil {
			log.Error(fn, err)
		}

		return nil
	}

	if err := h.s.DB.InsertVkGroup(ctx, &vkGroup); err != nil {
		return e.Wrap(fn, err)
	}

	go h.vkGroupNewsListener(vkGroup)

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

	v, ok := h.s.MM.GetFromMap(models.VkNewsGroupMapName, vkDomain)
	if !ok {
		if err := h.sendReplyTgMsg(msg, msgVkGroupNotFound); err != nil {
			log.Error(fn, err)
		}

		log.Error("Bad Request", ErrVkGroupNotFound)

		return e.Wrap(fn, models.ErrBadRequest)
	}

	stopCh := v.(chan struct{})

	vkGroupNewsListenerShutdown(stopCh)

	h.s.MM.DeleteFromMap(models.VkNewsGroupMapName, vkDomain)

	if err := h.sendReplyTgMsg(msg, msgSuccessfullyDeleteVkNewsGroup); err != nil {
		log.Error(fn, err)
	}

	if err := h.s.DB.DeleteVkGroup(ctx, vkDomain); err != nil {
		return e.Wrap(fn, err)
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

	user := args[1]

	chat, err := h.tdlib.SearchPublicChat(&client.SearchPublicChatRequest{
		Username: user,
	})
	if err != nil {
		return e.Wrap(fn, err)
	}

	log.Debug("SearchPublicChat result",
		slog.Int64("user id", chat.Id),
	)

	users, err := h.getTgUsersInfo([]int64{chat.Id})
	if err != nil {
		return e.Wrap(fn, err)
	}

	log.Debug("Get TgUsers Info",
		slog.String("Username", users[0].Username),
	)

	if err := h.s.DB.InsertUsers(ctx, users); err != nil {
		return e.Wrap(fn, err)
	}

	if err := h.sendReplyTgMsg(msg, msgSuccessfullyAddUser); err != nil {
		log.Error(fn, err)
	}

	if err := h.s.CDB.Set(ctx, strconv.FormatInt(chat.Id, 10), models.SubUserRole); err != nil {
		h.s.ReportCdbErr()

		return e.Wrap(fn, err)
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

	user := args[1]

	chat, err := h.tdlib.SearchPublicChat(&client.SearchPublicChatRequest{
		Username: user,
	})
	if err != nil {
		return e.Wrap(fn, err)
	}

	log.Debug("SearchPublicChat result", slog.Int64("user id", chat.Id))

	userRole, err := h.getRole(ctx, chat.Id)
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

	if err := h.s.DB.DeleteUser(ctx, chat.Id); err != nil {
		return e.Wrap(fn, err)
	}

	if err := h.sendReplyTgMsg(msg, msgSuccessfullyDeleteUser); err != nil {
		log.Error(fn, err)
	}

	if err := h.s.CDB.Del(ctx, strconv.FormatInt(chat.Id, 10)); err != nil {
		h.s.ReportCdbErr()

		return e.Wrap(fn, err)
	}

	return nil
}

func (h *Handler) getTgUsersInfo(userIDs []int64) ([]models.User, error) {
	const fn = "events.getTgUsersInfo"

	var users []models.User

	for _, userID := range userIDs {
		user, err := h.tdlib.GetUser(&client.GetUserRequest{UserId: userID})
		if err != nil {
			return nil, e.Wrap(fn, err)
		}

		roleID, _ := h.s.MM.GetFromMap(models.RoleIDsMapName, models.SubUserRole)

		users = append(users, models.User{
			UserID:    userID,
			Username:  user.Usernames.EditableUsername,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			RoleID:    roleID.(int64),
		})
	}

	return users, nil
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
	if userID == h.tg.Self.ID {
		return models.AdminRole, nil
	}

	if h.s.CdbErrStatus() {
		goto getFromDb
	}

	role, err = h.s.CDB.Get(ctx, strconv.FormatInt(userID, 10))
	if err != nil {
		switch {
		case errors.Is(err, cache.ErrKeyNotFound):
		default:
			h.s.ReportCdbErr()
		}

		goto getFromDb
	}

	return role, nil

getFromDb:
	role, err = h.s.DB.GetUserRole(ctx, userID)
	if err != nil {
		return "", err
	}

	if !h.s.CdbErrStatus() {
		if err := h.s.CDB.Set(ctx, strconv.FormatInt(userID, 10), role); err != nil {
			h.s.ReportCdbErr()
		}
	}

	return role, nil
}
