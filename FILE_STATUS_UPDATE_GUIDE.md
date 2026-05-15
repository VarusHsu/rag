# File Metadata Status Update After Embedding - Quick Reference

## What Was Implemented

The system now automatically updates file_metadata status when documents are successfully embedded into vectors:

- **Before**: File status stayed as "pending_upload" indefinitely
- **After**: File status is updated to "embedded" after the document-embedding service successfully processes the document

## How It Works

1. **File Upload Flow**:
   - User uploads a file via gateway
   - File is stored in MinIO
   - Message is published to RabbitMQ: `document.upload`
   - File status is set to: `"pending_upload"`

2. **Embedding Processing**:
   - Document-embedding service picks up the message
   - Downloads and parses the document into chunks
   - Generates embeddings for each chunk using the embedding API
   - Stores vectors in Qdrant database
   - Publishes completion message to RabbitMQ: `document.embedding.completed`

3. **Status Update**:
   - Gateway service listens for completion messages
   - Receives the message with file_id and status: `"embedded"`
   - Updates file_metadata.status in the database
   - File is now marked as successfully embedded

## Configuration

### Default Queues
- **Upload Queue**: `document.upload` (unchanged)
- **Completion Queue**: `document.embedding.completed` (new)

### Environment Variables

Both services read from the same RabbitMQ instance. Add this to your `.env` files:

```bash
# Gateway .env
RABBITMQ_URL=amqp://admin:password@rabbitmq:5672/
RABBITMQ_QUEUE=document.upload
RABBITMQ_COMPLETION_QUEUE=document.embedding.completed

# Document-embedding .env
RABBITMQ_URL=amqp://admin:password@rabbitmq:5672/
RABBITMQ_QUEUE=document.upload
RABBITMQ_COMPLETION_QUEUE=document.embedding.completed
```

## Database

No schema changes needed. The `file_metadata` table already has the required fields:
- `id`: File identifier
- `status`: Current status of the file
- `updated_at`: Automatically updated when record changes

## File Status States

```
pending_upload → (file uploaded) → embedded
                  ↓                    ↑
            [document uploaded]  [embedding complete]
```

## Monitoring

### Check File Status in Database
```sql
SELECT id, file_name, status, updated_at 
FROM file_metadata 
ORDER BY updated_at DESC 
LIMIT 10;
```

### RabbitMQ Queues to Monitor
- `document.upload`: Messages waiting to be embedded
- `document.embedding.completed`: Completion messages (should be processed quickly)

## Error Handling

**If a file stays in "pending_upload" status:**
1. Check if document-embedding service is running
2. Check RabbitMQ connection and queue health
3. Check logs for embedding errors:
   - File parsing errors (unsupported format)
   - Embedding API errors (timeout, authentication)
   - Vector storage errors (Qdrant connection)

**If documents embed but status doesn't update:**
1. Check if gateway's embedding consumer is running
2. Verify RABBITMQ_COMPLETION_QUEUE configuration matches
3. Check gateway logs for consumer errors

## Files Modified

### Gateway Service
- `gateway/internal/repository/file_repository.go` - Added UpdateStatus method
- `gateway/internal/messaging/publisher.go` - Added EmbeddingCompletedMessage type
- `gateway/internal/messaging/embedding_consumer.go` - Created new consumer
- `gateway/internal/config/config.go` - Added RABBITMQ_COMPLETION_QUEUE config
- `gateway/main.go` - Initialize and start embedding consumer

### Document-Embedding Service
- `document-embedding/internal/config/config.go` - Added RABBITMQ_COMPLETION_QUEUE config
- `document-embedding/internal/consumer/consumer.go` - Updated to publish completion messages
- `document-embedding/internal/consumer/completion_publisher.go` - Created publisher
- `document-embedding/main.go` - Initialize and wire completion publisher

## Testing the Feature

### Manual Test
1. Start all services (gateway, document-embedding, RabbitMQ, PostgreSQL, MinIO, Qdrant)
2. Upload a file through the gateway API
3. Query the database to see status progression:
   ```bash
   # Initial status
   SELECT status FROM file_metadata WHERE id = '<file_id>';
   # Result: pending_upload
   
   # Wait a few seconds for embedding...
   
   # Final status
   SELECT status FROM file_metadata WHERE id = '<file_id>';
   # Result: embedded
   ```

### Troubleshooting
- **No status change after 30 seconds?** Check document-embedding service logs
- **Error in gateway logs?** Check RabbitMQ connection and queue names
- **"embedded" status appearing correctly?** Feature is working!

## Performance Considerations

- Embedding time depends on document size and embedding API latency
- Status updates happen within milliseconds after embedding completes
- Completion messages are persistent (won't be lost if consumer restarts)

## Future Enhancements

Possible status states to add:
- `"pending_embedding"` - File uploaded, waiting for embedding
- `"embedding_failed"` - Document couldn't be embedded
- `"partially_embedded"` - Some chunks failed
- `"archived"` - File marked for deletion

## Support

For issues or questions, check:
1. Service logs: `docker logs <service-name>`
2. RabbitMQ management UI: `http://localhost:15672`
3. Database records: Check file_metadata table directly

