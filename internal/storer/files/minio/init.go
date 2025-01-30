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
	db   *minio.Client
	log  *slog.Logger
	addr string
}

func New(ctx context.Context, cfg *config.Files, log *slog.Logger) (*Files, error) {
	const fn = "minio.New"

	minioClient, err := minio.New(cfg.LocalAddr, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.KeyID, cfg.Secret, ""),
		Secure: false,
	})
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	return &Files{
		db:   minioClient,
		log:  log,
		addr: cfg.ExternalAddr,
	}, nil
}
