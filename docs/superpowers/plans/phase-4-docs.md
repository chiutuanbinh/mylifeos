# MyLifeOS Phase 4: Documentation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.
>
> **Prerequisite:** Phase 1–3 complete.

**Goal:** All setup guides and migration docs written so the app can be deployed to production and operators know how to run it.

---

### Task 1: Local dev guide

**Files:**
- Create: `docs/setup/local-dev.md`

- [ ] **Step 1: Write `docs/setup/local-dev.md`**

```markdown
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
```

- [ ] **Step 2: Commit**

```bash
git add docs/setup/local-dev.md
git commit -m "docs: local development guide"
```

---

### Task 2: Supabase setup guide

**Files:**
- Create: `docs/setup/supabase.md`

- [ ] **Step 1: Write `docs/setup/supabase.md`**

```markdown
# Supabase Setup

## Create project

1. Go to https://supabase.com → New project
2. Choose a region close to your users
3. Set a strong database password — save it, you'll need it
4. Wait ~2 minutes for provisioning

## Run schema migrations

1. In the Supabase dashboard → SQL Editor → New query
2. Paste the contents of `migrations/001_schema.sql` → Run
3. Paste the contents of `migrations/002_rls.sql` → Run

Verify tables exist: Table Editor should show all tables.

## Get API keys

Settings → API:

| Key | Where to use |
|-----|-------------|
| Project URL | `SUPABASE_URL` (Railway) and `VITE_SUPABASE_URL` (Vercel) |
| `anon` public key | `VITE_SUPABASE_ANON_KEY` (Vercel only) |
| `service_role` secret key | `SUPABASE_SERVICE_ROLE_KEY` (Railway only — NEVER expose in frontend) |

Settings → JWT:

| Key | Where to use |
|-----|-------------|
| JWT Secret | `SUPABASE_JWT_SECRET` (Railway) |

## Enable RLS

`migrations/002_rls.sql` already enables RLS and adds policies. Verify in Table Editor → each table should show "RLS enabled" badge.

## Security rules

- **Never** put `SUPABASE_SERVICE_ROLE_KEY` in frontend code or Vercel env vars.
- **Never** put `SUPABASE_JWT_SECRET` in frontend code.
- The `anon` key is safe in frontend — it has no access without a valid user JWT.
- RLS policies mean even if someone gets the `anon` key, they can only see their own rows.

## Enable OAuth (future SSO)

Authentication → Providers:

- **Google**: requires Google Cloud Console OAuth credentials
  1. Create OAuth 2.0 client at https://console.cloud.google.com
  2. Authorized redirect URI: `https://<your-supabase-project>.supabase.co/auth/v1/callback`
  3. Paste Client ID and Secret into Supabase Google provider

- **GitHub**: requires GitHub OAuth App
  1. Create at https://github.com/settings/developers
  2. Callback URL: `https://<your-supabase-project>.supabase.co/auth/v1/callback`
  3. Paste Client ID and Secret into Supabase GitHub provider

No backend code changes needed — Supabase issues the same JWT format for all providers.

## Free tier limits

| Resource | Limit |
|----------|-------|
| Database size | 500 MB |
| Monthly active users | 50,000 |
| Storage | 1 GB |
| Edge function invocations | 500,000 / month |

At typical personal use (~1 user, moderate data), these limits are not a concern.
```

- [ ] **Step 2: Commit**

```bash
git add docs/setup/supabase.md
git commit -m "docs: Supabase setup and security guide"
```

---

### Task 3: Railway deployment guide

**Files:**
- Create: `docs/setup/railway.md`
- Create: `backend/Dockerfile`

- [ ] **Step 1: Create production `backend/Dockerfile`**

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

- [ ] **Step 2: Write `docs/setup/railway.md`**

```markdown
# Railway Deployment

## Create project

1. Go to https://railway.app → New Project → Deploy from GitHub repo
2. Select `chiutuanbinh/mylifeos`
3. Set root directory to `backend/`

Railway auto-detects the `Dockerfile` and builds it.

## Set environment variables

In Railway → Service → Variables, add:

