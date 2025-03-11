package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"project/internal/app-cache"
	"project/internal/cache/rds"
	"project/internal/clients/vk"
	"project/internal/clients/vk_api"
	"project/internal/files/minio"
	"project/internal/server"
	"project/internal/server/telegram"
	"project/internal/server/telegram/channel"
	"project/internal/server/telegram/chat"
	"project/internal/server/telegram/group"
	"project/internal/server/web"
	"project/internal/storage/psql"
	"syscall"

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

	log = logger.SetSessionName(log)

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

	tgBot, err := tg_bot.New(cfg.Telegram.Host, cfg.Telegram.Token, log)
	if err != nil {
		panic(err)
	}

	vkApi, err := vk_api.New(cfg.VkApi.Token, log)
	if err != nil {
		panic(err)
	}

	vkHandler := vk.New(vkApi, storage, log)

	if err := server.Prepare(context.TODO(), storage, appCache, vkHandler); err != nil {
		panic(err)
	}

	// Telegram server
	processor := telegram.NewProcessor(
		chat.NewHandler(tgBot, vkHandler, storage, cache, appCache, log),
		group.NewHandler(tgBot, storage, cache, appCache, files, log),
		channel.NewHandler(tgBot, storage, cache, appCache, files, log),
	)

	tgSrv := telegram.NewServer(tgBot, processor, log)

	go tgSrv.Listener(cfg.Telegram.Timeout)

	// Web UI server
	handlers := web.NewHandler(storage, log)

	webSrv := web.NewServer(cfg.WebServer, log)

	go webSrv.Listener(handlers)

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
