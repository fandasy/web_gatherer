package channel

import (
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"project/internal/files"
	"project/internal/models"
	"project/internal/pkg/logger/sl"
	"project/internal/storage"
	"project/pkg/e"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	channelIdsList = "channel_ids"
)

func (h *Handler) ChannelCmd(ctx context.Context, update *tgbotapi.Update) error {

	switch {
	case update.ChannelPost != nil:
		return h.saveMsg(ctx, update.ChannelPost)

	case update.MyChatMember != nil && update.MyChatMember.NewChatMember.Status == "administrator":
		return h.initNewChannel(ctx, update.MyChatMember)

	case update.MyChatMember != nil && update.MyChatMember.NewChatMember.Status == "left":
		return h.leaveChannel(ctx, update.MyChatMember)

	default:
		return models.ErrSkipEvent
	}
}

func (h *Handler) initNewChannel(ctx context.Context, cmu *tgbotapi.ChatMemberUpdated) (err error) {
	const fn = "events.initNewChannel"

	h.log.Info("[TG CHANNEL]",
		slog.String("fn", fn),
		slog.Any("ID", ctx.Value("ID")),
		slog.String("username", cmu.From.UserName),
		slog.String("group", cmu.Chat.Title),
	)

	log := slog.With(
		slog.Any("ID", ctx.Value("ID")),
	)

	role, err := h.getRole(ctx, cmu.From.ID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNoRecordsFound):
			return models.ErrUnknownUser
		default:
			return err
		}
	}

	log.Debug("user role obtained",
		slog.String("role", role),
	)

	if !defineRole(role, models.SubUserRole, models.AdminRole) {
		if err = h.tg.LeaveChat(ctx, cmu.Chat.ID); err != nil {
			return e.Wrap(fn, err)
		}

		return models.ErrSkipEvent
	}

	channel := models.TgChannel{
		ChannelID:   cmu.Chat.ID,
		Name:        cmu.Chat.Title,
		Description: cmu.Chat.Description,
	}

	if err := h.db.CreateTgChannel(ctx, channel); err != nil {
		return e.Wrap(fn, err)
	}

	if err = h.cdb.SAdd(ctx, channelIdsList, strconv.FormatInt(cmu.Chat.ID, 10)); err != nil {
		log.Warn("Cache error",
			slog.String("fn", fn),
			sl.Err(err),
		)
	}

	return nil
}

func (h *Handler) leaveChannel(ctx context.Context, cmu *tgbotapi.ChatMemberUpdated) error {
	const fn = "events.leaveChannel"

	h.log.Info("[TG CHANNEL]",
		slog.String("fn", fn),
		slog.Any("ID", ctx.Value("ID")),
	)

	log := slog.With(
		slog.Any("ID", ctx.Value("ID")),
	)

	if err := h.db.DeleteTgChannel(ctx, cmu.Chat.ID); err != nil {
		if errors.Is(err, storage.ErrNoRecordsFound) {
			return models.ErrSkipEvent
		}

		return e.Wrap(fn, err)
	}

	if err := h.cdb.SRem(ctx, channelIdsList, strconv.FormatInt(cmu.Chat.ID, 10)); err != nil {
		log.Warn("Cache error",
			slog.String("fn", fn),
			sl.Err(err),
		)
	}

	return nil
}

func (h *Handler) saveMsg(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "events.saveMsg"

	h.log.Info("[TG CHANNEL]",
		slog.String("fn", fn),
		slog.Any("ID", ctx.Value("ID")),
		slog.String("group", msg.Chat.Title),
	)

	log := h.log.With(
		slog.Any("ID", ctx.Value("ID")),
	)

	exists, err := h.cdb.SIsMember(ctx, channelIdsList, strconv.FormatInt(msg.Chat.ID, 10))
	if err != nil {
		log.Warn("Cache error",
			slog.String("fn", fn),
			sl.Err(err),
		)

		exists, err = h.db.TgChannelIsExists(ctx, msg.Chat.ID)
		if err != nil {
			return e.Wrap(fn, err)
		}
	}

	if !exists {
		if err := h.tg.LeaveChat(ctx, msg.Chat.ID); err != nil {
			return e.Wrap(fn, err)
		}

		return models.ErrSkipEvent
	}

	switch {
	case msg.Text != "":
		return h.handleSaveTextMessage(ctx, msg)

	case msg.Caption != "":
		return h.handleSaveCaptionMessage(ctx, msg)

	case msg.MediaGroupID != "":
		return h.handleSaveMessageMedia(ctx, msg)
	}

	return nil
}

