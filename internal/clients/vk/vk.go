package vk

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
	"project/internal/storage"
	"project/pkg/e"
	"time"
)

func (h *Handler) PrepareNewsGatherer(ctx context.Context) error {
	const fn = "vk.PrepareNewsGatherer"

	vkGroups, err := h.db.GetVkGroups(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrNoRecordsFound) {
			return nil
		}

		return e.Wrap(fn, err)
	}

	for _, vkGroup := range vkGroups {
		vkGroup, err := h.validate(vkGroup.Domain)
		if err != nil {
			if errors.Is(err, ErrVkGroupNotFound) || errors.Is(err, ErrVkGroupIsPrivate) {
				h.log.Error(fn,
					sl.Err(err),
					slog.String("domain", vkGroup.Domain),
				)

				continue
			}

			h.log.Error(fn, sl.Err(err))

			continue
		}

		stopCh := make(chan struct{})

		listener := &Listener{
			stopCh,
		}

		h.ls.m[vkGroup.Domain] = listener

		go h.listen(vkGroup, stopCh)
	}

	return nil
}

func (h *Handler) Shutdown(ctx context.Context, domain string) error {
	const fn = "vk.Shutdown"

	h.ls.mu.Lock()
	defer h.ls.mu.Unlock()
	listener, ok := h.ls.m[domain]
	if !ok {
		return e.Wrap(fn, ErrVkGroupNotFound)
	}

	select {
	case listener.stop <- struct{}{}:
		if err := h.db.DeleteVkGroup(ctx, domain); err != nil {
			return e.Wrap(fn, err)
		}
		h.log.Info("[VK GROUP] Listener shutting down", slog.String("domain", domain))
		delete(h.ls.m, domain)
		close(listener.stop)
	default:
	}

	return nil
}

func (h *Handler) ListenStart(ctx context.Context, vkDomain string) error {
	const fn = "vk.ListenStart"

	h.ls.mu.RLock()
	_, ok := h.ls.m[vkDomain]
	if ok {
		h.ls.mu.RUnlock()
		return e.Wrap(fn, ErrVkGroupIsExists)
	}
	h.ls.mu.RUnlock()

	vkGroup, err := h.validate(vkDomain)
	if err != nil {
		return e.Wrap(fn, err)
	}

	h.ls.mu.Lock()
	defer h.ls.mu.Unlock()

	_, ok = h.ls.m[vkDomain]
	if ok {
		return e.Wrap(fn, ErrVkGroupIsExists)
	}

	if err = h.db.InsertVkGroup(ctx, vkGroup); err != nil {
		return e.Wrap(fn, err)
	}

	stopCh := make(chan struct{}, 1)

	listener := &Listener{
		stopCh,
	}

	h.ls.m[vkDomain] = listener

	go h.listen(vkGroup, stopCh)

	return nil
}

func (h *Handler) listen(vkGroup models.VkGroup, stopCh chan struct{}) {
	const fn = "vk.listen"

	log := h.log.With(
		slog.String("TgGroup domain", vkGroup.Domain),
		slog.String("TgGroup name", vkGroup.Name),
	)

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

	nextPollInterval := func(postsReceived bool) time.Duration {
		if postsReceived {
			pollIntervalBase = maxInterval

			return pollIntervalBase

		} else {
			pollIntervalBase -= pollIntervalBase / 3

			if pollIntervalBase < minInterval {
				pollIntervalBase = minInterval
			}

			return pollIntervalBase + time.Duration(rand.Intn(int(pollIntervalBase/10)))
		}
	}

	lastPostID := 0

	for {
		select {
		case <-stopCh:
			h.log.Info("[VK GROUP] Listener shutdown")
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

		if err := h.db.InsertVkMessages(context.TODO(), msgs); err != nil {
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

func (h *Handler) validate(domain string) (models.VkGroup, error) {
	const fn = "vk.validate"

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