| Variable | Value |
|----------|-------|
| `ENV` | `production` |
| `PORT` | `8080` |
| `DATABASE_URL` | From Supabase: Settings → Database → Connection string (URI format) |
| `SUPABASE_URL` | From Supabase: Settings → API → Project URL |
| `SUPABASE_JWT_SECRET` | From Supabase: Settings → JWT → JWT Secret |
| `SUPABASE_SERVICE_ROLE_KEY` | From Supabase: Settings → API → service_role key |
| `FRONTEND_URL` | Your Vercel URL (e.g. `https://mylifeos.vercel.app`) |

## Auto-deploy

Railway auto-deploys on every push to `main`. No additional CI config needed.

## Custom domain (optional)

Service → Settings → Domains → Add custom domain.

## Cold starts (free tier)

The free tier (500 hours/month) sleeps after ~10 minutes of inactivity. Cold starts take ~1–3 seconds.

**To avoid cold starts:**
- Upgrade to Railway Starter ($5/mo) — always-on
- Or migrate to Cloud Run (see `docs/migration/cloud-run.md`)

## Monitoring

Railway → Service → Logs tab shows real-time logs.
Health check: `GET https://<your-railway-url>/health` should return `{"status":"ok"}`.

## Redeploy manually

Railway dashboard → Deployments → Redeploy.
```

- [ ] **Step 3: Commit**

```bash
git add backend/Dockerfile docs/setup/railway.md
git commit -m "docs: Railway deployment guide and production Dockerfile"
```

---

### Task 4: Vercel deployment guide

**Files:**
- Create: `docs/setup/vercel.md`
- Create: `frontend/vercel.json`

- [ ] **Step 1: Create `frontend/vercel.json`**

```json
{
  "rewrites": [{ "source": "/(.*)", "destination": "/index.html" }]
}
```

This ensures React Router works on direct URL access (page refreshes, deep links).

- [ ] **Step 2: Write `docs/setup/vercel.md`**

```markdown
# Vercel Deployment

## Import project

1. Go to https://vercel.com → Add New → Project → Import from GitHub
2. Select `chiutuanbinh/mylifeos`
3. Set **Root Directory** to `frontend/`
4. Framework preset: **Vite** (auto-detected)

## Set environment variables

In Vercel → Project → Settings → Environment Variables:

| Variable | Value |
|----------|-------|
| `VITE_API_URL` | Your Railway backend URL + `/api/v1` (e.g. `https://mylifeos-production.up.railway.app/api/v1`) |
| `VITE_SUPABASE_URL` | From Supabase: Settings → API → Project URL |
| `VITE_SUPABASE_ANON_KEY` | From Supabase: Settings → API → anon public key |

## Auto-deploy

Vercel auto-deploys on every push to `main`. Pull requests get preview deployments automatically.

## Preview deployments

Each PR gets a unique preview URL (e.g. `https://mylifeos-git-feature-branch.vercel.app`).
The preview URL points to the same Railway backend — test end-to-end before merging.

## Free tier limits

| Resource | Limit |
|----------|-------|
| Bandwidth | 100 GB / month |
| Builds | 6,000 minutes / month |
| Deployments | Unlimited |

More than sufficient for personal use.

## Custom domain (optional)

Project → Domains → Add domain. Vercel handles SSL automatically.
```

- [ ] **Step 3: Commit**

```bash
git add frontend/vercel.json docs/setup/vercel.md
git commit -m "docs: Vercel deployment guide and SPA rewrite config"
```

---

### Task 5: Cloud Run migration guide

**Files:**
- Create: `docs/migration/cloud-run.md`

- [ ] **Step 1: Write `docs/migration/cloud-run.md`**

```markdown
# Migrating Backend to Google Cloud Run

## When to migrate

Railway free tier gives 500 hours/month — enough for ~20 days of continuous uptime.
Migrate when:
- You need always-on (no cold starts)
- Monthly requests exceed Railway Starter plan value ($5/mo)
- You want cost-per-request billing (~$0–3/mo at low personal traffic)

## Cost comparison

| Service | Always-on | Cost |
|---------|-----------|------|
| Railway free | No (sleeps) | $0 (500h/mo) |
| Railway Starter | Yes | $5/mo |
| Cloud Run | No (scales to 0) | ~$0–3/mo at low traffic |
| Cloud Run min-instances=1 | Yes | ~$7/mo (us-central1) |

Cloud Run billed only on requests — ideal for low-traffic personal apps.

## Prerequisites

