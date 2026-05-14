package handler

import (
	"gateway/internal/middleware"
	"gateway/internal/response"

	"github.com/gin-gonic/gin"
)

func writeSuccess(c *gin.Context, status int, data any) {
	response.Success(c, status, middleware.GetRequestID(c), data)
}

func writeError(c *gin.Context, status int, businessCode int, message string) {
	response.Error(c, status, businessCode, middleware.GetRequestID(c), message)
}
