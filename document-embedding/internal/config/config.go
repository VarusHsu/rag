package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	RabbitMQURL    string
	RabbitMQQueue  string
	QdrantHost     string
	QdrantPort     int
	QdrantAPIKey   string
	QdrantColl     string
	EmbeddingURL   string
	EmbeddingKey   string
	EmbeddingModel string
	LogLevel       string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		RabbitMQURL:    getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitMQQueue:  getEnv("RABBITMQ_QUEUE", "document.upload"),
		QdrantHost:     getEnv("QDRANT_HOST", "localhost"),
		QdrantPort:     getEnvAsInt("QDRANT_PORT", 6333),
		QdrantAPIKey:   getEnv("QDRANT_API_KEY", ""),
		QdrantColl:     getEnv("QDRANT_COLLECTION", "documents"),
		EmbeddingURL:   os.Getenv("EMBEDDING_API_URL"),
		EmbeddingKey:   os.Getenv("EMBEDDING_API_KEY"),
		EmbeddingModel: getEnv("EMBEDDING_MODEL", "text-embedding-3-small"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
	}

	if cfg.EmbeddingURL == "" {
		return nil, fmt.Errorf("EMBEDDING_API_URL is required")
	}
	if cfg.EmbeddingKey == "" {
		return nil, fmt.Errorf("EMBEDDING_API_KEY is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	var n int
	_, _ = fmt.Sscanf(val, "%d", &n)
	if n > 0 {
		return n
	}
	return fallback
}
