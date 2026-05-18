package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gateway/internal/middleware"
	"gateway/internal/response"
	"gateway/internal/security"
	"gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type mockFileUploadService struct {
	createUploadFn    func(ctx context.Context, input service.CreateUploadInput) (*service.CreateUploadResult, error)
	confirmUploadFn   func(ctx context.Context, input service.ConfirmUploadInput) error
	compensateEmbedFn func(ctx context.Context, input service.CompensateEmbeddingInput) (*service.CompensateEmbeddingResult, error)
}

func (m *mockFileUploadService) CreateUpload(ctx context.Context, input service.CreateUploadInput) (*service.CreateUploadResult, error) {
	if m.createUploadFn == nil {
		return nil, errors.New("createUploadFn not implemented")
	}
	return m.createUploadFn(ctx, input)
}

func (m *mockFileUploadService) ConfirmUpload(ctx context.Context, input service.ConfirmUploadInput) error {
	if m.confirmUploadFn == nil {
		return nil
	}
	return m.confirmUploadFn(ctx, input)
}

func (m *mockFileUploadService) CompensateEmbedding(ctx context.Context, input service.CompensateEmbeddingInput) (*service.CompensateEmbeddingResult, error) {
	if m.compensateEmbedFn == nil {
		return &service.CompensateEmbeddingResult{}, nil
	}
	return m.compensateEmbedFn(ctx, input)
}

func TestFileHandler_CreateUploadSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFileHandler(&mockFileUploadService{createUploadFn: func(ctx context.Context, input service.CreateUploadInput) (*service.CreateUploadResult, error) {
		return &service.CreateUploadResult{
			FileID:          "f1",
			Bucket:          "files",
			ObjectKey:       "uploads/u1/f1.pdf",
			UploadURL:       "http://minio.local/upload",
			UploadMethod:    "PUT",
			ExpiresInSecond: 900,
		}, nil
	}})

	r := gin.New()
	r.Use(middleware.RequestTrace())
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextClaims, &security.Claims{UserID: "u1"})
		c.Next()
	})
	r.POST("/files/presign-upload", h.CreateUpload)

	body := `{"file_name":"test.pdf","content_type":"application/pdf","file_size":123}`
	req := httptest.NewRequest(http.MethodPost, "/files/presign-upload", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
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
	data := resp["data"].(map[string]any)
	if data["upload_url"] == "" {
		t.Fatal("expected upload_url")
	}
}

func TestFileHandler_CreateUploadUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFileHandler(&mockFileUploadService{createUploadFn: func(ctx context.Context, input service.CreateUploadInput) (*service.CreateUploadResult, error) {
		return nil, service.ErrUnauthorized
	}})

	r := gin.New()
	r.Use(middleware.RequestTrace())
	r.POST("/files/presign-upload", h.CreateUpload)

	body := `{"file_name":"test.pdf","content_type":"application/pdf","file_size":123}`
	req := httptest.NewRequest(http.MethodPost, "/files/presign-upload", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestFileHandler_ConfirmUploadSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFileHandler(&mockFileUploadService{confirmUploadFn: func(ctx context.Context, input service.ConfirmUploadInput) error {
		if input.FileID != "f1" {
			t.Fatalf("expected file id f1, got %s", input.FileID)
		}
		if input.Claims == nil || input.Claims.UserID != "u1" {
			t.Fatalf("expected claims user u1, got %#v", input.Claims)
		}
		return nil
	}})

	r := gin.New()
	r.Use(middleware.RequestTrace())
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextClaims, &security.Claims{UserID: "u1"})
		c.Next()
	})
	r.POST("/files/:file_id/confirm-upload", h.ConfirmUpload)

	req := httptest.NewRequest(http.MethodPost, "/files/f1/confirm-upload", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestFileHandler_CompensateEmbeddingSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFileHandler(&mockFileUploadService{compensateEmbedFn: func(ctx context.Context, input service.CompensateEmbeddingInput) (*service.CompensateEmbeddingResult, error) {
		if input.Claims == nil || input.Claims.Role != "admin" {
			t.Fatalf("expected admin claims, got %#v", input.Claims)
		}
		if input.Limit != 50 {
			t.Fatalf("expected limit 50, got %d", input.Limit)
		}
		return &service.CompensateEmbeddingResult{Scanned: 10, Requeued: 9, Failed: 1, FailedFileIDs: []string{"f-bad"}}, nil
	}})

	r := gin.New()
	r.Use(middleware.RequestTrace())
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextClaims, &security.Claims{UserID: "admin-1", Role: "admin"})
		c.Next()
	})
	r.POST("/files/compensate-embedding", h.CompensateEmbedding)

	req := httptest.NewRequest(http.MethodPost, "/files/compensate-embedding", bytes.NewBufferString(`{"limit":50}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
}
