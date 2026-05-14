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

func TestMinIOUploaderPresignPutObject(t *testing.T) {
	u, _ := url.Parse("http://minio.local/test-bucket/object")
	uploader := NewMinIOUploader(&fakePresignClient{url: u})

	result, err := uploader.PresignPutObject(context.Background(), "test-bucket", "object", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignPutObject() error = %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty presigned url")
	}
}
