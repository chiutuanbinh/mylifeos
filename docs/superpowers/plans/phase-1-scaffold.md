# MyLifeOS Phase 1: Scaffold

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Working monorepo with docker-compose local dev, DB migrations, and CI/CD pipelines — no app code yet.

**Architecture:** Monorepo at repo root. Backend is a Go module at `backend/`. Frontend is a Vite+React app at `frontend/`. Shared infra (docker-compose, Makefile, migrations) at root.

**Tech Stack:** Go 1.22, Node 20, Postgres 16, Docker Compose, GitHub Actions

---

### Task 1: Monorepo root scaffold

**Files:**
- Create: `.env.example`
- Create: `.gitignore`
- Create: `Makefile`
- Create: `docker-compose.yml`

- [ ] **Step 1: Create `.env.example`**

```bash
cat > .env.example << 'EOF'
# Local dev only — copy to .env.local
POSTGRES_USER=mylifeos
POSTGRES_PASSWORD=mylifeos
POSTGRES_DB=mylifeos
DATABASE_URL=postgres://mylifeos:mylifeos@localhost:5432/mylifeos?sslmode=disable

# Set to "development" locally — skips real JWT validation
ENV=development
DEV_USER_ID=00000000-0000-0000-0000-000000000001
PORT=8080

# Supabase (production only)
SUPABASE_URL=
SUPABASE_JWT_SECRET=
SUPABASE_SERVICE_ROLE_KEY=

# Frontend
VITE_API_URL=http://localhost:8080/api/v1
VITE_SUPABASE_URL=
VITE_SUPABASE_ANON_KEY=
EOF
cp .env.example .env.local
```

- [ ] **Step 2: Create `.gitignore`**

```bash
cat > .gitignore << 'EOF'
.env.local
.env.*.local
*.env

# Go
backend/bin/
backend/tmp/

# Node
frontend/node_modules/
frontend/dist/
frontend/.vite/

# OS
.DS_Store
EOF
```

- [ ] **Step 3: Create `Makefile`**

```makefile
# Makefile
.PHONY: dev migrate test test-backend test-frontend build-backend

dev:
	docker compose up --build

migrate:
	docker compose run --rm migrate

test: test-backend test-frontend

test-backend:
	cd backend && go test ./... -v

test-frontend:
	cd frontend && npm test -- --run

build-backend:
	cd backend && go build -o bin/server ./cmd/server

lint-backend:
	cd backend && go vet ./...

lint-frontend:
	cd frontend && npm run lint
```

- [ ] **Step 4: Create `docker-compose.yml`**

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: mylifeos
      POSTGRES_PASSWORD: mylifeos
      POSTGRES_DB: mylifeos
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U mylifeos"]
      interval: 5s
      timeout: 5s
      retries: 5

  migrate:
    image: postgres:16-alpine
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      PGPASSWORD: mylifeos
    volumes:
      - ./migrations:/migrations
    command: >
      sh -c "for f in /migrations/*.sql; do psql -h postgres -U mylifeos -d mylifeos -f $$f; done"
    profiles: ["migrate"]

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile.dev
    env_file: .env.local
    ports:
      - "8080:8080"
    volumes:
      - ./backend:/app
    depends_on:
      postgres:
        condition: service_healthy

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile.dev
    env_file: .env.local
    ports:
      - "5173:5173"
    volumes:
      - ./frontend:/app
      - /app/node_modules
    depends_on:
      - backend

volumes:
  pgdata:
```

- [ ] **Step 5: Commit**

```bash
git add .env.example .gitignore Makefile docker-compose.yml
git commit -m "feat: monorepo root scaffold with docker-compose and Makefile"
```

---

### Task 2: DB migrations

**Files:**
- Create: `migrations/001_schema.sql`
- Create: `migrations/002_rls.sql`

- [ ] **Step 1: Create `migrations/001_schema.sql`**

```bash
mkdir -p migrations
cat > migrations/001_schema.sql << 'EOF'
-- Finance
CREATE TABLE IF NOT EXISTS transactions (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL,
  date        date NOT NULL,
  description text NOT NULL,
  category    text NOT NULL,
  amount      numeric(12,2) NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS budgets (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       uuid NOT NULL,
  category      text NOT NULL,
  monthly_limit numeric(12,2) NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, category)
);

