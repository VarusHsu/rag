package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gateway/internal/model"
	"gateway/internal/repository"
	"gateway/internal/security"

	"github.com/jackc/pgx/v5/pgconn"
)

type mockUserRepo struct {
	createFn          func(ctx context.Context, input repository.CreateUserParams) (*model.User, error)
	getByEmailFn      func(ctx context.Context, email string) (*model.User, error)
	updateLastLoginFn func(ctx context.Context, userID string, at time.Time) error
}

func (m *mockUserRepo) Create(ctx context.Context, input repository.CreateUserParams) (*model.User, error) {
	if m.createFn == nil {
		return nil, errors.New("createFn not implemented")
	}
	return m.createFn(ctx, input)
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.getByEmailFn == nil {
		return nil, errors.New("getByEmailFn not implemented")
	}
	return m.getByEmailFn(ctx, email)
}

func (m *mockUserRepo) UpdateLastLogin(ctx context.Context, userID string, at time.Time) error {
	if m.updateLastLoginFn == nil {
		return nil
	}
	return m.updateLastLoginFn(ctx, userID, at)
}

func TestAuthService_RegisterSuccess(t *testing.T) {
	repo := &mockUserRepo{
		createFn: func(ctx context.Context, input repository.CreateUserParams) (*model.User, error) {
			return &model.User{
				ID:            "u1",
				Username:      input.Username,
				Email:         input.Email,
				Phone:         input.Phone,
				PasswordHash:  input.PasswordHash,
				EmailVerified: false,
				Role:          "user",
				Status:        "active",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}, nil
		},
	}
	jwt := security.NewJWTManager("test-secret", 60)
	svc := NewAuthService(repo, jwt)

	res, err := svc.Register(context.Background(), RegisterInput{
		Username: "alice",
		Email:    "Alice@Example.com",
		Password: "12345678",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if res.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if res.User.Email != "alice@example.com" {
		t.Fatalf("expected normalized email, got %s", res.User.Email)
	}
}

func TestAuthService_RegisterConflict(t *testing.T) {
	repo := &mockUserRepo{
		createFn: func(ctx context.Context, input repository.CreateUserParams) (*model.User, error) {
			return nil, &pgconn.PgError{Code: "23505"}
		},
	}
	jwt := security.NewJWTManager("test-secret", 60)
	svc := NewAuthService(repo, jwt)

	_, err := svc.Register(context.Background(), RegisterInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "12345678",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestAuthService_LoginInvalidPassword(t *testing.T) {
	hash, err := security.HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	repo := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			return &model.User{
				ID:           "u1",
				Username:     "alice",
				Email:        "alice@example.com",
				PasswordHash: hash,
				Role:         "user",
				Status:       "active",
			}, nil
		},
	}
	jwt := security.NewJWTManager("test-secret", 60)
	svc := NewAuthService(repo, jwt)

	_, err = svc.Login(context.Background(), LoginInput{
		Email:    "alice@example.com",
		Password: "wrong-password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_LoginSuccess(t *testing.T) {
	hash, err := security.HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	updated := false
	repo := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			return &model.User{
				ID:           "u1",
				Username:     "alice",
				Email:        "alice@example.com",
				PasswordHash: hash,
				Role:         "user",
				Status:       "active",
			}, nil
		},
		updateLastLoginFn: func(ctx context.Context, userID string, at time.Time) error {
			if userID != "u1" {
				t.Fatalf("unexpected userID: %s", userID)
			}
			updated = true
			return nil
		},
	}
	jwt := security.NewJWTManager("test-secret", 60)
	svc := NewAuthService(repo, jwt)

	res, err := svc.Login(context.Background(), LoginInput{
		Email:    "alice@example.com",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if res.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if !updated {
		t.Fatal("expected UpdateLastLogin to be called")
	}
}
