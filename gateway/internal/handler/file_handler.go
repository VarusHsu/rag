package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"gateway/internal/middleware"
	"gateway/internal/response"
	"gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type FileUploadService interface {
	CreateUpload(ctx context.Context, input service.CreateUploadInput) (*service.CreateUploadResult, error)
}

type FileHandler struct {
	files FileUploadService
}

func NewFileHandler(files FileUploadService) *FileHandler {
	return &FileHandler{files: files}
}

type createUploadRequest struct {
	FileName    string `json:"file_name" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	FileSize    int64  `json:"file_size" binding:"required,gt=0"`
}

func (h *FileHandler) CreateUpload(c *gin.Context) {
	var req createUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.files.CreateUpload(ctx, service.CreateUploadInput{
		Claims:      middleware.GetClaims(c),
		FileName:    req.FileName,
		ContentType: req.ContentType,
		FileSize:    req.FileSize,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnauthorized):
			writeError(c, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		case errors.Is(err, service.ErrInvalidFileInput):
			writeError(c, http.StatusBadRequest, response.CodeInvalidParams, "invalid file input")
		default:
			writeError(c, http.StatusInternalServerError, response.CodeInternalError, "create upload url failed")
		}
		return
	}

	writeSuccess(c, http.StatusOK, result)
}