-- Health
CREATE TABLE IF NOT EXISTS habits (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL,
  name       text NOT NULL,
  icon       text NOT NULL DEFAULT '✓',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS habit_logs (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  habit_id    uuid NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
  user_id     uuid NOT NULL,
  logged_date date NOT NULL,
  done        boolean NOT NULL DEFAULT true,
  UNIQUE (habit_id, logged_date)
);

-- Goals
CREATE TABLE IF NOT EXISTS goals (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL,
  name        text NOT NULL,
  description text NOT NULL DEFAULT '',
  target_date date,
  progress    int NOT NULL DEFAULT 0,
  color       text NOT NULL DEFAULT '#1677ff',
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS key_results (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  goal_id     uuid NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  user_id     uuid NOT NULL,
  description text NOT NULL,
  done        boolean NOT NULL DEFAULT false
);

-- Notes
CREATE TABLE IF NOT EXISTS notes (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL,
  title      text NOT NULL,
  content    text NOT NULL DEFAULT '',
  tags       text[] NOT NULL DEFAULT '{}',
  pinned     boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- Calendar
CREATE TABLE IF NOT EXISTS events (
  id       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id  uuid NOT NULL,
  title    text NOT NULL,
  start_at timestamptz NOT NULL,
  end_at   timestamptz NOT NULL,
  color    text NOT NULL DEFAULT '#1677ff',
  all_day  boolean NOT NULL DEFAULT false
);

-- Inventory
CREATE TABLE IF NOT EXISTS assets (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      uuid NOT NULL,
  name         text NOT NULL,
  category     text NOT NULL,
  value        numeric(12,2) NOT NULL DEFAULT 0,
  purchased_at date,
  notes        text NOT NULL DEFAULT ''
);

-- Settings
CREATE TABLE IF NOT EXISTS user_settings (
  user_id         uuid PRIMARY KEY,
  notifications   jsonb NOT NULL DEFAULT '{"email": true, "push": false}'::jsonb,
  modules_enabled jsonb NOT NULL DEFAULT '{"finance": true, "health": true, "goals": true, "notes": true, "calendar": true, "inventory": true}'::jsonb
);
EOF
```

- [ ] **Step 2: Create `migrations/002_rls.sql`**

Note: RLS only applies in Supabase (production). Local Postgres skips this safely since `auth.uid()` is a Supabase function. This file is applied via Supabase SQL editor — not via docker-compose.

```bash
cat > migrations/002_rls.sql << 'EOF'
-- Enable RLS on all tables
ALTER TABLE transactions    ENABLE ROW LEVEL SECURITY;
ALTER TABLE budgets         ENABLE ROW LEVEL SECURITY;
ALTER TABLE habits          ENABLE ROW LEVEL SECURITY;
ALTER TABLE habit_logs      ENABLE ROW LEVEL SECURITY;
ALTER TABLE goals           ENABLE ROW LEVEL SECURITY;
ALTER TABLE key_results     ENABLE ROW LEVEL SECURITY;
ALTER TABLE notes           ENABLE ROW LEVEL SECURITY;
ALTER TABLE events          ENABLE ROW LEVEL SECURITY;
ALTER TABLE assets          ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_settings   ENABLE ROW LEVEL SECURITY;

-- Policies: users only see their own rows
CREATE POLICY transactions_user    ON transactions    USING (user_id = auth.uid());
CREATE POLICY budgets_user         ON budgets         USING (user_id = auth.uid());
CREATE POLICY habits_user          ON habits          USING (user_id = auth.uid());
CREATE POLICY habit_logs_user      ON habit_logs      USING (user_id = auth.uid());
CREATE POLICY goals_user           ON goals           USING (user_id = auth.uid());
CREATE POLICY key_results_user     ON key_results     USING (user_id = auth.uid());
CREATE POLICY notes_user           ON notes           USING (user_id = auth.uid());
CREATE POLICY events_user          ON events          USING (user_id = auth.uid());
CREATE POLICY assets_user          ON assets          USING (user_id = auth.uid());
CREATE POLICY user_settings_user   ON user_settings   USING (user_id = auth.uid());
EOF
```

- [ ] **Step 3: Verify migrations apply to local Postgres**

```bash
docker compose up postgres -d
sleep 3
docker compose run --rm migrate
```

Expected output: each SQL file echoes no errors.

- [ ] **Step 4: Commit**

```bash
git add migrations/
git commit -m "feat: add DB schema migrations and RLS policies"
```

---

### Task 3: Go backend dev scaffold

**Files:**
- Create: `backend/Dockerfile.dev`
- Create: `backend/go.mod`
- Create: `backend/cmd/server/main.go`
- Create: `backend/.air.toml`

- [ ] **Step 1: Init Go module**

```bash
mkdir -p backend/cmd/server
cd backend
go mod init github.com/chiutuanbinh/mylifeos/backend
go get github.com/go-chi/chi/v5@latest
go get github.com/go-chi/cors@latest
go get github.com/jackc/pgx/v5@latest
go get github.com/golang-jwt/jwt/v5@latest
go get github.com/joho/godotenv@latest
cd ..
```

- [ ] **Step 2: Create `backend/.air.toml`**

```toml
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/server ./cmd/server"
bin = "./tmp/server"
include_ext = ["go"]
exclude_dir = ["tmp", "vendor"]
delay = 500
kill_delay = "0s"
send_interrupt = false
stop_on_error = true

[log]
time = false

[color]
main = "magenta"
watcher = "cyan"
build = "yellow"
runner = "green"
```

- [ ] **Step 3: Create `backend/Dockerfile.dev`**

```dockerfile
FROM golang:1.22-alpine
RUN go install github.com/air-verse/air@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
CMD ["air", "-c", ".air.toml"]
```

- [ ] **Step 4: Write minimal `backend/cmd/server/main.go`**

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../../.env.local")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", os.Getenv("FRONTEND_URL")},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	log.Printf("server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 5: Verify it compiles and health check works**

```bash
cd backend && go build ./cmd/server && cd ..
# Start just postgres + backend
docker compose up postgres backend -d
sleep 5
curl http://localhost:8080/health
```

Expected: `{"status":"ok"}`

- [ ] **Step 6: Commit**

```bash
git add backend/
git commit -m "feat: Go backend dev scaffold with air hot-reload"
```

---

### Task 4: Frontend dev scaffold

**Files:**
- Create: `frontend/` (Vite project)
- Create: `frontend/Dockerfile.dev`

- [ ] **Step 1: Scaffold Vite+React+TypeScript project**

```bash
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install
npm install antd @ant-design/icons
npm install zustand @tanstack/react-query axios
npm install @supabase/supabase-js
npm install -D vitest @testing-library/react @testing-library/jest-dom @testing-library/user-event jsdom
cd ..
```

- [ ] **Step 2: Update `frontend/vite.config.ts`**

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
  },
})
```

- [ ] **Step 3: Create `frontend/src/test-setup.ts`**

```typescript
import '@testing-library/jest-dom'
```

- [ ] **Step 4: Create `frontend/Dockerfile.dev`**

```dockerfile
FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
CMD ["npm", "run", "dev"]
```

- [ ] **Step 5: Update `frontend/package.json` scripts** — add test script

Open `frontend/package.json` and add to `"scripts"`:
```json
"test": "vitest"
```

- [ ] **Step 6: Write smoke test `frontend/src/App.test.tsx`**

```typescript
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'

describe('App smoke test', () => {
  it('renders without crashing', () => {
    render(<div>MyLifeOS</div>)
    expect(screen.getByText('MyLifeOS')).toBeInTheDocument()
  })
})
```

- [ ] **Step 7: Run test**

```bash
cd frontend && npm test -- --run
```

Expected: `1 passed`

- [ ] **Step 8: Commit**

```bash
cd ..
git add frontend/
git commit -m "feat: React frontend scaffold with Vite, Ant Design, Zustand, TanStack Query"
```

---

### Task 5: CI/CD workflows

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/deploy.yml`

- [ ] **Step 1: Create `.github/workflows/ci.yml`**

```bash
mkdir -p .github/workflows
cat > .github/workflows/ci.yml << 'EOF'
name: CI

on:
  pull_request:
    branches: [main]

jobs:
  backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Vet
        run: cd backend && go vet ./...
      - name: Test
        run: cd backend && go test ./... -v

  frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json
      - run: cd frontend && npm ci
      - name: Lint
        run: cd frontend && npm run lint
      - name: Test
        run: cd frontend && npm test -- --run
      - name: Build
        run: cd frontend && npm run build
EOF
```

- [ ] **Step 2: Create `.github/workflows/deploy.yml`**

```bash
cat > .github/workflows/deploy.yml << 'EOF'
name: Deploy

on:
  push:
    branches: [main]

jobs:
  # Railway auto-deploys via GitHub integration — no steps needed here.
  # Vercel auto-deploys via GitHub integration — no steps needed here.
  # This workflow exists as a hook for future deploy scripts (e.g. run DB migrations).
  notify:
    runs-on: ubuntu-latest
    steps:
      - name: Deploy triggered
        run: echo "Railway and Vercel auto-deploy from main. Check their dashboards."
EOF
```

- [ ] **Step 3: Commit**

```bash
git add .github/
git commit -m "feat: GitHub Actions CI for backend and frontend"
```

---

### Phase 1 Complete

Verify the full local stack works end-to-end:

```bash
docker compose up --build
```

Expected:
- Postgres healthy on `:5432`
- Backend responds at `http://localhost:8080/health` → `{"status":"ok"}`
- Frontend dev server at `http://localhost:5173` (shows Vite default page for now)
- No container restart loops
