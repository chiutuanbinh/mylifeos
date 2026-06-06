# MyLifeOS — Design Spec

_Date: 2026-06-06_

## Overview

Personal ERP web app. Single user v1, SSO-ready for multi-user later. Always-online. React + Ant Design frontend, Go backend, Supabase (Postgres + Auth), deployed on Vercel (FE) + Railway (BE).

---

## Architecture

```
GitHub (git@github.com:chiutuanbinh/mylifeos.git)
  │
  ├── /frontend   →  Vercel (CDN, auto-deploy on push to main)
  └── /backend    →  Railway (Go binary, auto-deploy on push to main)
                         │
                    Supabase (Postgres + Auth)
```

**Request flow:**
1. Browser loads React SPA from Vercel
2. User authenticates via Supabase Auth (email+password v1, Google/GitHub OAuth later)
3. React calls Go API on Railway via HTTPS with `Authorization: Bearer <supabase_jwt>`
4. Go validates JWT, queries Supabase Postgres
5. Supabase RLS policies enforce user isolation at DB layer

**Monorepo layout:**
```
mylifeos/
├── frontend/                  # Vite + React 18 + Ant Design 5
│   ├── src/
│   │   ├── pages/             # one file per module
│   │   ├── components/        # shared UI
│   │   ├── hooks/             # data fetching hooks
│   │   ├── api/               # typed API client
│   │   └── store/             # auth state (Zustand or Context)
│   ├── vite.config.ts
│   └── package.json
├── backend/                   # Go 1.22+
│   ├── cmd/server/main.go
│   └── internal/
│       ├── handlers/          # HTTP handlers per module
│       ├── models/            # DB structs
│       ├── repo/              # Postgres queries (sqlc or raw pgx)
│       └── middleware/        # JWT validation, CORS, logging
├── docker-compose.yml         # local dev: Postgres + Go + Vite
├── Makefile                   # make dev, make test, make migrate
├── docs/
│   ├── setup/
│   │   ├── supabase.md
│   │   ├── railway.md
│   │   ├── vercel.md
│   │   └── local-dev.md
│   └── migration/
│       └── cloud-run.md
└── .github/workflows/
    ├── ci.yml                 # test + lint on PR
    └── deploy.yml             # deploy on push to main
```

---

## Modules (v1)

| Module | Description |
|--------|-------------|
| Dashboard | Summary stat cards, recent transactions, habits, goals, upcoming events |
| Finance & Budget | Transactions log, budget by category, spending vs budget chart |
| Health & Habits | Daily habit checklist, streak tracking, 12-week heatmap |
| Goals & OKRs | Goal cards with progress, key results checkboxes |
| Notes | Searchable card grid, tags, pinned notes |
| Calendar | Monthly grid, event list panel |
| Inventory | Asset register, category summary |
| Settings | Profile, notifications, module toggles |

---

## Data Models

All tables include `user_id uuid NOT NULL REFERENCES auth.users(id)` with RLS policy `USING (user_id = auth.uid())`.

```sql
-- Finance
transactions   (id uuid PK, user_id, date date, description text, category text, amount numeric, created_at)
budgets        (id uuid PK, user_id, category text, monthly_limit numeric, created_at)
               UNIQUE (user_id, category)

-- Health
habits         (id uuid PK, user_id, name text, icon text, created_at)
habit_logs     (id uuid PK, habit_id, user_id, logged_date date, done bool)
               UNIQUE (habit_id, logged_date)

-- Goals
goals          (id uuid PK, user_id, name text, description text, target_date date, progress int, color text, created_at)
key_results    (id uuid PK, goal_id, user_id, description text, done bool)

-- Notes
notes          (id uuid PK, user_id, title text, content text, tags text[], pinned bool, created_at, updated_at)

-- Calendar
events         (id uuid PK, user_id, title text, start_at timestamptz, end_at timestamptz, color text, all_day bool)

-- Inventory
assets         (id uuid PK, user_id, name text, category text, value numeric, purchased_at date, notes text)

-- Settings
user_settings  (user_id uuid PK, notifications jsonb, modules_enabled jsonb)
```

**Scaling pattern for future subsystems:** add tables with `user_id`, new Go handler, new React page, new sidebar entry. Nothing else changes.

