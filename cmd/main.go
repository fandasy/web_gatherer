package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"project/internal/clients/custom_tg_bot"
	"project/internal/clients/vk_api"
	"project/internal/server/telegram"
	"project/internal/server/telegram/events"
	"project/internal/server/web"
	"project/internal/server/web/handlers"
	"project/internal/storer"
	app_cache "project/internal/storer/app-cache"
	"project/internal/storer/cache/rds"
	"project/internal/storer/files/minio"
	"project/internal/storer/storage/psql"
	"syscall"

	"project/internal/clients/telegram_tdlib"
	"project/internal/clients/tg_bot"
	"project/internal/config"
	"project/internal/pkg/logger"
)

func main() {
	cfg, err := config.Load(mustGetFlags())
	if err != nil {
		panic(err)
	}

	log, err := logger.Setup(cfg.Slog)
	if err != nil {
		panic(err)
	}

	bot, err := tg_bot.New(cfg.Telegram.Token, log)
	if err != nil {
		panic(err)
	}

	customBot := custom_tg_bot.New("api.telegram.org", cfg.Telegram.Token)

	tdlibClient, err := telegram_tdlib.New(cfg.Telegram, log)
	if err != nil {
		panic(err)
	}

	vkApi, err := vk_api.New(cfg.VkApi.Token, log)
	if err != nil {
		panic(err)
	}

	appCache := app_cache.New()

	storage, err := psql.New(context.TODO(), cfg.Storage, cfg.MPath, log)
	if err != nil {
		panic(err)
	}

	cache, err := rds.New(context.TODO(), cfg.Redis, log)
	if err != nil {
		panic(err)
	}

	files, err := minio.New(context.TODO(), cfg.Files, log)
	if err != nil {
		panic(err)
	}

	s, err := storer.New(storage, cache, files, appCache, cfg, log)
	if err != nil {
		panic(err)
	}

	// Telegram server
	processor := events.NewProcessor(bot, customBot, tdlibClient, vkApi, s, log)

	tgSrv := telegram.NewServer(bot, processor, log)

	go tgSrv.Listener(cfg.Telegram.Timeout)

	// Web UI server
	handler := handlers.New(s, log)

	webSrv := web.New(handler, cfg.WebServer, log)

	go webSrv.Listener()

	// Server shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	sign := <-stop
	log.Info("got signal", slog.String("signal", sign.String()))

	tgSrv.Shutdown(context.TODO())

	webSrv.Shutdown(context.TODO())
}

func mustGetFlags() string {
	var path string

	flag.StringVar(&path,
		"config",
		"",
		"config file path",
	)
	flag.Parse()

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
		if path == "" {
			panic("config file path is required")
		}
	}

	return path
}
