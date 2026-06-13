# PR1: Data Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add DB schema for depreciation, goal status, net worth snapshots; extend backend repos/handlers with new fields, endpoints, and auto-computed values.

**Architecture:** Pure backend changes — new migration, model fields, repo logic, handler endpoints, route wiring. No frontend changes. All new logic covered by handler-level tests using mock repos.

**Tech Stack:** Go, chi, pgx/v5, PostgreSQL

---

## File Map

| Action | File |
|--------|------|
| Create | `supabase/migrations/20260613000001_improvements.sql` |
| Modify | `backend/internal/models/models.go` |
| Modify | `backend/internal/repo/assets.go` |
| Modify | `backend/internal/repo/goals.go` |
| Modify | `backend/internal/repo/habits.go` |
| Modify | `backend/internal/repo/dashboard.go` |
| Modify | `backend/internal/handlers/assets.go` |
| Modify | `backend/internal/handlers/goals.go` |
| Modify | `backend/internal/handlers/habits.go` |
| Modify | `backend/internal/handlers/assets_test.go` |
| Modify | `backend/internal/handlers/goals_test.go` |
| Modify | `backend/internal/handlers/habits_test.go` |
| Modify | `backend/internal/handlers/dashboard_test.go` |
| Modify | `backend/cmd/server/main.go` |

---

## Task 1: DB Migration

**Files:**
- Create: `supabase/migrations/20260613000001_improvements.sql`

- [ ] **Write migration file**

```sql
-- supabase/migrations/20260613000001_improvements.sql

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS purchase_value    NUMERIC(12,2),
  ADD COLUMN IF NOT EXISTS depreciation_rate NUMERIC(5,4) NOT NULL DEFAULT 0;

ALTER TABLE goals
  ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';

CREATE TABLE IF NOT EXISTS net_worth_snapshots (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       TEXT NOT NULL,
  snapshot_date DATE NOT NULL,
  assets_value  NUMERIC(12,2) NOT NULL,
  cash_position NUMERIC(12,2) NOT NULL,
  net_worth     NUMERIC(12,2) NOT NULL,
  UNIQUE(user_id, snapshot_date)
);
```

- [ ] **Commit**

```bash
git checkout -b feat/data-layer
git add supabase/migrations/20260613000001_improvements.sql
git commit -m "feat: add migration for depreciation, goal status, net worth snapshots"
```

---

## Task 2: Update Models

**Files:**
- Modify: `backend/internal/models/models.go`

- [ ] **Update `Asset` struct** — add `PurchaseValue`, `DepreciationRate`, `CurrentValue`

Replace the `Asset` struct:

```go
type Asset struct {
	ID               string   `json:"id"`
	UserID           string   `json:"user_id"`
	Name             string   `json:"name"`
	Category         string   `json:"category"`
	Value            float64  `json:"value"`
	PurchasedAt      *string  `json:"purchased_at"`
	Notes            string   `json:"notes"`
	PurchaseValue    *float64 `json:"purchase_value"`
	DepreciationRate float64  `json:"depreciation_rate"`
	CurrentValue     float64  `json:"current_value"`
}
```

- [ ] **Update `Goal` struct** — add `Status`

Replace the `Goal` struct:

```go
type Goal struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	TargetDate  *string     `json:"target_date"`
	Progress    int         `json:"progress"`
	Color       string      `json:"color"`
	Status      string      `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	KeyResults  []KeyResult `json:"key_results,omitempty"`
}
```

- [ ] **Add `NetWorthSnapshot` struct** at end of models.go

```go
type NetWorthSnapshot struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	SnapshotDate string    `json:"snapshot_date"`
	AssetsValue  float64   `json:"assets_value"`
	CashPosition float64   `json:"cash_position"`
	NetWorth     float64   `json:"net_worth"`
}
```

- [ ] **Update `DashboardSummary`** — replace hardcoded trend with real snapshots

```go
type DashboardSummary struct {
	NetWorthTrend    []float64     `json:"net_worth_trend"`
	NetWorth         float64       `json:"net_worth"`
	HabitsTotal      int           `json:"habits_total"`
	HabitsDoneToday  int           `json:"habits_done_today"`
	GoalsAvgProgress int           `json:"goals_avg_progress"`
	BudgetTotal      float64       `json:"budget_total"`
	BudgetSpent      float64       `json:"budget_spent"`
	RecentTx         []Transaction `json:"recent_transactions"`
}
```

- [ ] **Commit**

```bash
git add backend/internal/models/models.go
git commit -m "feat: add depreciation fields to Asset, status to Goal, NetWorthSnapshot model"
```

---

## Task 3: Update Assets Repo

**Files:**
- Modify: `backend/internal/repo/assets.go`

- [ ] **Add `computeCurrentValue` helper** and update `List`/`Create`/`Update` to scan new columns

Replace the entire file:

```go
package repo