---

## API Design

Base: `/api/v1/` — all routes require `Authorization: Bearer <supabase_jwt>`.

```
GET    /dashboard/summary          # single query: stat cards for all modules

GET    /transactions               # ?limit=&offset=&category=&from=&to=
POST   /transactions
DELETE /transactions/:id

GET    /budgets
PUT    /budgets/:category          # upsert

GET    /habits
POST   /habits
DELETE /habits/:id
GET    /habits/logs?date=
POST   /habits/:id/log             # toggle done for date

GET    /goals
POST   /goals
PATCH  /goals/:id
DELETE /goals/:id
POST   /goals/:id/key-results
PATCH  /goals/:id/key-results/:kr_id

GET    /notes?search=&tags=&pinned=
POST   /notes
PATCH  /notes/:id
DELETE /notes/:id

GET    /events?from=&to=
POST   /events
PATCH  /events/:id
DELETE /events/:id

GET    /assets
POST   /assets
PATCH  /assets/:id
DELETE /assets/:id

GET    /settings
PUT    /settings
```

---

## CI/CD

```yaml
# ci.yml — on every PR
- go test ./... && go vet ./...
- npm run lint && npm run build

# deploy.yml — on push to main
- Railway: auto-deploy via GitHub integration
- Vercel: auto-deploy via GitHub integration
```

**Environment variables:**

| Service | Vars |
|---------|------|
| Railway | `SUPABASE_URL`, `SUPABASE_SERVICE_ROLE_KEY`, `PORT` |
| Vercel | `VITE_API_URL`, `VITE_SUPABASE_URL`, `VITE_SUPABASE_ANON_KEY` |

---

## Local Dev Mode

`docker-compose.yml` runs:
- `postgres:16` — local DB, migrations applied on start
- Go backend with `air` hot-reload
- Vite dev server with HMR

```bash
make dev       # start all services
make migrate   # run DB migrations
make test      # go test + npm test
```

No Supabase account needed for local development. `.env.local` points to local Postgres.

---

## Documentation to Write

### `docs/setup/supabase.md`
- Create Supabase project
- Run schema migrations via Supabase SQL editor
- Enable RLS on all tables + add policies
- Get `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY`
- Enable Google/GitHub OAuth (future SSO)
- Security: never expose service role key in frontend

### `docs/setup/railway.md`
- Create Railway project, link GitHub repo
- Set environment variables
- Custom domain (optional)
- Monitor cold starts (free tier sleeps after inactivity)

### `docs/setup/vercel.md`
- Import GitHub repo, set root to `frontend/`
- Set environment variables
- Preview deployments per PR

### `docs/setup/local-dev.md`
- Prerequisites: Docker, Go 1.22+, Node 20+
- Clone repo, copy `.env.example` → `.env.local`
- `make dev` to start
- `make migrate` to apply schema

### `docs/migration/cloud-run.md`
- When to migrate: Railway free tier (500h/month) exhausted, or need always-warm instances
- Dockerfile for Go binary
- Deploy to Cloud Run (us-central1 cheapest region)
- Connect to Supabase (same JWT validation, same DB URL)
- Cost comparison: Railway Starter $5/mo vs Cloud Run ~$0–3/mo at low traffic
- Steps: `gcloud run deploy`, set env vars, update `VITE_API_URL` in Vercel

---

## Expected Behavior (Verification Checklist)

- [ ] Login page renders, Supabase auth works, JWT stored in memory (not localStorage)
- [ ] All 8 sidebar pages navigate without full reload
- [ ] Dashboard `/summary` loads all 4 stat cards in one request
- [ ] Transactions CRUD works, amounts display green (income) / red (expense)
- [ ] Habit log toggles persist across page refreshes
- [ ] Goal progress bars update when key results checked
- [ ] Notes search filters by title/content and tags
- [ ] Calendar shows events on correct dates
- [ ] Assets sum correctly in inventory category cards
- [ ] Settings module toggles hide/show sidebar items
- [ ] RLS: two different user_ids cannot access each other's data
- [ ] CI passes on PR before merge
- [ ] `make dev` starts all services locally with no Supabase account
- [ ] `make migrate` applies schema to local Postgres cleanly
