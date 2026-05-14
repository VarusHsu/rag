package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gateway/internal/config"
	"gateway/internal/db"
	"gateway/internal/handler"
	"gateway/internal/logx"
	"gateway/internal/repository"
	"gateway/internal/router"
	"gateway/internal/security"
	"gateway/internal/service"
)

func main() {
	defer logx.Sync()

	cfg, err := config.Load()
	if err != nil {
		logx.Fatal("load config failed", logx.Fields{"error": err.Error()})
	}

	ctx := context.Background()
	gormDB, err := db.NewGormDB(ctx, cfg.DatabaseURL)
	if err != nil {
		logx.Fatal("connect database failed", logx.Fields{"error": err.Error()})
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		logx.Fatal("open sql db failed", logx.Fields{"error": err.Error()})
	}
	defer sqlDB.Close()

	userRepo := repository.NewGormUserRepository(gormDB)
	jwtManager := security.NewJWTManager(cfg.JWTSecret, cfg.JWTExpireMinutes)
	authService := service.NewAuthService(userRepo, jwtManager)
	authHandler := handler.NewAuthHandler(authService)

	engine := router.New(authHandler)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logx.Info("gateway listening", logx.Fields{"port": cfg.Port})
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logx.Fatal("server error", logx.Fields{"error": err.Error()})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logx.Error("server shutdown error", logx.Fields{"error": err.Error()})
	}
}