import (
	"context"
	"math"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetRepo interface {
	List(ctx context.Context, userID string) ([]models.Asset, error)
	Create(ctx context.Context, a models.Asset) (models.Asset, error)
	Update(ctx context.Context, a models.Asset) (models.Asset, error)
	Delete(ctx context.Context, id, userID string) error
}

type pgAssetRepo struct{ db *pgxpool.Pool }

func NewAssetRepo(db *pgxpool.Pool) AssetRepo { return &pgAssetRepo{db} }

func computeCurrentValue(purchaseValue *float64, depreciationRate float64, purchasedAt *string) float64 {
	if purchaseValue == nil || *purchaseValue == 0 {
		return 0
	}
	if purchasedAt == nil || depreciationRate == 0 {
		return *purchaseValue
	}
	t, err := time.Parse("2006-01-02", *purchasedAt)
	if err != nil {
		return *purchaseValue
	}
	years := time.Since(t).Hours() / 8760
	return *purchaseValue * math.Pow(1-depreciationRate, years)
}

func scanAsset(row interface {
	Scan(...any) error
}) (models.Asset, error) {
	var a models.Asset
	var purchasedAt *time.Time
	err := row.Scan(&a.ID, &a.UserID, &a.Name, &a.Category, &a.Value, &purchasedAt, &a.Notes, &a.PurchaseValue, &a.DepreciationRate)
	if purchasedAt != nil {
		s := purchasedAt.Format("2006-01-02")
		a.PurchasedAt = &s
	}
	if a.PurchaseValue != nil {
		a.CurrentValue = computeCurrentValue(a.PurchaseValue, a.DepreciationRate, a.PurchasedAt)
	} else {
		a.CurrentValue = a.Value
	}
	return a, err
}

func (r *pgAssetRepo) List(ctx context.Context, userID string) ([]models.Asset, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, category, value, purchased_at, notes, purchase_value, depreciation_rate
		 FROM assets WHERE user_id = $1 ORDER BY category, name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if out == nil {
		out = []models.Asset{}
	}
	return out, rows.Err()
}

func (r *pgAssetRepo) Create(ctx context.Context, a models.Asset) (models.Asset, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO assets (user_id, name, category, value, purchased_at, notes, purchase_value, depreciation_rate)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, user_id, name, category, value, purchased_at, notes, purchase_value, depreciation_rate`,
		a.UserID, a.Name, a.Category, a.Value, a.PurchasedAt, a.Notes, a.PurchaseValue, a.DepreciationRate)
	return scanAsset(row)
}

func (r *pgAssetRepo) Update(ctx context.Context, a models.Asset) (models.Asset, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE assets SET name=$1, category=$2, value=$3, purchased_at=$4, notes=$5, purchase_value=$6, depreciation_rate=$7
		 WHERE id=$8 AND user_id=$9
		 RETURNING id, user_id, name, category, value, purchased_at, notes, purchase_value, depreciation_rate`,
		a.Name, a.Category, a.Value, a.PurchasedAt, a.Notes, a.PurchaseValue, a.DepreciationRate, a.ID, a.UserID)
	return scanAsset(row)
}

func (r *pgAssetRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM assets WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}
```

- [ ] **Commit**

```bash
git add backend/internal/repo/assets.go
git commit -m "feat: add depreciation computation to assets repo"
```

---

## Task 4: Update Goals Repo

**Files:**
- Modify: `backend/internal/repo/goals.go`

- [ ] **Add `DeleteKeyResult` to interface and impl; compute progress from KRs; add status column**

Replace the entire file:

```go
package repo

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GoalRepo interface {
	List(ctx context.Context, userID string) ([]models.Goal, error)
	Create(ctx context.Context, g models.Goal) (models.Goal, error)
	Update(ctx context.Context, g models.Goal) (models.Goal, error)
	Delete(ctx context.Context, id, userID string) error
	AddKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error)
	UpdateKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error)
	DeleteKeyResult(ctx context.Context, krID, userID string) error
}

