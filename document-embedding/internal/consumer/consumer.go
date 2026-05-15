package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"document-embedding/internal/parser"
	"document-embedding/internal/vector"
)

type UploadMessage struct {
	FileID      string `json:"file_id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
	FileURL     string `json:"file_url"`
	UserID      string `json:"user_id"`
	Bucket      string `json:"bucket"`
	ObjectKey   string `json:"object_key"`
}

type EmbeddingService interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type VectorStore interface {
	UpsertVectors(ctx context.Context, records []vector.VectorRecord) error
}

type DocumentConsumer struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	queue       string
	parser      parser.DocumentParser
	embedding   EmbeddingService
	vectorStore VectorStore
	logger      *zap.Logger
	httpClient  *http.Client
	publisher   *CompletionPublisher
}

func NewDocumentConsumer(
	conn *amqp.Connection,
	queue string,
	p parser.DocumentParser,
	e EmbeddingService,
	v VectorStore,
	logger *zap.Logger,
	publisher *CompletionPublisher,
) (*DocumentConsumer, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	_, err = channel.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	return &DocumentConsumer{
		conn:        conn,
		channel:     channel,
		queue:       queue,
		parser:      p,
		embedding:   e,
		vectorStore: v,
		logger:      logger,
		httpClient:  &http.Client{Timeout: 5 * time.Minute},
		publisher:   publisher,
	}, nil
}

func (c *DocumentConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume queue: %w", err)
	}

	c.logger.Info("consumer started", zap.String("queue", c.queue))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgs:
			if msg.Headers == nil && len(msg.Body) == 0 {
				continue
			}

			if err := c.handleMessage(ctx, msg); err != nil {
				c.logger.Error("handle message failed", zap.Error(err))
				if nackErr := msg.Nack(false, true); nackErr != nil {
					c.logger.Error("nack message failed", zap.Error(nackErr))
				}
			} else {
				if ackErr := msg.Ack(false); ackErr != nil {
					c.logger.Error("ack message failed", zap.Error(ackErr))
				}
			}
		}
	}
}

func (c *DocumentConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	var uploadMsg UploadMessage
	if err := json.Unmarshal(msg.Body, &uploadMsg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	c.logger.Info("processing document", zap.String("file_id", uploadMsg.FileID), zap.String("file_name", uploadMsg.FileName))

	fileData, err := c.downloadFile(uploadMsg.FileURL)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}

	doc, err := c.parser.Parse(fileData, uploadMsg.FileName)
	if err != nil {
		return fmt.Errorf("parse document: %w", err)
	}

	var vectorRecords []vector.VectorRecord
	for i, chunk := range doc.Chunks {
		emb, err := c.embedding.Embed(ctx, chunk.Content)
		if err != nil {
			return fmt.Errorf("embed chunk %d: %w", i, err)
		}

		vectorRecords = append(vectorRecords, vector.VectorRecord{
			ID:     buildPointID(uploadMsg.FileID, i),
			Vector: emb,
			Metadata: map[string]string{
				"file_id":     uploadMsg.FileID,
				"file_name":   uploadMsg.FileName,
				"user_id":     uploadMsg.UserID,
				"object_key":  uploadMsg.ObjectKey,
				"chunk_index": fmt.Sprintf("%d", i),
				"page":        fmt.Sprintf("%d", chunk.Page),
			},
		})
	}

	if err := c.vectorStore.UpsertVectors(ctx, vectorRecords); err != nil {
		return fmt.Errorf("upsert vectors: %w", err)
	}

	c.logger.Info("document processed successfully",
		zap.String("file_id", uploadMsg.FileID),
		zap.Int("chunks", len(vectorRecords)),
	)

	// Publish completion message
	if err := c.publisher.Publish(ctx, EmbeddingCompletedMessage{
		FileID:     uploadMsg.FileID,
		UserID:     uploadMsg.UserID,
		Status:     "embedded",
		ChunkCount: len(vectorRecords),
	}); err != nil {
		c.logger.Error("publish completion message failed",
			zap.String("file_id", uploadMsg.FileID),
			zap.Error(err),
		)
		// Don't return error here - we've already successfully embedded the document
	}

	return nil
}

func buildPointID(fileID string, chunkIndex int) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s:%d", fileID, chunkIndex))).String()
}

func (c *DocumentConsumer) downloadFile(fileURL string) ([]byte, error) {
	resp, err := c.httpClient.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: status=%d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return data, nil
}

func (c *DocumentConsumer) Close() error {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.publisher != nil {
		_ = c.publisher.Close()
	}
	return nil
}
