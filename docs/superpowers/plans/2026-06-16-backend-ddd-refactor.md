# Backend DDD Refactor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure `backend/internal/` into domain / port / service / infra / transport layers following DDD.

**Architecture:** Domain entities live in `internal/domain/<context>/` (stdlib-only). Repository interfaces live in `internal/port/repository/`. Business logic aggregating multiple repos lives in `internal/service/`. Postgres implementations live in `internal/infra/postgres/`. HTTP handlers live in `internal/transport/http/`. Net worth formula lives once in `service/wealth/` and is called by both dashboard and trends handlers.

**Tech Stack:** Go, chi, pgx/v5, Supabase/Postgres

**Module path:** `github.com/chiutuanbinh/mylifeos/backend`

---

## File Map

### New files (domain entities — stdlib only)
- `internal/domain/wealth/entity.go` — Asset, Liability
- `internal/domain/finance/entity.go` — Transaction, Budget
- `internal/domain/goals/entity.go` — Goal, KeyResult, KRLog
- `internal/domain/calendar/entity.go` — Event
- `internal/domain/notes/entity.go` — Note
- `internal/domain/settings/entity.go` — UserSettings
- `internal/domain/trends/entity.go` — NetWorthSnapshot, BenchmarkData, BankRate, NewsItem

### New files (port interfaces)
- `internal/port/repository/wealth.go` — AssetRepo, LiabilityRepo
- `internal/port/repository/finance.go` — TransactionRepo
- `internal/port/repository/goals.go` — GoalRepo, KRLogRepo
- `internal/port/repository/calendar.go` — EventRepo
- `internal/port/repository/notes.go` — NoteRepo
- `internal/port/repository/settings.go` — SettingsRepo
- `internal/port/repository/trends.go` — TrendsRepo

### New files (services)
- `internal/service/wealth/service.go` — CurrentValue(), NetWorth() pure functions
- `internal/service/wealth/service_test.go`
- `internal/service/dashboard/service.go` — Summary() aggregates repos
- `internal/service/dashboard/service_test.go`

### Moved + updated (infra/postgres — verbatim SQL, new pkg + imports)
- `internal/infra/postgres/db.go` ← `internal/repo/db.go`
- `internal/infra/postgres/assets.go` ← `internal/repo/assets.go`
- `internal/infra/postgres/liabilities.go` ← `internal/repo/liabilities.go`
- `internal/infra/postgres/transactions.go` ← `internal/repo/transactions.go`
- `internal/infra/postgres/goals.go` ← `internal/repo/goals.go`
- `internal/infra/postgres/kr_logs.go` ← `internal/repo/kr_logs.go`
- `internal/infra/postgres/events.go` ← `internal/repo/events.go`
- `internal/infra/postgres/notes.go` ← `internal/repo/notes.go`
- `internal/infra/postgres/settings.go` ← `internal/repo/settings.go`
- `internal/infra/postgres/trends.go` ← `internal/repo/trends.go`

### Moved + updated (transport/http — verbatim handler logic, new pkg + imports)
- `internal/transport/http/assets.go` ← `internal/handlers/assets.go`
- `internal/transport/http/liabilities.go` ← `internal/handlers/liabilities.go`
- `internal/transport/http/transactions.go` ← `internal/handlers/transactions.go`
- `internal/transport/http/goals.go` ← `internal/handlers/goals.go`
- `internal/transport/http/kr_logs.go` ← `internal/handlers/kr_logs.go`
- `internal/transport/http/events.go` ← `internal/handlers/events.go`
- `internal/transport/http/google_calendar.go` ← `internal/handlers/google_calendar.go`
- `internal/transport/http/notes.go` ← `internal/handlers/notes.go`
- `internal/transport/http/settings.go` ← `internal/handlers/settings.go`
- `internal/transport/http/dashboard.go` ← `internal/handlers/dashboard.go`
- `internal/transport/http/trends.go` ← `internal/handlers/trends.go`
- `internal/transport/http/*_test.go` ← `internal/handlers/*_test.go`

### Modified
- `cmd/server/main.go` — rewire all constructors to new paths

### Deleted after all tasks pass
- `internal/models/` (entire package)
- `internal/repo/` (entire package)
- `internal/handlers/` (entire package)

---

## Task 1: Branch setup

**Files:** none

- [ ] **Step 1: Create feature branch**

```bash
git checkout -b feat/ddd-backend-structure
```

- [ ] **Step 2: Create directory scaffold**

```bash
mkdir -p backend/internal/domain/{wealth,finance,goals,calendar,notes,settings,trends}
mkdir -p backend/internal/port/repository
mkdir -p backend/internal/service/{wealth,dashboard}
mkdir -p backend/internal/infra/postgres
mkdir -p backend/internal/transport/http
```

- [ ] **Step 3: Verify scaffold**

```bash
find backend/internal/domain backend/internal/port backend/internal/service backend/internal/infra backend/internal/transport -type d
```

Expected: all 14 directories listed.

---

## Task 2: Domain entities

**Files:**
- Create: `backend/internal/domain/wealth/entity.go`
- Create: `backend/internal/domain/finance/entity.go`
- Create: `backend/internal/domain/goals/entity.go`
- Create: `backend/internal/domain/calendar/entity.go`
- Create: `backend/internal/domain/notes/entity.go`
- Create: `backend/internal/domain/settings/entity.go`
- Create: `backend/internal/domain/trends/entity.go`

These are verbatim moves from `internal/models/models.go`, split by bounded context.

- [ ] **Step 1: Create wealth entities**

`backend/internal/domain/wealth/entity.go`:
```go
package wealth

import "time"

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

type Liability struct {
	ID                string   `json:"id"`
	UserID            string   `json:"user_id"`
	Name              string   `json:"name"`
	Category          string   `json:"category"`
	Balance           float64  `json:"balance"`
	OriginalPrincipal *float64 `json:"original_principal"`
	InterestRate      *float64 `json:"interest_rate"`
	StartedAt         *string  `json:"started_at"`
	DueAt             *string  `json:"due_at"`
	Notes             string   `json:"notes"`
}

// unused import guard
var _ = time.Time{}
```

Remove the `var _ = time.Time{}` line — `time` is not used here. The wealth package has no time imports needed (dates are `*string`).

Corrected `backend/internal/domain/wealth/entity.go`:
```go
package wealth

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

type Liability struct {
	ID                string   `json:"id"`
	UserID            string   `json:"user_id"`
	Name              string   `json:"name"`
	Category          string   `json:"category"`
	Balance           float64  `json:"balance"`
	OriginalPrincipal *float64 `json:"original_principal"`
	InterestRate      *float64 `json:"interest_rate"`
	StartedAt         *string  `json:"started_at"`
	DueAt             *string  `json:"due_at"`
	Notes             string   `json:"notes"`
}
```

- [ ] **Step 2: Create finance entities**

`backend/internal/domain/finance/entity.go`:
```go
package finance

import "time"

type Transaction struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Date        string    `json:"date"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Amount      float64   `json:"amount"`
	CreatedAt   time.Time `json:"created_at"`
}

type Budget struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Category     string    `json:"category"`
	MonthlyLimit float64   `json:"monthly_limit"`
	CreatedAt    time.Time `json:"created_at"`
}
```

- [ ] **Step 3: Create goals entities**

`backend/internal/domain/goals/entity.go`:
```go
package goals

import "time"

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