func nullDateString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func computeProgress(krs []models.KeyResult) int {
	if len(krs) == 0 {
		return 0
	}
	done := 0
	for _, kr := range krs {
		if kr.Done {
			done++
		}
	}
	return int(float64(done) / float64(len(krs)) * 100)
}

type pgGoalRepo struct{ db *pgxpool.Pool }

func NewGoalRepo(db *pgxpool.Pool) GoalRepo { return &pgGoalRepo{db} }

func (r *pgGoalRepo) List(ctx context.Context, userID string) ([]models.Goal, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, description, target_date, progress, color, status, created_at
		 FROM goals WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		var targetDate *time.Time
		rows.Scan(&g.ID, &g.UserID, &g.Name, &g.Description, &targetDate, &g.Progress, &g.Color, &g.Status, &g.CreatedAt)
		g.TargetDate = nullDateString(targetDate)
		goals = append(goals, g)
	}
	if goals == nil {
		goals = []models.Goal{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i, g := range goals {
		krows, err := r.db.Query(ctx,
			`SELECT id, goal_id, user_id, description, done FROM key_results WHERE goal_id = $1`, g.ID)
		if err != nil {
			return nil, err
		}
		var krs []models.KeyResult
		for krows.Next() {
			var kr models.KeyResult
			krows.Scan(&kr.ID, &kr.GoalID, &kr.UserID, &kr.Description, &kr.Done)
			krs = append(krs, kr)
		}
		krows.Close()
		if krs == nil {
			krs = []models.KeyResult{}
		}
		goals[i].KeyResults = krs
		goals[i].Progress = computeProgress(krs)
	}
	return goals, nil
}

func (r *pgGoalRepo) Create(ctx context.Context, g models.Goal) (models.Goal, error) {
	if g.Status == "" {
		g.Status = "active"
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO goals (user_id, name, description, target_date, progress, color, status)
		 VALUES ($1, $2, $3, $4, 0, $5, $6)
		 RETURNING id, user_id, name, description, target_date, progress, color, status, created_at`,
		g.UserID, g.Name, g.Description, g.TargetDate, g.Color, g.Status)
	var out models.Goal
	var targetDate *time.Time
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Description, &targetDate, &out.Progress, &out.Color, &out.Status, &out.CreatedAt)
	out.TargetDate = nullDateString(targetDate)
	out.KeyResults = []models.KeyResult{}
	return out, err
}

func (r *pgGoalRepo) Update(ctx context.Context, g models.Goal) (models.Goal, error) {
	if g.Status == "" {
		g.Status = "active"
	}
	row := r.db.QueryRow(ctx,
		`UPDATE goals SET name=$1, description=$2, target_date=$3, color=$4, status=$5
		 WHERE id=$6 AND user_id=$7
		 RETURNING id, user_id, name, description, target_date, progress, color, status, created_at`,
		g.Name, g.Description, g.TargetDate, g.Color, g.Status, g.ID, g.UserID)
	var out models.Goal
	var targetDate *time.Time
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Description, &targetDate, &out.Progress, &out.Color, &out.Status, &out.CreatedAt)
	out.TargetDate = nullDateString(targetDate)
	return out, err
}

func (r *pgGoalRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM goals WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *pgGoalRepo) AddKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO key_results (goal_id, user_id, description, done)
		 VALUES ($1, $2, $3, false)
		 RETURNING id, goal_id, user_id, description, done`,
		kr.GoalID, kr.UserID, kr.Description)
	var out models.KeyResult
	err := row.Scan(&out.ID, &out.GoalID, &out.UserID, &out.Description, &out.Done)
	return out, err
}

func (r *pgGoalRepo) UpdateKeyResult(ctx context.Context, kr models.KeyResult) (models.KeyResult, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE key_results SET description=$1, done=$2
		 WHERE id=$3 AND user_id=$4
		 RETURNING id, goal_id, user_id, description, done`,
		kr.Description, kr.Done, kr.ID, kr.UserID)
	var out models.KeyResult
	err := row.Scan(&out.ID, &out.GoalID, &out.UserID, &out.Description, &out.Done)
	return out, err
}

func (r *pgGoalRepo) DeleteKeyResult(ctx context.Context, krID, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM key_results WHERE id=$1 AND user_id=$2`, krID, userID)
	return err
}
```

- [ ] **Commit**

```bash
git add backend/internal/repo/goals.go
git commit -m "feat: auto-compute goal progress from KRs, add status, add DeleteKeyResult"
```

---

## Task 5: Update Habits Repo

**Files:**
- Modify: `backend/internal/repo/habits.go`

- [ ] **Add `Update` and `GetLogRange` to interface and implementation**

Replace the entire file:

```go
package repo

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HabitRepo interface {
	List(ctx context.Context, userID string) ([]models.Habit, error)
	Create(ctx context.Context, h models.Habit) (models.Habit, error)
	Update(ctx context.Context, h models.Habit) (models.Habit, error)
	Delete(ctx context.Context, id, userID string) error
	GetLogs(ctx context.Context, userID, date string) ([]models.HabitLog, error)
	GetLogRange(ctx context.Context, habitID, userID, from, to string) ([]models.HabitLog, error)
	ToggleLog(ctx context.Context, habitID, userID, date string) (models.HabitLog, error)
}

