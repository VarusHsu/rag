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
		logx.Error("load config failed", logx.Fields{"error": err.Error()})
		logx.Sync()
		os.Exit(1)
	}

	ctx := context.Background()
	gormDB, err := db.NewGormDB(ctx, cfg.DatabaseURL)
	if err != nil {
		logx.Error("connect database failed", logx.Fields{"error": err.Error()})
		logx.Sync()
		os.Exit(1)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		logx.Error("open sql db failed", logx.Fields{"error": err.Error()})
		logx.Sync()
		os.Exit(1)
	}
	defer sqlDB.Close()

	userRepo := repository.NewGormUserRepository(gormDB)
	jwtManager := security.NewJWTManager(cfg.JWTSecret, cfg.JWTExpireMinutes)
	tokenBlacklist := security.NewInMemoryTokenBlacklist()
	authService := service.NewAuthService(userRepo, jwtManager, tokenBlacklist)
	authHandler := handler.NewAuthHandler(authService)

	engine := router.New(authHandler, jwtManager, tokenBlacklist)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logx.Info("gateway listening", logx.Fields{"port": cfg.Port})
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logx.Error("server error", logx.Fields{"error": err.Error()})
			logx.Sync()
			os.Exit(1)
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
