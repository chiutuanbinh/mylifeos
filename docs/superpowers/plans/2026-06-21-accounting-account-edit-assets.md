# Accounting: Account Edit, Opening Balance, Physical Assets, Net Income — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add account editing (name/type/parent/sort order), opening balance on create, physical asset metadata on accounts (replacing `wealth.Asset`), and a net income YTD display card.

**Architecture:** Backend follows domain → service → repo → HTTP layering already in place. All four features share the same `Account` domain object; domain changes cascade upward through service, repo, and HTTP. Frontend extends existing `AccountingPage.tsx` with edit modal, opening balance field, Assets tab, and net income card.

**Tech Stack:** Go + chi (backend), pgx/v5 (postgres), shopspring/decimal, React + Ant Design + TanStack Query (frontend), Vite.

## Global Constraints

- Branch protection: never push to `main` directly — all work on a feature branch
- Backend test coverage: ≥80% per file in `transport/http` and `middleware` packages
- Run `bash scripts/hooks/pre-commit` from repo root before PR
- Frontend: `npm run lint && npm run build` must be clean before PR
- JWT in memory only — never localStorage
- Migration files go in `supabase/migrations/` with timestamp prefix `20260621HHMMSS_`
- `shopspring/decimal` for all monetary arithmetic — never `float64`
- `chi.URLParam(r, "id")` to extract path params in HTTP handlers

---

## File Map

| File | Change |
|------|--------|
| `supabase/migrations/20260621000001_account_asset_meta.sql` | CREATE — add 4 asset columns to `accounts` |
| `backend/internal/domain/accounting/account.go` | MODIFY — AssetMeta struct, mutation methods, updated constructors/accessors |
| `backend/internal/domain/accounting/account_test.go` | MODIFY — tests for mutation methods and AssetMeta |
| `backend/internal/service/accounting/commands.go` | MODIFY — UpdateAccountCmd, AssetMetaCmd, OpeningBalance on OpenAccountCmd |
| `backend/internal/port/repository/accounting.go` | MODIFY — add FindByNameAndType to AccountRepo interface |
| `backend/internal/infra/postgres/accounting_accounts.go` | MODIFY — FindByNameAndType impl, asset meta columns in Save/scan |
| `backend/internal/service/accounting/account_service.go` | MODIFY — UpdateAccount method, opening balance logic in OpenAccount |
| `backend/internal/service/accounting/account_service_test.go` | MODIFY — tests for UpdateAccount and opening balance |
| `backend/internal/service/accounting/networth_query.go` | MODIFY — add NetIncomeYTD to result |
| `backend/internal/transport/http/accounting_accounts.go` | MODIFY — PATCH handler, asset_meta in List response, opening_balance in Create |
| `backend/internal/transport/http/accounting_accounts_test.go` | MODIFY — tests for PATCH, asset_meta, opening_balance |
| `backend/cmd/server/main.go` | MODIFY — register `r.Patch("/accounts/{id}", ...)` |
| `frontend/src/api/types.ts` | MODIFY — AssetMeta interface, Account extended, NetWorthResult extended, UpdateAccountRequest |
| `frontend/src/api/endpoints.ts` | MODIFY — updateAccount function |
| `frontend/src/pages/AccountingPage.tsx` | MODIFY — edit modal, opening balance field, Assets tab, net income card |
| `frontend/src/pages/WealthPage.tsx` | MODIFY — deprecation banner on Assets tab |

---

## Task 1: DB Migration — Asset Metadata Columns

**Files:**
- Create: `supabase/migrations/20260621000001_account_asset_meta.sql`

**Interfaces:**
- Produces: `accounts` table with columns `purchase_value NUMERIC`, `purchased_at DATE`, `depreciation_rate NUMERIC`, `asset_notes TEXT` (all nullable)

- [ ] **Step 1: Create migration file**

```sql
-- supabase/migrations/20260621000001_account_asset_meta.sql
ALTER TABLE accounts
  ADD COLUMN IF NOT EXISTS purchase_value    NUMERIC,
  ADD COLUMN IF NOT EXISTS purchased_at      DATE,
  ADD COLUMN IF NOT EXISTS depreciation_rate NUMERIC,
  ADD COLUMN IF NOT EXISTS asset_notes       TEXT;
```

- [ ] **Step 2: Apply migration locally**

```bash
docker compose exec postgres psql -U postgres -d mylifeos -c "
  ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS purchase_value    NUMERIC,
    ADD COLUMN IF NOT EXISTS purchased_at      DATE,
    ADD COLUMN IF NOT EXISTS depreciation_rate NUMERIC,
    ADD COLUMN IF NOT EXISTS asset_notes       TEXT;
"
```

Expected: `ALTER TABLE`

- [ ] **Step 3: Commit**

```bash
git add supabase/migrations/20260621000001_account_asset_meta.sql
git commit -m "chore(db): add asset metadata columns to accounts"
```

---

## Task 2: Domain — AssetMeta + Account Mutation Methods

**Files:**
- Modify: `backend/internal/domain/accounting/account.go`
- Modify: `backend/internal/domain/accounting/account_test.go`

**Interfaces:**
- Consumes: nothing new
- Produces:
  - `type AssetMeta struct` with fields `PurchaseValue *decimal.Decimal`, `PurchasedAt *time.Time`, `DepreciationRate *decimal.Decimal`, `Notes string`
  - `(a *Account) AssetMeta() *AssetMeta`
  - `(a *Account) AttachAssetMeta(m *AssetMeta)`
  - `(a *Account) Rename(name string)`
  - `(a *Account) ChangeType(t AccountType)`
  - `(a *Account) Reparent(parentID *AccountID)`
  - `(a *Account) Reorder(n int)`
  - `NewAccount` and `ReconstituteAccount` signatures unchanged (asset meta set via `AttachAssetMeta`)

- [ ] **Step 1: Write failing tests**

Add to `backend/internal/domain/accounting/account_test.go`:

```go
func TestAccount_Rename(t *testing.T) {
	a := accounting.NewAccount("u1", nil, "Old", accounting.Asset, "VND", false, 0)
	a.Rename("New")
	if a.Name() != "New" {
		t.Errorf("want Name=New, got %s", a.Name())
	}
}

func TestAccount_ChangeType(t *testing.T) {
	a := accounting.NewAccount("u1", nil, "X", accounting.Asset, "VND", false, 0)
	a.ChangeType(accounting.Expense)
	if a.Type() != accounting.Expense {
		t.Errorf("want Expense, got %s", a.Type())
	}
}

func TestAccount_Reparent(t *testing.T) {
	pid := accounting.AccountID("parent-1")
	a := accounting.NewAccount("u1", nil, "X", accounting.Asset, "VND", false, 0)
	a.Reparent(&pid)
	if a.ParentID() == nil || *a.ParentID() != pid {
		t.Error("want ParentID set")
	}
	a.Reparent(nil)
	if a.ParentID() != nil {
		t.Error("want ParentID nil after clear")
	}
}

func TestAccount_Reorder(t *testing.T) {
	a := accounting.NewAccount("u1", nil, "X", accounting.Asset, "VND", false, 0)
	a.Reorder(5)
	if a.SortOrder() != 5 {
		t.Errorf("want SortOrder=5, got %d", a.SortOrder())
	}
}

func TestAccount_AttachAssetMeta(t *testing.T) {
	a := accounting.NewAccount("u1", nil, "Car", accounting.Asset, "VND", false, 0)
	if a.AssetMeta() != nil {
		t.Error("want nil AssetMeta initially")
	}
	pv := decimal.NewFromInt(500_000_000)
	now := time.Now()
	dr := decimal.NewFromFloat(0.15)
	a.AttachAssetMeta(&accounting.AssetMeta{
		PurchaseValue:    &pv,
		PurchasedAt:      &now,
		DepreciationRate: &dr,
		Notes:            "Toyota",
	})
	if a.AssetMeta() == nil {
		t.Fatal("want non-nil AssetMeta")
	}
	if a.AssetMeta().Notes != "Toyota" {
		t.Errorf("want Notes=Toyota, got %s", a.AssetMeta().Notes)
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
cd backend && go test ./internal/domain/accounting/... -run "TestAccount_Rename|TestAccount_ChangeType|TestAccount_Reparent|TestAccount_Reorder|TestAccount_AttachAssetMeta" -v
```

