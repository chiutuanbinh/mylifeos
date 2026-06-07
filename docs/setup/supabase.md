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
