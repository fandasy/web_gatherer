package minio

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
	"net/url"
	"project/internal/storer/files"
	"project/pkg/e"
)

func (f *Files) NewBucket(ctx context.Context, bucketName string, options minio.MakeBucketOptions) error {
	const fn = "minio.NewBucket"

	err := f.db.MakeBucket(ctx, bucketName, options)
	if err != nil {
		exists, errBucketExists := f.db.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			return e.Wrap(fn, files.ErrBucketIsExists)

		} else {
			return e.Wrap(fn, err)
		}
	}

	return nil
}

func (f *Files) InsertFile(ctx context.Context, bucketName, fileName string, content []byte, options minio.PutObjectOptions) (string, error) {
	const fn = "minio.InsertFile"

	_, err := f.db.PutObject(ctx, bucketName, fileName, bytes.NewReader(content), int64(len(content)), options)
	if err != nil {
		return "", e.Wrap(fn, err)
	}

	path, err := url.JoinPath(f.addr, bucketName, fileName)
	if err != nil {
		return "", e.Wrap(fn, err)
	}

	return path, nil
}

func (f *Files) GetFileUrl(bucketName, fileName string) (string, error) {
	const fn = "minio.GetFileUrl"

	path, err := url.JoinPath(f.addr, bucketName, fileName)
	if err != nil {
		return "", e.Wrap(fn, err)
	}

	return path, nil
}

func (f *Files) GetFile(ctx context.Context, bucketName, fileName string, options minio.GetObjectOptions) ([]byte, error) {
	const fn = "minio.GetFile"

	object, err := f.db.GetObject(ctx, bucketName, fileName, options)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(object)
	if err != nil {
		return nil, e.Wrap(fn, err)
	}

	return buf.Bytes(), nil
}

func (f *Files) DeleteFile(ctx context.Context, bucketName, fileName string, options minio.RemoveObjectOptions) error {
	const fn = "minio.DeleteFile"

	err := f.db.RemoveObject(ctx, bucketName, fileName, options)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}
