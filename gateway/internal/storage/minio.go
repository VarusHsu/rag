package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"
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
	client        PresignClient
	publicBaseURL string
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

func NewMinIOUploader(client PresignClient, publicBaseURL string) *MinIOUploader {
	return &MinIOUploader{client: client, publicBaseURL: strings.TrimSpace(publicBaseURL)}
}

func (u *MinIOUploader) PresignPutObject(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error) {
	presignedURL, err := u.client.PresignedPutObject(ctx, bucket, objectKey, expires)
	if err != nil {
		return "", fmt.Errorf("presign put object: %w", err)
	}
	if u.publicBaseURL == "" {
		return presignedURL.String(), nil
	}
	rewrittenURL, err := rewritePresignedURL(presignedURL, u.publicBaseURL)
	if err != nil {
		return "", err
	}
	return rewrittenURL, nil
}

func (u *MinIOUploader) PresignGetObject(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error) {
	u2, err := u.client.PresignedGetObject(ctx, bucket, objectKey, expires, nil)
	if err != nil {
		return "", fmt.Errorf("presign get object: %w", err)
	}
	return u2.String(), nil
}

func rewritePresignedURL(source *url.URL, publicBaseURL string) (string, error) {
	base, err := url.Parse(publicBaseURL)
	if err != nil {
		return "", fmt.Errorf("parse MINIO_PUBLIC_URL: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("MINIO_PUBLIC_URL must include scheme and host")
	}

	rewritten := *source
	rewritten.Scheme = base.Scheme
	rewritten.Host = base.Host
	if base.Path != "" && base.Path != "/" {
		rewritten.Path = strings.TrimRight(base.Path, "/") + source.Path
	}

	return rewritten.String(), nil
}
