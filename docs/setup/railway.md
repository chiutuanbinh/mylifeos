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
