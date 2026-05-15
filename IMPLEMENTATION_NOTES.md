# Implementation Summary: Update File Metadata Status After Embedding

## Overview
This implementation adds functionality to automatically update the `file_metadata` status after documents are parsed into vectors by the document-embedding service. The flow is:

1. Gateway receives file upload and publishes message to `document.upload` queue
2. Document-embedding service consumes the message, parses the document, and generates embeddings
3. After successful embedding, document-embedding service publishes a completion message to `document.embedding.completed` queue
4. Gateway consumes the completion message and updates the file_metadata status to "embedded"

## Changes Made

### 1. Gateway Service Changes

#### File: `gateway/internal/repository/file_repository.go`
- **Added**: `UpdateStatus(ctx context.Context, id string, status string) error` method to the FileRepository interface and implementation
- **Purpose**: Allows updating the status field of file_metadata records

#### File: `gateway/internal/messaging/publisher.go`
- **Added**: `EmbeddingCompletedMessage` struct with fields:
  - FileID: unique identifier for the file
  - UserID: user who owns the file
  - Status: status to update to (e.g., "embedded")
  - ChunkCount: number of chunks created during embedding

#### File: `gateway/internal/messaging/embedding_consumer.go` (NEW)
- **Created**: New consumer that listens for embedding completion messages
- **Functionality**:
  - Consumes `EmbeddingCompletedMessage` from RabbitMQ
  - Updates the file_metadata status in the database
  - Automatically acknowledges successfully processed messages

#### File: `gateway/internal/config/config.go`
- **Added**: `RabbitMQCompletionQueue` configuration field
- **Default Value**: `"document.embedding.completed"`
- **Environment Variable**: `RABBITMQ_COMPLETION_QUEUE`

#### File: `gateway/main.go`
- **Added**: RabbitMQ connection for the embedding consumer
- **Added**: Initialization of `EmbeddingConsumer`
- **Added**: Goroutine to start the consumer at application startup
- **Added**: Import of `amqp "github.com/rabbitmq/amqp091-go"`

### 2. Document-Embedding Service Changes

#### File: `document-embedding/internal/config/config.go`
- **Added**: `RabbitMQCompletionQueue` field to Config struct
- **Default Value**: `"document.embedding.completed"`
- **Environment Variable**: `RABBITMQ_COMPLETION_QUEUE`

#### File: `document-embedding/internal/consumer/completion_publisher.go` (NEW)
- **Created**: CompletionPublisher struct and methods
- **Functionality**:
  - Publishes `EmbeddingCompletedMessage` to the completion queue
  - Manages RabbitMQ channel lifecycle
  - Ensures messages are persisted (Persistent delivery mode)

#### File: `document-embedding/internal/consumer/consumer.go`
- **Added**: `publisher *CompletionPublisher` field to DocumentConsumer struct
- **Updated**: `NewDocumentConsumer()` function signature to accept CompletionPublisher
- **Updated**: `handleMessage()` method to publish completion message after successful embedding
  - Publishes message with status "embedded" and chunk count
  - Logs any errors but doesn't fail the message processing (documents are already embedded)
- **Updated**: `Close()` method to properly close the publisher

#### File: `document-embedding/main.go`
- **Added**: Creation of CompletionPublisher
- **Added**: Passing CompletionPublisher to DocumentConsumer initialization
- **Added**: Proper cleanup of CompletionPublisher on shutdown

## File Status Values

The implementation uses the following file status values:
- `"pending_upload"`: Initial status when file metadata is created
- `"pending_embedding"`: Status after file is uploaded (can be added to file confirmation)
- `"embedded"`: Status after document is successfully parsed into vectors

## Environment Configuration

### Required Environment Variables (Unchanged)
- `RABBITMQ_URL`
- `DATABASE_URL`
- `MINIO_ENDPOINT`
- etc.

### New Optional Environment Variables
- `RABBITMQ_COMPLETION_QUEUE` (defaults to "document.embedding.completed")

## Message Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                          Gateway Service                        │
├─────────────────────────────────────────────────────────────────┤
│ 1. File upload → Create file_metadata (status: pending_upload) │
│ 2. Publish to: document.upload queue                            │
│ 3. Listen on: document.embedding.completed queue                │
│ 4. Receive completion → Update status to "embedded"             │
└─────────────────────────────────────────────────────────────────┘
                            ↕ (RabbitMQ)
┌─────────────────────────────────────────────────────────────────┐
│                   Document-Embedding Service                    │
├─────────────────────────────────────────────────────────────────┤
│ 1. Consume from: document.upload queue                          │
│ 2. Download file from MinIO                                     │
│ 3. Parse document into chunks                                   │
│ 4. Generate embeddings for each chunk                           │
│ 5. Store vectors in Qdrant                                      │
│ 6. Publish completion → document.embedding.completed queue      │
└─────────────────────────────────────────────────────────────────┘
```

## Error Handling

- **Gateway Consumer**: 
  - Automatically retries failed messages (Nack with requeue=true)
  - Logs errors but continues processing

- **Document-Embedding Publisher**:
  - Logs completion message publication errors
  - Doesn't fail document processing if message publication fails
  - Documents are already embedded at this point

## Database

No database migration is required as the `file_metadata` table already has the `status` column defined in:
`gateway/migrations/002_create_file_metadata.sql`

## Testing

After deployment, you can verify the implementation by:
1. Uploading a file through the gateway
2. Checking the database to see when the status changes from `"pending_upload"` to `"embedded"`
3. Verifying RabbitMQ messages are being passed between services

## Backward Compatibility

- The changes are backward compatible
- Existing code that doesn't use the completion message feature will continue to work
- The `file_metadata.status` field will remain in its uploaded state if the consumer isn't running

