package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"gateway/internal/middleware"
	"gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Username string  `json:"username" binding:"required"`
	Email    string  `json:"email" binding:"required,email"`
	Phone    *string `json:"phone"`
	Password string  `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.auth.Register(ctx, service.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConflict):
			writeError(c, http.StatusConflict, "username or email already exists")
		case errors.Is(err, service.ErrInvalidInput):
			writeError(c, http.StatusBadRequest, "invalid input")
		default:
			writeError(c, http.StatusInternalServerError, "register failed")
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"request_id": middleware.GetRequestID(c),
		"token":      result.Token,
		"user":       result.User,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.auth.Login(ctx, service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(c, http.StatusUnauthorized, "invalid email or password")
		case errors.Is(err, service.ErrInvalidInput):
			writeError(c, http.StatusBadRequest, "invalid input")
		default:
			writeError(c, http.StatusInternalServerError, "login failed")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"request_id": middleware.GetRequestID(c),
		"token":      result.Token,
		"user":       result.User,
	})
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error":      message,
		"request_id": middleware.GetRequestID(c),
	})
}
