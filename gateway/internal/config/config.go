package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config stores runtime configuration values.
type Config struct {
	Port             string
	DatabaseURL      string
	JWTSecret        string
	JWTExpireMinutes int
	MinIOEndpoint    string
	MinIOPublicURL   string
	MinIOAccessKey   string
	MinIOSecretKey   string
	MinIOBucket      string
	MinIOUseSSL      bool
	PresignExpireMin int
	RabbitMQURL      string
	RabbitMQQueue    string
}

func Load() (Config, error) {
	cfg := Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		JWTExpireMinutes: getEnvAsInt("JWT_EXPIRE_MINUTES", 120),
		MinIOEndpoint:    os.Getenv("MINIO_ENDPOINT"),
		MinIOPublicURL:   getEnv("MINIO_PUBLIC_URL", ""),
		MinIOAccessKey:   os.Getenv("MINIO_ACCESS_KEY"),
		MinIOSecretKey:   os.Getenv("MINIO_SECRET_KEY"),
		MinIOBucket:      os.Getenv("MINIO_BUCKET"),
		MinIOUseSSL:      getEnvAsBool("MINIO_USE_SSL", false),
		PresignExpireMin: getEnvAsInt("PRESIGN_EXPIRE_MINUTES", 15),
		RabbitMQURL:      getEnv("RABBITMQ_URL", "amqp://admin:admin123456@localhost:5672/"),
		RabbitMQQueue:    getEnv("RABBITMQ_QUEUE", "document.upload"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	if cfg.MinIOEndpoint == "" {
		return Config{}, fmt.Errorf("MINIO_ENDPOINT is required")
	}
	if cfg.MinIOAccessKey == "" {
		return Config{}, fmt.Errorf("MINIO_ACCESS_KEY is required")
	}
	if cfg.MinIOSecretKey == "" {
		return Config{}, fmt.Errorf("MINIO_SECRET_KEY is required")
	}
	if cfg.MinIOBucket == "" {
		return Config{}, fmt.Errorf("MINIO_BUCKET is required")
	}

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvAsBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