type KeyResult struct {
	ID           string  `json:"id"`
	GoalID       string  `json:"goal_id"`
	UserID       string  `json:"user_id"`
	Description  string  `json:"description"`
	Done         bool    `json:"done"`
	Recurring    bool    `json:"recurring"`
	ReminderTime *string `json:"reminder_time,omitempty"`
}

type KRLog struct {
	ID         string `json:"id"`
	KRID       string `json:"kr_id"`
	UserID     string `json:"user_id"`
	LoggedDate string `json:"logged_date"`
	Done       bool   `json:"done"`
}
```

- [ ] **Step 4: Create calendar entities**

`backend/internal/domain/calendar/entity.go`:
```go
package calendar

type Event struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	Title         string  `json:"title"`
	StartAt       string  `json:"start_at"`
	EndAt         string  `json:"end_at"`
	Color         string  `json:"color"`
	AllDay        bool    `json:"all_day"`
	GoogleEventID *string `json:"google_event_id,omitempty"`
}
```

- [ ] **Step 5: Create notes entities**

`backend/internal/domain/notes/entity.go`:
```go
package notes

import "time"

type Note struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

- [ ] **Step 6: Create settings entities**

`backend/internal/domain/settings/entity.go`:
```go
package settings

type UserSettings struct {
	UserID         string         `json:"user_id"`
	Notifications  map[string]any `json:"notifications"`
	ModulesEnabled map[string]any `json:"modules_enabled"`
}
```

- [ ] **Step 7: Create trends entities**

`backend/internal/domain/trends/entity.go`:
```go
package trends

type NetWorthSnapshot struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	SnapshotDate string  `json:"snapshot_date"`
	AssetsValue  float64 `json:"assets_value"`
	CashPosition float64 `json:"cash_position"`
	NetWorth     float64 `json:"net_worth"`
	Note         string  `json:"note"`
}

type BenchmarkData struct {
	ID     string  `json:"id"`
	Source string  `json:"source"`
	Date   string  `json:"date"`
	Value  float64 `json:"value"`
}

type BankRate struct {
	Bank        string  `json:"bank"`
	Saving12m   float64 `json:"saving_12m"`
	Lending     float64 `json:"lending"`
	FetchedDate string  `json:"fetched_date"`
}

type NewsItem struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at"`
	Title       string `json:"title"`
	URL         string `json:"url"`
}
```

- [ ] **Step 8: Verify domain packages compile**

```bash
cd backend && go build ./internal/domain/...
```

Expected: no output (clean build).

- [ ] **Step 9: Commit**

```bash
git add backend/internal/domain/
git commit -m "feat(domain): add bounded context entity packages"
```

---

## Task 3: Port / repository interfaces

**Files:**
- Create: `backend/internal/port/repository/wealth.go`
- Create: `backend/internal/port/repository/finance.go`
- Create: `backend/internal/port/repository/goals.go`
- Create: `backend/internal/port/repository/calendar.go`
- Create: `backend/internal/port/repository/notes.go`
- Create: `backend/internal/port/repository/settings.go`
- Create: `backend/internal/port/repository/trends.go`

These mirror the old `repo/` interfaces but import `domain/<context>` types instead of `models`.

- [ ] **Step 1: Create wealth repository interfaces**

`backend/internal/port/repository/wealth.go`:
```go
package repository

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
)

type AssetRepo interface {
	List(ctx context.Context, userID string) ([]wealth.Asset, error)
	Create(ctx context.Context, a wealth.Asset) (wealth.Asset, error)
	Update(ctx context.Context, a wealth.Asset) (wealth.Asset, error)
	Delete(ctx context.Context, id, userID string) error
}

type LiabilityRepo interface {
	List(ctx context.Context, userID string) ([]wealth.Liability, error)
	Create(ctx context.Context, l wealth.Liability) (wealth.Liability, error)
	Update(ctx context.Context, l wealth.Liability) (wealth.Liability, error)
	Delete(ctx context.Context, id, userID string) error
	TotalBalance(ctx context.Context, userID string) (float64, error)
}
```

- [ ] **Step 2: Create finance repository interfaces**

`backend/internal/port/repository/finance.go`:
```go
package repository

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
)

type TransactionRepo interface {
	List(ctx context.Context, userID, category, from, to string, limit, offset int) ([]finance.Transaction, error)
	Create(ctx context.Context, t finance.Transaction) (finance.Transaction, error)
	Delete(ctx context.Context, id, userID string) error
	ListBudgets(ctx context.Context, userID string) ([]finance.Budget, error)
	UpsertBudget(ctx context.Context, b finance.Budget) (finance.Budget, error)
	SumByUser(ctx context.Context, userID string) (float64, error)
	SumSpentThisMonth(ctx context.Context, userID string) (float64, error)
}
```

Note: `SumByUser` and `SumSpentThisMonth` are new methods extracted from inline SQL in `repo/dashboard.go`. The postgres implementation in Task 5 will implement these.

- [ ] **Step 3: Create goals repository interfaces**

`backend/internal/port/repository/goals.go`:
```go
package repository

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
)

type GoalRepo interface {
	List(ctx context.Context, userID string) ([]goals.Goal, error)
	Create(ctx context.Context, g goals.Goal) (goals.Goal, error)
	Update(ctx context.Context, g goals.Goal) (goals.Goal, error)
	Delete(ctx context.Context, id, userID string) error
	AddKeyResult(ctx context.Context, kr goals.KeyResult) (goals.KeyResult, error)
	UpdateKeyResult(ctx context.Context, kr goals.KeyResult) (goals.KeyResult, error)
	DeleteKeyResult(ctx context.Context, krID, userID string) error
	HabitsSummary(ctx context.Context, userID string) (total, doneToday int, err error)
	GoalsAvgProgress(ctx context.Context, userID string) (int, error)
}

type KRLogRepo interface {
	GetLogs(ctx context.Context, userID, date string) ([]goals.KRLog, error)
	GetLogRange(ctx context.Context, krID, userID, from, to string) ([]goals.KRLog, error)
	ToggleLog(ctx context.Context, krID, userID, date string) (goals.KRLog, error)
}
```

Note: `HabitsSummary` and `GoalsAvgProgress` are new methods extracted from inline SQL in `repo/dashboard.go`.

- [ ] **Step 4: Create calendar repository interface**

`backend/internal/port/repository/calendar.go`:
```go
package repository

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/calendar"
)

type EventRepo interface {
	List(ctx context.Context, userID, from, to string) ([]calendar.Event, error)
	Create(ctx context.Context, e calendar.Event) (calendar.Event, error)
	Update(ctx context.Context, e calendar.Event) (calendar.Event, error)
	Delete(ctx context.Context, id, userID string) error
	UpsertFromGoogle(ctx context.Context, userID string, events []calendar.Event) (int, error)
}
```

- [ ] **Step 5: Create notes repository interface**

`backend/internal/port/repository/notes.go`:
```go
package repository

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/notes"
)

type NoteRepo interface {
	List(ctx context.Context, userID, search, tags string, pinned *bool) ([]notes.Note, error)
	Get(ctx context.Context, id, userID string) (notes.Note, error)
	Create(ctx context.Context, n notes.Note) (notes.Note, error)
	Update(ctx context.Context, n notes.Note) (notes.Note, error)
	Delete(ctx context.Context, id, userID string) error
}
```

- [ ] **Step 6: Create settings repository interface**

`backend/internal/port/repository/settings.go`:
```go
package repository

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/settings"
)

type SettingsRepo interface {
	Get(ctx context.Context, userID string) (settings.UserSettings, error)
	Upsert(ctx context.Context, s settings.UserSettings) (settings.UserSettings, error)
}
```

- [ ] **Step 7: Create trends repository interface**

`backend/internal/port/repository/trends.go`:
```go
package repository

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/trends"
)

