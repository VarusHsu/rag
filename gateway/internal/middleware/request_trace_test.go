package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestTrace_GeneratesAndEchoesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestTrace())
	r.GET("/healthz", func(c *gin.Context) {
		if GetRequestID(c) == "" {
			t.Fatal("expected request id in context")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get(HeaderRequestID); got == "" {
		t.Fatal("expected response header X-Request-Id")
	}
}

func TestRequestTrace_UsesIncomingRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestTrace())
	r.GET("/healthz", func(c *gin.Context) {
		if got := GetRequestID(c); got != "req-from-client" {
			t.Fatalf("expected request id req-from-client, got %s", got)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(HeaderRequestID, "req-from-client")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get(HeaderRequestID); got != "req-from-client" {
		t.Fatalf("expected response header req-from-client, got %s", got)
	}
}
