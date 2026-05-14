package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ok", func(c *gin.Context) {
		Success(c, http.StatusOK, "req-1", gin.H{"hello": "world"})
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if resp.Code != CodeSuccess {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if resp.Msg != "success" {
		t.Fatalf("expected success msg, got %s", resp.Msg)
	}
}
