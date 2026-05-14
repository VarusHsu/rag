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
