# MyLifeOS — Claude Instructions

## Branch workflow (REQUIRED)

- **Never push directly to `main`** — branch protection blocks it
- All changes go through a PR branch → CI → auto-merge
- Branch naming: `feat/<name>`, `fix/<name>`, `chore/<name>`

## Before creating a PR (REQUIRED steps in order)

1. **Run backend tests** — must pass with ≥80% coverage **per file**:
   ```bash
   cd backend && go test ./internal/handlers/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic
   bash scripts/hooks/pre-commit   # checks per-file ≥80%
   ```

2. **Run frontend lint + build** — must be clean:
   ```bash
   cd frontend && npm run lint && npm run build
   ```

3. **Run integration smoke tests** using agent-browser:
   ```bash
   bash scripts/integration-test.sh
   ```
   - Starts local stack if not running (`docker compose up -d`)
   - Opens browser, checks pages load, no JS crashes
   - Must pass before PR creation
   - Run with `--headed` to watch: `bash scripts/integration-test.sh --headed`

4. **Create PR** (only after steps 1–3 pass):
   ```bash
   git push -u origin <branch>
   gh pr create --title "..." --body "..."
   gh pr merge --auto --squash   # enable auto-merge immediately
   ```

## Auto-merge

Always run `gh pr merge --auto --squash` right after `gh pr create`.
GitHub will merge automatically when all CI checks pass.

## Test coverage rule

Coverage gate is **≥80% per file** in `handlers` and `middleware` packages.
Enforced by `scripts/hooks/pre-commit` (git hook) and CI.
If you add a new handler, add corresponding tests before the PR.

## Security constraints (never violate)

- JWT stored in memory only — never localStorage
- `SUPABASE_SERVICE_ROLE_KEY` — backend only, never frontend or Vercel env
- `SUPABASE_JWT_SECRET` — backend only
- `.env.local` — never commit
- `GOOGLE_CLIENT_SECRET`, `NOTION_API_TOKEN`, `FINNHUB_API_KEY` — backend env only

## Stack

- **Frontend**: Vite + React + Ant Design — runs on `localhost:5173`
- **Backend**: Go + chi — runs on `localhost:8080`
- **DB**: Supabase (prod) / PostgreSQL via Docker (local)
- **Deploy**: Railway (backend), Vercel (frontend)
- **Migrations**: Supabase CLI (`supabase/migrations/`) — run by CI on merge to main

## Local dev

```bash
docker compose up -d          # start postgres + backend + frontend
# or run individually:
cd backend && go run ./cmd/server
cd frontend && npm run dev
```
