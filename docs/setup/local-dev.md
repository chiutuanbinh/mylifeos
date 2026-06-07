# Local Development

## Prerequisites

- Docker Desktop 4.x+
- Go 1.22+ (`brew install go`)
- Node 20+ (`brew install node`)
- make

## Setup

```bash
git clone git@github.com:chiutuanbinh/mylifeos.git
cd mylifeos
cp .env.example .env.local
```

`.env.local` defaults work out of the box — no accounts needed locally.

## Start everything

```bash
make dev
```

This starts:
- Postgres on `:5432` (auto-seeded with schema from `migrations/001_schema.sql`)
- Go backend on `:8080` (hot-reload via `air`)
- React frontend on `:5173` (hot-reload via Vite)

Open: http://localhost:5173

Sign in with any email/password — local dev mode skips auth.

## Apply DB migrations manually

```bash
make migrate
```

Run this after pulling new migration files. Safe to run multiple times (all migrations are idempotent).

## Run tests

```bash
make test          # backend + frontend
make test-backend  # go test ./... -v
make test-frontend # vitest --run
```

## Stop everything

```bash
docker compose down
```

To also delete DB data:

```bash
docker compose down -v
```

## Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `ENV` | `development` | Set to `development` to skip JWT validation |
| `DEV_USER_ID` | `00000000-0000-0000-0000-000000000001` | User ID injected in dev mode |
| `DATABASE_URL` | `postgres://mylifeos:mylifeos@localhost:5432/mylifeos` | Postgres connection |
| `PORT` | `8080` | Backend port |
| `VITE_API_URL` | `http://localhost:8080/api/v1` | Frontend → backend URL |

Production variables (`SUPABASE_*`, `VITE_SUPABASE_*`) are not needed locally.
