# Health & Habit Sync Setup

Two integration paths: iOS Shortcuts (Apple Health) and Google Fit API (Xiaomi Watch via bridge).

---

## Path A: iOS Apple Health via Shortcuts

No app required. iOS Shortcuts reads HealthKit and POSTs to your backend.

### Step 1: Create the backend endpoint

Add to `backend/cmd/server/main.go`:
```
POST /api/v1/health/sync
```

Expected payload:
```json
{
  "date": "2026-06-07",
  "steps": 8500,
  "sleep_hours": 7.5,
  "resting_heart_rate": 62,
  "active_calories": 420,
  "workouts": [
    { "type": "running", "duration_min": 30, "calories": 280 }
  ]
}
```

### Step 2: Create the iOS Shortcut

1. Open **Shortcuts** app on iPhone
2. Tap **+** → **Add Action**
3. Add these actions in order:

**Get Health Samples — Steps**
- Action: `Find Health Samples` → Type: `Steps` → Last 1 day

**Get Health Samples — Sleep**
- Action: `Find Health Samples` → Type: `Sleep Analysis` → Last 1 day

**Get Health Samples — Resting Heart Rate**
- Action: `Find Health Samples` → Type: `Resting Heart Rate` → Last 1 day

**Build JSON payload**
- Action: `Dictionary`
- Keys: `date` (current date formatted `YYYY-MM-DD`), `steps`, `sleep_hours`, `resting_heart_rate`

**POST to backend**
- Action: `Get Contents of URL`
- URL: `https://your-railway-app.up.railway.app/api/v1/health/sync`
- Method: `POST`
- Headers: `Authorization: Bearer YOUR_TOKEN`, `Content-Type: application/json`
- Body: JSON → select Dictionary from previous step

### Step 3: Automate the Shortcut

1. Go to **Shortcuts** → **Automation** tab → **+**
2. Choose trigger: **Time of Day** (e.g. 9:00 PM daily)
3. Select your shortcut → **Don't Ask Before Running**

### Step 4: Get your auth token

For local dev, backend accepts any token in dev mode.
For production, get your JWT by logging into MyLifeOS and copying the token from browser DevTools → Application → Memory (Zustand store) — or add a `/api/v1/auth/token` endpoint that returns the current session token.

> **Security note:** Treat your token like a password. If compromised, it grants full API access. Rotate by logging out and back in.

---

## Path B: Google Fit API (Xiaomi Watch bridge)

Xiaomi Watch → Mi Fitness app → Google Fit → your backend via Google Fit REST API.

### Prerequisites

- Android phone with Mi Fitness app installed
- Google Fit app installed and linked to Mi Fitness
- In Mi Fitness: Settings → Connected Apps → Google Fit → Enable sync

### Step 1: Enable Google Fit API

1. Go to [console.cloud.google.com](https://console.cloud.google.com)
2. Select your `mylifeos` project
3. **APIs & Services** → **Library** → search `Fitness API` → **Enable**

### Step 2: Create OAuth 2.0 Credentials

1. **APIs & Services** → **Credentials** → **Create Credentials** → **OAuth client ID**
2. Type: **Web application**
3. Name: `mylifeos-fitness`
4. Redirect URI:
   - Local: `http://localhost:8080/api/v1/auth/google-fit/callback`
   - Production: `https://your-railway-app.up.railway.app/api/v1/auth/google-fit/callback`
5. Copy **Client ID** and **Client Secret**

### Step 3: Add OAuth scopes

On OAuth consent screen, add scopes:
- `https://www.googleapis.com/auth/fitness.activity.read`
- `https://www.googleapis.com/auth/fitness.sleep.read`
- `https://www.googleapis.com/auth/fitness.heart_rate.read`
- `https://www.googleapis.com/auth/fitness.body.read`

### Step 4: Store credentials

Add to `.env.local` and Railway:
```
GOOGLE_FIT_CLIENT_ID=your-client-id
GOOGLE_FIT_CLIENT_SECRET=your-client-secret
GOOGLE_FIT_REDIRECT_URL=http://localhost:8080/api/v1/auth/google-fit/callback
```

Store refresh token in DB after first auth (same pattern as calendar_integrations table — see google-calendar.md):
```sql
INSERT INTO calendar_integrations (user_id, provider, refresh_token, ...)
VALUES ($1, 'google_fit', $2, ...);
```

### Step 5: Fetch data from Google Fit API

**Endpoint:** `POST https://www.googleapis.com/fitness/v1/users/me/dataset:aggregate`

**Steps (last 24h):**
```json
{
  "aggregateBy": [{ "dataTypeName": "com.google.step_count.delta" }],
  "bucketByTime": { "durationMillis": 86400000 },
  "startTimeMillis": 1717718400000,
  "endTimeMillis": 1717804800000
}
```

**Sleep:**
```json
{
  "aggregateBy": [{ "dataTypeName": "com.google.sleep.segment" }],
  "bucketByTime": { "durationMillis": 86400000 },
  "startTimeMillis": ...,
  "endTimeMillis": ...
}
```

### Step 6: Add sync endpoint

```
POST /api/v1/health/google-fit/sync
```

Backend flow:
1. Load refresh token from DB for authenticated user
2. Exchange for access token
3. Call Google Fit aggregate API for today
4. Map response → your `habits` / `habit_logs` tables
5. Return sync summary

### Step 7: Automate sync

Options (pick one):

**Option A: Manual button** — Add "Sync from Google Fit" button in Health page
```tsx
// frontend/src/pages/Health.tsx
<Button onClick={() => syncMutation.mutate()}>Sync from Google Fit</Button>
```

**Option B: Cron job on Railway** — Add a daily sync job to backend
- Railway supports cron via a separate worker service
- Or use a simple time.Ticker in main.go for single-instance deployments

---

## Data Mapping

| Source | Field | MyLifeOS table | Column |
|--------|-------|----------------|--------|
| Apple Health / Google Fit | steps | habit_logs | value |
| Apple Health / Google Fit | sleep_hours | habit_logs | value |
| Apple Health / Google Fit | resting_heart_rate | habit_logs | value |
| Apple Health / Google Fit | active_calories | habit_logs | value |

Create habits with names like `steps`, `sleep`, `resting_hr` — sync populates logs automatically.

---

## Security Checklist

- [ ] `GOOGLE_FIT_CLIENT_SECRET` only in backend env, never Vercel
- [ ] Refresh tokens encrypted at rest in DB
- [ ] iOS Shortcut token rotated if device is lost
- [ ] Google Fit OAuth scopes minimal (read-only)
- [ ] Sync endpoint validates user owns the data being written
