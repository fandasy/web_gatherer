package chat

import (
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log/slog"
	"math/rand"
	"project/internal/clients/vk"
	"project/internal/models"
	"project/internal/pkg/logger/sl"
	"project/internal/storage"
	"project/pkg/e"
	"regexp"
	"strconv"
	"strings"
	"time"
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

	permissionCmd = "/perm "

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

		if strings.HasPrefix(update.Message.Text, permissionCmd) {

		}

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
	const fn = "chat.startCmd"

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
	const fn = "chat.helpCmd"

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

func (h *Handler) getPermission(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "chat.getPermission"

	h.log.Info("[CHAT]",
		slog.String("fn", fn),
		slog.Any("ID", ctx.Value("ID")),
		slog.String("username", msg.From.UserName),
		slog.String("cmd", msg.Text),
	)

	log := h.log.With(slog.Any("ID", ctx.Value("ID")))

	secretCode := strings.TrimPrefix(msg.Text, permissionCmd)

	correctCode, ok := h.ac.GetFromMap(secretCodeMap, msg.From.UserName)
	if !ok {
		if err := h.sendReplyTgMsg(msg, msgIncorrectArgs); err != nil {
			log.Error(fn, err)
		}

		log.Warn("Bad request", sl.Err(ErrUserNotFound))

		return e.Wrap(fn, models.ErrBadRequest)
	}

	if correctCode != secretCode {
		if err := h.sendReplyTgMsg(msg, msgIncorrectArgs); err != nil {
			log.Error(fn, err)
		}

		log.Warn("Bad request", sl.Err(ErrIncorrectArgs))

		return e.Wrap(fn, models.ErrBadRequest)
	}

	log.Debug("Correct code received", slog.String("Username", msg.From.UserName))

	roleID, _ := h.ac.GetFromMap(models.RoleIDsMapName, models.SubUserRole)

	from := msg.From

	users := []models.User{
		{
			UserID:    from.ID,
			Username:  from.UserName,
			FirstName: from.FirstName,
			LastName:  from.LastName,
			RoleID:    roleID.(int64),
		},
	}

	if err := h.db.InsertUsers(ctx, users); err != nil {
		return e.Wrap(fn, err)
	}

	h.ac.DeleteFromMap(secretCodeMap, msg.From.UserName)

	if err := h.sendReplyTgMsg(msg, msgSuccessfullyAddUser); err != nil {
		log.Error(fn, err)
	}

	if err := h.cdb.Set(ctx, strconv.FormatInt(from.ID, 10), models.SubUserRole); err != nil {
		log.Warn("Cache error",
			slog.String("fn", fn),
			sl.Err(err),
		)
	}

	return nil
}

func (h *Handler) getNewsSources(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "chat.getNewsSources"

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
	const fn = "chat.addVkGroup"

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
	const fn = "chat.deleteNewsVkGroup"

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
	const fn = "chat.addUser"

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

	secretCode := getSecretCode()

	h.ac.SetToMap(secretCodeMap, username, secretCode, 0)

	if err := h.sendReplyTgMsg(msg, fmt.Sprintf(msgSuccessfullyAddUser, secretCode)); err != nil {
		log.Error(fn, err)
	}

	return nil
}

func (h *Handler) deleteUser(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "chat.deleteUser"

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

	user, err := h.db.GetUserWithUsername(ctx, username)
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

	userRole, err := h.getRole(ctx, user.UserID)
	if err != nil {
		return e.Wrap(fn, err)
	}

	if defineRole(userRole, models.AdminRole) {
		if err := h.sendReplyTgMsg(msg, msgIncorrectArgs); err != nil {
			log.Error(fn, err)
		}

		log.Error("Bad Request", sl.Err(ErrIncorrectArgs))

		return e.Wrap(fn, models.ErrBadRequest)
	}

	if err := h.db.DeleteUser(ctx, user.UserID); err != nil {
		return e.Wrap(fn, err)
	}

	if err := h.sendReplyTgMsg(msg, msgSuccessfullyDeleteUser); err != nil {
		log.Error(fn, err)
	}

	if err := h.cdb.Del(ctx, strconv.FormatInt(user.UserID, 10)); err != nil {
		log.Warn("Cache error",
			slog.String("fn", fn),
			sl.Err(err),
		)
	}

	return nil
}

func (h *Handler) sendReplyTgMsg(msgInfo *tgbotapi.Message, text string) error {
	const fn = "chat.sendReplyTgMsg"

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

func getSecretCode() string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	keyLength := 10

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	shortKey := make([]byte, keyLength)
	for i := range shortKey {
		shortKey[i] = charset[r.Intn(len(charset)-1)]
	}

	return string(shortKey)
}