type pgHabitRepo struct{ db *pgxpool.Pool }

func NewHabitRepo(db *pgxpool.Pool) HabitRepo { return &pgHabitRepo{db} }

func (r *pgHabitRepo) List(ctx context.Context, userID string) ([]models.Habit, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, icon, created_at FROM habits WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Habit
	for rows.Next() {
		var h models.Habit
		rows.Scan(&h.ID, &h.UserID, &h.Name, &h.Icon, &h.CreatedAt)
		out = append(out, h)
	}
	if out == nil {
		out = []models.Habit{}
	}
	return out, rows.Err()
}

func (r *pgHabitRepo) Create(ctx context.Context, h models.Habit) (models.Habit, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO habits (user_id, name, icon) VALUES ($1, $2, $3)
		 RETURNING id, user_id, name, icon, created_at`,
		h.UserID, h.Name, h.Icon)
	var out models.Habit
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Icon, &out.CreatedAt)
	return out, err
}

func (r *pgHabitRepo) Update(ctx context.Context, h models.Habit) (models.Habit, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE habits SET name=$1, icon=$2 WHERE id=$3 AND user_id=$4
		 RETURNING id, user_id, name, icon, created_at`,
		h.Name, h.Icon, h.ID, h.UserID)
	var out models.Habit
	err := row.Scan(&out.ID, &out.UserID, &out.Name, &out.Icon, &out.CreatedAt)
	return out, err
}

