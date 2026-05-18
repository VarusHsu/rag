package repository

import (
	"context"
	"fmt"

	"gateway/internal/model"

	"gorm.io/gorm"
)

type CreateFileMetadataParams struct {
	UserID      string
	Bucket      string
	ObjectKey   string
	FileName    string
	ContentType string
	FileSize    int64
	Status      string
}

type FileRepository interface {
	Create(ctx context.Context, input CreateFileMetadataParams) (*model.FileMetadata, error)
	GetByID(ctx context.Context, id string) (*model.FileMetadata, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	ListNonEmbeddedText(ctx context.Context, limit int) ([]model.FileMetadata, error)
}

type GormFileRepository struct {
	db *gorm.DB
}

func NewGormFileRepository(db *gorm.DB) *GormFileRepository {
	return &GormFileRepository{db: db}
}

func (r *GormFileRepository) Create(ctx context.Context, input CreateFileMetadataParams) (*model.FileMetadata, error) {
	record := &model.FileMetadata{
		UserID:      input.UserID,
		Bucket:      input.Bucket,
		ObjectKey:   input.ObjectKey,
		FileName:    input.FileName,
		ContentType: input.ContentType,
		FileSize:    input.FileSize,
		Status:      input.Status,
	}

	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return nil, fmt.Errorf("create file metadata: %w", err)
	}

	return record, nil
}

func (r *GormFileRepository) GetByID(ctx context.Context, id string) (*model.FileMetadata, error) {
	var record model.FileMetadata
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		return nil, fmt.Errorf("get file metadata: %w", err)
	}
	return &record, nil
}

func (r *GormFileRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	if err := r.db.WithContext(ctx).Model(&model.FileMetadata{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return fmt.Errorf("update file status: %w", err)
	}
	return nil
}

func (r *GormFileRepository) ListNonEmbeddedText(ctx context.Context, limit int) ([]model.FileMetadata, error) {
	var records []model.FileMetadata

	db := r.db.WithContext(ctx).
		Where("status <> ?", "embedded").
		Where("content_type LIKE ?", "text/%").
		Order("created_at ASC")
	if limit > 0 {
		db = db.Limit(limit)
	}

	if err := db.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list non-embedded text files: %w", err)
	}

	return records, nil
}