Expected: compilation error or FAIL — methods don't exist yet.

- [ ] **Step 3: Implement in account.go**

Replace the `Account` struct and add all new code. Full replacement of `account.go`:

```go
package accounting

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type AccountID string
type AccountType string
type Side string

const (
	Asset     AccountType = "asset"
	Liability AccountType = "liability"
	Equity    AccountType = "equity"
	Income    AccountType = "income"
	Expense   AccountType = "expense"
)

const (
	Debit  Side = "debit"
	Credit Side = "credit"
)

type AssetMeta struct {
	PurchaseValue    *decimal.Decimal
	PurchasedAt      *time.Time
	DepreciationRate *decimal.Decimal
	Notes            string
}

type Account struct {
	id        AccountID
	userID    string
	parentID  *AccountID
	name      string
	acctType  AccountType
	currency  string
	isGroup   bool
	archived  bool
	sortOrder int
	assetMeta *AssetMeta
}

func NewAccount(userID string, parentID *string, name string, acctType AccountType, currency string, isGroup bool, sortOrder int) *Account {
	var pid *AccountID
	if parentID != nil {
		p := AccountID(*parentID)
		pid = &p
	}
	return &Account{
		id:        AccountID(newID()),
		userID:    userID,
		parentID:  pid,
		name:      name,
		acctType:  acctType,
		currency:  currency,
		isGroup:   isGroup,
		sortOrder: sortOrder,
	}
}

func ReconstituteAccount(id, userID string, parentID *string, name string, acctType AccountType, currency string, isGroup, archived bool, sortOrder int) *Account {
	var pid *AccountID
	if parentID != nil {
		p := AccountID(*parentID)
		pid = &p
	}
	return &Account{
		id:        AccountID(id),
		userID:    userID,
		parentID:  pid,
		name:      name,
		acctType:  acctType,
		currency:  currency,
		isGroup:   isGroup,
		archived:  archived,
		sortOrder: sortOrder,
	}
}

func (a *Account) ID() AccountID        { return a.id }
func (a *Account) UserID() string        { return a.userID }
func (a *Account) ParentID() *AccountID  { return a.parentID }
func (a *Account) Name() string          { return a.name }
func (a *Account) Type() AccountType     { return a.acctType }
func (a *Account) Currency() string      { return a.currency }
func (a *Account) IsGroup() bool         { return a.isGroup }
func (a *Account) Archived() bool        { return a.archived }
func (a *Account) SortOrder() int        { return a.sortOrder }
func (a *Account) AssetMeta() *AssetMeta { return a.assetMeta }

// Mutation methods
func (a *Account) Rename(name string)            { a.name = name }
func (a *Account) ChangeType(t AccountType)      { a.acctType = t }
func (a *Account) Reparent(parentID *AccountID)  { a.parentID = parentID }
func (a *Account) Reorder(n int)                 { a.sortOrder = n }
func (a *Account) AttachAssetMeta(m *AssetMeta)  { a.assetMeta = m }

func (a *Account) NormalBalance() Side {
	switch a.acctType {
	case Asset, Expense:
		return Debit
	default:
		return Credit
	}
}

func (a *Account) Balance(lines []JournalLine) (Money, error) {
	normal := a.NormalBalance()
	total := Money{Amount: zeroDecimal(), Currency: a.currency}
	for _, l := range lines {
		if l.AccountID() != a.id {
			continue
		}
		lineAmount := Money{Amount: l.Money().Amount, Currency: l.Money().Currency}
		var err error
		if l.Side() == normal {
			total, err = total.Add(lineAmount)
		} else {
			if total.Currency != lineAmount.Currency {
				return Money{}, fmt.Errorf("currency mismatch in account %s: %s vs %s", a.id, total.Currency, lineAmount.Currency)
			}
			total = Money{Amount: total.Amount.Sub(lineAmount.Amount), Currency: total.Currency}
		}
		if err != nil {
			return Money{}, fmt.Errorf("currency mismatch in account %s: %w", a.id, err)
		}
	}
	return total, nil
}
```

Note: add `"time"` and `"github.com/shopspring/decimal"` to imports.

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd backend && go test ./internal/domain/accounting/... -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/accounting/account.go backend/internal/domain/accounting/account_test.go
git commit -m "feat(accounting): AssetMeta struct and Account mutation methods"
```

---

## Task 3: Repo — FindByNameAndType + Asset Meta Persistence

**Files:**
- Modify: `backend/internal/port/repository/accounting.go`
- Modify: `backend/internal/infra/postgres/accounting_accounts.go`

**Interfaces:**
- Consumes: `AssetMeta` from Task 2
- Produces:
  - `AccountRepo` interface gains `FindByNameAndType(ctx context.Context, userID, name string, t accounting.AccountType) (*accounting.Account, error)`
  - `repo.Save` persists asset meta columns
  - `scanAccounts` reads asset meta columns and calls `AttachAssetMeta`
  - Error sentinel: `ErrAccountNotFound = errors.New("account not found")`

- [ ] **Step 1: Add FindByNameAndType to interface**

Full replacement of `backend/internal/port/repository/accounting.go`:

```go
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
)

var ErrAccountNotFound = errors.New("account not found")

type AccountRepo interface {
	Save(ctx context.Context, a *accounting.Account) error
	FindByUser(ctx context.Context, userID string) ([]*accounting.Account, error)
	FindByID(ctx context.Context, id accounting.AccountID) (*accounting.Account, error)
	FindByNameAndType(ctx context.Context, userID, name string, t accounting.AccountType) (*accounting.Account, error)
}

type JournalRepo interface {
	Save(ctx context.Context, e *accounting.JournalEntry) error
	FindByUser(ctx context.Context, userID string, from, to time.Time) ([]*accounting.JournalEntry, error)
}
```

- [ ] **Step 2: Update postgres implementation**

Full replacement of `backend/internal/infra/postgres/accounting_accounts.go`:

```go
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type pgAccountRepo struct{ db *pgxpool.Pool }

func NewAccountRepo(db *pgxpool.Pool) repository.AccountRepo {
	return &pgAccountRepo{db: db}
}

