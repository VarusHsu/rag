package router

import (
	"net/http"
	"time"

	"gateway/internal/handler"
	"gateway/internal/middleware"
	"gateway/internal/response"
	"gateway/internal/security"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(
	authHandler *handler.AuthHandler,
	jwtManager *security.JWTManager,
	tokenRevoker security.TokenRevoker,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestTrace())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", middleware.HeaderRequestID},
		ExposeHeaders:    []string{middleware.HeaderRequestID},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/healthz", func(c *gin.Context) {
		response.Success(c, http.StatusOK, middleware.GetRequestID(c), gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", middleware.RequireAuth(jwtManager, tokenRevoker), authHandler.Logout)
	}

	return r
}
