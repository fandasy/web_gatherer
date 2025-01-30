package files

import (
	"context"
	"errors"
	"github.com/minio/minio-go/v7"
)

type Files interface {
	NewBucket(ctx context.Context, bucketName string, options minio.MakeBucketOptions) error
	InsertFile(ctx context.Context, bucketName, fileName string, content []byte, options minio.PutObjectOptions) (string, error)
	GetFile(ctx context.Context, bucketName, fileName string, options minio.GetObjectOptions) ([]byte, error)
	GetFileUrl(bucketName, fileName string) (string, error)
	DeleteFile(ctx context.Context, bucketName, fileName string, options minio.RemoveObjectOptions) error
}

var (
	ErrBucketIsExists = errors.New("bucket is exists")
)
