package files

import (
	"errors"
)

var (
	ErrBucketIsExists = errors.New("bucket is exists")
)

type MakeBucketOptions struct{}

type PutObjectOptions struct {
	ContentType string
}

type GetObjectOptions struct{}

type RemoveObjectOptions struct{}