func (r *pgHabitRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM habits WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *pgHabitRepo) GetLogs(ctx context.Context, userID, date string) ([]models.HabitLog, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	rows, err := r.db.Query(ctx,
		`SELECT hl.id, hl.habit_id, hl.user_id, hl.logged_date, hl.done
		 FROM habit_logs hl
		 JOIN habits h ON h.id = hl.habit_id
		 WHERE hl.user_id = $1 AND hl.logged_date = $2::date`, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.HabitLog
	for rows.Next() {
		var l models.HabitLog
		var loggedDate time.Time
		rows.Scan(&l.ID, &l.HabitID, &l.UserID, &loggedDate, &l.Done)
		l.LoggedDate = loggedDate.Format("2006-01-02")
		out = append(out, l)
	}
	if out == nil {
		out = []models.HabitLog{}
	}
	return out, rows.Err()
}

func (r *pgHabitRepo) GetLogRange(ctx context.Context, habitID, userID, from, to string) ([]models.HabitLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, habit_id, user_id, logged_date, done
		 FROM habit_logs
		 WHERE habit_id = $1 AND user_id = $2 AND logged_date BETWEEN $3::date AND $4::date
		 ORDER BY logged_date`,
		habitID, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.HabitLog
	for rows.Next() {
		var l models.HabitLog
		var loggedDate time.Time
		rows.Scan(&l.ID, &l.HabitID, &l.UserID, &loggedDate, &l.Done)
		l.LoggedDate = loggedDate.Format("2006-01-02")
		out = append(out, l)
	}
	if out == nil {
		out = []models.HabitLog{}
	}
	return out, rows.Err()
}

func (r *pgHabitRepo) ToggleLog(ctx context.Context, habitID, userID, date string) (models.HabitLog, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO habit_logs (habit_id, user_id, logged_date, done)
		 VALUES ($1, $2, $3::date, true)
		 ON CONFLICT (habit_id, logged_date)
		 DO UPDATE SET done = NOT habit_logs.done
		 RETURNING id, habit_id, user_id, logged_date, done`,
		habitID, userID, date)
	var out models.HabitLog
	var loggedDate time.Time
	err := row.Scan(&out.ID, &out.HabitID, &out.UserID, &loggedDate, &out.Done)
	out.LoggedDate = loggedDate.Format("2006-01-02")
	return out, err
}
```

- [ ] **Commit**

```bash
git add backend/internal/repo/habits.go
git commit -m "feat: add Update and GetLogRange to habits repo"
```

---

## Task 6: Update Dashboard Repo

**Files:**
- Modify: `backend/internal/repo/dashboard.go`

- [ ] **Rewrite to compute real net worth, upsert snapshot, return sparkline from DB**

Replace the entire file:

```go
package repo

import (
	"context"
	"math"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardRepo interface {
	Summary(ctx context.Context, userID string) (models.DashboardSummary, error)
}

type pgDashboardRepo struct{ db *pgxpool.Pool }

func NewDashboardRepo(db *pgxpool.Pool) DashboardRepo {
	return &pgDashboardRepo{db}
}

func (r *pgDashboardRepo) Summary(ctx context.Context, userID string) (models.DashboardSummary, error) {
	var s models.DashboardSummary

	// Habits
	row := r.db.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN hl.done THEN 1 ELSE 0 END), 0)
		 FROM habits h
		 LEFT JOIN habit_logs hl ON hl.habit_id = h.id AND hl.logged_date = CURRENT_DATE
		 WHERE h.user_id = $1`, userID)
	row.Scan(&s.HabitsTotal, &s.HabitsDoneToday)

	// Goals avg progress (computed from KRs)
	row = r.db.QueryRow(ctx, `
		SELECT COALESCE(ROUND(AVG(
			CASE WHEN total = 0 THEN 0
			ELSE done_count::numeric / total * 100 END
		)), 0)
		FROM (
			SELECT g.id,
				COUNT(kr.id) AS total,
				COUNT(CASE WHEN kr.done THEN 1 END) AS done_count
			FROM goals g
			LEFT JOIN key_results kr ON kr.goal_id = g.id
			WHERE g.user_id = $1 AND g.status = 'active'
			GROUP BY g.id
		) sub`, userID)
	row.Scan(&s.GoalsAvgProgress)

	// Budget
	row = r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(monthly_limit), 0) FROM budgets WHERE user_id = $1`, userID)
	row.Scan(&s.BudgetTotal)

	row = r.db.QueryRow(ctx,
		`SELECT COALESCE(ABS(SUM(amount)), 0) FROM transactions
		 WHERE user_id = $1 AND amount < 0
		 AND date_trunc('month', date) = date_trunc('month', CURRENT_DATE)`, userID)
	row.Scan(&s.BudgetSpent)

	// Compute current assets value (with depreciation)
	assetRows, err := r.db.Query(ctx,
		`SELECT value, purchase_value, depreciation_rate, purchased_at FROM assets WHERE user_id = $1`, userID)
	if err != nil {
		return s, err
	}
	var assetsTotal float64
	for assetRows.Next() {
		var value float64
		var purchaseValue *float64
		var depreciationRate float64
		var purchasedAt *time.Time
		assetRows.Scan(&value, &purchaseValue, &depreciationRate, &purchasedAt)
		if purchaseValue != nil && *purchaseValue > 0 {
			var pDate *string
			if purchasedAt != nil {
				s2 := purchasedAt.Format("2006-01-02")
				pDate = &s2
			}
			cv := computeCurrentValue(purchaseValue, depreciationRate, pDate)
			assetsTotal += cv
		} else {
			assetsTotal += value
		}
	}
	assetRows.Close()

	// Cash position
	var cashPosition float64
	row = r.db.QueryRow(ctx, `SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE user_id = $1`, userID)
	row.Scan(&cashPosition)

	netWorth := assetsTotal + cashPosition
	s.NetWorth = netWorth

	// Upsert today's snapshot
	today := time.Now().Format("2006-01-02")
	r.db.Exec(ctx, `
		INSERT INTO net_worth_snapshots (user_id, snapshot_date, assets_value, cash_position, net_worth)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, snapshot_date)
		DO UPDATE SET assets_value=$3, cash_position=$4, net_worth=$5`,
		userID, today, assetsTotal, cashPosition, netWorth)

	// Sparkline: last 6 snapshots
	snapRows, err := r.db.Query(ctx,
		`SELECT net_worth FROM net_worth_snapshots
		 WHERE user_id = $1 ORDER BY snapshot_date DESC LIMIT 6`, userID)
	if err != nil {
		return s, err
	}
	var trend []float64
	for snapRows.Next() {
		var nw float64
		snapRows.Scan(&nw)
		trend = append(trend, nw)
	}
	snapRows.Close()
	// reverse to chronological order
	for i, j := 0, len(trend)-1; i < j; i, j = i+1, j-1 {
		trend[i], trend[j] = trend[j], trend[i]
	}
	if len(trend) == 0 {
		trend = []float64{netWorth}
	}
	s.NetWorthTrend = trend
	_ = math.Round // keep import used

	// Recent transactions
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, date, description, category, amount, created_at
		 FROM transactions WHERE user_id = $1 ORDER BY date DESC, created_at DESC LIMIT 6`, userID)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var t models.Transaction
		rows.Scan(&t.ID, &t.UserID, &t.Date, &t.Description, &t.Category, &t.Amount, &t.CreatedAt)
		s.RecentTx = append(s.RecentTx, t)
	}
	if s.RecentTx == nil {
		s.RecentTx = []models.Transaction{}
	}

	return s, rows.Err()
}
```