type TrendsRepo interface {
	ListSnapshots(ctx context.Context, userID string) ([]trends.NetWorthSnapshot, error)
	UpsertSnapshot(ctx context.Context, s trends.NetWorthSnapshot) (trends.NetWorthSnapshot, error)
	UpsertBenchmark(ctx context.Context, b trends.BenchmarkData) error
	ListBenchmarks(ctx context.Context, sources []string, from, to string) ([]trends.BenchmarkData, error)
	LatestBankRates(ctx context.Context) ([]trends.BankRate, error)
	UpsertBankRate(ctx context.Context, b trends.BankRate) error
	ListNews(ctx context.Context, limit int) ([]trends.NewsItem, error)
	UpsertNews(ctx context.Context, items []trends.NewsItem) error
}
```

- [ ] **Step 8: Verify port packages compile**

```bash
cd backend && go build ./internal/port/...
```

Expected: no output.

- [ ] **Step 9: Commit**

```bash
git add backend/internal/port/
git commit -m "feat(port): add repository interfaces"
```

---

## Task 4: WealthService — pure net worth logic

**Files:**
- Create: `backend/internal/service/wealth/service.go`
- Create: `backend/internal/service/wealth/service_test.go`

This pulls `computeCurrentValue` from `repo/assets.go` into a testable, pure service.

- [ ] **Step 1: Write failing tests**

`backend/internal/service/wealth/service_test.go`:
```go
package wealthsvc_test

import (
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
	wealthsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/wealth"
)

func TestCurrentValue_NoDepreciation(t *testing.T) {
	pv := 100.0
	a := wealth.Asset{Value: 80, PurchaseValue: &pv, DepreciationRate: 0}
	got := wealthsvc.CurrentValue(a)
	if got != 100.0 {
		t.Errorf("want 100, got %v", got)
	}
}

func TestCurrentValue_FallbackToValue(t *testing.T) {
	a := wealth.Asset{Value: 80}
	got := wealthsvc.CurrentValue(a)
	if got != 80.0 {
		t.Errorf("want 80, got %v", got)
	}
}

func TestNetWorth(t *testing.T) {
	liabilities := []wealth.Liability{
		{Balance: 30},
		{Balance: 20},
	}
	got := wealthsvc.NetWorth(150, 50, liabilities)
	// 150 + 50 - 50 = 150
	if got != 150 {
		t.Errorf("want 150, got %v", got)
	}
}

func TestNetWorth_NoLiabilities(t *testing.T) {
	got := wealthsvc.NetWorth(100, 25, nil)
	if got != 125 {
		t.Errorf("want 125, got %v", got)
	}
}
```

- [ ] **Step 2: Run tests — expect failure**

```bash
cd backend && go test ./internal/service/wealth/...
```

Expected: build error — package not found.

- [ ] **Step 3: Implement WealthService**

`backend/internal/service/wealth/service.go`:
```go
package wealthsvc

import (
	"math"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
)

// CurrentValue returns the depreciation-adjusted value of an asset.
// If no PurchaseValue is set, falls back to Value.
func CurrentValue(a wealth.Asset) float64 {
	if a.PurchaseValue == nil || *a.PurchaseValue == 0 {
		return a.Value
	}
	if a.PurchasedAt == nil || a.DepreciationRate == 0 {
		return *a.PurchaseValue
	}
	t, err := time.Parse("2006-01-02", *a.PurchasedAt)
	if err != nil {
		return *a.PurchaseValue
	}
	years := time.Since(t).Hours() / 8760
	return *a.PurchaseValue * math.Pow(1-a.DepreciationRate, years)
}

