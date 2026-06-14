# Design: Unified Objectives + Wealth Trends
**Date:** 2026-06-14

---

## 1. Scope

Two major feature areas:

1. **Unified Objectives page** — merge Goals + Habits into one OKR-style page where habits are recurring key results under goals, with a daily gate view and reminder mockup.
2. **Wealth Trends tab** — net worth history, benchmark comparisons (VN-Index, SJC gold, GSO CPI, bank rates), and a Vietnamese finance news feed.

---

## 2. Data Model

### 2.1 Key Results (extended)

Add two columns to `key_results`:

```sql
ALTER TABLE key_results ADD COLUMN recurring BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE key_results ADD COLUMN reminder_time TIME DEFAULT NULL;
```

- `recurring = true` → this KR repeats daily (was a habit)
- `reminder_time` → stored for future push notification wiring; not yet functional

### 2.2 KR Logs (rename from habit_logs)

`habit_logs` is reused as the completion log for recurring KRs. Rename column `habit_id` → `kr_id`. Migrate existing rows.

```sql
ALTER TABLE habit_logs RENAME COLUMN habit_id TO kr_id;
ALTER TABLE habit_logs RENAME TO kr_logs;
```

### 2.3 Habits migration

All existing `habits` rows migrate to recurring KRs under an auto-created goal per user: `"Daily Routines"` (color `#52c41a`). Habit `name` → KR `description`, habit `icon` prepended to description.

After migration, `habits` table is dropped.

### 2.4 Net Worth Snapshots

```sql
CREATE TABLE net_worth_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id),
  date DATE NOT NULL,
  net_worth NUMERIC(18,2) NOT NULL,
  note TEXT DEFAULT '',
  UNIQUE(user_id, date)
);
```

- Daily cron auto-inserts by summing `current_value` from assets.
- Manual backfill via UI (upsert on conflict).

### 2.5 Benchmark Data

```sql
CREATE TABLE benchmark_data (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source TEXT NOT NULL,       -- e.g. 'vn_index', 'sjc_gold', 'gso_cpi', 'vcb_saving_12m'
  date DATE NOT NULL,
  value NUMERIC(18,4) NOT NULL,
  UNIQUE(source, date)
);
```

Sources fetched daily by backend cron:
- `vn_index` — VN-Index closing price (cafef.vn or vndirect public endpoint)
- `sjc_gold` — SJC buy price per tael (sjc.com.vn)
- `gso_cpi` — Vietnam CPI index (gso.gov.vn, quarterly, interpolated monthly)
- `vcb_saving_12m`, `bidv_saving_12m`, `agr_saving_12m`, `tcb_saving_12m` — 12-month saving rates
- `vcb_lending`, `bidv_lending`, `agr_lending`, `tcb_lending` — standard lending rates

### 2.6 News Cache

```sql
CREATE TABLE news_cache (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source TEXT NOT NULL,
  published_at TIMESTAMPTZ NOT NULL,
  title TEXT NOT NULL,
  url TEXT NOT NULL,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Backend fetches cafef.vn RSS every 6h, stores latest 50 items, truncates older.

---

## 3. Backend

### 3.1 New / changed endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/kr-logs?date=` | Today's completion logs (was habit-logs) |
| POST | `/api/kr-logs/toggle` | Toggle a recurring KR done/undone for a date |
| GET | `/api/kr-logs/range?kr_id=&from=&to=` | Log range for heatmap |
| GET | `/api/net-worth-snapshots` | All snapshots for current user |
| POST | `/api/net-worth-snapshots` | Manual backfill entry |
| GET | `/api/benchmarks?sources=&from=&to=` | Benchmark time series |
| GET | `/api/bank-rates` | Latest bank interest rates |
| GET | `/api/news` | Latest cached news items |

Existing `/api/habits/*` endpoints removed after migration.

### 3.2 Daily cron job

Runs at 23:50 VN time (UTC+7) each day:
1. For each user: sum `current_value` from assets → upsert into `net_worth_snapshots`
2. Fetch VN-Index closing price
3. Fetch SJC gold price
4. Fetch bank rates (Vietcombank, BIDV, Agribank, Techcombank public pages)
5. Fetch cafef.vn RSS news, upsert into `news_cache`
6. Upsert all into `benchmark_data`

Scraping is best-effort; failures logged, not fatal.

---

## 4. Frontend

### 4.1 Navigation