- [ ] **Commit**

```bash
git add backend/internal/repo/dashboard.go
git commit -m "feat: dashboard uses real net worth from depreciated assets + snapshots"
```

---

## Task 7: Update Asset Handler (Validation)

**Files:**
- Modify: `backend/internal/handlers/assets.go`

- [ ] **Add validation: name required, category required, purchase_value ≥ 0, depreciation_rate in [0,1]**

Replace `Create` and `Update` functions (keep rest unchanged):

```go
func validateAsset(a models.Asset) string {
	if a.Name == "" {
		return "name is required"
	}
	if a.Category == "" {
		return "category is required"
	}
	if a.PurchaseValue != nil && *a.PurchaseValue < 0 {
		return "purchase_value must be >= 0"
	}
	if a.DepreciationRate < 0 || a.DepreciationRate > 1 {
		return "depreciation_rate must be between 0 and 1"
	}
	return ""
}

func (h *AssetHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var a models.Asset
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if msg := validateAsset(a); msg != "" {
		http.Error(w, `{"error":"`+msg+`"}`, 400)
		return
	}
	a.UserID = uid
	out, err := h.repo.Create(r.Context(), a)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(out)
}

func (h *AssetHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var a models.Asset
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if msg := validateAsset(a); msg != "" {
		http.Error(w, `{"error":"`+msg+`"}`, 400)
		return
	}
	a.ID = chi.URLParam(r, "id")
	a.UserID = uid
	out, err := h.repo.Update(r.Context(), a)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
```

- [ ] **Commit**

```bash
git add backend/internal/handlers/assets.go
git commit -m "feat: add validation to asset handler"
```

---

## Task 8: Update Goal Handler (DeleteKeyResult + validation)

**Files:**
- Modify: `backend/internal/handlers/goals.go`

- [ ] **Add `DeleteKeyResult` handler and name validation**

Add after the existing `UpdateKeyResult` function:

```go
func (h *GoalHandler) DeleteKeyResult(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if err := h.repo.DeleteKeyResult(r.Context(), chi.URLParam(r, "kr_id"), uid); err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.WriteHeader(204)
}
```