// NetWorth computes net worth from pre-summed asset value, cash position, and liabilities.
func NetWorth(assetsTotal, cashPosition float64, liabilities []wealth.Liability) float64 {
	var totalLiabilities float64
	for _, l := range liabilities {
		totalLiabilities += l.Balance
	}
	return assetsTotal + cashPosition - totalLiabilities
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd backend && go test ./internal/service/wealth/... -v
```

Expected:
```
--- PASS: TestCurrentValue_NoDepreciation
--- PASS: TestCurrentValue_FallbackToValue
--- PASS: TestNetWorth
--- PASS: TestNetWorth_NoLiabilities
PASS
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/wealth/
git commit -m "feat(service/wealth): add pure CurrentValue and NetWorth functions"
```

---

## Task 5: infra/postgres — db + wealth repos

**Files:**
- Create: `backend/internal/infra/postgres/db.go`
- Create: `backend/internal/infra/postgres/assets.go`
- Create: `backend/internal/infra/postgres/liabilities.go`

These are verbatim moves from `repo/db.go`, `repo/assets.go`, `repo/liabilities.go` with:
1. `package repo` → `package postgres`
2. `"github.com/.../internal/models"` → `"github.com/.../internal/domain/wealth"`
3. `models.Asset` → `wealth.Asset`, `models.Liability` → `wealth.Liability`
4. `computeCurrentValue(...)` in assets.go replaced by call to `wealthsvc.CurrentValue(a)`
5. Interface types change from `repo.AssetRepo` to `repository.AssetRepo`

- [ ] **Step 1: Copy and update db.go**

`backend/internal/infra/postgres/db.go` — copy `internal/repo/db.go` verbatim, change only:
```go
// old
package repo

// new
package postgres
```

No other changes needed (it only sets up pgxpool).

- [ ] **Step 2: Copy and update assets.go**

`backend/internal/infra/postgres/assets.go`:

```go
package postgres

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	wealthsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/wealth"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type pgAssetRepo struct{ db *pgxpool.Pool }

func NewAssetRepo(db *pgxpool.Pool) repository.AssetRepo { return &pgAssetRepo{db} }

func scanAsset(row interface {
	Scan(...any) error
}) (wealth.Asset, error) {
	var a wealth.Asset
	var purchasedAt *time.Time
	err := row.Scan(&a.ID, &a.UserID, &a.Name, &a.Category, &a.Value, &purchasedAt, &a.Notes, &a.PurchaseValue, &a.DepreciationRate)
	if purchasedAt != nil {
		s := purchasedAt.Format("2006-01-02")
		a.PurchasedAt = &s
	}
	a.CurrentValue = wealthsvc.CurrentValue(a)
	return a, err
}

func (r *pgAssetRepo) List(ctx context.Context, userID string) ([]wealth.Asset, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, category, value, purchased_at, notes, purchase_value, depreciation_rate
		 FROM assets WHERE user_id = $1 ORDER BY category, name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []wealth.Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if out == nil {
		out = []wealth.Asset{}
	}
	return out, rows.Err()
}

func (r *pgAssetRepo) Create(ctx context.Context, a wealth.Asset) (wealth.Asset, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO assets (user_id, name, category, value, purchased_at, notes, purchase_value, depreciation_rate)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, user_id, name, category, value, purchased_at, notes, purchase_value, depreciation_rate`,
		a.UserID, a.Name, a.Category, a.Value, a.PurchasedAt, a.Notes, a.PurchaseValue, a.DepreciationRate)
	return scanAsset(row)
}

func (r *pgAssetRepo) Update(ctx context.Context, a wealth.Asset) (wealth.Asset, error) {
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

- [ ] **Step 3: Copy and update liabilities.go**

`backend/internal/infra/postgres/liabilities.go`:

```go
package postgres

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

const liabilityCols = `id, user_id, name, category, balance, original_principal, interest_rate, started_at, due_at, notes`

type pgLiabilityRepo struct{ db *pgxpool.Pool }

func NewLiabilityRepo(db *pgxpool.Pool) repository.LiabilityRepo { return &pgLiabilityRepo{db} }

func scanLiability(row interface{ Scan(...any) error }) (wealth.Liability, error) {
	var l wealth.Liability
	var startedAt, dueAt *time.Time
	err := row.Scan(&l.ID, &l.UserID, &l.Name, &l.Category, &l.Balance,
		&l.OriginalPrincipal, &l.InterestRate, &startedAt, &dueAt, &l.Notes)
	if startedAt != nil {
		s := startedAt.Format("2006-01-02")
		l.StartedAt = &s
	}
	if dueAt != nil {
		s := dueAt.Format("2006-01-02")
		l.DueAt = &s
	}
	return l, err
}

func (r *pgLiabilityRepo) List(ctx context.Context, userID string) ([]wealth.Liability, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+liabilityCols+` FROM liabilities WHERE user_id=$1 ORDER BY category, name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []wealth.Liability
	for rows.Next() {
		l, err := scanLiability(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	if out == nil {
		out = []wealth.Liability{}
	}
	return out, rows.Err()
}

func (r *pgLiabilityRepo) Create(ctx context.Context, l wealth.Liability) (wealth.Liability, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO liabilities (user_id, name, category, balance, original_principal, interest_rate, started_at, due_at, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING `+liabilityCols,
		l.UserID, l.Name, l.Category, l.Balance, l.OriginalPrincipal, l.InterestRate, l.StartedAt, l.DueAt, l.Notes)
	return scanLiability(row)
}

func (r *pgLiabilityRepo) Update(ctx context.Context, l wealth.Liability) (wealth.Liability, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE liabilities SET name=$1, category=$2, balance=$3, original_principal=$4,
		 interest_rate=$5, started_at=$6, due_at=$7, notes=$8
		 WHERE id=$9 AND user_id=$10
		 RETURNING `+liabilityCols,
		l.Name, l.Category, l.Balance, l.OriginalPrincipal, l.InterestRate, l.StartedAt, l.DueAt, l.Notes, l.ID, l.UserID)
	return scanLiability(row)
}

func (r *pgLiabilityRepo) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM liabilities WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *pgLiabilityRepo) TotalBalance(ctx context.Context, userID string) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx, `SELECT COALESCE(SUM(balance),0) FROM liabilities WHERE user_id=$1`, userID).Scan(&total)
	return total, err
}
```

- [ ] **Step 4: Verify compile**

```bash
cd backend && go build ./internal/infra/postgres/
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/infra/postgres/
git commit -m "feat(infra/postgres): add db, assets, liabilities postgres implementations"
```

---

## Task 6: infra/postgres — finance, goals, kr_logs

**Files:**
- Create: `backend/internal/infra/postgres/transactions.go`
- Create: `backend/internal/infra/postgres/goals.go`
- Create: `backend/internal/infra/postgres/kr_logs.go`

Copy from `repo/transactions.go`, `repo/goals.go`, `repo/kr_logs.go`. Changes:
1. `package repo` → `package postgres`
2. `models.Transaction/Budget` → `finance.Transaction/Budget`
3. `models.Goal/KeyResult/KRLog` → `goals.Goal/KeyResult/KRLog`
4. Return type of `New*` funcs changes to `repository.*Repo`
5. Add `SumByUser`, `SumSpentThisMonth` to `pgTransactionRepo`
6. Add `HabitsSummary`, `GoalsAvgProgress` to `pgGoalRepo`

- [ ] **Step 1: Create transactions.go**

Copy `repo/transactions.go` to `infra/postgres/transactions.go`. Apply these changes:

**Header** — change package and imports:
```go
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)
```

**Constructor** — change return type:
```go
func NewTransactionRepo(db *pgxpool.Pool) repository.TransactionRepo { return &pgTransactionRepo{db} }
```

**All `models.Transaction`** → `finance.Transaction`, **`models.Budget`** → `finance.Budget`.

**Add these two new methods** at the end of the file (extracted from `repo/dashboard.go`):
```go
func (r *pgTransactionRepo) SumByUser(ctx context.Context, userID string) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE user_id = $1`, userID).Scan(&total)
	return total, err
}

func (r *pgTransactionRepo) SumSpentThisMonth(ctx context.Context, userID string) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(ABS(SUM(amount)), 0) FROM transactions
		 WHERE user_id = $1 AND amount < 0
		 AND date_trunc('month', date) = date_trunc('month', CURRENT_DATE)`, userID).Scan(&total)
	return total, err
}
```

- [ ] **Step 2: Create goals.go**

Copy `repo/goals.go` to `infra/postgres/goals.go`. Apply:

**Header**:
```go
package postgres

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)
```

**Constructor**:
```go
func NewGoalRepo(db *pgxpool.Pool) repository.GoalRepo { return &pgGoalRepo{db} }
```

**All `models.Goal`** → `goals.Goal`, `models.KeyResult` → `goals.KeyResult`.

**Add these two new methods** at the end (extracted from `repo/dashboard.go`):
```go
func (r *pgGoalRepo) HabitsSummary(ctx context.Context, userID string) (total, doneToday int, err error) {
	row := r.db.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN kl.done THEN 1 ELSE 0 END), 0)
		 FROM key_results kr
		 LEFT JOIN kr_logs kl ON kl.kr_id = kr.id AND kl.logged_date = CURRENT_DATE
		 WHERE kr.user_id = $1 AND kr.recurring = TRUE`, userID)
	err = row.Scan(&total, &doneToday)
	return
}

func (r *pgGoalRepo) GoalsAvgProgress(ctx context.Context, userID string) (int, error) {
	var avg int
	err := r.db.QueryRow(ctx, `
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
		) sub`, userID).Scan(&avg)
	return avg, err
}
```

- [ ] **Step 3: Create kr_logs.go**

Copy `repo/kr_logs.go` to `infra/postgres/kr_logs.go`. Apply:

**Header**:
```go
package postgres

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)
```

**Constructor**:
```go
func NewKRLogRepo(db *pgxpool.Pool) repository.KRLogRepo { return &pgKRLogRepo{db} }
```

`models.KRLog` → `goals.KRLog` everywhere.

- [ ] **Step 4: Verify compile**

```bash
cd backend && go build ./internal/infra/postgres/
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/infra/postgres/
git commit -m "feat(infra/postgres): add transactions, goals, kr_logs implementations"
```

---

## Task 7: infra/postgres — events, notes, settings, trends

**Files:**
- Create: `backend/internal/infra/postgres/events.go`
- Create: `backend/internal/infra/postgres/notes.go`
- Create: `backend/internal/infra/postgres/settings.go`
- Create: `backend/internal/infra/postgres/trends.go`

Same pattern as Task 6.

- [ ] **Step 1: Create events.go**

Copy `repo/events.go`. Changes:
- `package postgres`
- Import `calendar` domain, `repository` port
- `models.Event` → `calendar.Event`
- Constructor returns `repository.EventRepo`

- [ ] **Step 2: Create notes.go**

Copy `repo/notes.go`. Changes:
- `package postgres`
- Import `notes` domain, `repository` port
- `models.Note` → `notes.Note`
- Constructor returns `repository.NoteRepo`

- [ ] **Step 3: Create settings.go**

Copy `repo/settings.go`. Changes:
- `package postgres`
- Import `settings` domain, `repository` port
- `models.UserSettings` → `settings.UserSettings`
- Constructor returns `repository.SettingsRepo`

- [ ] **Step 4: Create trends.go**

Copy `repo/trends.go`. Changes:
- `package postgres`
- Import `trends` domain, `repository` port
- `models.NetWorthSnapshot` → `trends.NetWorthSnapshot`
- `models.BenchmarkData` → `trends.BenchmarkData`
- `models.BankRate` → `trends.BankRate`
- `models.NewsItem` → `trends.NewsItem`
- Constructor returns `repository.TrendsRepo`

- [ ] **Step 5: Verify compile**

```bash
cd backend && go build ./internal/infra/postgres/
```

Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/infra/postgres/
git commit -m "feat(infra/postgres): add events, notes, settings, trends implementations"
```

