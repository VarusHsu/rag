package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gateway/internal/model"

	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")

type CreateUserParams struct {
	Username     string
	Email        string
	Phone        *string
	PasswordHash string
}

type UserRepository interface {
	Create(ctx context.Context, input CreateUserParams) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateLastLogin(ctx context.Context, userID string, at time.Time) error
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(ctx context.Context, input CreateUserParams) (*model.User, error) {
	user := &model.User{
		Username:     input.Username,
		Email:        strings.ToLower(input.Email),
		Phone:        input.Phone,
		PasswordHash: input.PasswordHash,
		Role:         "user",
		Status:       "active",
	}

	err := r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (r *GormUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	err := r.db.WithContext(ctx).Where("email = ?", strings.ToLower(email)).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}

func (r *GormUserRepository) UpdateLastLogin(ctx context.Context, userID string, at time.Time) error {
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"last_login_at": at, "updated_at": at}).Error
	if err != nil {
		return fmt.Errorf("update last_login_at: %w", err)
	}
	return nil
}
