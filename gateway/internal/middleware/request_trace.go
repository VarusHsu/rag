package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"gateway/internal/logx"

	"github.com/gin-gonic/gin"
)

const (
	HeaderRequestID  = "X-Request-Id"
	ContextRequestID = "request_id"
)

// RequestTrace ensures every request has a request id and logs it for tracing.
func RequestTrace() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(HeaderRequestID))
		if requestID == "" {
			requestID = newRequestID()
		}

		c.Set(ContextRequestID, requestID)
		c.Writer.Header().Set(HeaderRequestID, requestID)

		start := time.Now()
		c.Next()

		latency := time.Since(start)
		logx.Info("http_request", logx.Fields{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"latency_ms": latency.Milliseconds(),
			"client_ip":  c.ClientIP(),
		})
	}
}

func GetRequestID(c *gin.Context) string {
	value, ok := c.Get(ContextRequestID)
	if !ok {
		return ""
	}
	requestID, ok := value.(string)
	if !ok {
		return ""
	}
	return requestID
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}