---

## Task 8: DashboardService

**Files:**
- Create: `backend/internal/service/dashboard/service.go`
- Create: `backend/internal/service/dashboard/service_test.go`

This replaces `repo/dashboard.go`. Business logic extracted from that file moves here. SQL stays in infra.

- [ ] **Step 1: Write failing test**

`backend/internal/service/dashboard/service_test.go`:
```go
package dashboardsvc_test

import (
	"context"
	"testing"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/trends"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
	dashboardsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/dashboard"
)

// --- stubs ---

type stubAssets struct{ assets []wealth.Asset }

func (s *stubAssets) List(_ context.Context, _ string) ([]wealth.Asset, error) { return s.assets, nil }
func (s *stubAssets) Create(_ context.Context, a wealth.Asset) (wealth.Asset, error) { return a, nil }
func (s *stubAssets) Update(_ context.Context, a wealth.Asset) (wealth.Asset, error) { return a, nil }
func (s *stubAssets) Delete(_ context.Context, _, _ string) error                    { return nil }

type stubLiabilities struct{ liabilities []wealth.Liability }

func (s *stubLiabilities) List(_ context.Context, _ string) ([]wealth.Liability, error) {
	return s.liabilities, nil
}
func (s *stubLiabilities) Create(_ context.Context, l wealth.Liability) (wealth.Liability, error) {
	return l, nil
}
func (s *stubLiabilities) Update(_ context.Context, l wealth.Liability) (wealth.Liability, error) {
	return l, nil
}
func (s *stubLiabilities) Delete(_ context.Context, _, _ string) error { return nil }
func (s *stubLiabilities) TotalBalance(_ context.Context, _ string) (float64, error) {
	var total float64
	for _, l := range s.liabilities {
		total += l.Balance
	}
	return total, nil
}

type stubTxs struct{ cash float64 }

func (s *stubTxs) List(_ context.Context, _, _, _, _ string, _, _ int) ([]finance.Transaction, error) {
	return nil, nil
}
func (s *stubTxs) Create(_ context.Context, t finance.Transaction) (finance.Transaction, error) {
	return t, nil
}
func (s *stubTxs) Delete(_ context.Context, _, _ string) error { return nil }
func (s *stubTxs) ListBudgets(_ context.Context, _ string) ([]finance.Budget, error) {
	return []finance.Budget{{MonthlyLimit: 5_000_000}}, nil
}
func (s *stubTxs) UpsertBudget(_ context.Context, b finance.Budget) (finance.Budget, error) {
	return b, nil
}
func (s *stubTxs) SumByUser(_ context.Context, _ string) (float64, error) { return s.cash, nil }
func (s *stubTxs) SumSpentThisMonth(_ context.Context, _ string) (float64, error) {
	return 1_000_000, nil
}

type stubGoals struct{}

func (s *stubGoals) List(_ context.Context, _ string) ([]goals.Goal, error)   { return nil, nil }
func (s *stubGoals) Create(_ context.Context, g goals.Goal) (goals.Goal, error) { return g, nil }
func (s *stubGoals) Update(_ context.Context, g goals.Goal) (goals.Goal, error) { return g, nil }
func (s *stubGoals) Delete(_ context.Context, _, _ string) error                { return nil }
func (s *stubGoals) AddKeyResult(_ context.Context, kr goals.KeyResult) (goals.KeyResult, error) {
	return kr, nil
}
func (s *stubGoals) UpdateKeyResult(_ context.Context, kr goals.KeyResult) (goals.KeyResult, error) {
	return kr, nil
}
func (s *stubGoals) DeleteKeyResult(_ context.Context, _, _ string) error { return nil }
func (s *stubGoals) HabitsSummary(_ context.Context, _ string) (int, int, error) {
	return 5, 3, nil
}
func (s *stubGoals) GoalsAvgProgress(_ context.Context, _ string) (int, error) { return 65, nil }

type stubKRLogs struct{}

func (s *stubKRLogs) GetLogs(_ context.Context, _, _ string) ([]goals.KRLog, error) { return nil, nil }
func (s *stubKRLogs) GetLogRange(_ context.Context, _, _, _, _ string) ([]goals.KRLog, error) {
	return nil, nil
}
func (s *stubKRLogs) ToggleLog(_ context.Context, _, _, _ string) (goals.KRLog, error) {
	return goals.KRLog{}, nil
}

type stubSnapshots struct{}

func (s *stubSnapshots) ListSnapshots(_ context.Context, _ string) ([]trends.NetWorthSnapshot, error) {
	return []trends.NetWorthSnapshot{
		{SnapshotDate: "2026-06-10", NetWorth: 100_000_000},
		{SnapshotDate: "2026-06-16", NetWorth: 110_000_000},
	}, nil
}
func (s *stubSnapshots) UpsertSnapshot(_ context.Context, snap trends.NetWorthSnapshot) (trends.NetWorthSnapshot, error) {
	return snap, nil
}
func (s *stubSnapshots) UpsertBenchmark(_ context.Context, _ trends.BenchmarkData) error { return nil }
func (s *stubSnapshots) ListBenchmarks(_ context.Context, _ []string, _, _ string) ([]trends.BenchmarkData, error) {
	return nil, nil
}
func (s *stubSnapshots) LatestBankRates(_ context.Context) ([]trends.BankRate, error) { return nil, nil }
func (s *stubSnapshots) UpsertBankRate(_ context.Context, _ trends.BankRate) error    { return nil }
func (s *stubSnapshots) ListNews(_ context.Context, _ int) ([]trends.NewsItem, error)  { return nil, nil }
func (s *stubSnapshots) UpsertNews(_ context.Context, _ []trends.NewsItem) error       { return nil }

// --- tests ---

func TestSummary_NetWorth(t *testing.T) {
	svc := dashboardsvc.New(
		&stubAssets{assets: []wealth.Asset{
			{CurrentValue: 200_000_000},
		}},
		&stubLiabilities{liabilities: []wealth.Liability{
			{Balance: 50_000_000},
		}},
		&stubTxs{cash: 10_000_000},
		&stubGoals{},
		&stubKRLogs{},
		&stubSnapshots{},
	)

	summary, err := svc.Summary(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 200M assets + 10M cash - 50M liabilities = 160M
	want := 160_000_000.0
	if summary.NetWorth != want {
		t.Errorf("NetWorth: want %v, got %v", want, summary.NetWorth)
	}
}

func TestSummary_HabitsAndGoals(t *testing.T) {
	svc := dashboardsvc.New(
		&stubAssets{},
		&stubLiabilities{},
		&stubTxs{},
		&stubGoals{},
		&stubKRLogs{},
		&stubSnapshots{},
	)

	summary, err := svc.Summary(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.HabitsTotal != 5 {
		t.Errorf("HabitsTotal: want 5, got %d", summary.HabitsTotal)
	}
	if summary.HabitsDoneToday != 3 {
		t.Errorf("HabitsDoneToday: want 3, got %d", summary.HabitsDoneToday)
	}
	if summary.GoalsAvgProgress != 65 {
		t.Errorf("GoalsAvgProgress: want 65, got %d", summary.GoalsAvgProgress)
	}
}

// Ensure unused import doesn't cause compile error
var _ = time.Now
```

