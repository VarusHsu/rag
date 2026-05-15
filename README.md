# Retrieval-Augmented-Generation

A full local demo stack for:

- `gateway`: Gin + PostgreSQL + MinIO + RabbitMQ
- `document-embedding`: consumes upload events, calls Ollama embeddings, stores vectors in Qdrant
- `web`: Vue frontend for register/login/upload

## One-command startup

1. Copy the root env template if you want to customize passwords or model:

```bash
cp .env.example .env
```

2. Start the whole stack:

```bash
docker compose up --build
```

After startup, the main entry points are:

- Web UI: `http://localhost:5173`
- Gateway API: `http://localhost:8080`
- RabbitMQ management: `http://localhost:15672`
- MinIO API: `http://localhost:9000`
- MinIO console: `http://localhost:9001`
- Qdrant: `http://localhost:6333`
- Ollama: `http://localhost:11434`

## What the compose stack does

The root `docker-compose.yml` starts and wires together:

- PostgreSQL
- RabbitMQ
- Qdrant
- MinIO
- MinIO bucket initializer
- DB migration job
- Ollama
- Ollama model pull job (`bge-m3` by default)
- Gateway service
- Document embedding service
- Web frontend

## Notes

- The frontend is built with `VITE_API_BASE_URL=http://localhost:8080` by default.
- The gateway talks to MinIO over the internal Docker network (`minio:9000`) but rewrites browser-facing presigned upload URLs to `${MINIO_PUBLIC_URL}`. By default this is `http://localhost:9000`.
- The default embedding model is `bge-m3`, and the vector dimension is configured as `1024`.
- The first `ollama pull` can take a while because the model needs to be downloaded.

## Stop and clean up

Stop services:

```bash
docker compose down
```

Stop and remove persistent data too:

```bash
docker compose down -v
```

