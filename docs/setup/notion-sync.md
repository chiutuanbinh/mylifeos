# Notion Notes Sync Setup

Two-way sync between MyLifeOS notes and a Notion database. MyLifeOS is source of truth; Notion is the editor interface.

**Sync strategy:** Last-modified wins. Track `notion_page_id` and `notion_last_edited` per note to detect which side changed.

---

## Step 1: Create a Notion Integration

1. Go to [notion.so/my-integrations](https://www.notion.so/my-integrations)
2. Click **+ New integration**
3. Name: `MyLifeOS`
4. Associated workspace: your personal workspace
5. Capabilities: check **Read content**, **Update content**, **Insert content**
6. Click **Submit** → copy the **Internal Integration Secret** (starts with `secret_...`)

> **Security note:** This token grants full access to any page shared with the integration. Treat like a password. Store only in backend env vars — never in frontend or Vercel.

---

## Step 2: Create the Notion Database

1. In Notion, create a new page: `MyLifeOS Notes`
2. Add a **Database — Full page** block
3. Set up columns (properties):

| Property | Type | Purpose |
|----------|------|---------|
| `Title` | Title | Note title (default) |
| `Content` | Text | Note body |
| `Tags` | Multi-select | Note tags |
| `MyLifeOSID` | Text | UUID from your notes table — **do not edit** |
| `LastSyncedAt` | Date | Timestamp of last sync |
| `Status` | Select | Options: `Synced`, `Modified`, `New` |

4. Click **Share** (top right) → **Invite** → search your integration name → **Invite**
5. Copy the **Database ID** from the URL:
   `https://notion.so/your-workspace/DATABASE_ID_HERE?v=...`

---

## Step 3: Store Credentials

Add to `.env.local` and Railway:
```
NOTION_API_TOKEN=secret_your_token_here
NOTION_NOTES_DATABASE_ID=your-database-id
```

> **Never commit these.** `.env.local` is in `.gitignore`. Add to Railway Variables for production.

---

## Step 4: DB Schema Changes

Add columns to the `notes` table:

```sql
ALTER TABLE notes
  ADD COLUMN notion_page_id TEXT,
  ADD COLUMN notion_last_edited TIMESTAMPTZ,
  ADD COLUMN synced_to_notion_at TIMESTAMPTZ;

CREATE INDEX idx_notes_notion_page_id ON notes(notion_page_id);
```

Create a new migration file: `migrations/003_notion_sync.sql`

---

## Step 5: Sync Logic

Add endpoint to `backend/cmd/server/main.go`:
```
POST /api/v1/notes/notion/sync
```

### Sync algorithm

```
1. PUSH (MyLifeOS → Notion)
   For each note where updated_at > synced_to_notion_at OR notion_page_id IS NULL:
     - If notion_page_id IS NULL: create new Notion page, store page ID
     - Else: update existing Notion page
     - Set synced_to_notion_at = now()

2. PULL (Notion → MyLifeOS)
   Query Notion DB for pages where LastSyncedAt < last_edited_time:
     - Extract MyLifeOSID from page properties
     - If MyLifeOSID found: update note in Postgres (only if Notion edit time > notes.updated_at)
     - If MyLifeOSID missing: treat as new note created in Notion → insert into notes table
     - Update synced_to_notion_at = now()

3. CONFLICT RULE
   If both sides modified since last sync: Notion wins (last editor wins).
   Log conflict for user visibility.
```

### Notion API calls

**Create page:**
```
POST https://api.notion.com/v1/pages
Authorization: Bearer secret_...
Notion-Version: 2022-06-28

{
  "parent": { "database_id": "DATABASE_ID" },
  "properties": {
    "Title": { "title": [{ "text": { "content": "Note title" } }] },
    "MyLifeOSID": { "rich_text": [{ "text": { "content": "uuid-here" } }] },
    "Status": { "select": { "name": "Synced" } }
  },
  "children": [
    {
      "object": "block",
      "type": "paragraph",
      "paragraph": {
        "rich_text": [{ "type": "text", "text": { "content": "Note body here" } }]
      }
    }
  ]
}
```

**Update page:**
```
PATCH https://api.notion.com/v1/pages/{page_id}

{
  "properties": {
    "Title": { "title": [{ "text": { "content": "Updated title" } }] },
    "Status": { "select": { "name": "Synced" } }
  }
}
```

**Update page content (blocks):**
```
# First delete existing blocks, then append new ones
DELETE https://api.notion.com/v1/blocks/{block_id}
PATCH https://api.notion.com/v1/blocks/{page_id}/children
```

**Query for modified pages:**
```
POST https://api.notion.com/v1/databases/{database_id}/query

{
  "filter": {
    "property": "Status",
    "select": { "equals": "Modified" }
  }
}
```

> **Note:** Notion has no webhooks. You must poll or trigger sync manually.

---

## Step 6: Frontend Sync Button

Add to `frontend/src/pages/Notes.tsx`:

```tsx
const syncMutation = useMutation({
  mutationFn: () => api.post('/notes/notion/sync'),
  onSuccess: (data) => {
    message.success(`Synced: ${data.pushed} pushed, ${data.pulled} pulled`);
    queryClient.invalidateQueries({ queryKey: ['notes'] });
  },
});

// In the page header:
<Button
  icon={<SyncOutlined />}
  loading={syncMutation.isPending}
  onClick={() => syncMutation.mutate()}
>
  Sync with Notion
</Button>
```

---

## Step 7: Rich Text Limitations

Notion rich text does not map 1:1 to Markdown. Known lossy conversions:

| Notion element | MyLifeOS (plain text) |
|----------------|-----------------------|
| Toggle | Lost — content preserved, toggle wrapper dropped |
| Callout | Converted to blockquote |
| Table | Converted to pipe-delimited text |
| Column layout | Columns merged sequentially |
| Embedded files | URL reference only |

For notes with heavy formatting, store the raw Notion block JSON in a `notion_blocks JSONB` column if you want lossless round-trips.

---

## Automation Options

### Option A: Manual sync button (recommended to start)
User clicks sync when needed. Simple, no background jobs.

### Option B: Scheduled sync on Railway
Add a time.Ticker in backend that runs sync every N minutes:

```go
// In main.go, after router setup:
go func() {
    ticker := time.NewTicker(15 * time.Minute)
    for range ticker.C {
        if err := syncNotesForAllUsers(pool); err != nil {
            log.Printf("notion sync error: %v", err)
        }
    }
}()
```

> Caution: Notion rate limit is 3 req/sec. With many notes or users, add a rate limiter.

### Option C: Trigger sync on note save
Call sync in the `POST /notes` and `PATCH /notes/{id}` handlers after DB write. Adds latency to every save — only viable if sync is fast (< 500ms).

---

## Security Checklist

- [ ] `NOTION_API_TOKEN` only in backend env (Railway), never Vercel
- [ ] `NOTION_NOTES_DATABASE_ID` only in backend env
- [ ] Notion database shared only with the integration, not publicly
- [ ] Sync endpoint scoped to authenticated user — never sync another user's notes
- [ ] `MyLifeOSID` column in Notion marked as read-only in docs/instructions to users (editing it breaks sync)