```bash
brew install google-cloud-sdk
gcloud auth login
gcloud config set project YOUR_PROJECT_ID
gcloud services enable run.googleapis.com artifactregistry.googleapis.com
```

## Build and push Docker image

```bash
# From repo root
gcloud artifacts repositories create mylifeos \
  --repository-format=docker \
  --location=us-central1

docker build -t us-central1-docker.pkg.dev/YOUR_PROJECT_ID/mylifeos/backend:latest ./backend
docker push us-central1-docker.pkg.dev/YOUR_PROJECT_ID/mylifeos/backend:latest
```

## Deploy to Cloud Run

```bash
gcloud run deploy mylifeos-backend \
  --image us-central1-docker.pkg.dev/YOUR_PROJECT_ID/mylifeos/backend:latest \
  --region us-central1 \
  --platform managed \
  --allow-unauthenticated \
  --port 8080 \
  --set-env-vars "ENV=production,SUPABASE_URL=...,SUPABASE_JWT_SECRET=...,DATABASE_URL=..."
```

## Update frontend to point to Cloud Run

In Vercel → Environment Variables, update:

```
VITE_API_URL = https://mylifeos-backend-xxxx-uc.a.run.app/api/v1
```

Redeploy Vercel (push an empty commit or trigger manually).

## Set up CI/CD for Cloud Run

Add to `.github/workflows/deploy.yml`:

```yaml
deploy-backend:
  runs-on: ubuntu-latest
  if: github.ref == 'refs/heads/main'
  steps:
    - uses: actions/checkout@v4
    - uses: google-github-actions/auth@v2
      with:
        credentials_json: ${{ secrets.GCP_SA_KEY }}
    - uses: google-github-actions/deploy-cloudrun@v2
      with:
        service: mylifeos-backend
        region: us-central1
        image: us-central1-docker.pkg.dev/${{ vars.GCP_PROJECT_ID }}/mylifeos/backend:latest
```

Create a GCP Service Account with `Cloud Run Admin` + `Artifact Registry Writer` roles.
Add the JSON key as `GCP_SA_KEY` secret in GitHub repo settings.

## Disable Railway

Once Cloud Run is verified working, delete the Railway service to avoid wasted hours.

## No backend code changes needed

The same `Dockerfile` and environment variables work identically on Cloud Run.
The only change is where the image runs.
```

- [ ] **Step 2: Commit**

```bash
git add docs/migration/cloud-run.md
git commit -m "docs: Cloud Run migration guide with cost comparison"
```

---

### Phase 4 Complete — Push to GitHub

```bash
git push -u origin main
```

All 4 phases committed. Verify CI passes on GitHub → Actions tab.
```

---

## Self-Review Against Spec

| Spec requirement | Covered by |
|-----------------|-----------|
| Login page renders, Supabase auth works | Phase 3 Task 3 |
| JWT stored in memory (not localStorage) | Phase 3 Task 1 — Zustand store, no localStorage |
| All 8 sidebar pages navigate without full reload | Phase 3 Task 2 — React Router SPA |
| Dashboard `/summary` loads in one request | Phase 2 Task 2, Phase 3 Task 4 |
| Transactions CRUD | Phase 2 Task 3, Phase 3 Task 5 |
| Habit log toggles persist | Phase 2 Task 4, Phase 3 Task 6 |
| Goal progress bars + key results | Phase 2 Task 5, Phase 3 Task 7 |
| Notes search | Phase 2 Task 6, Phase 3 Task 8 |
| Calendar events on correct dates | Phase 2 Task 6, Phase 3 Task 9 |
| Assets sum by category | Phase 2 Task 6, Phase 3 Task 10 |
| Settings module toggles | Phase 2 Task 6, Phase 3 Task 11 |
| RLS user isolation | Phase 1 Task 2 — `migrations/002_rls.sql` |
| CI passes on PR | Phase 1 Task 5 |
| `make dev` starts all services | Phase 1 Task 1–4 |
| `make migrate` applies schema | Phase 1 Task 1, 2 |
| Supabase setup doc | Phase 4 Task 2 |
| Railway doc | Phase 4 Task 3 |
| Vercel doc | Phase 4 Task 4 |
| Local dev doc | Phase 4 Task 1 |
| Cloud Run migration doc | Phase 4 Task 5 |
