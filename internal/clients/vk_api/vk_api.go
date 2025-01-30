package vk_api

import (
	"github.com/SevereCloud/vksdk/v3/api"
	vk_params "github.com/SevereCloud/vksdk/v3/api/params"
	"log/slog"
	"project/pkg/e"
)

func New(token string, log *slog.Logger) (*api.VK, error) {
	const fn = "vk_api.New"

	vk := api.NewVK(token)

	params := vk_params.NewUsersGetBuilder()
	params.Fields([]string{"id"})

	_, err := vk.UsersGet(params.Params)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	log.Info("[OK] vk_api successfully connected")

	return vk, nil
}
