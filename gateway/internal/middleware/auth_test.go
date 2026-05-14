package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gateway/internal/security"

	"github.com/gin-gonic/gin"
)

func TestRequireAuth_AllowsValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager := security.NewJWTManager("test-secret", 60)
	blacklist := security.NewInMemoryTokenBlacklist()
	token, err := jwtManager.GenerateToken("u1", "alice@example.com", "user")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	r := gin.New()
	r.Use(RequestTrace())
	r.Use(RequireAuth(jwtManager, blacklist))
	r.GET("/protected", func(c *gin.Context) {
		if GetToken(c) == "" || GetClaims(c) == nil {
			t.Fatal("expected auth data in context")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestRequireAuth_RejectsRevokedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager := security.NewJWTManager("test-secret", 60)
	blacklist := security.NewInMemoryTokenBlacklist()
	token, err := jwtManager.GenerateToken("u1", "alice@example.com", "user")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	blacklist.Revoke(token, time.Now().Add(time.Minute))

	r := gin.New()
	r.Use(RequestTrace())
	r.Use(RequireAuth(jwtManager, blacklist))
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestExtractBearerToken(t *testing.T) {
	if got := extractBearerToken("Bearer token-1"); got != "token-1" {
		t.Fatalf("expected token-1, got %s", got)
	}
	if got := extractBearerToken("invalid"); got != "" {
		t.Fatalf("expected empty token, got %s", got)
	}
}
