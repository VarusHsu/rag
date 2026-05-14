# Gateway Auth Service

Gin + PostgreSQL login/register service.

## Features

- `POST /api/v1/auth/register` for registration
- `POST /api/v1/auth/login` for authentication
- Password hashing with bcrypt
- JWT token generation
- PostgreSQL user repository

## Prerequisites

- Go 1.22+
- PostgreSQL 13+

## Setup

1. Copy environment variables and adjust values:

```bash
cp .env.example .env
```

2. Create the `users` table:

```bash
psql "$DATABASE_URL" -f migrations/001_create_users.sql
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
  -d '{"username":"alice","email":"alice@example.com","password":"12345678","phone":"13800138000"}'
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"12345678"}'
```

## Test

```bash
go test ./...
```

