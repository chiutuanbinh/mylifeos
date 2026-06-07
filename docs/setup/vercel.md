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
