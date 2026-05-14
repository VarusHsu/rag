# Document Embedding Service

A Go service that consumes document upload events from RabbitMQ, parses files, generates embeddings, and stores them in Qdrant vector database.

## Features

- **RabbitMQ Consumer**: Consumes document upload events
- **File Parsing**: Supports PDF and text files
- **Embedding Generation**: Uses OpenAI embedding API
- **Vector Storage**: Stores embeddings in Qdrant

## Architecture

```
RabbitMQ → Consumer → Parser → Embedding → Qdrant
```

## Setup

### Prerequisites

- Go 1.25+
- RabbitMQ running
- Qdrant server running
- OpenAI API key

### Installation

```bash
cd /path/to/document-embedding
cp .env.example .env
# Edit .env with your configuration
go mod download
```

### Configuration

Update `.env` with:
- `RABBITMQ_URL`: RabbitMQ connection string
- `RABBITMQ_QUEUE`: Queue name to consume from
- `QDRANT_HOST`: Qdrant server host
- `QDRANT_PORT`: Qdrant server port
- `QDRANT_COLLECTION`: Collection name
- `EMBEDDING_API_URL`: OpenAI API endpoint
- `EMBEDDING_API_KEY`: OpenAI API key
- `EMBEDDING_MODEL`: Model name (e.g., text-embedding-3-small)

## Running

```bash
go run main.go
```

## Docker

```bash
docker run -d \
  -e RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/ \
  -e QDRANT_HOST=qdrant \
  -e EMBEDDING_API_URL=https://api.openai.com/v1 \
  -e EMBEDDING_API_KEY=your-key \
  document-embedding
```

## Message Format

Expected RabbitMQ message:

```json
{
  "file_id": "uuid",
  "file_name": "document.pdf",
  "content_type": "application/pdf",
  "file_size": 12345,
  "file_url": "https://minio.../uploads/...",
  "user_id": "user-uuid",
  "bucket": "files",
  "object_key": "uploads/user-uuid/..."
}
```

## Testing

```bash
go test ./...
```

## Performance

- Processes documents sequentially
- Chunks large documents for better embedding quality
- Automatic retry on transient failures (requires manual nack configuration)

