package config

import "testing"

func TestLoadReturnsErrorWhenEmbeddingURLMissing(t *testing.T) {
	t.Setenv("EMBEDDING_API_URL", "")
	t.Setenv("EMBEDDING_API_KEY", "")
	t.Setenv("EMBEDDING_MODEL", "")
	t.Setenv("EMBEDDING_DIM", "")
	t.Setenv("RABBITMQ_URL", "")
	t.Setenv("RABBITMQ_QUEUE", "")
	t.Setenv("QDRANT_HOST", "")
	t.Setenv("QDRANT_PORT", "")
	t.Setenv("QDRANT_API_KEY", "")
	t.Setenv("QDRANT_COLLECTION", "")
	t.Setenv("LOG_LEVEL", "")

	cfg, err := Load()
	if err == nil {
		t.Fatalf("Load() error = nil, want error")
	}
	if cfg != nil {
		t.Fatalf("Load() cfg = %#v, want nil", cfg)
	}
}

func TestLoadAppliesDefaultsAndEnvOverrides(t *testing.T) {
	t.Setenv("EMBEDDING_API_URL", "http://localhost:11434/api/embed")
	t.Setenv("RABBITMQ_URL", "amqp://custom:pass@mq:5672/")
	t.Setenv("RABBITMQ_QUEUE", "documents.queue")
	t.Setenv("QDRANT_HOST", "qdrant")
	t.Setenv("QDRANT_PORT", "7000")
	t.Setenv("QDRANT_API_KEY", "secret")
	t.Setenv("QDRANT_COLLECTION", "knowledge")
	t.Setenv("EMBEDDING_API_KEY", "")
	t.Setenv("EMBEDDING_MODEL", "bge-m3")
	t.Setenv("EMBEDDING_DIM", "2048")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RabbitMQURL != "amqp://custom:pass@mq:5672/" {
		t.Fatalf("RabbitMQURL = %q, want custom value", cfg.RabbitMQURL)
	}
	if cfg.RabbitMQQueue != "documents.queue" {
		t.Fatalf("RabbitMQQueue = %q, want documents.queue", cfg.RabbitMQQueue)
	}
	if cfg.QdrantHost != "qdrant" || cfg.QdrantPort != 7000 {
		t.Fatalf("Qdrant = %s:%d, want qdrant:7000", cfg.QdrantHost, cfg.QdrantPort)
	}
	if cfg.QdrantAPIKey != "secret" {
		t.Fatalf("QdrantAPIKey = %q, want secret", cfg.QdrantAPIKey)
	}
	if cfg.QdrantColl != "knowledge" {
		t.Fatalf("QdrantColl = %q, want knowledge", cfg.QdrantColl)
	}
	if cfg.EmbeddingURL != "http://localhost:11434/api/embed" {
		t.Fatalf("EmbeddingURL = %q, want env override", cfg.EmbeddingURL)
	}
	if cfg.EmbeddingKey != "" {
		t.Fatalf("EmbeddingKey = %q, want empty", cfg.EmbeddingKey)
	}
	if cfg.EmbeddingModel != "bge-m3" {
		t.Fatalf("EmbeddingModel = %q, want bge-m3", cfg.EmbeddingModel)
	}
	if cfg.EmbeddingDim != 2048 {
		t.Fatalf("EmbeddingDim = %d, want 2048", cfg.EmbeddingDim)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want debug", cfg.LogLevel)
	}
}

func TestLoadFallsBackForInvalidEmbeddingDim(t *testing.T) {
	t.Setenv("EMBEDDING_API_URL", "http://localhost:11434/api/embed")
	t.Setenv("EMBEDDING_DIM", "invalid")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.EmbeddingDim != 1024 {
		t.Fatalf("EmbeddingDim = %d, want fallback 1024", cfg.EmbeddingDim)
	}
}