Add validation to `Create` (after decode, before setting UserID):

```go
	if g.Name == "" {
		http.Error(w, `{"error":"name is required"}`, 400)
		return
	}
	if len(g.Name) > 100 {
		http.Error(w, `{"error":"name too long"}`, 400)
		return
	}
```

Add the same name validation to `Update`.

- [ ] **Commit**

```bash
git add backend/internal/handlers/goals.go
git commit -m "feat: add DeleteKeyResult handler and goal name validation"
```

---

## Task 9: Update Habit Handler (Update + GetLogRange)

**Files:**
- Modify: `backend/internal/handlers/habits.go`

- [ ] **Add `Update` and `GetLogRange` handlers**

Add these two functions to `habits.go`:

```go
func (h *HabitHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	var habit models.Habit
	if err := json.NewDecoder(r.Body).Decode(&habit); err != nil {
		http.Error(w, `{"error":"bad request"}`, 400)
		return
	}
	if habit.Name == "" {
		http.Error(w, `{"error":"name is required"}`, 400)
		return
	}
	if len(habit.Name) > 80 {
		http.Error(w, `{"error":"name too long"}`, 400)
		return
	}
	habit.ID = chi.URLParam(r, "id")
	habit.UserID = uid
	if habit.Icon == "" {
		habit.Icon = "✓"
	}
	out, err := h.repo.Update(r.Context(), habit)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *HabitHandler) GetLogRange(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		http.Error(w, `{"error":"from and to are required"}`, 400)
		return
	}
	logs, err := h.repo.GetLogRange(r.Context(), chi.URLParam(r, "id"), uid, from, to)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
```

- [ ] **Commit**

```bash
git add backend/internal/handlers/habits.go
git commit -m "feat: add Update and GetLogRange handlers to habits"
```

---

## Task 10: Wire New Routes

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Add new routes to the router**

Inside `r.Route("/api/v1", ...)`, update the habits and goals sections:

```go
		// Habits — add new routes
		r.Get("/habits",                habitHandler.List)
		r.Post("/habits",               habitHandler.Create)
		r.Put("/habits/{id}",           habitHandler.Update)       // NEW
		r.Delete("/habits/{id}",        habitHandler.Delete)
		r.Get("/habits/logs",           habitHandler.GetLogs)
		r.Get("/habits/{id}/logs",      habitHandler.GetLogRange)  // NEW
		r.Post("/habits/{id}/log",      habitHandler.ToggleLog)

		// Goals — add DeleteKeyResult route
		r.Get("/goals",                                   goalHandler.List)
		r.Post("/goals",                                  goalHandler.Create)
		r.Patch("/goals/{id}",                            goalHandler.Update)
		r.Delete("/goals/{id}",                           goalHandler.Delete)
		r.Post("/goals/{id}/key-results",                 goalHandler.AddKeyResult)
		r.Patch("/goals/{id}/key-results/{kr_id}",        goalHandler.UpdateKeyResult)
		r.Delete("/goals/{id}/key-results/{kr_id}",       goalHandler.DeleteKeyResult)  // NEW
```

