package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gateway/internal/model"
	"gateway/internal/repository"
	"gateway/internal/security"
)

type mockFileRepo struct {
	createFn func(ctx context.Context, input repository.CreateFileMetadataParams) (*model.FileMetadata, error)
}

func (m *mockFileRepo) Create(ctx context.Context, input repository.CreateFileMetadataParams) (*model.FileMetadata, error) {
	if m.createFn == nil {
		return nil, errors.New("createFn not implemented")
	}
	return m.createFn(ctx, input)
}

type mockPresignUploader struct {
	presignFn func(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error)
}

func (m *mockPresignUploader) PresignPutObject(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error) {
	if m.presignFn == nil {
		return "", errors.New("presignFn not implemented")
	}
	return m.presignFn(ctx, bucket, objectKey, expires)
}

func TestFileService_CreateUploadSuccess(t *testing.T) {
	repo := &mockFileRepo{createFn: func(ctx context.Context, input repository.CreateFileMetadataParams) (*model.FileMetadata, error) {
		if input.UserID != "u1" {
			t.Fatalf("expected user u1, got %s", input.UserID)
		}
		return &model.FileMetadata{ID: "file-1"}, nil
	}}
	uploader := &mockPresignUploader{presignFn: func(ctx context.Context, bucket, objectKey string, expires time.Duration) (string, error) {
		if bucket != "files" {
			t.Fatalf("expected bucket files, got %s", bucket)
		}
		if objectKey == "" {
			t.Fatal("expected non-empty object key")
		}
		return "http://minio.local/upload", nil
	}}

	svc := NewFileService(repo, uploader, "files", 15*time.Minute)
	result, err := svc.CreateUpload(context.Background(), CreateUploadInput{
		Claims:      &security.Claims{UserID: "u1"},
		FileName:    "a b.pdf",
		ContentType: "application/pdf",
		FileSize:    123,
	})
	if err != nil {
		t.Fatalf("CreateUpload() error = %v", err)
	}
	if result.FileID != "file-1" {
		t.Fatalf("expected file-1, got %s", result.FileID)
	}
	if result.UploadURL == "" {
		t.Fatal("expected upload url")
	}
}

func TestFileService_CreateUploadInvalidInput(t *testing.T) {
	svc := NewFileService(&mockFileRepo{}, &mockPresignUploader{}, "files", 15*time.Minute)
	_, err := svc.CreateUpload(context.Background(), CreateUploadInput{
		Claims:      &security.Claims{UserID: "u1"},
		FileName:    "",
		ContentType: "application/pdf",
		FileSize:    10,
	})
	if !errors.Is(err, ErrInvalidFileInput) {
		t.Fatalf("expected ErrInvalidFileInput, got %v", err)
	}
}
