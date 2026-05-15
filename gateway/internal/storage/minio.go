package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type PresignClient interface {
	PresignedPutObject(ctx context.Context, bucketName string, objectName string, expires time.Duration) (*url.URL, error)
	PresignedGetObject(ctx context.Context, bucketName string, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error)
}

type PresignUploader interface {
	PresignPutObject(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error)
	PresignGetObject(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error)
}

type MinIOUploader struct {
	client PresignClient
}

func NewMinIOClient(endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	return client, nil
}

func NewMinIOUploader(client PresignClient) *MinIOUploader {
	return &MinIOUploader{client: client}
}

func (u *MinIOUploader) PresignPutObject(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error) {
	url, err := u.client.PresignedPutObject(ctx, bucket, objectKey, expires)
	if err != nil {
		return "", fmt.Errorf("presign put object: %w", err)
	}
	return url.String(), nil
}

func (u *MinIOUploader) PresignGetObject(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error) {
	u2, err := u.client.PresignedGetObject(ctx, bucket, objectKey, expires, nil)
	if err != nil {
		return "", fmt.Errorf("presign get object: %w", err)
	}
	return u2.String(), nil
}