func (r *pgAccountRepo) Save(ctx context.Context, a *accounting.Account) error {
	var parentID *string
	if a.ParentID() != nil {
		s := string(*a.ParentID())
		parentID = &s
	}
	var (
		purchaseValue    *decimal.Decimal
		purchasedAt      *time.Time
		depreciationRate *decimal.Decimal
		assetNotes       *string
	)
	if m := a.AssetMeta(); m != nil {
		purchaseValue = m.PurchaseValue
		purchasedAt = m.PurchasedAt
		depreciationRate = m.DepreciationRate
		if m.Notes != "" {
			assetNotes = &m.Notes
		}
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO accounts (id, user_id, parent_id, name, type, currency, is_group, archived, sort_order,
		                      purchase_value, purchased_at, depreciation_rate, asset_notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO UPDATE SET
			parent_id=$3, name=$4, type=$5, currency=$6, is_group=$7, archived=$8, sort_order=$9,
			purchase_value=$10, purchased_at=$11, depreciation_rate=$12, asset_notes=$13`,
		string(a.ID()), a.UserID(), parentID, a.Name(),
		string(a.Type()), a.Currency(), a.IsGroup(), a.Archived(), a.SortOrder(),
		purchaseValue, purchasedAt, depreciationRate, assetNotes,
	)
	return err
}

func (r *pgAccountRepo) FindByUser(ctx context.Context, userID string) ([]*accounting.Account, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, parent_id, name, type, currency, is_group, archived, sort_order,
		       purchase_value, purchased_at, depreciation_rate, asset_notes
		FROM accounts WHERE user_id = $1 AND archived = false ORDER BY sort_order, name`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAccounts(rows)
}

func (r *pgAccountRepo) FindByID(ctx context.Context, id accounting.AccountID) (*accounting.Account, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, parent_id, name, type, currency, is_group, archived, sort_order,
		       purchase_value, purchased_at, depreciation_rate, asset_notes
		FROM accounts WHERE id = $1`,
		string(id),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts, err := scanAccounts(rows)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, repository.ErrAccountNotFound
	}
	return accounts[0], nil
}

func (r *pgAccountRepo) FindByNameAndType(ctx context.Context, userID, name string, t accounting.AccountType) (*accounting.Account, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, parent_id, name, type, currency, is_group, archived, sort_order,
		       purchase_value, purchased_at, depreciation_rate, asset_notes
		FROM accounts WHERE user_id = $1 AND name = $2 AND type = $3 AND archived = false LIMIT 1`,
		userID, name, string(t),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts, err := scanAccounts(rows)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, repository.ErrAccountNotFound
	}
	return accounts[0], nil
}

func scanAccounts(rows pgx.Rows) ([]*accounting.Account, error) {
	var result []*accounting.Account
	for rows.Next() {
		var (
			id, userID, name, acctType, currency string
			parentID                             *string
			isGroup, archived                    bool
			sortOrder                            int
			purchaseValue                        *decimal.Decimal
			purchasedAt                          *time.Time
			depreciationRate                     *decimal.Decimal
			assetNotes                           *string
		)
		if err := rows.Scan(
			&id, &userID, &parentID, &name, &acctType, &currency, &isGroup, &archived, &sortOrder,
			&purchaseValue, &purchasedAt, &depreciationRate, &assetNotes,
		); err != nil {
			return nil, err
		}
		a := accounting.ReconstituteAccount(
			id, userID, parentID, name,
			accounting.AccountType(acctType), currency, isGroup, archived, sortOrder,
		)
		if purchaseValue != nil || purchasedAt != nil || depreciationRate != nil || assetNotes != nil {
			meta := &accounting.AssetMeta{
				PurchaseValue:    purchaseValue,
				PurchasedAt:      purchasedAt,
				DepreciationRate: depreciationRate,
			}
			if assetNotes != nil {
				meta.Notes = *assetNotes
			}
			a.AttachAssetMeta(meta)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
```

- [ ] **Step 3: Update testAccountRepo in test file to implement new interface**

In `backend/internal/transport/http/accounting_accounts_test.go`, add `FindByNameAndType` to `testAccountRepo`.

Also update all calls to `accountingsvc.NewAccountService(repo)` → `accountingsvc.NewAccountService(repo, &testJournalRepo{})` since the constructor gains a `journal` parameter in Task 4. Search for them:
```bash
grep -n "NewAccountService" backend/internal/transport/http/accounting_accounts_test.go
```

```go
func (r *testAccountRepo) FindByNameAndType(_ context.Context, userID, name string, t accounting.AccountType) (*accounting.Account, error) {
	for _, a := range r.accounts {
		if a.UserID() == userID && a.Name() == name && a.Type() == t {
			return a, nil
		}
	}
	return nil, repository.ErrAccountNotFound
}
```

Also add import `"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"` to the test file.

- [ ] **Step 4: Build to verify no compilation errors**

```bash
cd backend && go build ./...
```

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add backend/internal/port/repository/accounting.go \
        backend/internal/infra/postgres/accounting_accounts.go \
        backend/internal/transport/http/accounting_accounts_test.go
git commit -m "feat(accounting): repo FindByNameAndType + asset meta persistence"
```

---

## Task 4: Service — UpdateAccount + Opening Balance

**Files:**
- Modify: `backend/internal/service/accounting/commands.go`
- Modify: `backend/internal/service/accounting/account_service.go`
- Modify: `backend/internal/service/accounting/account_service_test.go`

**Interfaces:**
- Consumes: `repository.ErrAccountNotFound`, `Account` mutation methods from Task 2, `FindByNameAndType` from Task 3
- Produces:
  - `UpdateAccountCmd{ID, UserID, Name string; Type accounting.AccountType; ParentID *string; SortOrder int; AssetMeta *AssetMetaCmd}`
  - `AssetMetaCmd{PurchaseValue *decimal.Decimal; PurchasedAt *time.Time; DepreciationRate *decimal.Decimal; Notes string}`
  - `OpenAccountCmd` gains `OpeningBalance *decimal.Decimal`
  - `AccountService.UpdateAccount(ctx, UpdateAccountCmd) error`

- [ ] **Step 1: Write failing tests**

Add to `backend/internal/service/accounting/account_service_test.go`:

```go
func TestAccountService_UpdateAccount(t *testing.T) {
	repo := newMemAccountRepo()
	svc := accountingsvc.NewAccountService(repo, nil)

	// create an account first
	id, err := svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{
		UserID: "u1", Name: "Old Name", Type: accounting.Asset, Currency: "VND",
	})
	if err != nil {
		t.Fatal(err)
	}

	// update it
	err = svc.UpdateAccount(context.Background(), accountingsvc.UpdateAccountCmd{
		ID: string(id), UserID: "u1", Name: "New Name", Type: accounting.Expense, SortOrder: 3,
	})
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}

	accounts, _ := svc.ListAccounts(context.Background(), "u1")
	if len(accounts) != 1 || accounts[0].Name() != "New Name" || accounts[0].Type() != accounting.Expense {
		t.Errorf("want updated account, got %+v", accounts)
	}
}

func TestAccountService_UpdateAccount_WrongUser(t *testing.T) {
	repo := newMemAccountRepo()
	svc := accountingsvc.NewAccountService(repo, nil)

	id, _ := svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{
		UserID: "u1", Name: "X", Type: accounting.Asset, Currency: "VND",
	})

	err := svc.UpdateAccount(context.Background(), accountingsvc.UpdateAccountCmd{
		ID: string(id), UserID: "u2", Name: "Hacked",
	})
	if err == nil {
		t.Error("want error for wrong user")
	}
}

func TestAccountService_OpenAccount_WithOpeningBalance(t *testing.T) {
	repo := newMemAccountRepo()
	journalRepo := newMemJournalRepo()
	svc := accountingsvc.NewAccountService(repo, journalRepo)

	// create Opening Balance equity account first
	_, err := svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{
		UserID: "u1", Name: "Opening Balance", Type: accounting.Equity, Currency: "VND", IsGroup: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	ob := decimal.NewFromInt(1_000_000)
	_, err = svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{
		UserID: "u1", Name: "Cash", Type: accounting.Asset, Currency: "VND",
		OpeningBalance: &ob,
	})
	if err != nil {
		t.Fatalf("OpenAccount with opening balance: %v", err)
	}

	entries, _ := journalRepo.FindByUser(context.Background(), "u1", time.Time{}, time.Now())
	if len(entries) != 1 {
		t.Fatalf("want 1 journal entry, got %d", len(entries))
	}
	if len(entries[0].Lines()) != 2 {
		t.Errorf("want 2 lines, got %d", len(entries[0].Lines()))
	}
}

func TestAccountService_OpenAccount_OpeningBalance_NoEquityAccount(t *testing.T) {
	repo := newMemAccountRepo()
	journalRepo := newMemJournalRepo()
	svc := accountingsvc.NewAccountService(repo, journalRepo)

	ob := decimal.NewFromInt(500_000)
	_, err := svc.OpenAccount(context.Background(), accountingsvc.OpenAccountCmd{
		UserID: "u1", Name: "Cash", Type: accounting.Asset, Currency: "VND",
		OpeningBalance: &ob,
	})
	if err == nil {
		t.Error("want error when Opening Balance account missing")
	}
}
```

Also ensure `account_service_test.go` has a `newMemJournalRepo()` helper that implements `repository.JournalRepo`. Check existing test file — if it already exists, reuse it. The `newMemAccountRepo()` must also implement `FindByNameAndType`. Full helpers:

```go
type memAccountRepo struct {
	accounts map[accounting.AccountID]*accounting.Account
}

func newMemAccountRepo() *memAccountRepo {
	return &memAccountRepo{accounts: map[accounting.AccountID]*accounting.Account{}}
}

func (r *memAccountRepo) Save(_ context.Context, a *accounting.Account) error {
	r.accounts[a.ID()] = a
	return nil
}

func (r *memAccountRepo) FindByUser(_ context.Context, userID string) ([]*accounting.Account, error) {
	var res []*accounting.Account
	for _, a := range r.accounts {
		if a.UserID() == userID {
			res = append(res, a)
		}
	}
	return res, nil
}

func (r *memAccountRepo) FindByID(_ context.Context, id accounting.AccountID) (*accounting.Account, error) {
	a, ok := r.accounts[id]
	if !ok {
		return nil, repository.ErrAccountNotFound
	}
	return a, nil
}

func (r *memAccountRepo) FindByNameAndType(_ context.Context, userID, name string, t accounting.AccountType) (*accounting.Account, error) {
	for _, a := range r.accounts {
		if a.UserID() == userID && a.Name() == name && a.Type() == t {
			return a, nil
		}
	}
	return nil, repository.ErrAccountNotFound
}

type memJournalRepo struct {
	entries []*accounting.JournalEntry
}

func newMemJournalRepo() *memJournalRepo { return &memJournalRepo{} }

func (r *memJournalRepo) Save(_ context.Context, e *accounting.JournalEntry) error {
	r.entries = append(r.entries, e)
	return nil
}

func (r *memJournalRepo) FindByUser(_ context.Context, userID string, from, to time.Time) ([]*accounting.JournalEntry, error) {
	var res []*accounting.JournalEntry
	for _, e := range r.entries {
		if e.UserID() == userID {
			res = append(res, e)
		}
	}
	return res, nil
}
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
cd backend && go test ./internal/service/accounting/... -run "TestAccountService_UpdateAccount|TestAccountService_OpenAccount_WithOpeningBalance|TestAccountService_OpenAccount_OpeningBalance" -v
```

Expected: compilation error — UpdateAccount not defined.

- [ ] **Step 3: Update commands.go**

Full replacement of `backend/internal/service/accounting/commands.go`:

```go
package accountingsvc

import (
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/shopspring/decimal"
)

type LineCmd struct {
	AccountID string
	Amount    decimal.Decimal
	Currency  string
	Side      accounting.Side
}

type RecordTransactionCmd struct {
	UserID      string
	Date        time.Time
	Description string
	Memo        string
	Lines       []LineCmd
}

type AssetMetaCmd struct {
	PurchaseValue    *decimal.Decimal
	PurchasedAt      *time.Time
	DepreciationRate *decimal.Decimal
	Notes            string
}

type OpenAccountCmd struct {
	UserID         string
	ParentID       *string
	Name           string
	Type           accounting.AccountType
	Currency       string
	IsGroup        bool
	SortOrder      int
	OpeningBalance *decimal.Decimal
	AssetMeta      *AssetMetaCmd
}

type UpdateAccountCmd struct {
	ID        string
	UserID    string
	Name      string
	Type      accounting.AccountType
	ParentID  *string
	SortOrder int
	AssetMeta *AssetMetaCmd
}
```

- [ ] **Step 4: Update account_service.go**

Full replacement of `backend/internal/service/accounting/account_service.go`:

```go
package accountingsvc

import (
	"context"
	"errors"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type AccountService struct {
	accounts repository.AccountRepo
	journal  repository.JournalRepo
}

func NewAccountService(accounts repository.AccountRepo, journal repository.JournalRepo) *AccountService {
	return &AccountService{accounts: accounts, journal: journal}
}

func (s *AccountService) OpenAccount(ctx context.Context, cmd OpenAccountCmd) (accounting.AccountID, error) {
	if cmd.ParentID != nil {
		parent, err := s.accounts.FindByID(ctx, accounting.AccountID(*cmd.ParentID))
		if err != nil {
			return "", err
		}
		if parent.UserID() != cmd.UserID {
			return "", errors.New("parent account not found")
		}
		if !parent.IsGroup() {
			return "", errors.New("parent account must be a group")
		}
	}
	if cmd.Currency == "" {
		cmd.Currency = "VND"
	}
	a := accounting.NewAccount(cmd.UserID, cmd.ParentID, cmd.Name, cmd.Type, cmd.Currency, cmd.IsGroup, cmd.SortOrder)
	if cmd.AssetMeta != nil {
		a.AttachAssetMeta(&accounting.AssetMeta{
			PurchaseValue:    cmd.AssetMeta.PurchaseValue,
			PurchasedAt:      cmd.AssetMeta.PurchasedAt,
			DepreciationRate: cmd.AssetMeta.DepreciationRate,
			Notes:            cmd.AssetMeta.Notes,
		})
	}
	if err := s.accounts.Save(ctx, a); err != nil {
		return "", err
	}
	if cmd.OpeningBalance != nil && cmd.OpeningBalance.IsPositive() {
		ob, err := s.accounts.FindByNameAndType(ctx, cmd.UserID, "Opening Balance", accounting.Equity)
		if err != nil {
			return "", errors.New("Opening Balance equity account not found; run account setup first")
		}
		entry := accounting.NewJournalEntry(
			cmd.UserID,
			time.Now(),
			"Opening balance — "+cmd.Name,
			"",
			[]LineCmd{
				{AccountID: string(a.ID()), Amount: *cmd.OpeningBalance, Currency: cmd.Currency, Side: accounting.Debit},
				{AccountID: string(ob.ID()), Amount: *cmd.OpeningBalance, Currency: cmd.Currency, Side: accounting.Credit},
			},
		)
		if err := s.journal.Save(ctx, entry); err != nil {
			return "", err
		}
	}
	return a.ID(), nil
}

func (s *AccountService) UpdateAccount(ctx context.Context, cmd UpdateAccountCmd) error {
	a, err := s.accounts.FindByID(ctx, accounting.AccountID(cmd.ID))
	if err != nil {
		return err
	}
	if a.UserID() != cmd.UserID {
		return errors.New("account not found")
	}
	if cmd.ParentID != nil {
		parent, err := s.accounts.FindByID(ctx, accounting.AccountID(*cmd.ParentID))
		if err != nil {
			return errors.New("parent account not found")
		}
		if parent.UserID() != cmd.UserID {
			return errors.New("parent account not found")
		}
		if !parent.IsGroup() {
			return errors.New("parent account must be a group")
		}
		pid := accounting.AccountID(*cmd.ParentID)
		a.Reparent(&pid)
	} else {
		a.Reparent(nil)
	}
	a.Rename(cmd.Name)
	a.ChangeType(cmd.Type)
	a.Reorder(cmd.SortOrder)
	if cmd.AssetMeta != nil {
		a.AttachAssetMeta(&accounting.AssetMeta{
			PurchaseValue:    cmd.AssetMeta.PurchaseValue,
			PurchasedAt:      cmd.AssetMeta.PurchasedAt,
			DepreciationRate: cmd.AssetMeta.DepreciationRate,
			Notes:            cmd.AssetMeta.Notes,
		})
	} else {
		a.AttachAssetMeta(nil)
	}
	return s.accounts.Save(ctx, a)
}

func (s *AccountService) ListAccounts(ctx context.Context, userID string) ([]*accounting.Account, error) {
	return s.accounts.FindByUser(ctx, userID)
}
```

Note: `accounting.NewJournalEntry` must accept `[]LineCmd` or the service must convert — check the existing `JournalService.RecordTransaction` to see how `NewJournalEntry` works and match that pattern. If `NewJournalEntry` takes domain types directly, convert `LineCmd` inline.

- [ ] **Step 4b: Check NewJournalEntry signature and fix if needed**

```bash
grep -n "func NewJournalEntry" /Users/binhct/workspace/mylifeos/backend/internal/domain/accounting/journal_entry.go
```

If `NewJournalEntry` does not take `[]LineCmd`, look at `journal_service.go` for how it builds a `JournalEntry` and replicate that pattern inside `OpenAccount` in `account_service.go`.

- [ ] **Step 5: Run tests — expect PASS**

```bash
cd backend && go test ./internal/service/accounting/... -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/accounting/commands.go \
        backend/internal/service/accounting/account_service.go \
        backend/internal/service/accounting/account_service_test.go
git commit -m "feat(accounting): UpdateAccount service + opening balance on create"
```

---

## Task 5: HTTP — PATCH /accounts/{id} + Asset Meta in Responses

**Files:**
- Modify: `backend/internal/transport/http/accounting_accounts.go`
- Modify: `backend/internal/transport/http/accounting_accounts_test.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: `AccountService.UpdateAccount`, `UpdateAccountCmd`, `AssetMetaCmd` from Task 4
- Produces:
  - `PATCH /accounts/{id}` → 204 No Content
  - `GET /accounts` response rows include `asset_meta` object (null if not set)
  - `POST /accounts` accepts `opening_balance` and `asset_meta`

- [ ] **Step 1: Write failing tests for PATCH**

Add to `backend/internal/transport/http/accounting_accounts_test.go`:

```go
func TestAccountsHandler_Update_Success(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	// create account
	createBody, _ := json.Marshal(map[string]interface{}{
		"name": "Old", "type": "asset", "currency": "VND", "is_group": false, "sort_order": 0,
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(createBody))
	r = r.WithContext(setUserID(r.Context(), "user1"))
	h.Create(w, r)
	var created map[string]string
	json.NewDecoder(w.Body).Decode(&created)
	id := created["id"]

	// patch it
	patchBody, _ := json.Marshal(map[string]interface{}{
		"name": "New", "type": "expense", "sort_order": 2,
	})
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPatch, "/accounts/"+id, bytes.NewReader(patchBody))
	r2 = r2.WithContext(setUserID(r2.Context(), "user1"))
	r2 = setChiURLParam(r2, "id", id)
	h.Update(w2, r2)
	if w2.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestAccountsHandler_Update_WrongUser(t *testing.T) {
	repo := newTestAccountRepo()
	svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
	h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

	createBody, _ := json.Marshal(map[string]interface{}{
		"name": "X", "type": "asset", "currency": "VND", "is_group": false, "sort_order": 0,
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(createBody))
	r = r.WithContext(setUserID(r.Context(), "user1"))
	h.Create(w, r)
	var created map[string]string
	json.NewDecoder(w.Body).Decode(&created)
	id := created["id"]

	patchBody, _ := json.Marshal(map[string]interface{}{"name": "Hacked", "type": "expense"})
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPatch, "/accounts/"+id, bytes.NewReader(patchBody))
	r2 = r2.WithContext(setUserID(r2.Context(), "user2"))
	r2 = setChiURLParam(r2, "id", id)
	h.Update(w2, r2)
	if w2.Code == http.StatusNoContent {
		t.Error("want non-204 for wrong user")
	}
}
```

Add helper `setChiURLParam` if not already present in the test file:

```go
import "github.com/go-chi/chi/v5"

func setChiURLParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
```

Also check for existing `setUserID` helper in the test file — if missing, add:

```go
func setUserID(ctx context.Context, userID string) context.Context {
	return middleware.WithUserID(ctx, userID)
}
```

(Check `middleware` package for `WithUserID` function name.)

- [ ] **Step 2: Run tests — expect FAIL**

```bash
cd backend && go test ./internal/transport/http/... -run "TestAccountsHandler_Update" -v
```

Expected: compilation error — `h.Update` not defined.

- [ ] **Step 3: Update accounting_accounts.go**

Full replacement of `backend/internal/transport/http/accounting_accounts.go`:

```go
package httphandler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
)

type AccountsHandler struct {
	svc     *accountingsvc.AccountService
	journal repository.JournalRepo
}

func NewAccountsHandler(svc *accountingsvc.AccountService, journal repository.JournalRepo) *AccountsHandler {
	return &AccountsHandler{svc: svc, journal: journal}
}

type assetMetaResponse struct {
	PurchaseValue    *string `json:"purchase_value,omitempty"`
	PurchasedAt      *string `json:"purchased_at,omitempty"`
	DepreciationRate *string `json:"depreciation_rate,omitempty"`
	Notes            string  `json:"notes,omitempty"`
}

func (h *AccountsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	accounts, err := h.svc.ListAccounts(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	entries, err := h.journal.FindByUser(r.Context(), userID, time.Time{}, time.Now())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var allLines []accounting.JournalLine
	for _, e := range entries {
		allLines = append(allLines, e.Lines()...)
	}

	leafBalance := map[string]string{}
	for _, a := range accounts {
		if !a.IsGroup() {
			m, err := a.Balance(allLines)
			if err == nil {
				leafBalance[string(a.ID())] = m.Amount.String()
			}
		}
	}

	children := map[string][]string{}
	for _, a := range accounts {
		if a.ParentID() != nil {
			pid := string(*a.ParentID())
			children[pid] = append(children[pid], string(a.ID()))
		}
	}

	var sumDescendants func(id string) float64
	sumDescendants = func(id string) float64 {
		if bal, ok := leafBalance[id]; ok {
			v, _ := decimal.NewFromString(bal)
			f, _ := v.Float64()
			return f
		}
		var total float64
		for _, cid := range children[id] {
			total += sumDescendants(cid)
		}
		return total
	}

	type row struct {
		ID        string             `json:"id"`
		ParentID  string             `json:"parent_id,omitempty"`
		Name      string             `json:"name"`
		Type      string             `json:"type"`
		Currency  string             `json:"currency"`
		IsGroup   bool               `json:"is_group"`
		SortOrder int                `json:"sort_order"`
		Balance   float64            `json:"balance"`
		AssetMeta *assetMetaResponse `json:"asset_meta,omitempty"`
	}

	resp := make([]row, len(accounts))
	for i, a := range accounts {
		var pid string
		if a.ParentID() != nil {
			pid = string(*a.ParentID())
		}
		var bal float64
		if a.IsGroup() {
			bal = sumDescendants(string(a.ID()))
		} else {
			if s, ok := leafBalance[string(a.ID())]; ok {
				d, _ := decimal.NewFromString(s)
				bal, _ = d.Float64()
			}
		}
		var amr *assetMetaResponse
		if m := a.AssetMeta(); m != nil {
			amr = &assetMetaResponse{Notes: m.Notes}
			if m.PurchaseValue != nil {
				s := m.PurchaseValue.String()
				amr.PurchaseValue = &s
			}
			if m.PurchasedAt != nil {
				s := m.PurchasedAt.Format("2006-01-02")
				amr.PurchasedAt = &s
			}
			if m.DepreciationRate != nil {
				s := m.DepreciationRate.String()
				amr.DepreciationRate = &s
			}
		}
		resp[i] = row{
			ID:        string(a.ID()),
			ParentID:  pid,
			Name:      a.Name(),
			Type:      string(a.Type()),
			Currency:  a.Currency(),
			IsGroup:   a.IsGroup(),
			SortOrder: a.SortOrder(),
			Balance:   bal,
			AssetMeta: amr,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AccountsHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	var req struct {
		ParentID       *string  `json:"parent_id"`
		Name           string   `json:"name"`
		Type           string   `json:"type"`
		Currency       string   `json:"currency"`
		IsGroup        bool     `json:"is_group"`
		SortOrder      int      `json:"sort_order"`
		OpeningBalance *float64 `json:"opening_balance"`
		AssetMeta      *struct {
			PurchaseValue    *float64 `json:"purchase_value"`
			PurchasedAt      *string  `json:"purchased_at"`
			DepreciationRate *float64 `json:"depreciation_rate"`
			Notes            string   `json:"notes"`
		} `json:"asset_meta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Type == "" {
		http.Error(w, "name and type required", http.StatusBadRequest)
		return
	}
	if req.Currency == "" {
		req.Currency = "VND"
	}

	cmd := accountingsvc.OpenAccountCmd{
		UserID:    userID,
		ParentID:  req.ParentID,
		Name:      req.Name,
		Type:      accounting.AccountType(req.Type),
		Currency:  req.Currency,
		IsGroup:   req.IsGroup,
		SortOrder: req.SortOrder,
	}
	if req.OpeningBalance != nil && *req.OpeningBalance > 0 {
		ob := decimal.NewFromFloat(*req.OpeningBalance)
		cmd.OpeningBalance = &ob
	}
	if req.AssetMeta != nil {
		amc := &accountingsvc.AssetMetaCmd{Notes: req.AssetMeta.Notes}
		if req.AssetMeta.PurchaseValue != nil {
			pv := decimal.NewFromFloat(*req.AssetMeta.PurchaseValue)
			amc.PurchaseValue = &pv
		}
		if req.AssetMeta.PurchasedAt != nil {
			t, err := time.Parse("2006-01-02", *req.AssetMeta.PurchasedAt)
			if err == nil {
				amc.PurchasedAt = &t
			}
		}
		if req.AssetMeta.DepreciationRate != nil {
			dr := decimal.NewFromFloat(*req.AssetMeta.DepreciationRate)
			amc.DepreciationRate = &dr
		}
		cmd.AssetMeta = amc
	}

	id, err := h.svc.OpenAccount(r.Context(), cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": string(id)})
}

func (h *AccountsHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	id := chi.URLParam(r, "id")
	var req struct {
		Name      string   `json:"name"`
		Type      string   `json:"type"`
		ParentID  *string  `json:"parent_id"`
		SortOrder int      `json:"sort_order"`
		AssetMeta *struct {
			PurchaseValue    *float64 `json:"purchase_value"`
			PurchasedAt      *string  `json:"purchased_at"`
			DepreciationRate *float64 `json:"depreciation_rate"`
			Notes            string   `json:"notes"`
		} `json:"asset_meta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Type == "" {
		http.Error(w, "name and type required", http.StatusBadRequest)
		return
	}

	cmd := accountingsvc.UpdateAccountCmd{
		ID:        id,
		UserID:    userID,
		Name:      req.Name,
		Type:      accounting.AccountType(req.Type),
		ParentID:  req.ParentID,
		SortOrder: req.SortOrder,
	}
	if req.AssetMeta != nil {
		amc := &accountingsvc.AssetMetaCmd{Notes: req.AssetMeta.Notes}
		if req.AssetMeta.PurchaseValue != nil {
			pv := decimal.NewFromFloat(*req.AssetMeta.PurchaseValue)
			amc.PurchaseValue = &pv
		}
		if req.AssetMeta.PurchasedAt != nil {
			t, err := time.Parse("2006-01-02", *req.AssetMeta.PurchasedAt)
			if err == nil {
				amc.PurchasedAt = &t
			}
		}
		if req.AssetMeta.DepreciationRate != nil {
			dr := decimal.NewFromFloat(*req.AssetMeta.DepreciationRate)
			amc.DepreciationRate = &dr
		}
		cmd.AssetMeta = amc
	}

	if err := h.svc.UpdateAccount(r.Context(), cmd); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Register route in main.go**

In `backend/cmd/server/main.go`, find the line:
```go
r.Post("/accounts",          accountsHandler.Create)
```
Add after it:
```go
r.Patch("/accounts/{id}",    accountsHandler.Update)
```

- [ ] **Step 5: Run tests — expect PASS**

```bash
cd backend && go test ./internal/transport/http/... -v
```

Expected: all PASS including new Update tests. Check coverage ≥80% per file.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/transport/http/accounting_accounts.go \
        backend/internal/transport/http/accounting_accounts_test.go \
        backend/cmd/server/main.go
git commit -m "feat(accounting): PATCH /accounts/{id} + asset_meta in responses"
```

---

## Task 6: Net Income YTD

**Files:**
- Modify: `backend/internal/service/accounting/networth_query.go`
- Modify: `backend/internal/transport/http/accounting_journal.go`

**Interfaces:**
- Consumes: existing `NetWorthQuery`, journal lines
- Produces:
  - `NetWorthQuery.Current` returns `(nwMoney accounting.Money, netIncomeYTD accounting.Money, err error)`
  - `GET /journal/networth` response: `{"net_worth": "...", "currency": "...", "net_income_ytd": "..."}`

- [ ] **Step 1: Update networth_query.go**

Full replacement of `backend/internal/service/accounting/networth_query.go`:

```go
package accountingsvc

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type NetWorthQuery struct {
	accounts repository.AccountRepo
	journal  repository.JournalRepo
}

func NewNetWorthQuery(accounts repository.AccountRepo, journal repository.JournalRepo) *NetWorthQuery {
	return &NetWorthQuery{accounts: accounts, journal: journal}
}

type NetWorthResult struct {
	NetWorth     accounting.Money
	NetIncomeYTD accounting.Money
}

func (q *NetWorthQuery) Current(ctx context.Context, userID string) (NetWorthResult, error) {
	accounts, err := q.accounts.FindByUser(ctx, userID)
	if err != nil {
		return NetWorthResult{}, err
	}
	allEntries, err := q.journal.FindByUser(ctx, userID, time.Time{}, time.Now())
	if err != nil {
		return NetWorthResult{}, err
	}
	var allLines []accounting.JournalLine
	for _, e := range allEntries {
		allLines = append(allLines, e.Lines()...)
	}

	nw, err := accounting.NetWorthService{}.Calculate(accounts, allLines)
	if err != nil {
		return NetWorthResult{}, err
	}

	// Net income YTD: income credits - expense debits since Jan 1
	ytdStart := time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	ytdEntries, err := q.journal.FindByUser(ctx, userID, ytdStart, time.Now())
	if err != nil {
		return NetWorthResult{}, err
	}

	// build account type index
	acctType := map[accounting.AccountID]accounting.AccountType{}
	for _, a := range accounts {
		acctType[a.ID()] = a.Type()
	}

	netIncome := accounting.ZeroMoney("VND")
	for _, e := range ytdEntries {
		for _, l := range e.Lines() {
			t := acctType[l.AccountID()]
			switch {
			case t == accounting.Income && l.Side() == accounting.Credit:
				netIncome, _ = netIncome.Add(accounting.Money{Amount: l.Money().Amount, Currency: l.Money().Currency})
			case t == accounting.Expense && l.Side() == accounting.Debit:
				netIncome = accounting.Money{Amount: netIncome.Amount.Sub(l.Money().Amount), Currency: netIncome.Currency}
			}
		}
	}

	return NetWorthResult{NetWorth: nw, NetIncomeYTD: netIncome}, nil
}
```

Note: `accounting.ZeroMoney` may not exist yet — check `money.go`. If it doesn't exist, add it:

```go
// In backend/internal/domain/accounting/money.go
func ZeroMoney(currency string) Money {
	return Money{Amount: zeroDecimal(), Currency: currency}
}
```

- [ ] **Step 2: Update NetWorth HTTP handler**

In `backend/internal/transport/http/accounting_journal.go`, find the `NetWorth` method and replace:

```go
func (h *JournalHandler) NetWorth(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	result, err := h.networth.Current(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"net_worth":      result.NetWorth.Amount,
		"currency":       result.NetWorth.Currency,
		"net_income_ytd": result.NetIncomeYTD.Amount,
	})
}
```

- [ ] **Step 3: Fix all callers of networth.Current**

Search for other callers:
```bash
grep -rn "networth.Current\|\.Current(" backend/ --include="*.go"
```

Update each to use `result.NetWorth` instead of the old single return value.

- [ ] **Step 4: Build**

```bash
cd backend && go build ./...
```

Expected: no errors

- [ ] **Step 5: Run all backend tests**

```bash
cd backend && go test ./internal/transport/http/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic && bash ../scripts/hooks/pre-commit
```

Expected: all PASS, ≥80% coverage per file

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/accounting/networth_query.go \
        backend/internal/transport/http/accounting_journal.go \
        backend/internal/domain/accounting/money.go
git commit -m "feat(accounting): net income YTD in networth response"
```

---

## Task 7: Frontend — Types + updateAccount API

**Files:**
- Modify: `frontend/src/api/types.ts`
- Modify: `frontend/src/api/endpoints.ts`

**Interfaces:**
- Produces:
  - `AssetMeta` interface
  - `Account` extended with `asset_meta: AssetMeta | null`
  - `UpdateAccountRequest` interface
  - `NetWorthResult` extended with `net_income_ytd: string`
  - `updateAccount(id: string, data: UpdateAccountRequest): Promise<void>`

- [ ] **Step 1: Update types.ts**

Add `AssetMeta` interface and update `Account` and `NetWorthResult`. Find and replace these sections in `frontend/src/api/types.ts`:

Replace the existing `Account` interface:
```ts
export interface AssetMeta {
  purchase_value: string | null
  purchased_at: string | null
  depreciation_rate: string | null
  notes: string
}

export interface Account {
  id: string
  user_id: string
  parent_id: string | null
  name: string
  type: 'asset' | 'liability' | 'equity' | 'income' | 'expense'
  currency: string
  is_group: boolean
  archived: boolean
  sort_order: number
  balance: number
  asset_meta: AssetMeta | null
}
```

Add `UpdateAccountRequest` after `CreateAccountRequest`:
```ts
export interface UpdateAccountRequest {
  name: string
  type: 'asset' | 'liability' | 'equity' | 'income' | 'expense'
  parent_id?: string | null
  sort_order: number
  asset_meta?: {
    purchase_value?: number | null
    purchased_at?: string | null
    depreciation_rate?: number | null
    notes?: string
  } | null
}
```

Replace `NetWorthResult`:
```ts
export interface NetWorthResult {
  net_worth: string
  currency: string
  net_income_ytd: string
}
```

- [ ] **Step 2: Add updateAccount to endpoints.ts**

Find the `getAccounts` / `createAccount` block and add after `createAccount`:

```ts
export const updateAccount = (id: string, data: UpdateAccountRequest) =>
  apiClient.patch(`/accounts/${id}`, data)
```

Also add `UpdateAccountRequest` to the import from `./types` at the top of `endpoints.ts`.

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd frontend && npm run build 2>&1 | head -30
```

Expected: no type errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/types.ts frontend/src/api/endpoints.ts
git commit -m "feat(accounting): frontend types for asset meta, update account, net income YTD"
```

---

## Task 8: Frontend — Edit Modal + Opening Balance Field

**Files:**
- Modify: `frontend/src/pages/AccountingPage.tsx`

**Interfaces:**
- Consumes: `updateAccount`, `UpdateAccountRequest`, `AssetMeta` from Task 7
- Produces:
  - Edit button (`EditOutlined`) in accounts table action column
  - Edit modal pre-filled with current account values
  - "Opening Balance (VND)" optional field in Create modal (non-group accounts only)
  - "Asset Details" collapsible section in Create/Edit modal (shown when type=asset and is_group=false)

- [ ] **Step 1: Update imports in AccountingPage.tsx**

Add to existing antd import: `Collapse`
Add to existing antd/icons import: `EditOutlined`
Add to API imports: `updateAccount`
Add to API types import: `UpdateAccountRequest`

- [ ] **Step 2: Add edit state and mutation to AccountsTab**

Inside `AccountsTab`, add:

```tsx
const [editOpen, setEditOpen] = useState(false)
const [editTarget, setEditTarget] = useState<Account | null>(null)
const [editForm] = Form.useForm()

const editMutation = useMutation({
  mutationFn: ({ id, data }: { id: string; data: UpdateAccountRequest }) => updateAccount(id, data),
  onSuccess: () => {
    qc.invalidateQueries({ queryKey: ['accounts'] })
    setEditOpen(false)
    setEditTarget(null)
    editForm.resetFields()
  },
})

const openEdit = (account: Account) => {
  setEditTarget(account)
  editForm.setFieldsValue({
    name: account.name,
    type: account.type,
    parent_id: account.parent_id,
    sort_order: account.sort_order,
    is_group: account.is_group,
    asset_meta_purchase_value: account.asset_meta?.purchase_value ? parseFloat(account.asset_meta.purchase_value) : undefined,
    asset_meta_purchased_at: account.asset_meta?.purchased_at ?? undefined,
    asset_meta_depreciation_rate: account.asset_meta?.depreciation_rate ? parseFloat(account.asset_meta.depreciation_rate) : undefined,
    asset_meta_notes: account.asset_meta?.notes ?? '',
  })
  setEditOpen(true)
}
```

- [ ] **Step 3: Add Edit column to accounts table**

In `columns`, add a new column after Balance:

```tsx
{
  title: '',
  width: 48,
  render: (_: unknown, row: AccountTreeNode) => (
    <Button
      type="text"
      size="small"
      icon={<EditOutlined />}
      onClick={() => openEdit(row)}
    />
  ),
},
```

- [ ] **Step 4: Add opening balance field to Create modal**

Inside the Create modal `Form`, add after the `sort_order` field and before the submit button, wrapped in a conditional render using `Form.Item shouldUpdate`:

```tsx
<Form.Item noStyle shouldUpdate={(prev, cur) => prev.is_group !== cur.is_group}>
  {({ getFieldValue }) =>
    !getFieldValue('is_group') && (
      <Form.Item name="opening_balance" label="Opening Balance (VND)">
        <InputNumber min={0} style={{ width: '100%' }} placeholder="0 — leave empty to skip" />
      </Form.Item>
    )
  }
</Form.Item>
```

Update `onFinish` handler to pass `opening_balance` in the create request. Modify the existing `createMutation.mutate(values)` call:

```tsx
onFinish={(values) => {
  const req: CreateAccountRequest & { opening_balance?: number } = {
    name: values.name,
    type: values.type,
    currency: values.currency ?? 'VND',
    is_group: values.is_group ?? false,
    sort_order: values.sort_order ?? 0,
    parent_id: values.parent_id ?? null,
    opening_balance: values.opening_balance ?? undefined,
  }
  createMutation.mutate(req as CreateAccountRequest)
}}
```

Update `CreateAccountRequest` usage: the API call needs `opening_balance` on the body — since `createAccount` in endpoints.ts takes `CreateAccountRequest`, cast to `any` or extend the type. Simplest: cast `req as any` in the mutate call.

- [ ] **Step 5: Add Asset Details section to Create modal**

Inside the Create modal form, add after the `opening_balance` field:

```tsx
<Form.Item noStyle shouldUpdate={(prev, cur) => prev.type !== cur.type || prev.is_group !== cur.is_group}>
  {({ getFieldValue }) =>
    getFieldValue('type') === 'asset' && !getFieldValue('is_group') && (
      <Collapse ghost size="small" style={{ marginBottom: 8 }}>
        <Collapse.Panel header="Asset Details (optional)" key="asset">
          <Form.Item name="asset_meta_purchase_value" label="Purchase Value (VND)">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="asset_meta_purchased_at" label="Purchase Date">
            <Input type="date" />
          </Form.Item>
          <Form.Item name="asset_meta_depreciation_rate" label="Annual Depreciation Rate (0–1)">
            <InputNumber min={0} max={1} step={0.01} style={{ width: '100%' }} placeholder="e.g. 0.15 for 15%" />
          </Form.Item>
          <Form.Item name="asset_meta_notes" label="Notes">
            <Input />
          </Form.Item>
        </Collapse.Panel>
      </Collapse>
    )
  }
</Form.Item>
```

- [ ] **Step 6: Add Edit modal**

After the closing `</Modal>` of the Create modal, add:

```tsx
<Modal
  title="Edit Account"
  open={editOpen}
  onCancel={() => { setEditOpen(false); setEditTarget(null); editForm.resetFields() }}
  footer={null}
>
  <Form
    form={editForm}
    layout="vertical"
    onFinish={(values) => {
      if (!editTarget) return
      const assetMeta = (values.type === 'asset' && !editTarget.is_group && (
        values.asset_meta_purchase_value || values.asset_meta_purchased_at || values.asset_meta_depreciation_rate
      )) ? {
        purchase_value: values.asset_meta_purchase_value ?? null,
        purchased_at: values.asset_meta_purchased_at ?? null,
        depreciation_rate: values.asset_meta_depreciation_rate ?? null,
        notes: values.asset_meta_notes ?? '',
      } : null
      editMutation.mutate({
        id: editTarget.id,
        data: {
          name: values.name,
          type: values.type,
          parent_id: values.parent_id ?? null,
          sort_order: values.sort_order ?? 0,
          asset_meta: assetMeta,
        },
      })
    }}
  >
    <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Required' }]}>
      <Input />
    </Form.Item>
    <Form.Item name="type" label="Type" rules={[{ required: true }]}>
      <Select options={['asset','liability','equity','income','expense'].map(t => ({ value: t, label: t }))} />
    </Form.Item>
    <Form.Item name="parent_id" label="Parent Group">
      <Select
        allowClear
        placeholder="None (root)"
        options={groupAccounts.filter(a => a.id !== editTarget?.id).map(a => ({ value: a.id, label: a.name }))}
      />
    </Form.Item>
    <Form.Item name="sort_order" label="Sort Order">
      <InputNumber min={0} style={{ width: '100%' }} />
    </Form.Item>
    <Form.Item noStyle shouldUpdate={(prev, cur) => prev.type !== cur.type}>
      {({ getFieldValue }) =>
        getFieldValue('type') === 'asset' && editTarget && !editTarget.is_group && (
          <Collapse ghost size="small" style={{ marginBottom: 8 }}>
            <Collapse.Panel header="Asset Details (optional)" key="asset">
              <Form.Item name="asset_meta_purchase_value" label="Purchase Value (VND)">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item name="asset_meta_purchased_at" label="Purchase Date">
                <Input type="date" />
              </Form.Item>
              <Form.Item name="asset_meta_depreciation_rate" label="Annual Depreciation Rate (0–1)">
                <InputNumber min={0} max={1} step={0.01} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item name="asset_meta_notes" label="Notes">
                <Input />
              </Form.Item>
            </Collapse.Panel>
          </Collapse>
        )
      }
    </Form.Item>
    <Form.Item>
      <Button type="primary" htmlType="submit" loading={editMutation.isPending} block>
        Save Changes
      </Button>
    </Form.Item>
  </Form>
</Modal>
```

- [ ] **Step 7: Lint + build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add frontend/src/pages/AccountingPage.tsx
git commit -m "feat(accounting): edit modal + opening balance + asset details in create/edit"
```

---

## Task 9: Frontend — Assets Tab + Net Income Card + WealthPage Deprecation

**Files:**
- Modify: `frontend/src/pages/AccountingPage.tsx`
- Modify: `frontend/src/pages/WealthPage.tsx`

**Interfaces:**
- Consumes: `Account.asset_meta`, `NetWorthResult.net_income_ytd` from Task 7
- Produces:
  - New "Assets" tab in `AccountingPage` — lists accounts where `asset_meta != null`
  - Net Income YTD card in Journal tab
  - Deprecation banner in WealthPage Assets tab

- [ ] **Step 1: Add Assets tab to AccountingPage**

Add a new `AssetsTab` component in `AccountingPage.tsx` before the `AccountingPage` export:

```tsx
function AssetsTab() {
  const { data: accounts = [], isLoading } = useQuery({
    queryKey: ['accounts'],
    queryFn: getAccounts,
  })

  const assetAccounts = accounts.filter(a => a.asset_meta !== null && !a.is_group)

  const columns: ColumnsType<Account> = [
    {
      title: 'Name', dataIndex: 'name',
      render: (name) => <span><FileOutlined style={{ marginRight: 6, color: '#8c8c8c' }} />{name}</span>,
    },
    { title: 'Purchase Value', dataIndex: ['asset_meta', 'purchase_value'], width: 160, align: 'right',
      render: (v: string | null) => v ? fmtVND(v) : '—' },
    { title: 'Purchased', dataIndex: ['asset_meta', 'purchased_at'], width: 110,
      render: (v: string | null) => v ?? '—' },
    { title: 'Depr. Rate', dataIndex: ['asset_meta', 'depreciation_rate'], width: 100,
      render: (v: string | null) => v ? `${(parseFloat(v) * 100).toFixed(0)}%/yr` : '—' },
    { title: 'Current Balance', dataIndex: 'balance', width: 160, align: 'right',
      render: (bal: number) => fmtVND(String(bal)) },
    { title: 'Notes', dataIndex: ['asset_meta', 'notes'],
      render: (v: string) => v || '—' },
  ]

  return (
    <Card size="small" title="Physical Assets">
      {isLoading ? <Spin /> : (
        <Table<Account>
          dataSource={assetAccounts}
          columns={columns}
          size="small"
          rowKey="id"
          pagination={false}
          scroll={{ x: true }}
          locale={{ emptyText: 'No physical assets tracked. Add an account with type "asset" and fill in Asset Details.' }}
        />
      )}
    </Card>
  )
}
```

- [ ] **Step 2: Add Assets tab to AccountingPage tabs**

In the `AccountingPage` component, update the `Tabs` items array:

```tsx
items={[
  { key: 'journal', label: 'Journal', children: <JournalTab /> },
  { key: 'accounts', label: 'Accounts', children: <AccountsTab /> },
  { key: 'assets', label: 'Assets', children: <AssetsTab /> },
]}
```

- [ ] **Step 3: Add Net Income YTD card to JournalTab**

In `JournalTab`, the existing `nw` query result is `NetWorthResult`. Update the net worth card section to also show `net_income_ytd`:

Find the existing net worth card:
```tsx
{nw && (
  <Card size="small" style={{ marginBottom: 12 }}>
    <div style={{ fontSize: 12, color: '#999' }}>Live Net Worth</div>
    <div style={{ fontSize: 28, fontWeight: 700, color: '#1677ff' }}>
      {fmtVND(nw.net_worth)}
    </div>
  </Card>
)}
```

Replace with:
```tsx
{nw && (
  <Row gutter={12} style={{ marginBottom: 12 }}>
    <Col xs={24} sm={12}>
      <Card size="small">
        <div style={{ fontSize: 12, color: '#999' }}>Live Net Worth</div>
        <div style={{ fontSize: 28, fontWeight: 700, color: '#1677ff' }}>
          {fmtVND(nw.net_worth)}
        </div>
      </Card>
    </Col>
    <Col xs={24} sm={12}>
      <Card size="small">
        <div style={{ fontSize: 12, color: '#999' }}>Net Income (YTD)</div>
        <div style={{
          fontSize: 28, fontWeight: 700,
          color: parseFloat(nw.net_income_ytd) >= 0 ? '#52c41a' : '#ff4d4f',
        }}>
          {parseFloat(nw.net_income_ytd) < 0 ? '-' : ''}{fmtVND(nw.net_income_ytd)}
        </div>
      </Card>
    </Col>
  </Row>
)}
```

Add `Row` and `Col` to the antd imports in `AccountingPage.tsx` if not already present.

- [ ] **Step 4: Add deprecation banner to WealthPage Assets tab**

In `frontend/src/pages/WealthPage.tsx`, find the Assets tab render (search for `getAssets` usage). At the top of the assets tab content, add:

```tsx
import { Alert } from 'antd'
// ...
<Alert
  type="info"
  showIcon
  message="Assets are now tracked in Accounting → Assets tab."
  style={{ marginBottom: 12 }}
/>
```

- [ ] **Step 5: Lint + build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add frontend/src/pages/AccountingPage.tsx frontend/src/pages/WealthPage.tsx
git commit -m "feat(accounting): Assets tab, net income YTD card, WealthPage deprecation banner"
```

---

## Task 10: Pre-PR Verification + PR

**Files:** none new

- [ ] **Step 1: Run backend tests with coverage**

```bash
cd backend && go test ./internal/transport/http/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic
bash scripts/hooks/pre-commit
```

Expected: all files ≥80% coverage

- [ ] **Step 2: Run frontend lint + build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: clean

- [ ] **Step 3: Run integration smoke tests**

```bash
bash scripts/integration-test.sh
```

Expected: PASS

- [ ] **Step 4: Create PR**

```bash
git push -u origin feat/accounting-edit-assets
gh pr create --title "feat(accounting): account edit, opening balance, physical assets, net income YTD" --body "$(cat <<'EOF'
## Summary
- `PATCH /accounts/{id}` — edit name, type, parent, sort order, asset metadata
- Opening balance field on account create — auto-posts journal entry against Opening Balance equity account
- Physical asset metadata (purchase value, date, depreciation rate) on accounting accounts, replacing `wealth.Asset`
- Net income YTD card in Journal tab
- New Assets tab in AccountingPage listing accounts with asset metadata
- Deprecation banner on WealthPage Assets tab

## Test plan
- [ ] Create account with opening balance — verify journal entry posted
- [ ] Edit account name/type/parent — verify changes persist
- [ ] Create asset account with asset details — verify appears in Assets tab
- [ ] Check Net Income YTD card shows green/red based on income vs expenses
- [ ] WealthPage Assets tab shows deprecation banner

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
gh pr merge --auto --squash
```
