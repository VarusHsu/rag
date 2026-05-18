package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"gateway/internal/messaging"
	"gateway/internal/repository"
	"gateway/internal/security"
	"gateway/internal/storage"
)

var ErrInvalidFileInput = errors.New("invalid file input")
var ErrFileNotFound = errors.New("file not found")
var ErrForbidden = errors.New("forbidden")

type MessagePublisher interface {
	Publish(ctx context.Context, msg messaging.UploadMessage) error
}

type CreateUploadInput struct {
	Claims      *security.Claims
	FileName    string
	ContentType string
	FileSize    int64
}

type CreateUploadResult struct {
	FileID          string `json:"file_id"`
	Bucket          string `json:"bucket"`
	ObjectKey       string `json:"object_key"`
	UploadURL       string `json:"upload_url"`
	UploadMethod    string `json:"upload_method"`
	ExpiresInSecond int64  `json:"expires_in_seconds"`
}

type ConfirmUploadInput struct {
	Claims *security.Claims
	FileID string
}

type CompensateEmbeddingInput struct {
	Claims *security.Claims
	Limit  int
}

type CompensateEmbeddingResult struct {
	Scanned       int      `json:"scanned"`
	Requeued      int      `json:"requeued"`
	Failed        int      `json:"failed"`
	FailedFileIDs []string `json:"failed_file_ids"`
}

type FileService struct {
	files         repository.FileRepository
	uploader      storage.PresignUploader
	publisher     MessagePublisher
	bucket        string
	presignExpiry time.Duration
}

func NewFileService(
	files repository.FileRepository,
	uploader storage.PresignUploader,
	publisher MessagePublisher,
	bucket string,
	presignExpiry time.Duration,
) *FileService {
	return &FileService{files: files, uploader: uploader, publisher: publisher, bucket: bucket, presignExpiry: presignExpiry}
}

func (s *FileService) CreateUpload(ctx context.Context, input CreateUploadInput) (*CreateUploadResult, error) {
	if input.Claims == nil || strings.TrimSpace(input.Claims.UserID) == "" {
		return nil, ErrUnauthorized
	}

	fileName := strings.TrimSpace(input.FileName)
	contentType := strings.TrimSpace(input.ContentType)
	if fileName == "" || contentType == "" || input.FileSize <= 0 {
		return nil, ErrInvalidFileInput
	}

	objectKey := buildObjectKey(input.Claims.UserID, fileName)

	record, err := s.files.Create(ctx, repository.CreateFileMetadataParams{
		UserID:      input.Claims.UserID,
		Bucket:      s.bucket,
		ObjectKey:   objectKey,
		FileName:    fileName,
		ContentType: contentType,
		FileSize:    input.FileSize,
		Status:      "pending_upload",
	})
	if err != nil {
		return nil, err
	}

	uploadURL, err := s.uploader.PresignPutObject(ctx, s.bucket, objectKey, s.presignExpiry)
	if err != nil {
		return nil, fmt.Errorf("create presigned url: %w", err)
	}

	return &CreateUploadResult{
		FileID:          record.ID,
		Bucket:          s.bucket,
		ObjectKey:       objectKey,
		UploadURL:       uploadURL,
		UploadMethod:    "PUT",
		ExpiresInSecond: int64(s.presignExpiry.Seconds()),
	}, nil
}

func (s *FileService) ConfirmUpload(ctx context.Context, input ConfirmUploadInput) error {
	if input.Claims == nil || strings.TrimSpace(input.Claims.UserID) == "" {
		return ErrUnauthorized
	}

	record, err := s.files.GetByID(ctx, input.FileID)
	if err != nil {
		return ErrFileNotFound
	}

	if record.UserID != input.Claims.UserID {
		return ErrUnauthorized
	}

	fileURL, err := s.uploader.PresignGetObject(ctx, record.Bucket, record.ObjectKey, 30*time.Minute)
	if err != nil {
		return fmt.Errorf("create presigned get url: %w", err)
	}

	return s.publisher.Publish(ctx, messaging.UploadMessage{
		FileID:      record.ID,
		FileName:    record.FileName,
		ContentType: record.ContentType,
		FileSize:    record.FileSize,
		FileURL:     fileURL,
		UserID:      record.UserID,
		Bucket:      record.Bucket,
		ObjectKey:   record.ObjectKey,
	})
}

func (s *FileService) CompensateEmbedding(ctx context.Context, input CompensateEmbeddingInput) (*CompensateEmbeddingResult, error) {
	if input.Claims == nil || strings.TrimSpace(input.Claims.UserID) == "" {
		return nil, ErrUnauthorized
	}
	if input.Claims.Role != "admin" {
		return nil, ErrForbidden
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 200
	}

	records, err := s.files.ListNonEmbeddedText(ctx, limit)
	if err != nil {
		return nil, err
	}

	result := &CompensateEmbeddingResult{Scanned: len(records)}
	for _, record := range records {
		fileURL, presignErr := s.uploader.PresignGetObject(ctx, record.Bucket, record.ObjectKey, 30*time.Minute)
		if presignErr != nil {
			result.Failed++
			result.FailedFileIDs = append(result.FailedFileIDs, record.ID)
			continue
		}

		publishErr := s.publisher.Publish(ctx, messaging.UploadMessage{
			FileID:      record.ID,
			FileName:    record.FileName,
			ContentType: record.ContentType,
			FileSize:    record.FileSize,
			FileURL:     fileURL,
			UserID:      record.UserID,
			Bucket:      record.Bucket,
			ObjectKey:   record.ObjectKey,
		})
		if publishErr != nil {
			result.Failed++
			result.FailedFileIDs = append(result.FailedFileIDs, record.ID)
			continue
		}

		result.Requeued++
	}

	return result, nil
}

func buildObjectKey(userID string, fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	base = sanitizeName(base)
	if base == "" {
		base = "file"
	}
	return fmt.Sprintf("uploads/%s/%d-%s%s", userID, time.Now().UnixNano(), base, ext)
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}
