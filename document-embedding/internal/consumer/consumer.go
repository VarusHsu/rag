package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"document-embedding/internal/embedding"
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

type DocumentConsumer struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	queue       string
	parser      parser.DocumentParser
	embedding   *embedding.EmbeddingClient
	vectorStore *vector.QdrantStore
	logger      *zap.Logger
	httpClient  *http.Client
}

func NewDocumentConsumer(
	conn *amqp.Connection,
	queue string,
	p parser.DocumentParser,
	e *embedding.EmbeddingClient,
	v *vector.QdrantStore,
	logger *zap.Logger,
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
				msg.Nack(false, true)
			} else {
				msg.Ack(false)
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
			ID:     fmt.Sprintf("%s-%d", uploadMsg.FileID, i),
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

	return nil
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
	return nil
}
