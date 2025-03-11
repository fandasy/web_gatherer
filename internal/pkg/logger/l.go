package logger

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"project/internal/config"
	"strings"
	"time"
)

const (
	outputConsole = "console"
	outputFile    = "file"

	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func Setup(cfg *config.Slog) (*slog.Logger, error) {
	var (
		log *slog.Logger
		w   io.Writer
		err error
	)

	switch cfg.Output {
	case outputFile:
		w, err = createLogFile()
		if err != nil {
			return nil, err
		}
	case outputConsole:
		w = os.Stdout
	default:
		w = os.Stdout
	}

	switch cfg.Env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)

	case envDev:
		log = slog.New(
			slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)

	case envProd:
		log = slog.New(
			slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)

	default:
		log = slog.Default()
	}

	return log, nil
}

func SetSessionName(log *slog.Logger) *slog.Logger {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	keyLength := 10

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	shortKey := make([]byte, keyLength)
	for i := range shortKey {
		shortKey[i] = charset[r.Intn(len(charset)-1)]
	}

	return log.With(slog.String("Session", string(shortKey)))
}

func createLogFile() (*os.File, error) {
	logDir := "logs"

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.Mkdir(logDir, 0774); err != nil {
			return nil, fmt.Errorf("can't create a logs dir: %w", err)
		}
	}

	nowDate := time.Now().Format(time.DateOnly)
	nowTime := strings.ReplaceAll(time.Now().Format(time.TimeOnly), ":", ".")

	file, err := os.Create(logDir + "/" + nowDate + "_" + nowTime + ".log")
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return file, nil
}
