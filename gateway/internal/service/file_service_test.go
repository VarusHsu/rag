package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gateway/internal/messaging"
	"gateway/internal/model"
	"gateway/internal/repository"
	"gateway/internal/security"
)

type mockFileRepo struct {
	createFn              func(ctx context.Context, input repository.CreateFileMetadataParams) (*model.FileMetadata, error)
	getByIDFn             func(ctx context.Context, id string) (*model.FileMetadata, error)
	updateStatusFn        func(ctx context.Context, id string, status string) error
	listNonEmbeddedTextFn func(ctx context.Context, limit int) ([]model.FileMetadata, error)
}

func (m *mockFileRepo) Create(ctx context.Context, input repository.CreateFileMetadataParams) (*model.FileMetadata, error) {
	if m.createFn == nil {
		return nil, errors.New("createFn not implemented")
	}
	return m.createFn(ctx, input)
}

func (m *mockFileRepo) GetByID(ctx context.Context, id string) (*model.FileMetadata, error) {
	if m.getByIDFn == nil {
		return nil, errors.New("getByIDFn not implemented")
	}
	return m.getByIDFn(ctx, id)
}

func (m *mockFileRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	if m.updateStatusFn == nil {
		return nil
	}
	return m.updateStatusFn(ctx, id, status)
}

func (m *mockFileRepo) ListNonEmbeddedText(ctx context.Context, limit int) ([]model.FileMetadata, error) {
	if m.listNonEmbeddedTextFn == nil {
		return nil, nil
	}
	return m.listNonEmbeddedTextFn(ctx, limit)
}

type mockPresignUploader struct {
	presignPutFn func(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error)
	presignGetFn func(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error)
}

func (m *mockPresignUploader) PresignPutObject(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error) {
	if m.presignPutFn == nil {
		return "", errors.New("presignPutFn not implemented")
	}
	return m.presignPutFn(ctx, bucket, objectKey, expires)
}

func (m *mockPresignUploader) PresignGetObject(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error) {
	if m.presignGetFn == nil {
		return "", errors.New("presignGetFn not implemented")
	}
	return m.presignGetFn(ctx, bucket, objectKey, expires)
}

type mockPublisher struct {
	publishFn func(ctx context.Context, msg messaging.UploadMessage) error
}

func (m *mockPublisher) Publish(ctx context.Context, msg messaging.UploadMessage) error {
	if m.publishFn == nil {
		return errors.New("publishFn not implemented")
	}
	return m.publishFn(ctx, msg)
}

