package model

import "time"

// FileMetadata tracks presigned upload tasks and object metadata.
type FileMetadata struct {
	ID          string     `json:"id" gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID      string     `json:"user_id" gorm:"column:user_id;type:uuid;not null"`
	Bucket      string     `json:"bucket" gorm:"column:bucket;not null"`
	ObjectKey   string     `json:"object_key" gorm:"column:object_key;not null;uniqueIndex"`
	FileName    string     `json:"file_name" gorm:"column:file_name;not null"`
	ContentType string     `json:"content_type" gorm:"column:content_type;not null"`
	FileSize    int64      `json:"file_size" gorm:"column:file_size;not null"`
	Status      string     `json:"status" gorm:"column:status;not null"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at"`
	UploadedAt  *time.Time `json:"uploaded_at,omitempty" gorm:"column:uploaded_at"`
}

func (FileMetadata) TableName() string {
	return "file_metadata"
}