- [ ] **Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat: wire new habit update/log-range and goal KR delete routes"
```

---

## Task 11: Tests

**Files:**
- Modify: `backend/internal/handlers/assets_test.go`
- Modify: `backend/internal/handlers/goals_test.go`
- Modify: `backend/internal/handlers/habits_test.go`
- Modify: `backend/internal/handlers/dashboard_test.go`

- [ ] **Update mock repos to satisfy new interfaces**

In `assets_test.go`, update `mockAssetRepo.List` to return an asset with new fields:

```go
func (m *mockAssetRepo) List(_ context.Context, _ string) ([]models.Asset, error) {
	pv := 12000.0
	return []models.Asset{{ID: "a-1", Name: "Car", Category: "vehicle", Value: 10000, PurchaseValue: &pv, DepreciationRate: 0.15, CurrentValue: 9800}}, nil
}
```

In `goals_test.go`, add `DeleteKeyResult` to `mockGoalRepo`:

```go
func (m *mockGoalRepo) DeleteKeyResult(_ context.Context, _, _ string) error { return nil }
```

In `habits_test.go`, add `Update` and `GetLogRange` to `mockHabitRepo`:

```go
func (m *mockHabitRepo) Update(_ context.Context, h models.Habit) (models.Habit, error) { return h, nil }
func (m *mockHabitRepo) GetLogRange(_ context.Context, _, _, _, _ string) ([]models.HabitLog, error) {
	return []models.HabitLog{}, nil
}
```

- [ ] **Add test for asset validation**

In `assets_test.go`, add:

```go
func TestAssetCreate_MissingName(t *testing.T) {
	devEnv(t)
	h := handlers.NewAssetHandler(&mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"category": "electronics", "value": 1500.0})
	req := httptest.NewRequest("POST", "/api/v1/assets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAssetCreate_InvalidDepreciationRate(t *testing.T) {
	devEnv(t)
	h := handlers.NewAssetHandler(&mockAssetRepo{})
	handler := middleware.Auth(http.HandlerFunc(h.Create))

	body, _ := json.Marshal(map[string]any{"name": "Car", "category": "vehicle", "value": 10000.0, "depreciation_rate": 1.5})
	req := httptest.NewRequest("POST", "/api/v1/assets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
```

- [ ] **Add test for habit Update**

In `habits_test.go`, add:

```go
func TestHabitUpdate(t *testing.T) {
	devEnv(t)
	h := handlers.NewHabitHandler(&mockHabitRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "h-1")

	body, _ := json.Marshal(map[string]any{"name": "Updated Habit", "icon": "🏃"})
	req := httptest.NewRequest("PUT", "/api/v1/habits/h-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := middleware.Auth(http.HandlerFunc(h.Update))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHabitUpdate_MissingName(t *testing.T) {
	devEnv(t)
	h := handlers.NewHabitHandler(&mockHabitRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "h-1")

	body, _ := json.Marshal(map[string]any{"name": ""})
	req := httptest.NewRequest("PUT", "/api/v1/habits/h-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := middleware.Auth(http.HandlerFunc(h.Update))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHabitGetLogRange(t *testing.T) {
	devEnv(t)
	h := handlers.NewHabitHandler(&mockHabitRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "h-1")

	req := httptest.NewRequest("GET", "/api/v1/habits/h-1/logs?from=2026-06-01&to=2026-06-30", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := middleware.Auth(http.HandlerFunc(h.GetLogRange))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
```

- [ ] **Add test for goal DeleteKeyResult**

In `goals_test.go`, add:

```go
func TestGoalDeleteKeyResult(t *testing.T) {
	devEnv(t)
	h := handlers.NewGoalHandler(&mockGoalRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "g-1")
	rctx.URLParams.Add("kr_id", "kr-1")

	req := httptest.NewRequest("DELETE", "/api/v1/goals/g-1/key-results/kr-1", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := middleware.Auth(http.HandlerFunc(h.DeleteKeyResult))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}
```

- [ ] **Run tests — must pass with ≥80% coverage**

```bash
cd backend && go test ./internal/handlers/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out | grep total
```

Expected: `PASS`, total coverage ≥ 80%

- [ ] **Commit**

```bash
git add backend/internal/handlers/
git commit -m "test: update mocks and add tests for new endpoints"
```

---

## Task 12: Final PR

- [ ] **Push and create PR**

```bash
git push -u origin feat/data-layer
gh pr create --title "feat: data layer — depreciation, goal status, habit edit, real net worth" --body "$(cat <<'EOF'
## Summary
- DB migration: asset depreciation fields, goal status, net_worth_snapshots table
- Assets: purchase_value + depreciation_rate, current_value computed in Go
- Goals: status field, progress auto-computed from KRs, DeleteKeyResult endpoint
- Habits: Update endpoint (name/icon), GetLogRange endpoint for heatmap
- Dashboard: real net worth = depreciated assets + cash, sparkline from DB snapshots

## Test plan
- [ ] `go test ./internal/handlers/... ./internal/middleware/...` passes with ≥80% coverage
- [ ] All new endpoints return correct status codes
- [ ] Asset validation rejects missing name, invalid depreciation_rate
- [ ] Habit update rejects empty name

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
gh pr merge --auto --squash
```
