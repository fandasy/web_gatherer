package telegram_tdlib

import (
	"log/slog"
	"path/filepath"
	"project/internal/config"
	"project/pkg/e"
	"strconv"

	"github.com/zelenin/go-tdlib/client"
)

func New(cfg *config.Telegram, log *slog.Logger) (*client.Client, error) {
	const fn = "clients.telegram_tdlib.New"

	apiID, err := strconv.Atoi(cfg.ApiID)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	tdlibParameters := &client.SetTdlibParametersRequest{
		UseTestDc:           false,
		DatabaseDirectory:   filepath.Join(".tdlib", "database"),
		FilesDirectory:      filepath.Join(".tdlib", "files"),
		UseFileDatabase:     true,
		UseChatInfoDatabase: true,
		UseMessageDatabase:  true,
		UseSecretChats:      false,
		ApiId:               int32(apiID),
		ApiHash:             cfg.ApiHash,
		SystemLanguageCode:  "en",
		DeviceModel:         "Server",
		SystemVersion:       "1.0.0",
		ApplicationVersion:  "1.0.0",
	}

	authorizer := client.ClientAuthorizer(tdlibParameters)
	go client.CliInteractor(authorizer)

	_, err = client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 1,
	})
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	tdlibClient, err := client.NewClient(authorizer)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	versionOption, err := client.GetOption(&client.GetOptionRequest{
		Name: "version",
	})
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	commitOption, err := client.GetOption(&client.GetOptionRequest{
		Name: "commit_hash",
	})
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	log.Debug("TDLib",
		slog.String("version", versionOption.(*client.OptionValueString).Value),
		slog.String("commit", commitOption.(*client.OptionValueString).Value),
	)

	me, err := tdlibClient.GetMe()
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	log.Debug("TDLib Me",
		slog.String("First name", me.FirstName),
		slog.String("Last name", me.LastName),
	)

	log.Info("[OK] tg tdlib client created")

	return tdlibClient, nil
}
