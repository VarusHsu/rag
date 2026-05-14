package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gateway/internal/middleware"
	"gateway/internal/model"
	"gateway/internal/repository"
	"gateway/internal/security"
	"gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type mockUserRepo struct {
	createFn          func(ctx context.Context, input repository.CreateUserParams) (*model.User, error)
	getByEmailFn      func(ctx context.Context, email string) (*model.User, error)
	updateLastLoginFn func(ctx context.Context, userID string, at time.Time) error
}

func (m *mockUserRepo) Create(ctx context.Context, input repository.CreateUserParams) (*model.User, error) {
	if m.createFn == nil {
		return nil, repository.ErrUserNotFound
	}
	return m.createFn(ctx, input)
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.getByEmailFn == nil {
		return nil, repository.ErrUserNotFound
	}
	return m.getByEmailFn(ctx, email)
}

func (m *mockUserRepo) UpdateLastLogin(ctx context.Context, userID string, at time.Time) error {
	if m.updateLastLoginFn == nil {
		return nil
	}
	return m.updateLastLoginFn(ctx, userID, at)
}

func newTestAuthHandler(repo repository.UserRepository) *AuthHandler {
	jwt := security.NewJWTManager("test-secret", 60)
	svc := service.NewAuthService(repo, jwt)
	return NewAuthHandler(svc)
}

func TestAuthHandler_RegisterValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := newTestAuthHandler(&mockUserRepo{})
	r.POST("/register", h.Register)

	body := `{"username":"alice","email":"alice@example.com","password":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_RegisterSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := &mockUserRepo{
		createFn: func(ctx context.Context, input repository.CreateUserParams) (*model.User, error) {
			return &model.User{
				ID:            "u1",
				Username:      input.Username,
				Email:         input.Email,
				PasswordHash:  input.PasswordHash,
				EmailVerified: false,
				Role:          "user",
				Status:        "active",
			}, nil
		},
	}
	h := newTestAuthHandler(repo)
	r.POST("/register", h.Register)

	body := `{"username":"alice","email":"alice@example.com","password":"12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if resp["token"] == "" {
		t.Fatal("expected token in response")
	}
}

func TestAuthHandler_LoginUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			return nil, repository.ErrUserNotFound
		},
	}
	h := newTestAuthHandler(repo)
	r.POST("/login", h.Login)

	body := `{"email":"alice@example.com","password":"12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_LoginSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	hash, err := security.HashPassword("12345678")
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
		updateLastLoginFn: func(ctx context.Context, userID string, at time.Time) error {
			return nil
		},
	}
	h := newTestAuthHandler(repo)
	r.POST("/login", h.Login)

	body := `{"email":"alice@example.com","password":"12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_RegisterSuccessIncludesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestTrace())
	repo := &mockUserRepo{
		createFn: func(ctx context.Context, input repository.CreateUserParams) (*model.User, error) {
			return &model.User{
				ID:           "u1",
				Username:     input.Username,
				Email:        input.Email,
				PasswordHash: input.PasswordHash,
				Role:         "user",
				Status:       "active",
			}, nil
		},
	}
	h := newTestAuthHandler(repo)
	r.POST("/register", h.Register)

	body := `{"username":"alice","email":"alice@example.com","password":"12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.HeaderRequestID, "req-register-1")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body=%s", w.Code, w.Body.String())
	}

	if got := w.Header().Get(middleware.HeaderRequestID); got != "req-register-1" {
		t.Fatalf("expected response header request id req-register-1, got %s", got)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if got, _ := resp["request_id"].(string); got != "req-register-1" {
		t.Fatalf("expected response request_id req-register-1, got %v", resp["request_id"])
	}
}

func TestAuthHandler_LoginUnauthorizedIncludesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestTrace())
	repo := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			return nil, repository.ErrUserNotFound
		},
	}
	h := newTestAuthHandler(repo)
	r.POST("/login", h.Login)

	body := `{"email":"alice@example.com","password":"12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.HeaderRequestID, "req-login-unauth")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if got, _ := resp["request_id"].(string); got != "req-login-unauth" {
		t.Fatalf("expected response request_id req-login-unauth, got %v", resp["request_id"])
	}
}

func TestAuthHandler_LogoutSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockUserRepo{}
	jwt := security.NewJWTManager("test-secret", 60)
	blacklist := security.NewInMemoryTokenBlacklist()
	svc := service.NewAuthService(repo, jwt, blacklist)
	h := NewAuthHandler(svc)
	r := gin.New()
	r.Use(middleware.RequestTrace())
	r.POST("/logout", middleware.RequireAuth(jwt, blacklist), h.Logout)

	token, err := jwt.GenerateToken("u1", "alice@example.com", "user")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/logout", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(middleware.HeaderRequestID, "req-logout-1")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !blacklist.IsRevoked(token) {
		t.Fatal("expected token to be revoked")
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if got, _ := resp["message"].(string); got != "logout success" {
		t.Fatalf("expected logout success message, got %v", resp["message"])
	}
	if got, _ := resp["request_id"].(string); got != "req-logout-1" {
		t.Fatalf("expected response request_id req-logout-1, got %v", resp["request_id"])
	}
}

func TestAuthHandler_LogoutUnauthorizedWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockUserRepo{}
	jwt := security.NewJWTManager("test-secret", 60)
	blacklist := security.NewInMemoryTokenBlacklist()
	svc := service.NewAuthService(repo, jwt, blacklist)
	h := NewAuthHandler(svc)
	r := gin.New()
	r.Use(middleware.RequestTrace())
	r.POST("/logout", middleware.RequireAuth(jwt, blacklist), h.Logout)

	req := httptest.NewRequest(http.MethodPost, "/logout", http.NoBody)
	req.Header.Set(middleware.HeaderRequestID, "req-logout-unauth")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if got, _ := resp["request_id"].(string); got != "req-logout-unauth" {
		t.Fatalf("expected response request_id req-logout-unauth, got %v", resp["request_id"])
	}
}
