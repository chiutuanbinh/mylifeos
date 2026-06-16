# Backend DDD Refactor — Design Spec

Date: 2026-06-16

## Goal

Restructure `backend/internal/` using Domain-Driven Design so that:
- Business logic lives in domain + service layers (not handlers or repos)
- SQL is isolated in `infra/postgres/` — swappable without touching domain
- Handlers are thin HTTP adapters
- `DashboardService` replaces the bogus `repo/dashboard.go` aggregate

## Bounded Contexts

| Context    | Entities / Value Objects                        |
|------------|-------------------------------------------------|
| `wealth`   | Asset, Liability, NetWorth (computed VO)        |
| `finance`  | Transaction, Budget                             |
| `goals`    | Goal, KeyResult, KRLog                          |
| `calendar` | Event                                           |
| `notes`    | Note                                            |
| `settings` | UserSettings                                    |

Each context is a package under `internal/domain/<context>/`. Allowed imports: stdlib only (`time`, `encoding/json`, etc.).

## Target Directory Layout

```
backend/
  cmd/server/main.go              — wire everything together
  internal/
    domain/
      wealth/
        entity.go                 — Asset, Liability structs + NetWorth value object
      finance/
        entity.go                 — Transaction, Budget
      goals/
        entity.go                 — Goal, KeyResult, KRLog
      calendar/
        entity.go                 — Event
      notes/
        entity.go                 — Note
      settings/
        entity.go                 — UserSettings
      trends/
        entity.go                 — NetWorthSnapshot, BenchmarkData, BankRate, NewsItem
    port/
      repository/
        wealth.go                 — AssetRepo, LiabilityRepo interfaces
        finance.go                — TransactionRepo interface
        goals.go                  — GoalRepo, KRLogRepo interfaces
        calendar.go               — EventRepo interface
        notes.go                  — NoteRepo interface
        settings.go               — SettingsRepo interface
        trends.go                 — TrendsRepo interface
    service/
      dashboard/
        service.go                — DashboardService: Summary()
        service_test.go
      wealth/
        service.go                — WealthService: NetWorth(), CurrentValue()
        service_test.go
      trends/
        service.go                — TrendsService: ListSnapshots(), UpsertSnapshot()
        service_test.go
    infra/
      postgres/
        assets.go                 — implements port/repository.AssetRepo
        liabilities.go            — implements port/repository.LiabilityRepo
        transactions.go
        goals.go
        kr_logs.go
        events.go
        notes.go
        settings.go
        trends.go
        db.go                     — pool setup (moved from repo/db.go)
      scraper/
        scraper.go                — moved from internal/scraper/
    transport/
      http/
        assets.go                 — thin handler: parse → service/repo → encode
        liabilities.go
        transactions.go
        goals.go
        kr_logs.go
        events.go
        google_calendar.go
        notes.go
        settings.go
        dashboard.go
        trends.go
    middleware/
      auth.go                     — unchanged
    migrate/
      migrate.go                  — unchanged
```

## Layer Rules

```
transport/http  →  service/*  →  port/repository (interface)
                              ↑
                         infra/postgres (implements)
```

- `domain/*` — zero external imports (stdlib only)
- `port/repository` — imports `domain/*` only
- `service/*` — imports `domain/*` + `port/repository` only
- `infra/postgres` — imports `domain/*` + `port/repository` + pgx
- `transport/http` — imports `service/*` + `port/repository` + `domain/*` + chi

## DashboardService Design

```go
type DashboardService struct {
    assets      repository.AssetRepo
    liabilities repository.LiabilityRepo
    txs         repository.TransactionRepo
    goals       repository.GoalRepo
    krLogs      repository.KRLogRepo
    budgets     repository.BudgetRepo
    snapshots   repository.TrendsRepo
}

func (s *DashboardService) Summary(ctx context.Context, userID string) (domain.DashboardSummary, error)
```

Net worth formula lives in `service/wealth/service.go`:

```go
func NetWorth(assets []wealth.Asset, cashPosition float64, liabilities []wealth.Liability) float64
func CurrentValue(a wealth.Asset) float64   // depreciation logic moved here
```

`DashboardService.Summary` calls `WealthService` methods — no SQL formula duplication.

## WealthService Design

```go
type WealthService struct{}   // stateless, pure functions

func (s WealthService) CurrentValue(a wealth.Asset) float64
func (s WealthService) NetWorth(assets []wealth.Asset, cash float64, liabilities []wealth.Liability) float64
```

Both dashboard and trends handlers use `WealthService` — single source of truth for net worth math.

## Migration Strategy

Rename/move, don't rewrite logic:
1. Move structs from `models/` → `domain/<context>/entity.go`
2. Copy repo interfaces → `port/repository/`
3. Move repo implementations → `infra/postgres/` (rename receivers, same SQL)
4. Extract business logic from `repo/dashboard.go` → `service/dashboard/` + `service/wealth/`
5. Move handlers → `transport/http/` (update imports only)
6. Update `cmd/server/main.go` to wire new paths
7. Delete old `models/`, `repo/`, `handlers/` packages

Tests move with their files. No logic changes during migration — only structural moves + import updates.

## What Does NOT Change

- SQL queries (verbatim moves)
- HTTP route paths
- JSON field names
- Test logic
- `middleware/` and `migrate/` packages
- Frontend API contract

## Success Criteria

- `go build ./...` passes
- All existing tests pass at ≥80% coverage per file
- `grep -r "internal/repo" .` returns nothing
- `grep -r "internal/models" .` returns nothing
- `grep -r "internal/handlers" .` returns nothing
