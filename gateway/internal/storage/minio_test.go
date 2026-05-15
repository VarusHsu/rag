package storage

import (
	"context"
	"net/url"
	"testing"
	"time"
)

type fakePresignClient struct {
	url *url.URL
	err error
}

func (f *fakePresignClient) PresignedPutObject(_ context.Context, _ string, _ string, _ time.Duration) (*url.URL, error) {
	return f.url, f.err
}

func (f *fakePresignClient) PresignedGetObject(_ context.Context, _ string, _ string, _ time.Duration, _ url.Values) (*url.URL, error) {
	return f.url, f.err
}

func TestMinIOUploaderPresignPutObject(t *testing.T) {
	u, _ := url.Parse("http://minio.local/test-bucket/object")
	uploader := NewMinIOUploader(&fakePresignClient{url: u}, "")

	result, err := uploader.PresignPutObject(context.Background(), "test-bucket", "object", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignPutObject() error = %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty presigned url")
	}
}

func TestMinIOUploaderPresignPutObjectRewritesToPublicURL(t *testing.T) {
	u, _ := url.Parse("http://minio:9000/test-bucket/object?X-Amz-Signature=abc")
	uploader := NewMinIOUploader(&fakePresignClient{url: u}, "http://localhost:9000")

	result, err := uploader.PresignPutObject(context.Background(), "test-bucket", "object", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignPutObject() error = %v", err)
	}
	if result != "http://localhost:9000/test-bucket/object?X-Amz-Signature=abc" {
		t.Fatalf("PresignPutObject() = %q, want rewritten localhost URL", result)
	}
}

func TestMinIOUploaderPresignGetObjectKeepsInternalURL(t *testing.T) {
	u, _ := url.Parse("http://minio:9000/test-bucket/object?X-Amz-Signature=abc")
	uploader := NewMinIOUploader(&fakePresignClient{url: u}, "http://localhost:9000")

	result, err := uploader.PresignGetObject(context.Background(), "test-bucket", "object", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignGetObject() error = %v", err)
	}
	if result != u.String() {
		t.Fatalf("PresignGetObject() = %q, want internal URL %q", result, u.String())
	}
}