func (h *Handler) handleSaveTextMessage(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "events.handleSaveTextMessage"

	var msgText = msg.Text

	message := models.TgChMessage{
		MessageID: msg.MessageID,
		ChannelID: msg.Chat.ID,
		Text:      msgText,
		Metadata:  nil,
		CreatedAt: time.Unix(int64(msg.Date), 0),
	}

	msgs := []models.TgChMessage{message}

	if err := h.db.InsertTgChannelMessages(ctx, msgs); err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

const (
	mediaGroupMsgWaitingTime = 500 * time.Millisecond

	loadMediaFromTgTimeout = 5 * time.Minute
)

func (h *Handler) handleSaveCaptionMessage(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "events.handleSaveCaptionMessage"

	log := h.log.With(slog.Any("ID", ctx.Value("ID")))

	var msgText = msg.Caption

	msgMetadata := getMetadataFromTgMsg(msg)

	log.Debug("message received",
		slog.String("text", msgText),
		slog.String("meta type", msgMetadata.Type),
		slog.String("metadata id", msgMetadata.ID),
	)

	metaPair := models.TgMetaPair{
		ID:   msgMetadata.ID,
		Type: msgMetadata.Type,
	}

	if msg.MediaGroupID != "" {

		saveMsgFunc := func() {
			const fn = "events.saveMsgFunc"
			resp, _ := h.ac.GetFromMap(models.MediaGroupMapName, msg.MediaGroupID)

			readyMsg := resp.(*models.TgChMessage)

			wg := sync.WaitGroup{}

			var mu sync.Mutex
			metaPairs := make([]models.MetaPair, 0, len(readyMsg.MetadataID))

			for _, pairID := range readyMsg.MetadataID {
				wg.Add(1)

				go func() {
					defer wg.Done()

					metaUrl, err := h.loadMetaByTgID(pairID.ID, loadMediaFromTgTimeout)
					if err != nil {
						h.log.Error(fn, err)
						return
					}

					mu.Lock()
					defer mu.Unlock()

					metaPairs = append(metaPairs, models.MetaPair{
						Url:  metaUrl,
						Type: pairID.Type,
					})
				}()
			}

			wg.Wait()

			readyMsg.Metadata = metaPairs

			msgs := []models.TgChMessage{*readyMsg}

			log.Debug(fn, slog.Any("media group message", *readyMsg))

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := h.db.InsertTgChannelMessages(ctx, msgs); err != nil {
				log.Error(fn, sl.Err(err))
			}
		}

		resp, ok := h.ac.GetFromMap(models.MediaGroupMapName, msg.MediaGroupID)
		if !ok {

			metadata := []models.TgMetaPair{metaPair}

			message := &models.TgChMessage{
				MessageID:  msg.MessageID,
				ChannelID:  msg.Chat.ID,
				Text:       msgText,
				MetadataID: metadata,
				CreatedAt:  time.Unix(int64(msg.Date), 0),
			}

			h.ac.SetToMapWithFunc(
				models.MediaGroupMapName,
				msg.MediaGroupID,
				message,
				mediaGroupMsgWaitingTime,
				saveMsgFunc,
			)

		} else {

			existingMsg := resp.(*models.TgChMessage)

			h.ac.Mutex(models.MediaGroupMapName, func() {

				existingMsg.MessageID = msg.MessageID
				existingMsg.ChannelID = msg.Chat.ID
				existingMsg.Text = msgText
				existingMsg.CreatedAt = time.Unix(int64(msg.Date), 0)

				existingMsg.MetadataID = append(existingMsg.MetadataID, metaPair)

			})

			h.ac.SetToMapWithFunc(
				models.MediaGroupMapName,
				msg.MediaGroupID,
				existingMsg,
				mediaGroupMsgWaitingTime,
				saveMsgFunc,
			)
		}

	} else {

		metaUrl, err := h.loadMetaByTgID(metaPair.ID, loadMediaFromTgTimeout)
		if err != nil {
			return e.Wrap(fn, err)
		}

		message := models.TgChMessage{
			MessageID: msg.MessageID,
			ChannelID: msg.Chat.ID,
			Text:      msgText,
			Metadata: []models.MetaPair{{
				Url:  metaUrl,
				Type: metaPair.Type,
			}},
			CreatedAt: time.Unix(int64(msg.Date), 0),
		}

		msgs := []models.TgChMessage{message}

		if err := h.db.InsertTgChannelMessages(ctx, msgs); err != nil {
			return e.Wrap(fn, err)
		}
	}

	return nil
}

func (h *Handler) handleSaveMessageMedia(ctx context.Context, msg *tgbotapi.Message) error {
	const fn = "events.handleSaveMessageMedia"

	log := h.log.With(slog.Any("ID", ctx.Value("ID")))

	msgMetadata := getMetadataFromTgMsg(msg)

	log.Debug("message received",
		slog.String("meta type", msgMetadata.Type),
		slog.String("metadata id", msgMetadata.ID),
	)

	metaPair := models.TgMetaPair{
		ID:   msgMetadata.ID,
		Type: msgMetadata.Type,
	}

	resp, ok := h.ac.GetFromMap(models.MediaGroupMapName, msg.MediaGroupID)
	if !ok {

		message := &models.TgChMessage{
			MetadataID: []models.TgMetaPair{metaPair},
		}

		h.ac.SetToMap(
			models.MediaGroupMapName,
			msg.MediaGroupID,
			message,
			mediaGroupMsgWaitingTime,
		)

	} else {

		existingMsg := resp.(*models.TgChMessage)

		h.ac.Mutex(models.MediaGroupMapName, func() {

			existingMsg.MetadataID = append(existingMsg.MetadataID, metaPair)

		})

		h.ac.SetToMap(
			models.MediaGroupMapName,
			msg.MediaGroupID,
			existingMsg,
			mediaGroupMsgWaitingTime,
		)
	}

	return nil
}

func getMetadataFromTgMsg(msg *tgbotapi.Message) models.TgMetaPair {
	switch {
	case msg.Photo != nil:
		return models.TgMetaPair{
			ID:   msg.Photo[len(msg.Photo)-1].FileID,
			Type: models.MsgPhoto,
		}

	case msg.Video != nil:
		return models.TgMetaPair{
			ID:   msg.Video.FileID,
			Type: models.MsgVideo,
		}

	case msg.Audio != nil:
		return models.TgMetaPair{
			ID:   msg.Audio.FileID,
			Type: models.MsgAudio,
		}

	case msg.Document != nil:
		return models.TgMetaPair{
			ID:   msg.Document.FileID,
			Type: models.MsgDocument,
		}

	default:
		return models.TgMetaPair{}
	}
}

const mediaBucket = models.MediaBucket

func (h *Handler) loadMetaByTgID(fileID string, timeout time.Duration) (string, error) {
	const fn = "web-socket.loadMetaByTgID"

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	file, err := h.tg.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", e.Wrap(fn, err)
	}

	path := strings.Split(file.FilePath, "/")
	mediaType := path[0]
	fileName := path[1]

	h.log.Debug(fn, slog.String("type", mediaType), slog.String("file", fileName))

	tgUrl := file.Link(h.tg.Token)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tgUrl, nil)
	if err != nil {
		return "", e.Wrap(fn, err)
	}

	req.Close = true

	client := &http.Client{
		Timeout: timeout / 2,
	}

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return "", e.Wrap(fn, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", e.Wrap(fn, fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}

	pr, pw := io.Pipe()
	defer pr.Close()

	go func() {
		defer pw.Close()

		_, err := io.Copy(pw, resp.Body)
		if err != nil {
			pw.CloseWithError(e.Wrap(fn, err))
		}
	}()

	ext := filepath.Ext(fileName)
	contentType := mime.TypeByExtension(ext)

	fileUrl, err := h.fdb.SaveFile(ctx, mediaBucket, fileName, pr, files.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", e.Wrap(fn, err)
	}

	return fileUrl, nil
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
