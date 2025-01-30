package chat

import (
	"context"
	"errors"
	"fmt"
	"github.com/SevereCloud/vksdk/v3/api"
	vk_params "github.com/SevereCloud/vksdk/v3/api/params"
	"github.com/SevereCloud/vksdk/v3/object"
	"log/slog"
	"math/rand"
	"project/internal/models"
	"project/internal/pkg/logger/sl"
	storage "project/internal/storer/storage"
	"project/pkg/e"
	"time"
)

var (
	ErrVkGroupIsPrivate = errors.New("vk group is private")
	ErrVkGroupNotFound  = errors.New("vk group not found")
	ErrVkGroupIsExists  = errors.New("vk group is exists")
)

func (h *Handler) PrepareGroupNewsGatherer() {
	const fn = "events.PrepareGroupNewsGatherer"

	vkGroups, err := h.s.DB.GetVkGroups(context.TODO())
	if err != nil {
		if errors.Is(err, storage.ErrNoRecordsFound) {
			return
		}

		h.log.Error(fn, sl.Err(err))
		return
	}

	for _, vkGroup := range vkGroups {
		vkGroup, err := h.validateVkGroup(vkGroup.Domain)
		if err != nil {
			if errors.Is(err, ErrVkGroupNotFound) || errors.Is(err, ErrVkGroupIsPrivate) {
				h.log.Error(fn,
					sl.Err(err),
					slog.String("vk_group_id", vkGroup.Domain),
				)

				continue
			}

			h.log.Error(fn, sl.Err(err))

			continue
		}

		go h.vkGroupNewsListener(vkGroup)
	}
}

func vkGroupNewsListenerShutdown(stopCh chan struct{}) {
	stopCh <- struct{}{}
}

func (h *Handler) vkGroupNewsListener(vkGroup models.VkGroup) {
	const fn = "events.vkGroupNewsListener"

	log := h.log.With(
		slog.String("TgGroup domain", vkGroup.Domain),
		slog.String("TgGroup name", vkGroup.Name),
	)

	stopCh := make(chan struct{})

	h.s.MM.SetToMap(models.VkNewsGroupMapName, vkGroup.Domain, stopCh, 0)

	log.Info("[VK GROUP] Listener started")

	params := vk_params.NewWallGetBuilder()
	params.Domain(vkGroup.Domain)
	params.Count(5)

	const (
		minInterval = 1 * time.Minute
		maxInterval = 6 * time.Hour
	)

	timer := time.NewTimer(time.Nanosecond)

	pollIntervalBase := minInterval
	var maxIntervalUse bool

	nextPollInterval := func(postsReceived bool) time.Duration {
		if postsReceived {
			pollIntervalBase = minInterval

			return pollIntervalBase

		} else {
			pollIntervalBase *= 2

			if pollIntervalBase > maxInterval {
				if maxIntervalUse {
					pollIntervalBase = maxInterval / 2
					maxIntervalUse = false

				} else {
					pollIntervalBase = maxInterval
					maxIntervalUse = true
				}

			}

			return pollIntervalBase + time.Duration(rand.Intn(int(pollIntervalBase/10)))
		}
	}

	lastPostID := 0

	for {
		select {
		case <-stopCh:
			log.Info("[VK GROUP] Listener shutting down")
			return
		case <-timer.C:
		}

		timer.Reset(nextPollInterval(false))

		timeNow := time.Now()

		posts, err := h.vk.WallGet(params.Params)
		if err != nil {
			log.Error(fn, sl.Err(err))

			continue
		}

		if posts.Count == 0 {
			continue
		}

		msgs := make([]models.VkMessage, 0, posts.Count)

		for i := len(posts.Items) - 1; i >= 0; i-- {
			post := posts.Items[i]

			if post.ID <= lastPostID {
				continue
			}

			metadata := getMetadataFromVkMsg(post.Attachments)

			msg := models.VkMessage{
				MessageID: post.ID,
				GroupID:   vkGroup.ID,
				Text:      post.Text,
				Metadata:  metadata,
				CreatedAt: time.Unix(int64(post.Date), 0),
			}

			msgs = append(msgs, msg)
		}

		lastPostID = posts.Items[0].ID

		if len(msgs) == 0 {
			continue
		}

		log.Info("[VK GROUP] new messages received",
			slog.Int("quantity", len(msgs)),
			slog.String("duration", time.Since(timeNow).String()),
		)

		timer.Reset(nextPollInterval(true))

		if err := h.s.DB.InsertVkMessages(context.TODO(), msgs); err != nil {
			if err != nil {
				log.Error(fn, sl.Err(err))
			}
		}
	}
}

func getMetadataFromVkMsg(attachments []object.WallWallpostAttachment) []models.MetaPair {
	var pairs []models.MetaPair

	for _, attachment := range attachments {
		switch attachment.Type {
		case object.AttachmentTypePhoto:
			pairs = append(pairs, models.MetaPair{
				Url:  attachment.Photo.Sizes[len(attachment.Photo.Sizes)-1].URL,
				Type: models.MsgPhoto,
			})

		case object.AttachmentTypeVideo:
			pairs = append(pairs, models.MetaPair{
				Url: fmt.Sprintf(
					"https://vk.com/video_ext.php?oid=%d&id=%d&hd=2",
					attachment.Video.OwnerID,
					attachment.Video.ID,
				),
				Type: models.MsgIframe,
			})

		case object.AttachmentTypeAudio:
			pairs = append(pairs, models.MetaPair{
				Url:  attachment.Audio.URL,
				Type: models.MsgAudio,
			})

		case object.AttachmentTypeDoc:
			pairs = append(pairs, models.MetaPair{
				Url:  attachment.Doc.URL,
				Type: models.MsgDocument,
			})

		default:
		}
	}

	return pairs
}

func (h *Handler) validateVkGroup(domain string) (models.VkGroup, error) {
	const fn = "events.validateVkGroup"

	params := vk_params.NewGroupsGetByIDBuilder()
	params.GroupID(domain)

	res, err := h.vk.GroupsGetByID(params.Params)
	if err != nil {
		if errors.Is(err, api.ErrPermission) {
			return models.VkGroup{}, ErrVkGroupIsPrivate
		}

		return models.VkGroup{}, e.Wrap(fn, err)
	}

	if len(res.Groups) == 0 {
		return models.VkGroup{}, ErrVkGroupNotFound
	}

	groupID := res.Groups[0].ID
	if groupID > 0 {
		groupID *= -1
	}

	return models.VkGroup{
		ID:     groupID,
		Domain: domain,
		Name:   res.Groups[0].Name,
	}, nil
}
