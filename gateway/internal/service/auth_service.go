package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gateway/internal/model"
	"gateway/internal/repository"
	"gateway/internal/security"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrConflict           = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUnauthorized       = errors.New("unauthorized")
)

type RegisterInput struct {
	Username string
	Email    string
	Phone    *string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	Token string     `json:"token"`
	User  PublicUser `json:"user"`
}

type PublicUser struct {
	ID            string     `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email"`
	Phone         *string    `json:"phone,omitempty"`
	EmailVerified bool       `json:"email_verified"`
	Role          string     `json:"role"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
}

type AuthService struct {
	users        repository.UserRepository
	jwt          *security.JWTManager
	tokenRevoker security.TokenRevoker
}

func NewAuthService(users repository.UserRepository, jwt *security.JWTManager, tokenRevoker ...security.TokenRevoker) *AuthService {
	var revoker security.TokenRevoker
	if len(tokenRevoker) > 0 && tokenRevoker[0] != nil {
		revoker = tokenRevoker[0]
	} else {
		revoker = security.NewInMemoryTokenBlacklist()
	}

	return &AuthService{users: users, jwt: jwt, tokenRevoker: revoker}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Username == "" || input.Email == "" || len(input.Password) < 8 {
		return nil, ErrInvalidInput
	}

	hash, err := security.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, repository.CreateUserParams{
		Username:     input.Username,
		Email:        input.Email,
		Phone:        input.Phone,
		PasswordHash: hash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrConflict
		}
		return nil, err
	}

	token, err := s.jwt.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{Token: token, User: toPublicUser(*user)}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Email == "" || input.Password == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.users.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := security.CheckPassword(input.Password, user.PasswordHash); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now()
	if err := s.users.UpdateLastLogin(ctx, user.ID, now); err == nil {
		user.LastLoginAt = &now
		user.UpdatedAt = now
	}

	token, err := s.jwt.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{Token: token, User: toPublicUser(*user)}, nil
}

func (s *AuthService) Logout(ctx context.Context, token string, claims *security.Claims) error {
	_ = ctx

	if token == "" || claims == nil {
		return ErrUnauthorized
	}

	expiresAt := time.Now().Add(time.Minute)
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	s.tokenRevoker.Revoke(token, expiresAt)
	return nil
}

func toPublicUser(user model.User) PublicUser {
	return PublicUser{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		Phone:         user.Phone,
		EmailVerified: user.EmailVerified,
		Role:          user.Role,
		Status:        user.Status,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
		LastLoginAt:   user.LastLoginAt,
	}
}
