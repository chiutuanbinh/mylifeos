# Google Calendar Sync

Import events from Google Calendar into MyLifeOS.

---

## How It Works

```
User clicks "Sync GCal"
  → frontend calls supabase.auth.getSession() → gets provider_token
  → POST /api/v1/calendar/google/sync { provider_token, time_min, time_max }
  → backend calls Google Calendar API with the token
  → events upserted into DB by google_event_id
  → calendar re-fetches, showing merged events
```

Google-synced events are **read-only** — they cannot be edited/deleted locally (the `google_event_id` tracks their origin). Manual events you create in MyLifeOS are not pushed to Google Calendar.

---

## Setup

### 1. Google Cloud Console — add Calendar scope

In your OAuth client (used for Google SSO):

1. **APIs & Services → OAuth consent screen → Scopes**
2. Add scope: `https://www.googleapis.com/auth/calendar.readonly`
3. Save

### 2. Enable Google Calendar API

1. **APIs & Services → Library**
2. Search "Google Calendar API" → **Enable**

### 3. Run DB migration

```bash
psql $DATABASE_URL -f migrations/003_google_calendar.sql
```

Or run it in Supabase SQL Editor.

---

## Usage

1. Sign in with Google (grants calendar read access)
2. Navigate to **Calendar**
3. Click **Sync GCal** — imports events for the current month
4. Navigate months then sync again to import other months

> If you see "No Google access token": sign out and sign back in — the new `calendar.readonly` scope requires re-consent.

---

## Troubleshooting

| Error | Fix |
|-------|-----|
| "No Google access token" | Sign out → sign in again (re-grants calendar scope) |
| "google calendar fetch failed: status 403" | Calendar API not enabled in Google Cloud Console |
| "google calendar fetch failed: status 401" | provider_token expired — sign out and back in |
| Synced 0 events | No events in the selected month range in Google Calendar |
