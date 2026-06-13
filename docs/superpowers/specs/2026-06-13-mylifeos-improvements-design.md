# MyLifeOS Improvements Design

**Date:** 2026-06-13  
**Scope:** Dashboard net worth, Goals (OKR mechanics), Wealth Management (Finance+Assets merge), Habits (edit + heatmap)  
**Delivery:** 3 sequential PRs

---

## PR1 — Data Layer

### DB Migrations

```sql
-- Assets: depreciation fields
ALTER TABLE assets ADD COLUMN purchase_value NUMERIC(12,2);
ALTER TABLE assets ADD COLUMN depreciation_rate NUMERIC(5,4) DEFAULT 0; -- 0.20 = 20%/yr

-- Goals: status
ALTER TABLE goals ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active';
-- values: 'active' | 'completed' | 'archived'

-- Net worth snapshots (upserted on each /dashboard fetch)
CREATE TABLE net_worth_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  snapshot_date DATE NOT NULL,
  assets_value NUMERIC(12,2) NOT NULL,
  cash_position NUMERIC(12,2) NOT NULL,
  net_worth NUMERIC(12,2) NOT NULL,
  UNIQUE(user_id, snapshot_date)
);
```

### Backend Changes

| Method | Path | Change |
|--------|------|--------|
| `PUT` | `/assets/:id` | add `purchase_value`, `depreciation_rate` to model + handler |
| `PUT` | `/goals/:id` | add `status` field; backend recomputes `progress` from KRs |
| `POST` | `/goals/:id/key-results` | endpoint exists — no change |
| `DELETE` | `/goals/:id/key-results/:kr_id` | new |
| `PUT` | `/habits/:id` | new — edit name + icon |
| `GET` | `/habits/:id/logs` | new — query params `from`, `to` (date range for heatmap) |
| `GET` | `/dashboard` | upsert today's snapshot; sparkline from last 6 snapshots |

### Net Worth Computation

```
current_value = purchase_value * (1 - depreciation_rate)^(years_since_purchase)
```

- Computed in Go on read, not stored.
- If `purchase_value` is NULL, `current_value = value` (legacy compatibility).
- `net_worth = SUM(current_value of assets) + SUM(transactions.amount)`
- Dashboard handler upserts `net_worth_snapshots` row for today on each fetch.

### Goal Progress Computation

```
progress = done_krs / total_krs * 100   (0 if no KRs)
```

- Computed in Go after fetching key results.
- `progress` column in DB updated as cache whenever KR is toggled or goal is saved.
- `progress` field in Create/Update request body is ignored.

### Validation (backend)

- Asset: `name` required, `category` required, `purchase_value` >= 0, `depreciation_rate` in [0, 1]
- Goal: `name` required, max 100 chars; `status` must be one of `active|completed|archived`
- Habit: `name` required, max 80 chars

---

## PR2 — Wealth Page

### Route

`/inventory` → `/wealth`. Nav link in `AppShell.tsx` updated. `App.tsx` route updated. `InventoryPage` replaced by `WealthPage`.

### WealthPage Layout

Ant Design `Tabs` — three tabs:

1. **Transactions** — exact current FinancePage transactions table + add modal
2. **Budgets** — exact current FinancePage budgets section
3. **Assets** — new content (see below)

`FinancePage` is deleted; its content moves into tabs 1 and 2.

### Assets Tab

**Summary row (top):**
- Total Assets (depreciated value)
- Cash Position (SUM of all transactions)
- Net Worth (assets + cash), highlighted

**Table columns:** Name, Category, Current Value, Purchase Value, Depr. Rate, Bought, Notes, Edit (pencil), Delete

- Current Value tooltip shows: `$X (purchased at $Y, depreciating at Z%/yr)`
- Edit opens modal pre-filled with all fields

**Add/Edit modal fields:**
- Name (required)
- Category (required)
- Purchase Value (required, ≥ 0)
- Depreciation Rate (0–100%, stored as 0–1 decimal, default 0%)
- Purchase Date
- Notes

### Dashboard Net Worth Card

- Value: live `net_worth` from dashboard summary endpoint
- Sparkline: last 6 `net_worth_snapshots` values
- Sub-label: `+X.X% vs last month` computed from `snapshots[n-1]` vs `snapshots[n-2]`; `—` if < 2 snapshots

---

## PR3 — Goals + Habits

### Goals

**Card UI:**
- Edit button (pencil) → edit modal: Name, Description, Target Date, Color, Status (Active/Completed/Archived)
- Progress bar = auto-computed from KRs (no manual input anywhere)
- "+ Key Result" inline text input at card bottom; Enter/submit adds KR
- Each KR row: checkbox (toggle done) + description text + delete (×) button
- `completed` goals: green checkmark badge on title
- `archived` goals: 50% opacity, sorted to bottom of list

**No manual progress field** in Create or Edit modals.

### Habits

**Edit:**
- Pencil icon per habit row → modal with Name + Icon fields

**Month Heatmap:**
- 28–31 square grid below each habit name
- Green = done, grey = not logged for that day
- Fetched via `GET /habits/:id/logs?from=YYYY-MM-01&to=YYYY-MM-31`
- All habits load their month logs on page mount via `Promise.all`

**Streak:**
- "🔥 N days" shown next to habit name
- Computed frontend-side: count consecutive done days ending today from the log array

---

## Delivery Order

```
PR1 (data layer) → merge → PR2 (wealth page) → merge → PR3 (goals + habits)
```

PR2 and PR3 can be developed in parallel after PR1 merges, but PR2 should merge before PR3 since PR3's goal progress depends on the updated goal model from PR1.
