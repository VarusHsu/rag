# Gateway Auth Service

Gin + PostgreSQL login/register service.

## Response Format

All APIs return a unified envelope:

```json
{
  "code": 0,
  "msg": "success",
  "request_id": "req-demo-001",
  "data": {}
}
```

- `code = 0` means success
- `code != 0` means business error

## Features

- `POST /api/v1/auth/register` for registration
- `POST /api/v1/auth/login` for authentication
- `POST /api/v1/auth/logout` for logout and token revocation
- `POST /api/v1/files/presign-upload` for presigned upload URL generation
- `POST /api/v1/files/compensate-embedding` for retrying non-embedded text vectorization (admin)
- Password hashing with bcrypt
- JWT token generation
- PostgreSQL user repository
- Request tracing with `X-Request-Id` (auto generated if missing)

## Prerequisites

- Go 1.22+
- PostgreSQL 13+

## Setup

1. Copy environment variables and adjust values:

```bash
cp .env.example .env
```

2. Create database tables:

```bash
psql "$DATABASE_URL" -f migrations/001_create_users.sql
psql "$DATABASE_URL" -f migrations/002_create_file_metadata.sql
```

3. Run service:

```bash
go run .
```

Server starts at `http://localhost:8080` by default.

## API Examples

### Register

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","email":"alice@example.com","password":"12345678","confirm_password":"12345678","phone":"13800138000"}'
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -H 'X-Request-Id: req-demo-001' \
  -d '{"email":"alice@example.com","password":"12345678"}'
```

All API responses include `X-Request-Id` header, and backend logs print `request_id=...` for each request.

### Logout

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H 'Authorization: Bearer <your-jwt-token>' \
  -H 'X-Request-Id: req-demo-logout-001'
```

### Presigned Upload URL

```bash
curl -X POST http://localhost:8080/api/v1/files/presign-upload \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <your-jwt-token>' \
  -d '{"file_name":"resume.pdf","content_type":"application/pdf","file_size":12345}'
```

Use `data.upload_url` with `PUT` to upload bytes directly to MinIO from frontend.

### Compensate Embedding (Admin)

```bash
curl -X POST http://localhost:8080/api/v1/files/compensate-embedding \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <admin-jwt-token>' \
  -d '{"limit":200}'
```

`limit` is optional (`1-1000`), default is `200`.

## Test

```bash
go test ./...
```

