package response

import "github.com/gin-gonic/gin"

const (
	CodeSuccess            = 0
	CodeInvalidParams      = 1001
	CodeUnauthorized       = 1002
	CodeConflict           = 1003
	CodeInvalidCredentials = 1004
	CodeInternalError      = 1005
)

type Envelope struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	RequestID string `json:"request_id,omitempty"`
	Data      any    `json:"data,omitempty"`
}

func Success(c *gin.Context, httpStatus int, requestID string, data any) {
	c.JSON(httpStatus, Envelope{
		Code:      CodeSuccess,
		Msg:       "success",
		RequestID: requestID,
		Data:      data,
	})
}

func Error(c *gin.Context, httpStatus int, businessCode int, requestID string, msg string) {
	c.JSON(httpStatus, Envelope{
		Code:      businessCode,
		Msg:       msg,
		RequestID: requestID,
	})
}

func AbortError(c *gin.Context, httpStatus int, businessCode int, requestID string, msg string) {
	c.AbortWithStatusJSON(httpStatus, Envelope{
		Code:      businessCode,
		Msg:       msg,
		RequestID: requestID,
	})
}
