package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"gateway/internal/repository"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EmbeddingConsumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	queue    string
	fileRepo repository.FileRepository
}

func NewEmbeddingConsumer(
	conn *amqp.Connection,
	queue string,
	fileRepo repository.FileRepository,
) (*EmbeddingConsumer, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	_, err = channel.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	return &EmbeddingConsumer{
		conn:     conn,
		channel:  channel,
		queue:    queue,
		fileRepo: fileRepo,
	}, nil
}

func (c *EmbeddingConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume queue: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgs:
			if msg.Headers == nil && len(msg.Body) == 0 {
				continue
			}

			if err := c.handleMessage(ctx, msg); err != nil {
				if nackErr := msg.Nack(false, true); nackErr != nil {
					// Log error if nack fails
				}
			} else {
				if ackErr := msg.Ack(false); ackErr != nil {
					// Log error if ack fails
				}
			}
		}
	}
}

func (c *EmbeddingConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	var completedMsg EmbeddingCompletedMessage
	if err := json.Unmarshal(msg.Body, &completedMsg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	if err := c.fileRepo.UpdateStatus(ctx, completedMsg.FileID, completedMsg.Status); err != nil {
		return fmt.Errorf("update file status: %w", err)
	}

	return nil
}

func (c *EmbeddingConsumer) Close() error {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	return nil
}
