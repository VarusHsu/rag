package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EmbeddingCompletedMessage struct {
	FileID     string `json:"file_id"`
	UserID     string `json:"user_id"`
	Status     string `json:"status"`
	ChunkCount int    `json:"chunk_count"`
}

type CompletionPublisher struct {
	channel *amqp.Channel
	queue   string
}

func NewCompletionPublisher(conn *amqp.Connection, queue string) (*CompletionPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	_, err = ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	return &CompletionPublisher{
		channel: ch,
		queue:   queue,
	}, nil
}

func (p *CompletionPublisher) Publish(ctx context.Context, msg EmbeddingCompletedMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	return p.channel.PublishWithContext(ctx, "", p.queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *CompletionPublisher) Close() error {
	if p.channel != nil {
		return p.channel.Close()
	}
	return nil
}
