# Google SSO Setup

Enable Google sign-in via Supabase OAuth.

---

## 1. Google Cloud Console

1. Go to [console.cloud.google.com](https://console.cloud.google.com)
2. Select your project (or create one)
3. **APIs & Services → Credentials → Create Credentials → OAuth 2.0 Client ID**
4. Application type: **Web application**
5. Name: `MyLifeOS`
6. **Authorized redirect URIs** — add:
   ```
   https://<your-project-ref>.supabase.co/auth/v1/callback
   ```
   Replace `<your-project-ref>` with your Supabase project reference ID (e.g. `xvwpvhbiiwhdeimxxgxg`).
7. Click **Create** — copy the **Client ID** and **Client Secret**

> If you don't have an OAuth consent screen configured yet, Google will prompt you to create one first. Set User Type to **External**, add your email as a test user, and publish when ready.

---

## 2. Supabase Dashboard

1. Go to **Authentication → Providers → Google**
2. Toggle **Enable**
3. Paste the **Client ID** and **Client Secret** from step 1
4. Save

---

## 3. Add Redirect URL

1. In Supabase: **Authentication → URL Configuration**
2. Under **Redirect URLs**, add:
   ```
   https://<your-vercel-domain>/auth/callback
   ```
   Example: `https://mylifeos.vercel.app/auth/callback`
3. Also add `http://localhost:5173/auth/callback` for local dev

---

## 4. Local Dev

No extra env vars needed. The frontend reads `VITE_SUPABASE_URL` and `VITE_SUPABASE_ANON_KEY` (already set in `.env.local`).

Test locally:
```bash
cd frontend && npm run dev
```
Open `http://localhost:5173/login` → click **Continue with Google**.

---

## 5. How It Works

```
User clicks "Continue with Google"
  → supabase.auth.signInWithOAuth({ provider: 'google' })
  → browser redirects to Google login
  → Google redirects to Supabase callback URL
  → Supabase redirects to /auth/callback on your app
  → AuthCallbackPage reads session token
  → user is signed in, redirected to /
```

**Relevant files:**
- `frontend/src/pages/LoginPage.tsx` — Google button
- `frontend/src/pages/AuthCallbackPage.tsx` — handles redirect, sets session
- `frontend/src/store/auth.ts` — `signInWithGoogle()`, `setSession()`

---

## Troubleshooting

| Error | Fix |
|-------|-----|
| `redirect_uri_mismatch` | Redirect URI in Google Cloud doesn't match Supabase callback URL exactly |
| Stuck on Google screen / no redirect | Check Supabase provider is enabled and Client ID/Secret are correct |
| `/auth/callback` returns to login | Supabase redirect URL list doesn't include your app domain |
| Works locally, fails on Vercel | Add production domain to Supabase redirect URL list |
