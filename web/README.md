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

