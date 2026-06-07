# Google Sheets Integration Setup

## Overview

Two-way sync between MyLifeOS Postgres data and Google Sheets using a **service account** (no user OAuth required). Sheets acts as a convenient editor; Postgres remains source of truth.

**Recommended modules to sync:** Finance (transactions, budgets), Inventory (assets)

---

## Step 1: Enable Google Sheets API

1. Go to [console.cloud.google.com](https://console.cloud.google.com)
2. Select your `mylifeos` project (created in google-calendar.md) or create a new one
3. Go to **APIs & Services** → **Library**
4. Search `Google Sheets API` → **Enable**
5. Also enable `Google Drive API` (required to share sheets programmatically)

---

## Step 2: Create a Service Account

1. Go to **APIs & Services** → **Credentials** → **Create Credentials** → **Service account**
2. Name: `mylifeos-sheets`
3. Click **Create and Continue** — skip optional role grants → **Done**
4. Click the new service account → **Keys** tab → **Add Key** → **Create new key** → **JSON**
5. Download the JSON key file — store securely, treat like a password

> **Security note:** This key grants write access to any Sheet shared with it. Never commit it to git. Never expose it in frontend or Vercel env vars.

---

## Step 3: Store the Service Account Key

### Local dev

Option A — file path:
```
GOOGLE_SERVICE_ACCOUNT_KEY_PATH=/path/to/mylifeos-sheets-key.json
```

Option B — inline JSON (better for Railway):
```
GOOGLE_SERVICE_ACCOUNT_KEY_JSON={"type":"service_account","project_id":...}
```

Add to `.env.local`. Never commit.

### Railway (production)

Paste the entire JSON content as a single-line string into a Railway variable:
- Variable name: `GOOGLE_SERVICE_ACCOUNT_KEY_JSON`
- Value: minified JSON (use `cat key.json | jq -c .`)

> **Security note:** Do not put service account credentials in Vercel (frontend). Backend only.

---

## Step 4: Create and Share the Target Sheet

1. Create a new Google Sheet for each module you want to sync, e.g.:
   - `MyLifeOS - Transactions`
   - `MyLifeOS - Assets`
2. Click **Share** → paste the service account email (looks like `mylifeos-sheets@your-project.iam.gserviceaccount.com`)
3. Grant **Editor** access
4. Copy the **Sheet ID** from the URL:
   `https://docs.google.com/spreadsheets/d/SHEET_ID_HERE/edit`

Store Sheet IDs in env vars:
```
SHEETS_TRANSACTIONS_ID=your-sheet-id
SHEETS_ASSETS_ID=your-sheet-id
```

---

## Step 5: Sheet Structure

### Transactions sheet

| id | date | amount | category | description | type | user_id |
|----|------|--------|----------|-------------|------|---------|
| uuid | 2026-01-15 | 50.00 | food | Lunch | expense | uuid |

- Row 1: headers (exact column names matter for import)
- `id` column: used to match rows on sync (upsert logic)
- Do not edit `id` or `user_id` columns

### Assets sheet

| id | name | category | value | purchase_date | notes | user_id |
|----|------|----------|-------|---------------|-------|---------|

---

## Step 6: Sync Endpoints (when you implement)

Add these routes to `backend/cmd/server/main.go`:

```
POST /api/v1/export/transactions   → Postgres → write to Sheet
POST /api/v1/import/transactions   → read Sheet → upsert to Postgres
POST /api/v1/export/assets         → Postgres → write to Sheet
POST /api/v1/import/assets         → read Sheet → upsert to Postgres
```

### Export logic (Postgres → Sheet)
1. Query all rows for `user_id`
2. Clear sheet rows (keep header)
3. Write all rows via `spreadsheets.values.update`

### Import logic (Sheet → Postgres)
1. Read all rows via `spreadsheets.values.get`
2. Validate each row (required fields, types)
3. Upsert by `id` — insert new, update existing, skip rows missing `id`
4. Return summary: `{inserted: N, updated: N, skipped: N, errors: [...]}`

> **Conflict handling:** Export always overwrites Sheet. Import always overwrites Postgres. Do not run both simultaneously. Last write wins — no merge logic.

---

## Step 7: Frontend Trigger (when you implement)

Add buttons to Finance and Inventory pages in the frontend:

```tsx
// In frontend/src/pages/Finance.tsx
<Button onClick={() => exportMutation.mutate()}>Export to Sheets</Button>
<Button onClick={() => importMutation.mutate()}>Import from Sheets</Button>
```

Show import summary in a modal so user sees what changed.

---

## Option: Apps Script Auto-Push (advanced)

If you want Sheet edits to sync automatically (without clicking Import):

1. In the Sheet: **Extensions** → **Apps Script**
2. Add an `onEdit` trigger that calls your backend:

```javascript
function onEdit(e) {
  const url = 'https://your-railway-app.up.railway.app/api/v1/import/transactions';
  const token = PropertiesService.getScriptProperties().getProperty('API_TOKEN');
  UrlFetchApp.fetch(url, {
    method: 'post',
    headers: { 'Authorization': 'Bearer ' + token }
  });
}
```

3. Store your API token in **Project Settings** → **Script Properties**

> **Limitation:** `onEdit` fires on every cell edit — throttle or debounce to avoid hammering the API. Consider using `onChange` with a time-based trigger instead (e.g., sync every 5 minutes).

---

## Security Checklist

- [ ] Service account JSON key not committed to git
- [ ] `GOOGLE_SERVICE_ACCOUNT_KEY_JSON` only in backend env (Railway), never Vercel
- [ ] Sheets shared only with service account, not publicly
- [ ] Import endpoint validates `user_id` matches authenticated user (prevent overwriting other users' data)
- [ ] Import endpoint rejects rows with unknown columns
