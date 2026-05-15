package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"document-embedding/internal/config"
	"document-embedding/internal/consumer"
	"document-embedding/internal/embedding"
	"document-embedding/internal/parser"
	"document-embedding/internal/vector"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := createLogger(cfg.LogLevel)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		logger.Fatal("connect rabbitmq", zap.Error(err))
	}
	defer conn.Close()
	logger.Info("connected to rabbitmq", zap.String("url", cfg.RabbitMQURL))

	vectorStore, err := vector.NewQdrantStore(cfg.QdrantHost, cfg.QdrantPort, cfg.QdrantAPIKey, cfg.QdrantColl)
	if err != nil {
		logger.Fatal("create qdrant store", zap.Error(err))
	}
	defer vectorStore.Close()

	ctx := context.Background()
	if err := vectorStore.CreateCollection(ctx, cfg.EmbeddingDim); err != nil {
		logger.Error("create collection", zap.Error(err))
	}
	logger.Info("qdrant store initialized", zap.String("collection", cfg.QdrantColl))

	embeddingClient := embedding.NewEmbeddingClient(cfg.EmbeddingURL, cfg.EmbeddingKey, cfg.EmbeddingModel)
	logger.Info("embedding client initialized", zap.String("model", cfg.EmbeddingModel))

	docParser := parser.NewSimpleParser()

	completionPublisher, err := consumer.NewCompletionPublisher(conn, cfg.RabbitMQCompletionQueue)
	if err != nil {
		logger.Fatal("create completion publisher", zap.Error(err))
	}
	defer completionPublisher.Close()

	docConsumer, err := consumer.NewDocumentConsumer(
		conn,
		cfg.RabbitMQQueue,
		docParser,
		embeddingClient,
		vectorStore,
		logger,
		completionPublisher,
	)
	if err != nil {
		logger.Fatal("create consumer", zap.Error(err))
	}
	defer docConsumer.Close()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := docConsumer.Start(ctx); err != nil && err != context.Canceled {
			logger.Error("consumer error", zap.Error(err))
		}
	}()

	logger.Info("document embedding service started")
	<-quit
	logger.Info("shutting down...")
}

func createLogger(level string) (*zap.Logger, error) {
	switch level {
	case "debug":
		return zap.NewDevelopment()
	case "production":
		return zap.NewProduction()
	default:
		return zap.NewDevelopment()
	}
}
