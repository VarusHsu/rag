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
		writeError(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
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
			writeError(c, http.StatusConflict, response.CodeConflict, "username or email already exists")
		case errors.Is(err, service.ErrInvalidInput):
			writeError(c, http.StatusBadRequest, response.CodeInvalidParams, "invalid input")
		default:
			writeError(c, http.StatusInternalServerError, response.CodeInternalError, "register failed")
		}
		return
	}

	writeSuccess(c, http.StatusCreated, gin.H{
		"token": result.Token,
		"user":  result.User,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
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
			writeError(c, http.StatusUnauthorized, response.CodeInvalidCredentials, "invalid email or password")
		case errors.Is(err, service.ErrInvalidInput):
			writeError(c, http.StatusBadRequest, response.CodeInvalidParams, "invalid input")
		default:
			writeError(c, http.StatusInternalServerError, response.CodeInternalError, "login failed")
		}
		return
	}

	writeSuccess(c, http.StatusOK, gin.H{
		"token": result.Token,
		"user":  result.User,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err := h.auth.Logout(ctx, middleware.GetToken(c), middleware.GetClaims(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnauthorized):
			writeError(c, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		default:
			writeError(c, http.StatusInternalServerError, response.CodeInternalError, "logout failed")
		}
		return
	}

	writeSuccess(c, http.StatusOK, gin.H{
		"message": "logout success",
	})
}
