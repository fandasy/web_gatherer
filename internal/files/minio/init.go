package minio

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log/slog"
	"project/internal/config"
	"project/pkg/e"
)

type Files struct {
	db  *minio.Client
	log *slog.Logger
}

func New(ctx context.Context, cfg *config.Files, log *slog.Logger) (*Files, error) {
	const fn = "minio.New"

	minioClient, err := minio.New(cfg.Addr, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.KeyID, cfg.Secret, ""),
		Secure: false,
	})
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	// Проверка клиента
	_, err = minioClient.BucketExists(ctx, "test")
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	log.Info(fn, slog.String("[OK]", "Minio successfully started"))

	return &Files{
		db:  minioClient,
		log: log,
	}, nil
}