Remove "Health" nav item. Rename "Goals" → "Objectives". Single `ObjectivesPage`.

### 4.2 ObjectivesPage layout

```
┌─────────────────────────────────────────────┐
│  TODAY  [X/Y done]  ████████░░░  80%        │  ← sticky gate zone
│  ┌ Daily Routines ────────────────────────┐ │
│  │ ☑ 🏃 Morning run        🔥 5 streak   │ │
│  │ ☐ 📚 Read 30 min                      │ │
│  └────────────────────────────────────────┘ │
│  ┌ Fitness Goal ──────────────────────────┐ │
│  │ ☑ 💧 Drink 2L water     🔥 12 streak  │ │
│  └────────────────────────────────────────┘ │
├─────────────────────────────────────────────┤
│  GOALS                          [+ Add Goal]│  ← goals grid below
│  ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│  │ Goal A   │ │ Goal B   │ │ Goal C   │   │
│  │ ████ 60% │ │ ██░░ 40% │ │ ████ 80% │   │
│  │ KRs...   │ │ KRs...   │ │ KRs...   │   │
│  │ Daily 2/3│ │ Daily 1/2│ │ Daily 3/3│   │
│  └──────────┘ └──────────┘ └──────────┘   │
└─────────────────────────────────────────────┘
```

- Gate zone groups recurring KRs by parent goal, shows streak badge per KR
- Goal cards show one-time KRs (existing) + "Daily" sub-section with today's recurring KR count
- "Add KR" flow: toggle "Recurring" → shows reminder time picker (stored, labeled "Reminder — coming soon")
- Heatmap moved inside each goal card's expanded view (click to expand)

### 4.3 WealthPage — Trends tab

New fourth tab. Layout:

```
┌─────────────────────────────────────────────┐
│  Net Worth    VN-Index   SJC Gold    CPI    │  ← summary stat cards
│  +12.3%       +8.1%      +15.2%     +3.1%  │
│  vs 1 year ago                              │
├─────────────────────────────────────────────┤
│  [Net worth trend chart — line, 1Y default] │
│  Toggles: [1M] [3M] [6M] [1Y] [All]        │
│  Overlay: ☑ VN-Index  ☑ SJC Gold  ☐ CPI   │
│  (all normalized to % change from start)    │
├─────────────────────────────────────────────┤
│  Bank Interest Rates                        │
│  ┌──────────┬────────────┬──────────┐      │
│  │ Bank     │ Saving 12m │ Lending  │      │
│  │ VCB      │ 5.5%       │ 9.0%     │      │
│  │ BIDV     │ 5.6%       │ 9.2%     │      │
│  │ Agribank │ 5.4%       │ 8.8%     │      │
│  │ TCB      │ 5.8%       │ 9.5%     │      │
│  └──────────┴────────────┴──────────┘      │
│  Updated: 2026-06-14                        │
├─────────────────────────────────────────────┤
│  Finance News  (cafef.vn)                   │
│  • Title 1 — 2h ago                        │
│  • Title 2 — 5h ago                        │
│  • Title 3 — 1d ago                        │
└─────────────────────────────────────────────┘
```

Manual backfill: "Add past data point" button → modal with date + net worth fields.

Chart library: Recharts (add as dependency).

### 4.4 Settings — Reminders mockup

New "Reminders" section in SettingsPage. Lists recurring KRs that have `reminder_time` set. Each entry renders a phone notification mockup card:

```
┌────────────────────────────────┐
│ 🔔 MyLifeOS          09:00 AM  │
│ Daily Routines                 │
│ Time to: 🏃 Morning run        │
└────────────────────────────────┘
```

Label beneath: "Push notifications — coming soon. Reminder times are saved and will activate when mobile app is available."

---

## 5. Migration Plan

1. Add `recurring` + `reminder_time` columns to `key_results`
2. Create `kr_logs` (rename `habit_logs`)
3. Run migration script: for each user, create "Daily Routines" goal, insert habits as recurring KRs, copy habit_logs → kr_logs
4. Drop `habits` table
5. Create `net_worth_snapshots`, `benchmark_data`, `news_cache` tables
6. Deploy backend with new endpoints + cron job
7. Deploy frontend with ObjectivesPage + Trends tab

---

## 6. Out of Scope

- Actual push notification delivery (mobile app not built)
- Historical benchmark data before deployment date (benchmarks start accumulating from first cron run)
- Real-time stock prices (daily close only)
- Authentication for external scrapers (all public endpoints)
