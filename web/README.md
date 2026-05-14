# Web (Vue)

Simple Vue 3 page for register/login against gateway auth APIs.

## Setup

```bash
cp .env.example .env
npm install
npm run dev
```

Default dev URL: `http://localhost:5173`

## Build

```bash
npm run build
npm run preview
```

## API Base URL

Configure in `.env`:

```bash
VITE_API_BASE_URL=http://localhost:8080
```

The page calls:

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`

## Tracing

- Frontend uses a request interceptor to inject `X-Request-Id` on every request.
- The UI displays the latest `request_id` from response headers or response body.

## API Envelope

- Frontend expects backend responses in unified format: `code`, `msg`, `request_id`, `data`.
- `code = 0` means success.