- [ ] **Step 2: Run test — expect failure**

```bash
cd backend && go test ./internal/service/dashboard/...
```

Expected: build error — package not found.

- [ ] **Step 3: Implement DashboardService**

`backend/internal/service/dashboard/service.go`:
```go
package dashboardsvc

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	wealthsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/wealth"
)

// Summary is the dashboard aggregate output.
type Summary struct {
	NetWorth         float64               `json:"net_worth"`
	NetWorthTrend    []float64             `json:"net_worth_trend"`
	HabitsTotal      int                   `json:"habits_total"`
	HabitsDoneToday  int                   `json:"habits_done_today"`
	GoalsAvgProgress int                   `json:"goals_avg_progress"`
	BudgetTotal      float64               `json:"budget_total"`
	BudgetSpent      float64               `json:"budget_spent"`
	RecentTx         []finance.Transaction `json:"recent_transactions"`
}

type Service struct {
	assets      repository.AssetRepo
	liabilities repository.LiabilityRepo
	txs         repository.TransactionRepo
	goals       repository.GoalRepo
	krLogs      repository.KRLogRepo
	snapshots   repository.TrendsRepo
}

func New(
	assets repository.AssetRepo,
	liabilities repository.LiabilityRepo,
	txs repository.TransactionRepo,
	goals repository.GoalRepo,
	krLogs repository.KRLogRepo,
	snapshots repository.TrendsRepo,
) *Service {
	return &Service{assets, liabilities, txs, goals, krLogs, snapshots}
}

func (s *Service) Summary(ctx context.Context, userID string) (Summary, error) {
	var sum Summary

	// Habits
	sum.HabitsTotal, sum.HabitsDoneToday, _ = s.goals.HabitsSummary(ctx, userID)

	// Goals
	sum.GoalsAvgProgress, _ = s.goals.GoalsAvgProgress(ctx, userID)

	// Budget
	budgets, _ := s.txs.ListBudgets(ctx, userID)
	for _, b := range budgets {
		sum.BudgetTotal += b.MonthlyLimit
	}
	sum.BudgetSpent, _ = s.txs.SumSpentThisMonth(ctx, userID)

	// Net worth
	assets, err := s.assets.List(ctx, userID)
	if err != nil {
		return sum, err
	}
	var assetsTotal float64
	for _, a := range assets {
		assetsTotal += a.CurrentValue
	}

	cash, _ := s.txs.SumByUser(ctx, userID)
	liabilities, _ := s.liabilities.List(ctx, userID)
	sum.NetWorth = wealthsvc.NetWorth(assetsTotal, cash, liabilities)

	// Upsert today's snapshot
	today := time.Now().Format("2006-01-02")
	s.snapshots.UpsertSnapshot(ctx, snapshot(userID, today, assetsTotal, cash, sum.NetWorth))

	// Sparkline — last 6 snapshots chronological
	snaps, err := s.snapshots.ListSnapshots(ctx, userID)
	if err != nil {
		return sum, err
	}
	// snapshots are ordered by date ASC from ListSnapshots
	start := len(snaps) - 6
	if start < 0 {
		start = 0
	}
	for _, sn := range snaps[start:] {
		sum.NetWorthTrend = append(sum.NetWorthTrend, sn.NetWorth)
	}
	if len(sum.NetWorthTrend) == 0 {
		sum.NetWorthTrend = []float64{sum.NetWorth}
	}

	// Recent transactions
	sum.RecentTx, _ = s.txs.List(ctx, userID, "", "", "", 6, 0)
	if sum.RecentTx == nil {
		sum.RecentTx = []finance.Transaction{}
	}

	return sum, nil
}
```

Add a private helper in the same file:
```go
func snapshot(userID, date string, assetsTotal, cash, netWorth float64) trendsdomain.NetWorthSnapshot {
	return trendsdomain.NetWorthSnapshot{
		UserID:       userID,
		SnapshotDate: date,
		AssetsValue:  assetsTotal,
		CashPosition: cash,
		NetWorth:     netWorth,
	}
}
```

Add the missing import for `trendsdomain`:
```go
import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	trendsdomain "github.com/chiutuanbinh/mylifeos/backend/internal/domain/trends"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	wealthsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/wealth"
)
```

Note on `ListSnapshots`: the current `repo/trends.go` orders by `snapshot_date` ASC — verify this in `infra/postgres/trends.go` and ensure it's `ORDER BY snapshot_date ASC`. The old dashboard code ordered DESC then reversed; this service assumes ASC so it just takes the last 6.

- [ ] **Step 4: Fix TrendsRepo.ListSnapshots order**

In `backend/internal/infra/postgres/trends.go`, ensure the list query uses `ORDER BY snapshot_date ASC` (change from DESC if needed):
```sql
SELECT id, user_id, snapshot_date, assets_value, cash_position, net_worth, note
FROM net_worth_snapshots WHERE user_id = $1 ORDER BY snapshot_date ASC
```

- [ ] **Step 5: Run tests — expect pass**

```bash
cd backend && go test ./internal/service/dashboard/... -v
```

Expected:
```
--- PASS: TestSummary_NetWorth
--- PASS: TestSummary_HabitsAndGoals
PASS
```

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/dashboard/ backend/internal/infra/postgres/trends.go
git commit -m "feat(service/dashboard): DashboardService aggregates repos for summary"
```

---

## Task 9: transport/http — wealth handlers

**Files:**
- Create: `backend/internal/transport/http/assets.go`
- Create: `backend/internal/transport/http/liabilities.go`
- Create: `backend/internal/transport/http/assets_test.go`
- Create: `backend/internal/transport/http/liabilities_test.go`

Copy from `handlers/assets.go` + `handlers/liabilities.go` + their tests. Changes:
1. `package handlers` → `package httphandler`
2. `"github.com/.../internal/models"` → `"github.com/.../internal/domain/wealth"`
3. `repo.AssetRepo` → `repository.AssetRepo`, `repo.LiabilityRepo` → `repository.LiabilityRepo`
4. `models.Asset` → `wealth.Asset`, `models.Liability` → `wealth.Liability`
5. Test file: `package handlers_test` → `package httphandler_test`; update mock struct types to match new domain + port packages.

- [ ] **Step 1: Copy and update assets handler**

Copy `internal/handlers/assets.go` → `internal/transport/http/assets.go`. Apply changes listed above. The handler logic (JSON parsing, http.Error calls, chi routing) is verbatim.

- [ ] **Step 2: Copy and update liabilities handler**

Copy `internal/handlers/liabilities.go` → `internal/transport/http/liabilities.go`. Apply changes listed above.

- [ ] **Step 3: Copy and update test files**

Copy `internal/handlers/assets_test.go` → `internal/transport/http/assets_test.go`.
Copy `internal/handlers/liabilities_test.go` → `internal/transport/http/liabilities_test.go`.

In each test file:
- `package handlers_test` → `package httphandler_test`
- `"github.com/.../internal/handlers"` → `"github.com/.../internal/transport/http"`; use alias: `httphandler "github.com/.../internal/transport/http"`
- `"github.com/.../internal/models"` → appropriate domain package (`wealth`)
- `"github.com/.../internal/repo"` → `"github.com/.../internal/port/repository"`
- Mock structs implement `repository.AssetRepo` / `repository.LiabilityRepo` (same method signatures, just updated types)
- Handler constructor calls: `handlers.NewAssetHandler(...)` → `httphandler.NewAssetHandler(...)`

- [ ] **Step 4: Verify compile + tests**

```bash
cd backend && go test ./internal/transport/http/ -run "TestAsset|TestLiabilit" -v
```

Expected: all asset and liability tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/transport/http/
git commit -m "feat(transport/http): add assets and liabilities handlers"
```

