package router

import (
	"context"
	"encoding/json"
	"gateway/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gateway/internal/handler"
	"gateway/internal/repository"
	"gateway/internal/response"
	"gateway/internal/security"
	"gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type noopUserRepo struct{}

func (n *noopUserRepo) Create(ctx context.Context, input repository.CreateUserParams) (*model.User, error) {
	return nil, repository.ErrUserNotFound
}

func (n *noopUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return nil, repository.ErrUserNotFound
}

func (n *noopUserRepo) UpdateLastLogin(ctx context.Context, userID string, at time.Time) error {
	return nil
}

func TestHealthzReturnsEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwt := security.NewJWTManager("test-secret", 60)
	blacklist := security.NewInMemoryTokenBlacklist()
	authHandler := handler.NewAuthHandler(service.NewAuthService(&noopUserRepo{}, jwt, blacklist))
	fileHandler := handler.NewFileHandler(nil)
	r := New(authHandler, fileHandler, jwt, blacklist)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if got := int(resp["code"].(float64)); got != response.CodeSuccess {
		t.Fatalf("expected business code 0, got %d", got)
	}
}