func TestFileService_CreateUploadSuccess(t *testing.T) {
	repo := &mockFileRepo{createFn: func(ctx context.Context, input repository.CreateFileMetadataParams) (*model.FileMetadata, error) {
		if input.UserID != "u1" {
			t.Fatalf("expected user u1, got %s", input.UserID)
		}
		return &model.FileMetadata{ID: "file-1"}, nil
	}}
	uploader := &mockPresignUploader{presignPutFn: func(ctx context.Context, bucket, objectKey string, expires time.Duration) (string, error) {
		if bucket != "files" {
			t.Fatalf("expected bucket files, got %s", bucket)
		}
		if objectKey == "" {
			t.Fatal("expected non-empty object key")
		}
		return "http://minio.local/upload", nil
	}}

	svc := NewFileService(repo, uploader, &mockPublisher{}, "files", 15*time.Minute)
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
	svc := NewFileService(&mockFileRepo{}, &mockPresignUploader{}, &mockPublisher{}, "files", 15*time.Minute)
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

func TestFileService_ConfirmUploadSuccess(t *testing.T) {
	repo := &mockFileRepo{getByIDFn: func(ctx context.Context, id string) (*model.FileMetadata, error) {
		if id != "file-1" {
			t.Fatalf("expected file-1, got %s", id)
		}
		return &model.FileMetadata{
			ID:          "file-1",
			UserID:      "u1",
			Bucket:      "files",
			ObjectKey:   "uploads/u1/test.txt",
			FileName:    "test.txt",
			ContentType: "text/plain",
			FileSize:    12,
		}, nil
	}}
	uploader := &mockPresignUploader{presignGetFn: func(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error) {
		if bucket != "files" || objectKey != "uploads/u1/test.txt" {
			t.Fatalf("unexpected get input bucket=%s objectKey=%s", bucket, objectKey)
		}
		return "http://minio.internal/files/uploads/u1/test.txt?sig=1", nil
	}}
	publisher := &mockPublisher{publishFn: func(ctx context.Context, uploadMsg messaging.UploadMessage) error {
		if uploadMsg.FileURL == "" || uploadMsg.FileID != "file-1" || uploadMsg.UserID != "u1" {
			t.Fatalf("unexpected publish message: %#v", uploadMsg)
		}
		return nil
	}}

	svc := NewFileService(repo, uploader, publisher, "files", 15*time.Minute)
	err := svc.ConfirmUpload(context.Background(), ConfirmUploadInput{
		Claims: &security.Claims{UserID: "u1"},
		FileID: "file-1",
	})
	if err != nil {
		t.Fatalf("ConfirmUpload() error = %v", err)
	}
}

func TestFileService_ConfirmUploadUnauthorizedForOtherUser(t *testing.T) {
	repo := &mockFileRepo{getByIDFn: func(ctx context.Context, id string) (*model.FileMetadata, error) {
		return &model.FileMetadata{ID: id, UserID: "other-user"}, nil
	}}

	svc := NewFileService(repo, &mockPresignUploader{}, &mockPublisher{}, "files", 15*time.Minute)
	err := svc.ConfirmUpload(context.Background(), ConfirmUploadInput{
		Claims: &security.Claims{UserID: "u1"},
		FileID: "file-1",
	})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestFileService_CompensateEmbeddingSuccess(t *testing.T) {
	var published int
	repo := &mockFileRepo{listNonEmbeddedTextFn: func(ctx context.Context, limit int) ([]model.FileMetadata, error) {
		if limit != 2 {
			t.Fatalf("expected limit 2, got %d", limit)
		}
		return []model.FileMetadata{
			{ID: "f1", UserID: "u1", FileName: "a.txt", ContentType: "text/plain", FileSize: 10, Bucket: "files", ObjectKey: "uploads/u1/a.txt", Status: "pending_upload"},
			{ID: "f2", UserID: "u2", FileName: "b.md", ContentType: "text/markdown", FileSize: 20, Bucket: "files", ObjectKey: "uploads/u2/b.md", Status: "failed"},
		}, nil
	}}
	uploader := &mockPresignUploader{presignGetFn: func(ctx context.Context, bucket string, objectKey string, expires time.Duration) (string, error) {
		return "http://minio.internal/" + objectKey, nil
	}}
	publisher := &mockPublisher{publishFn: func(ctx context.Context, uploadMsg messaging.UploadMessage) error {
		published++
		if uploadMsg.FileID == "" || uploadMsg.FileURL == "" {
			t.Fatalf("unexpected publish payload: %#v", uploadMsg)
		}
		return nil
	}}

	svc := NewFileService(repo, uploader, publisher, "files", 15*time.Minute)
	result, err := svc.CompensateEmbedding(context.Background(), CompensateEmbeddingInput{
		Claims: &security.Claims{UserID: "admin-1", Role: "admin"},
		Limit:  2,
	})
	if err != nil {
		t.Fatalf("CompensateEmbedding() error = %v", err)
	}
	if result.Scanned != 2 || result.Requeued != 2 || result.Failed != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if published != 2 {
		t.Fatalf("expected 2 publishes, got %d", published)
	}
}

func TestFileService_CompensateEmbeddingForbiddenForNonAdmin(t *testing.T) {
	svc := NewFileService(&mockFileRepo{}, &mockPresignUploader{}, &mockPublisher{}, "files", 15*time.Minute)
	_, err := svc.CompensateEmbedding(context.Background(), CompensateEmbeddingInput{
		Claims: &security.Claims{UserID: "u1", Role: "user"},
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