---

## Task 10: transport/http — remaining handlers (all except dashboard)

**Files:**
- Create: `backend/internal/transport/http/transactions.go` + test
- Create: `backend/internal/transport/http/goals.go` + test
- Create: `backend/internal/transport/http/kr_logs.go` + test
- Create: `backend/internal/transport/http/events.go` + test
- Create: `backend/internal/transport/http/google_calendar.go` + test
- Create: `backend/internal/transport/http/notes.go` + test
- Create: `backend/internal/transport/http/settings.go` + test
- Create: `backend/internal/transport/http/trends.go` + test

Same copy-and-update pattern as Task 9 for each handler+test pair:
1. `package handlers` → `package httphandler`
2. `package handlers_test` → `package httphandler_test`
3. Imports: `models` → appropriate `domain/<context>`, `repo` → `port/repository`
4. Type references updated to match domain types
5. Handler constructor alias in tests: `httphandler "github.com/.../internal/transport/http"`

Domain mapping for each:
- `transactions.go` → `finance.Transaction`, `finance.Budget`
- `goals.go` → `goals.Goal`, `goals.KeyResult`
- `kr_logs.go` → `goals.KRLog`
- `events.go` → `calendar.Event`
- `google_calendar.go` → `calendar.Event`
- `notes.go` → `notes.Note`
- `settings.go` → `settings.UserSettings`
- `trends.go` → `trends.NetWorthSnapshot`, `trends.BenchmarkData`, `trends.BankRate`, `trends.NewsItem`

- [ ] **Step 1: Copy and update all 8 handlers + tests** (apply the pattern above to each file)

- [ ] **Step 2: Verify compile + tests**

```bash
cd backend && go test ./internal/transport/http/... -v 2>&1 | tail -30
```

Expected: all tests PASS. Zero build errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/transport/http/
git commit -m "feat(transport/http): add remaining handlers (transactions, goals, events, notes, settings, trends)"
```

---

## Task 11: transport/http — dashboard handler

**Files:**
- Create: `backend/internal/transport/http/dashboard.go`
- Create: `backend/internal/transport/http/dashboard_test.go`

This handler is the only one that changes more than imports — it now calls `*dashboardsvc.Service` instead of `repo.DashboardRepo`.

- [ ] **Step 1: Create dashboard handler**

`backend/internal/transport/http/dashboard.go`:
```go
package httphandler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	dashboardsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/dashboard"
)

type DashboardHandler struct{ svc *dashboardsvc.Service }

func NewDashboardHandler(svc *dashboardsvc.Service) *DashboardHandler {
	return &DashboardHandler{svc}
}

