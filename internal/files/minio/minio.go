package minio

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
	"io"
	"net/url"
	"project/internal/files"
	"project/pkg/e"
)

func (f *Files) NewBucket(ctx context.Context, bucketName string, options files.MakeBucketOptions) error {
	const fn = "minio.NewBucket"

	err := f.db.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
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

func (f *Files) SaveFile(ctx context.Context, bucketName, fileName string, reader io.Reader, options files.PutObjectOptions) (string, error) {
	const fn = "minio.InsertFile"

	opts := minio.PutObjectOptions{
		ContentType: options.ContentType,
	}

	info, err := f.db.PutObject(ctx, bucketName, fileName, reader, -1, opts)
	if err != nil {
		return "", e.Wrap(fn, err)
	}

	return info.Location, nil
}

func (f *Files) GetFileUrl(bucketName, fileName string) (string, error) {
	const fn = "minio.GetFileUrl"

	res, err := url.JoinPath(f.db.EndpointURL().String(), bucketName, fileName)
	if err != nil {
		return "", e.Wrap(fn, err)
	}

	return res, nil
}

func (f *Files) GetFile(ctx context.Context, bucketName, fileName string, options files.GetObjectOptions) ([]byte, error) {
	const fn = "minio.GetFile"

	object, err := f.db.GetObject(ctx, bucketName, fileName, minio.GetObjectOptions{})
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

func (f *Files) DeleteFile(ctx context.Context, bucketName, fileName string, options files.RemoveObjectOptions) error {
	const fn = "minio.DeleteFile"

	err := f.db.RemoveObject(ctx, bucketName, fileName, minio.RemoveObjectOptions{})
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}
