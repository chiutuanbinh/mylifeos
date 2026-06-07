# Google Calendar Integration Setup

## Prerequisites

- Google account with access to Google Cloud Console
- MyLifeOS backend deployed (or running locally)

---

## Step 1: Create a Google Cloud Project

1. Go to [console.cloud.google.com](https://console.cloud.google.com)
2. Click **Select a project** → **New Project**
3. Name it `mylifeos` → **Create**

---

## Step 2: Enable the Calendar API

1. In the project, go to **APIs & Services** → **Library**
2. Search `Google Calendar API` → **Enable**

---

## Step 3: Configure OAuth Consent Screen

1. Go to **APIs & Services** → **OAuth consent screen**
2. Choose **External** (for personal use; Internal requires Google Workspace)
3. Fill required fields:
   - **App name**: MyLifeOS
   - **User support email**: your email
   - **Developer contact email**: your email
4. Click **Save and Continue** through Scopes (add later)
5. Under **Test users**, add your Google account email
6. Status stays **Testing** — fine for personal use (100-user limit)

> **Security note:** Do NOT publish the app. Keep it in Testing mode. Publishing requires Google verification and exposes OAuth to anyone.

---

## Step 4: Create OAuth 2.0 Credentials

1. Go to **APIs & Services** → **Credentials** → **Create Credentials** → **OAuth client ID**
2. Application type: **Web application**
3. Name: `mylifeos-backend`
4. **Authorized redirect URIs** — add:
   - Local dev: `http://localhost:8080/api/v1/auth/google/callback`
   - Production: `https://your-railway-app.up.railway.app/api/v1/auth/google/callback`
5. Click **Create**
6. Copy **Client ID** and **Client Secret** — store securely

---

## Step 5: Add Scopes

1. Go back to **OAuth consent screen** → **Edit App** → **Scopes**
2. Add scope: `https://www.googleapis.com/auth/calendar.readonly`
   - Read-only is sufficient for displaying events
   - Use `calendar.events` if you need write access (create/edit events)
3. Save

---

## Step 6: Store Credentials Securely

### Local dev — `.env.local`

```
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback
```

> **Never commit `.env.local`** — it is already in `.gitignore`.

### Railway (production backend)

1. Railway dashboard → your backend service → **Variables**
2. Add:
   - `GOOGLE_CLIENT_ID`
   - `GOOGLE_CLIENT_SECRET`
   - `GOOGLE_REDIRECT_URL` (production URL)

> **Security note:** Never put `GOOGLE_CLIENT_SECRET` in frontend env vars or Vercel. It must only live in the backend.

---

## Step 7: Token Storage (when you implement the feature)

When a user connects their calendar, your backend receives a **refresh token** after the OAuth callback. Store it encrypted in the database, never in logs or responses.

Suggested table:

```sql
CREATE TABLE calendar_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT 'google',
    access_token TEXT,
    refresh_token TEXT NOT NULL,
    token_expiry TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(user_id, provider)
);
```

> **Security note:** `refresh_token` grants long-lived access. Encrypt at rest using `pgcrypto` or application-level AES before storing.

---

## Revoking Access

Users can revoke at any time:
- [myaccount.google.com/permissions](https://myaccount.google.com/permissions) → remove MyLifeOS
- Your app should also provide a "Disconnect" button that calls `DELETE /api/v1/calendar/integration` and deletes the stored tokens

---

## Microsoft Calendar (alternative)

Same OAuth 2.0 pattern via Microsoft Graph API:

1. Register app at [portal.azure.com](https://portal.azure.com) → **Azure Active Directory** → **App registrations**
2. Add redirect URI, generate client secret
3. Grant API permission: `Calendars.Read` (delegated)
4. Endpoint: `GET https://graph.microsoft.com/v1.0/me/events`

> More complex than Google due to Azure AD tenant configuration. Recommended only if users are primarily on Outlook/Teams.