func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	log.Printf("dashboard: summary request for uid=%q", uid)
	summary, err := h.svc.Summary(r.Context(), uid)
	if err != nil {
		log.Printf("dashboard: summary error: %v", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
```

- [ ] **Step 2: Create dashboard handler test**

`backend/internal/transport/http/dashboard_test.go`:
```go
package httphandler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/finance"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/goals"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/trends"
	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/wealth"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	dashboardsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/dashboard"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
)

// stubAssetsDash, stubLiabilitiesDash, stubTxsDash, stubGoalsDash, stubKRLogsDash, stubSnapshotsDash
// are identical to the stubs in service/dashboard/service_test.go
// Copy them verbatim here (Go requires stubs per package).

type stubAssetsDash struct{}
func (s *stubAssetsDash) List(_ context.Context, _ string) ([]wealth.Asset, error) { return []wealth.Asset{}, nil }
func (s *stubAssetsDash) Create(_ context.Context, a wealth.Asset) (wealth.Asset, error) { return a, nil }
func (s *stubAssetsDash) Update(_ context.Context, a wealth.Asset) (wealth.Asset, error) { return a, nil }
func (s *stubAssetsDash) Delete(_ context.Context, _, _ string) error { return nil }

type stubLiabilitiesDash struct{}
func (s *stubLiabilitiesDash) List(_ context.Context, _ string) ([]wealth.Liability, error) { return []wealth.Liability{}, nil }
func (s *stubLiabilitiesDash) Create(_ context.Context, l wealth.Liability) (wealth.Liability, error) { return l, nil }
func (s *stubLiabilitiesDash) Update(_ context.Context, l wealth.Liability) (wealth.Liability, error) { return l, nil }
func (s *stubLiabilitiesDash) Delete(_ context.Context, _, _ string) error { return nil }
func (s *stubLiabilitiesDash) TotalBalance(_ context.Context, _ string) (float64, error) { return 0, nil }

type stubTxsDash struct{}
func (s *stubTxsDash) List(_ context.Context, _, _, _, _ string, _, _ int) ([]finance.Transaction, error) { return []finance.Transaction{}, nil }
func (s *stubTxsDash) Create(_ context.Context, t finance.Transaction) (finance.Transaction, error) { return t, nil }
func (s *stubTxsDash) Delete(_ context.Context, _, _ string) error { return nil }
func (s *stubTxsDash) ListBudgets(_ context.Context, _ string) ([]finance.Budget, error) { return []finance.Budget{}, nil }
func (s *stubTxsDash) UpsertBudget(_ context.Context, b finance.Budget) (finance.Budget, error) { return b, nil }
func (s *stubTxsDash) SumByUser(_ context.Context, _ string) (float64, error) { return 0, nil }
func (s *stubTxsDash) SumSpentThisMonth(_ context.Context, _ string) (float64, error) { return 0, nil }

type stubGoalsDash struct{}
func (s *stubGoalsDash) List(_ context.Context, _ string) ([]goals.Goal, error) { return nil, nil }
func (s *stubGoalsDash) Create(_ context.Context, g goals.Goal) (goals.Goal, error) { return g, nil }
func (s *stubGoalsDash) Update(_ context.Context, g goals.Goal) (goals.Goal, error) { return g, nil }
func (s *stubGoalsDash) Delete(_ context.Context, _, _ string) error { return nil }
func (s *stubGoalsDash) AddKeyResult(_ context.Context, kr goals.KeyResult) (goals.KeyResult, error) { return kr, nil }
func (s *stubGoalsDash) UpdateKeyResult(_ context.Context, kr goals.KeyResult) (goals.KeyResult, error) { return kr, nil }
func (s *stubGoalsDash) DeleteKeyResult(_ context.Context, _, _ string) error { return nil }
func (s *stubGoalsDash) HabitsSummary(_ context.Context, _ string) (int, int, error) { return 5, 3, nil }
func (s *stubGoalsDash) GoalsAvgProgress(_ context.Context, _ string) (int, error) { return 65, nil }

type stubKRLogsDash struct{}
func (s *stubKRLogsDash) GetLogs(_ context.Context, _, _ string) ([]goals.KRLog, error) { return nil, nil }
func (s *stubKRLogsDash) GetLogRange(_ context.Context, _, _, _, _ string) ([]goals.KRLog, error) { return nil, nil }
func (s *stubKRLogsDash) ToggleLog(_ context.Context, _, _, _ string) (goals.KRLog, error) { return goals.KRLog{}, nil }

type stubSnapshotsDash struct{}
func (s *stubSnapshotsDash) ListSnapshots(_ context.Context, _ string) ([]trends.NetWorthSnapshot, error) {
	return []trends.NetWorthSnapshot{{NetWorth: 127450}}, nil
}
func (s *stubSnapshotsDash) UpsertSnapshot(_ context.Context, snap trends.NetWorthSnapshot) (trends.NetWorthSnapshot, error) { return snap, nil }
func (s *stubSnapshotsDash) UpsertBenchmark(_ context.Context, _ trends.BenchmarkData) error { return nil }
func (s *stubSnapshotsDash) ListBenchmarks(_ context.Context, _ []string, _, _ string) ([]trends.BenchmarkData, error) { return nil, nil }
func (s *stubSnapshotsDash) LatestBankRates(_ context.Context) ([]trends.BankRate, error) { return nil, nil }
func (s *stubSnapshotsDash) UpsertBankRate(_ context.Context, _ trends.BankRate) error { return nil }
func (s *stubSnapshotsDash) ListNews(_ context.Context, _ int) ([]trends.NewsItem, error) { return nil, nil }
func (s *stubSnapshotsDash) UpsertNews(_ context.Context, _ []trends.NewsItem) error { return nil }

func TestDashboardSummary(t *testing.T) {
	if os.Getenv("SUPABASE_JWT_SECRET") == "" {
		os.Setenv("SUPABASE_JWT_SECRET", "test-secret")
	}
	svc := dashboardsvc.New(
		&stubAssetsDash{}, &stubLiabilitiesDash{}, &stubTxsDash{},
		&stubGoalsDash{}, &stubKRLogsDash{}, &stubSnapshotsDash{},
	)
	h := httphandler.NewDashboardHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/summary", nil)
	req = req.WithContext(middleware.WithUserID(req.Context(), "user1"))
	w := httptest.NewRecorder()
	h.Summary(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var result dashboardsvc.Summary
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.HabitsTotal != 5 {
		t.Errorf("HabitsTotal: want 5, got %d", result.HabitsTotal)
	}
}
```

Note: `middleware.WithUserID` may not exist — check `internal/middleware/auth.go`. If the middleware sets context using a key, the test needs to inject the userID the same way. Look at how existing handler tests do it and replicate exactly.

- [ ] **Step 3: Check middleware helper**

```bash
grep -n "WithUserID\|GetUserID\|userIDKey" backend/internal/middleware/auth.go
```

Use whatever context setter the middleware exports (or create a `WithUserID` test helper if only `GetUserID` exists).

- [ ] **Step 4: Run test**

```bash
cd backend && go test ./internal/transport/http/ -run TestDashboard -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/transport/http/dashboard.go backend/internal/transport/http/dashboard_test.go
git commit -m "feat(transport/http): add dashboard handler using DashboardService"
```

---

## Task 12: Wire cmd/server/main.go

**Files:**
- Modify: `backend/cmd/server/main.go`

Replace all `internal/repo`, `internal/handlers`, `internal/models` imports with new paths.

- [ ] **Step 1: Update imports**

In `backend/cmd/server/main.go`, replace the import block:

```go
import (
	// ... keep existing non-repo imports ...
	"github.com/chiutuanbinh/mylifeos/backend/internal/infra/postgres"
	dashboardsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/dashboard"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
	// remove: internal/repo, internal/handlers, internal/models
)
```

- [ ] **Step 2: Update constructor calls**

Replace the wiring section:

```go
// Repos
txRepo      := postgres.NewTransactionRepo(db)
krLogRepo   := postgres.NewKRLogRepo(db)
goalRepo    := postgres.NewGoalRepo(db)
noteRepo    := postgres.NewNoteRepo(db)
eventRepo   := postgres.NewEventRepo(db)
assetRepo   := postgres.NewAssetRepo(db)
liabRepo    := postgres.NewLiabilityRepo(db)
settingsRepo := postgres.NewSettingsRepo(db)
trendsRepo  := postgres.NewTrendsRepo(db)

// Services
dashSvc := dashboardsvc.New(assetRepo, liabRepo, txRepo, goalRepo, krLogRepo, trendsRepo)

// Handlers
dashHandler     := httphandler.NewDashboardHandler(dashSvc)
txHandler       := httphandler.NewTransactionHandler(txRepo)
krLogHandler    := httphandler.NewKRLogHandler(krLogRepo)
goalHandler     := httphandler.NewGoalHandler(goalRepo)
noteHandler     := httphandler.NewNoteHandler(noteRepo)
eventHandler    := httphandler.NewEventHandler(eventRepo)
gcalHandler     := httphandler.NewGoogleCalendarHandler(eventRepo)
assetHandler    := httphandler.NewAssetHandler(assetRepo)
liabHandler     := httphandler.NewLiabilityHandler(liabRepo)
settingHandler  := httphandler.NewSettingsHandler(settingsRepo)
trendsHandler   := httphandler.NewTrendsHandler(trendsRepo, assetRepo)
```

- [ ] **Step 3: Verify build**

```bash
cd backend && go build ./...
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "chore(server): rewire main.go to new DDD packages"
```

---

## Task 13: Delete old packages and final verification

**Files:**
- Delete: `backend/internal/models/` (entire dir)
- Delete: `backend/internal/repo/` (entire dir)
- Delete: `backend/internal/handlers/` (entire dir)

- [ ] **Step 1: Delete old packages**

```bash
rm -rf backend/internal/models backend/internal/repo backend/internal/handlers
```

- [ ] **Step 2: Verify no stale references**

```bash
grep -r "internal/models\|internal/repo\|internal/handlers" backend/ --include="*.go"
```

Expected: no output.

- [ ] **Step 3: Full build**

```bash
cd backend && go build ./...
```

Expected: no output.

- [ ] **Step 4: Run all tests with coverage**

```bash
cd backend && go test ./internal/... -coverprofile=coverage.out -covermode=atomic 2>&1
```

Expected: all packages pass.

- [ ] **Step 5: Run pre-commit coverage gate**

```bash
bash backend/scripts/hooks/pre-commit
```

Expected: `✓ Coverage OK`

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "chore: delete legacy models, repo, handlers packages"
```

---

## Task 14: PR

- [ ] **Step 1: Push branch**

```bash
git push -u origin feat/ddd-backend-structure
```

- [ ] **Step 2: Create PR**

```bash
gh pr create \
  --title "feat: DDD backend restructure (domain / port / service / infra / transport)" \
  --body "$(cat <<'EOF'
## Summary
- Split `internal/models/` into bounded-context domain packages (`domain/wealth`, `domain/finance`, `domain/goals`, `domain/calendar`, `domain/notes`, `domain/settings`, `domain/trends`)
- Added `port/repository/` with typed interfaces (no pgx/SQL)
- Moved SQL implementations to `infra/postgres/`
- Moved HTTP handlers to `transport/http/`
- Extracted `service/wealth/` with pure `CurrentValue()` and `NetWorth()` functions
- Replaced fake `repo/dashboard.go` with `service/dashboard/DashboardService` that aggregates repos
- Deleted `internal/models`, `internal/repo`, `internal/handlers`

## No behaviour changes
SQL queries, HTTP routes, JSON field names, and test logic are unchanged.

## Test plan
- [ ] `go build ./...` passes
- [ ] All tests pass at ≥80% per file
- [ ] `grep -r "internal/repo\|internal/models\|internal/handlers" .` returns nothing
EOF
)"
```

- [ ] **Step 3: Enable auto-merge**

```bash
gh pr merge --auto --squash
```
