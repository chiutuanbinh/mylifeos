# Double-Entry Accounting — Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a full double-entry accounting domain (accounts, journal entries, net worth) as a clean DDD backend, following existing project layer conventions.

**Architecture:** Domain layer holds pure aggregates (Account, JournalEntry) and a NetWorthService with no I/O. Application services orchestrate domain + repos. Postgres repos implement port interfaces. HTTP handlers are thin translators. Old `transactions` table is untouched — new system coexists.

**Tech Stack:** Go 1.25, pgx/v5, chi router, shopspring/decimal (new dep), google/uuid (new dep)

## Global Constraints

- Module path: `github.com/chiutuanbinh/mylifeos/backend`
- Layer conventions: `domain/<pkg>/`, `port/repository/`, `service/<pkg>/`, `infra/postgres/`, `transport/http/`
- No business logic in transport or infra layers
- All money: `github.com/shopspring/decimal` — never `float64`
- Repository interfaces defined in `port/repository/`, implemented in `infra/postgres/`
- Coverage gate: ≥80% per file in `transport/http/` and `middleware/` — new handler files must have tests
- Both migration files must be kept in sync: `backend/internal/migrate/` and `supabase/migrations/`

---

## File Map

**New files — domain:**
- `backend/internal/domain/accounting/money.go` — Money value object
- `backend/internal/domain/accounting/account.go` — Account aggregate, AccountType, NormalBalance, Balance
- `backend/internal/domain/accounting/journal_entry.go` — JournalEntry aggregate, AddLine, Post, reconstitution
- `backend/internal/domain/accounting/events.go` — DomainEvent interface, EntryPosted
- `backend/internal/domain/accounting/networth_service.go` — NetWorthService.Calculate

**New files — ports:**
- `backend/internal/port/repository/accounting.go` — AccountRepo, JournalRepo interfaces
- `backend/internal/port/events/publisher.go` — EventPublisher interface

**New files — application services:**
- `backend/internal/service/accounting/commands.go` — RecordTransactionCmd, OpenAccountCmd, LineCmd
- `backend/internal/service/accounting/journal_service.go` — JournalService
- `backend/internal/service/accounting/account_service.go` — AccountService
- `backend/internal/service/accounting/networth_query.go` — NetWorthQuery

**New files — infra:**
- `backend/internal/infra/postgres/accounting_accounts.go` — pgAccountRepo
- `backend/internal/infra/postgres/accounting_journal.go` — pgJournalRepo
- `backend/internal/infra/events/publisher.go` — InProcessPublisher

**New files — transport:**
- `backend/internal/transport/http/accounting_accounts.go` — AccountsHandler (CRUD accounts)
- `backend/internal/transport/http/accounting_journal.go` — JournalHandler (record tx, net worth)

**New files — migrations:**
- `backend/internal/migrate/008_double_entry.sql`
- `supabase/migrations/20260617000001_double_entry.sql` (identical content)

**Modified files:**
- `backend/go.mod` + `backend/go.sum` — add shopspring/decimal, google/uuid
- `backend/cmd/server/main.go` — wire new repos/services/handlers

**Test files:**
- `backend/internal/domain/accounting/money_test.go`
- `backend/internal/domain/accounting/account_test.go`
- `backend/internal/domain/accounting/journal_entry_test.go`
- `backend/internal/domain/accounting/networth_service_test.go`
- `backend/internal/service/accounting/journal_service_test.go`
- `backend/internal/service/accounting/account_service_test.go`
- `backend/internal/transport/http/accounting_accounts_test.go`
- `backend/internal/transport/http/accounting_journal_test.go`

---

## Task 1: Add dependencies + Money value object

**Files:**
- Modify: `backend/go.mod`, `backend/go.sum`
- Create: `backend/internal/domain/accounting/money.go`
- Create: `backend/internal/domain/accounting/money_test.go`

**Interfaces:**
- Produces: `Money{Amount decimal.Decimal, Currency string}`, `NewMoney(amount decimal.Decimal, currency string) (Money, error)`

- [ ] **Step 1: Add dependencies**

```bash
cd backend
go get github.com/shopspring/decimal@v1.4.0
go get github.com/google/uuid@v1.6.0
```

Expected: `go.mod` and `go.sum` updated, no errors.

- [ ] **Step 2: Write failing tests**

Create `backend/internal/domain/accounting/money_test.go`:

```go
package accounting_test

import (
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/shopspring/decimal"
)

func TestNewMoney_RejectsNegative(t *testing.T) {
	_, err := accounting.NewMoney(decimal.NewFromInt(-1), "VND")
	if err == nil {
		t.Fatal("want error for negative amount")
	}
}

func TestNewMoney_AcceptsZero(t *testing.T) {
	m, err := accounting.NewMoney(decimal.Zero, "VND")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Amount.IsZero() {
		t.Error("want zero amount")
	}
}

func TestMoney_Add_SameCurrency(t *testing.T) {
	a, _ := accounting.NewMoney(decimal.NewFromInt(100), "VND")
	b, _ := accounting.NewMoney(decimal.NewFromInt(50), "VND")
	got, err := a.Add(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Amount.Equal(decimal.NewFromInt(150)) {
		t.Errorf("want 150, got %s", got.Amount)
	}
}

func TestMoney_Add_CurrencyMismatch(t *testing.T) {
	a, _ := accounting.NewMoney(decimal.NewFromInt(100), "VND")
	b, _ := accounting.NewMoney(decimal.NewFromInt(50), "USD")
	_, err := a.Add(b)
	if err == nil {
		t.Fatal("want error for currency mismatch")
	}
}
```

- [ ] **Step 3: Run tests — expect FAIL**

```bash
cd backend && go test ./internal/domain/accounting/... -v
```

Expected: compile error — package does not exist yet.

- [ ] **Step 4: Implement Money**

Create `backend/internal/domain/accounting/money.go`:

```go
package accounting

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

type Money struct {
	Amount   decimal.Decimal
	Currency string
}

func NewMoney(amount decimal.Decimal, currency string) (Money, error) {
	if amount.IsNegative() {
		return Money{}, errors.New("money amount cannot be negative")
	}
	return Money{Amount: amount, Currency: currency}, nil
}

func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: %s vs %s", m.Currency, other.Currency)
	}
	return Money{Amount: m.Amount.Add(other.Amount), Currency: m.Currency}, nil
}
```

- [ ] **Step 5: Run tests — expect PASS**

```bash
cd backend && go test ./internal/domain/accounting/... -v
```

Expected: all 4 tests PASS.

- [ ] **Step 6: Commit**

```bash
cd backend
git add go.mod go.sum internal/domain/accounting/
git commit -m "feat(accounting): add decimal/uuid deps and Money value object"
```

---

## Task 2: Account aggregate

**Files:**
- Create: `backend/internal/domain/accounting/account.go`
- Create: `backend/internal/domain/accounting/account_test.go`

**Interfaces:**
- Consumes: `Money` from Task 1
- Produces:
  - `AccountID string`
  - `AccountType` (asset, liability, equity, income, expense)
  - `Side` (debit, credit)
  - `Account` struct with unexported fields
  - `NewAccount(userID, parentID *string, name string, acctType AccountType, currency string, isGroup bool, sortOrder int) *Account`
  - `ReconstitueAccount(id, userID string, parentID *string, name string, acctType AccountType, currency string, isGroup, archived bool, sortOrder int) *Account`
  - `(a *Account) ID() AccountID`
  - `(a *Account) UserID() string`
  - `(a *Account) ParentID() *AccountID`
  - `(a *Account) Name() string`
  - `(a *Account) Type() AccountType`
  - `(a *Account) Currency() string`
  - `(a *Account) IsGroup() bool`
  - `(a *Account) Archived() bool`
  - `(a *Account) SortOrder() int`
  - `(a *Account) NormalBalance() Side`
  - `(a *Account) Balance(lines []JournalLine) Money`

- [ ] **Step 1: Write failing tests**

Create `backend/internal/domain/accounting/account_test.go`:

```go
package accounting_test

import (
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/shopspring/decimal"
)

func TestAccount_NormalBalance(t *testing.T) {
	cases := []struct {
		t    accounting.AccountType
		want accounting.Side
	}{
		{accounting.Asset, accounting.Debit},
		{accounting.Expense, accounting.Debit},
		{accounting.Liability, accounting.Credit},
		{accounting.Equity, accounting.Credit},
		{accounting.Income, accounting.Credit},
	}
	for _, c := range cases {
		a := accounting.NewAccount("user1", nil, "Test", c.t, "VND", false, 0)
		if a.NormalBalance() != c.want {
			t.Errorf("type %s: want %s, got %s", c.t, c.want, a.NormalBalance())
		}
	}
}

func TestAccount_Balance_Asset(t *testing.T) {
	a := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
	lines := []accounting.JournalLine{
		accounting.TestJournalLine(string(a.ID()), decimal.NewFromInt(100), accounting.Debit),
		accounting.TestJournalLine(string(a.ID()), decimal.NewFromInt(30), accounting.Credit),
	}
	bal := a.Balance(lines)
	if !bal.Amount.Equal(decimal.NewFromInt(70)) {
		t.Errorf("want 70, got %s", bal.Amount)
	}
}

func TestAccount_Balance_Liability(t *testing.T) {
	a := accounting.NewAccount("user1", nil, "Visa", accounting.Liability, "VND", false, 0)
	lines := []accounting.JournalLine{
		accounting.TestJournalLine(string(a.ID()), decimal.NewFromInt(200), accounting.Credit),
		accounting.TestJournalLine(string(a.ID()), decimal.NewFromInt(50), accounting.Debit),
	}
	bal := a.Balance(lines)
	if !bal.Amount.Equal(decimal.NewFromInt(150)) {
		t.Errorf("want 150, got %s", bal.Amount)
	}
}

func TestAccount_Balance_IgnoresOtherAccounts(t *testing.T) {
	a := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
	lines := []accounting.JournalLine{
		accounting.TestJournalLine("other-account-id", decimal.NewFromInt(999), accounting.Debit),
	}
	bal := a.Balance(lines)
	if !bal.Amount.IsZero() {
		t.Errorf("want 0, got %s", bal.Amount)
	}
}
```

- [ ] **Step 2: Run — expect FAIL (compile)**

```bash
cd backend && go test ./internal/domain/accounting/... -v
```

Expected: compile error.

- [ ] **Step 3: Implement Account**

Create `backend/internal/domain/accounting/account.go`:

```go
package accounting

import "github.com/google/uuid"

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
}

func NewAccount(userID string, parentID *string, name string, acctType AccountType, currency string, isGroup bool, sortOrder int) *Account {
	var pid *AccountID
	if parentID != nil {
		p := AccountID(*parentID)
		pid = &p
	}
	return &Account{
		id:        AccountID(uuid.New().String()),
		userID:    userID,
		parentID:  pid,
		name:      name,
		acctType:  acctType,
		currency:  currency,
		isGroup:   isGroup,
		sortOrder: sortOrder,
	}
}

func ReconstitueAccount(id, userID string, parentID *string, name string, acctType AccountType, currency string, isGroup, archived bool, sortOrder int) *Account {
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

func (a *Account) NormalBalance() Side {
	switch a.acctType {
	case Asset, Expense:
		return Debit
	default:
		return Credit
	}
}

func (a *Account) Balance(lines []JournalLine) Money {
	normal := a.NormalBalance()
	total := zeroDecimal()
	for _, l := range lines {
		if l.AccountID() != a.id {
			continue
		}
		if l.Side() == normal {
			total = total.Add(l.Money().Amount)
		} else {
			total = total.Sub(l.Money().Amount)
		}
	}
	return Money{Amount: total, Currency: a.currency}
}

// TestJournalLine constructs a minimal JournalLine for use in tests only.
func TestJournalLine(accountID string, amount interface{ String() string }, side Side) JournalLine {
	panic("implemented in journal_entry.go")
}
```

Wait — `TestJournalLine` and `JournalLine` are forward references. To avoid circular dependency between account_test and journal_entry, define `JournalLine` as a struct in a shared file and expose a test helper. See Task 3 for the full implementation — the account_test file above uses `accounting.TestJournalLine` and `accounting.JournalLine` which are defined in Task 3. **Write the test file in Task 3, not Task 2.**

Instead, write a simpler account_test that only tests `NormalBalance` in Task 2, and add the `Balance` tests in Task 3 once `JournalLine` exists.

- [ ] **Step 3 (revised): Implement Account (without Balance test)**

Create `backend/internal/domain/accounting/account.go` as above but **omit** `TestJournalLine` stub. Keep the `Balance` method — it compiles as long as `JournalLine` is defined in the same package.

The `account_test.go` for this task should only test `NormalBalance` (no `Balance` calls yet):

Replace `backend/internal/domain/accounting/account_test.go` with:

```go
package accounting_test

import (
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
)

func TestAccount_NormalBalance(t *testing.T) {
	cases := []struct {
		t    accounting.AccountType
		want accounting.Side
	}{
		{accounting.Asset, accounting.Debit},
		{accounting.Expense, accounting.Debit},
		{accounting.Liability, accounting.Credit},
		{accounting.Equity, accounting.Credit},
		{accounting.Income, accounting.Credit},
	}
	for _, c := range cases {
		a := accounting.NewAccount("user1", nil, "Test", c.t, "VND", false, 0)
		if a.NormalBalance() != c.want {
			t.Errorf("type %s: want %s, got %s", c.t, c.want, a.NormalBalance())
		}
	}
}

func TestNewAccount_IDNotEmpty(t *testing.T) {
	a := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
	if string(a.ID()) == "" {
		t.Error("want non-empty ID")
	}
}

func TestNewAccount_ParentID_Nil(t *testing.T) {
	a := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
	if a.ParentID() != nil {
		t.Error("want nil ParentID")
	}
}

func TestNewAccount_ParentID_Set(t *testing.T) {
	pid := "parent-123"
	a := accounting.NewAccount("user1", &pid, "Cash", accounting.Asset, "VND", false, 0)
	if a.ParentID() == nil || string(*a.ParentID()) != pid {
		t.Error("want ParentID set")
	}
}
```

Create `backend/internal/domain/accounting/account.go`:

```go
package accounting

import "github.com/google/uuid"

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
}

func NewAccount(userID string, parentID *string, name string, acctType AccountType, currency string, isGroup bool, sortOrder int) *Account {
	var pid *AccountID
	if parentID != nil {
		p := AccountID(*parentID)
		pid = &p
	}
	return &Account{
		id:        AccountID(uuid.New().String()),
		userID:    userID,
		parentID:  pid,
		name:      name,
		acctType:  acctType,
		currency:  currency,
		isGroup:   isGroup,
		sortOrder: sortOrder,
	}
}

func ReconstitueAccount(id, userID string, parentID *string, name string, acctType AccountType, currency string, isGroup, archived bool, sortOrder int) *Account {
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

func (a *Account) ID() AccountID       { return a.id }
func (a *Account) UserID() string       { return a.userID }
func (a *Account) ParentID() *AccountID { return a.parentID }
func (a *Account) Name() string         { return a.name }
func (a *Account) Type() AccountType    { return a.acctType }
func (a *Account) Currency() string     { return a.currency }
func (a *Account) IsGroup() bool        { return a.isGroup }
func (a *Account) Archived() bool       { return a.archived }
func (a *Account) SortOrder() int       { return a.sortOrder }

func (a *Account) NormalBalance() Side {
	switch a.acctType {
	case Asset, Expense:
		return Debit
	default:
		return Credit
	}
}

func (a *Account) Balance(lines []JournalLine) Money {
	normal := a.NormalBalance()
	total := zeroDecimal()
	for _, l := range lines {
		if l.AccountID() != a.id {
			continue
		}
		if l.Side() == normal {
			total = total.Add(l.Money().Amount)
		} else {
			total = total.Sub(l.Money().Amount)
		}
	}
	return Money{Amount: total, Currency: a.currency}
}
```

`zeroDecimal()` is a helper — add to `money.go`:

```go
// add to money.go
import "github.com/shopspring/decimal"

func zeroDecimal() decimal.Decimal { return decimal.Zero }
```

- [ ] **Step 4: Run tests — expect FAIL (JournalLine undefined)**

```bash
cd backend && go test ./internal/domain/accounting/... -v
```

Expected: compile error — `JournalLine` undefined. This is expected — defined in Task 3.

- [ ] **Step 5: Stub JournalLine to unblock compilation**

Add to `backend/internal/domain/accounting/account.go` at the bottom (will be replaced in Task 3):

```go
// JournalLine is defined in journal_entry.go — stub here so account.go compiles during incremental build.
// Remove this comment once journal_entry.go is added to the package.
```

Instead: create a minimal stub file so `account.go` compiles now:

Create `backend/internal/domain/accounting/journal_line_stub.go`:

```go
package accounting

// JournalLine stub — replaced by full implementation in Task 3.
// Delete this file when journal_entry.go is created.
type JournalLine struct {
	accountID AccountID
	money     Money
	side      Side
}

func (l JournalLine) AccountID() AccountID { return l.accountID }
func (l JournalLine) Money() Money         { return l.money }
func (l JournalLine) Side() Side           { return l.side }
```

- [ ] **Step 6: Run tests — expect PASS**

```bash
cd backend && go test ./internal/domain/accounting/... -v
```

Expected: 4 tests PASS.

- [ ] **Step 7: Commit**

```bash
cd backend
git add internal/domain/accounting/
git commit -m "feat(accounting): Account aggregate with NormalBalance"
```

---

## Task 3: JournalEntry aggregate + domain events

**Files:**
- Delete: `backend/internal/domain/accounting/journal_line_stub.go`
- Create: `backend/internal/domain/accounting/journal_entry.go`
- Create: `backend/internal/domain/accounting/events.go`
- Create: `backend/internal/domain/accounting/journal_entry_test.go`

**Interfaces:**
- Consumes: `Money`, `AccountID`, `Side` from Tasks 1–2
- Produces:
  - `EntryID string`
  - `JournalLine` struct with accessors: `ID() string`, `AccountID() AccountID`, `Money() Money`, `Side() Side`
  - `JournalEntry` struct with:
    - `NewJournalEntry(userID string, date time.Time, description string) *JournalEntry`
    - `ReconstitueEntry(id, userID string, date time.Time, desc, memo string) *JournalEntry`
    - `(e *JournalEntry) AddLine(accountID AccountID, money Money, side Side) error`
    - `(e *JournalEntry) SetMemo(memo string)`
    - `(e *JournalEntry) Post() error`
    - `(e *JournalEntry) ReconstituteLine(id string, acctID AccountID, money Money, side Side)`
    - `(e *JournalEntry) ID() EntryID`
    - `(e *JournalEntry) UserID() string`
    - `(e *JournalEntry) Date() time.Time`
    - `(e *JournalEntry) Description() string`
    - `(e *JournalEntry) Memo() string`
    - `(e *JournalEntry) Lines() []JournalLine`
    - `(e *JournalEntry) Events() []DomainEvent`
  - `DomainEvent` interface
  - `EntryPosted{EntryID EntryID, UserID string, Date time.Time}`

- [ ] **Step 1: Write failing tests**

Create `backend/internal/domain/accounting/journal_entry_test.go`:

```go
package accounting_test

import (
	"testing"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/shopspring/decimal"
)

func mustMoney(amount int64, currency string) accounting.Money {
	m, _ := accounting.NewMoney(decimal.NewFromInt(amount), currency)
	return m
}

func TestJournalEntry_Post_Balanced(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "Coffee")
	e.AddLine("account-expense", mustMoney(150000, "VND"), accounting.Debit)
	e.AddLine("account-visa", mustMoney(150000, "VND"), accounting.Credit)
	if err := e.Post(); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
}

func TestJournalEntry_Post_Unbalanced(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "Bad entry")
	e.AddLine("account-a", mustMoney(100, "VND"), accounting.Debit)
	e.AddLine("account-b", mustMoney(50, "VND"), accounting.Credit)
	if err := e.Post(); err == nil {
		t.Fatal("want error for unbalanced entry")
	}
}

func TestJournalEntry_Post_TooFewLines(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "One line")
	e.AddLine("account-a", mustMoney(100, "VND"), accounting.Debit)
	if err := e.Post(); err == nil {
		t.Fatal("want error for single line")
	}
}

func TestJournalEntry_Post_EmitsEvent(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "Coffee")
	e.AddLine("account-expense", mustMoney(100, "VND"), accounting.Debit)
	e.AddLine("account-visa", mustMoney(100, "VND"), accounting.Credit)
	e.Post()
	evs := e.Events()
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %d", len(evs))
	}
	ep, ok := evs[0].(accounting.EntryPosted)
	if !ok {
		t.Fatal("want EntryPosted event")
	}
	if ep.UserID != "user1" {
		t.Errorf("want userID user1, got %s", ep.UserID)
	}
}

func TestJournalEntry_AddLine_ZeroAmountRejected(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "Zero")
	m, _ := accounting.NewMoney(decimal.Zero, "VND")
	if err := e.AddLine("account-a", m, accounting.Debit); err == nil {
		t.Fatal("want error for zero amount line")
	}
}

func TestJournalEntry_Lines_DefensiveCopy(t *testing.T) {
	e := accounting.NewJournalEntry("user1", time.Now(), "Test")
	e.AddLine("a", mustMoney(100, "VND"), accounting.Debit)
	e.AddLine("b", mustMoney(100, "VND"), accounting.Credit)
	lines := e.Lines()
	lines[0] = accounting.JournalLine{}
	if e.Lines()[0].Money().Amount.IsZero() {
		t.Error("Lines() should return a defensive copy")
	}
}

func TestAccount_Balance_WithRealLines(t *testing.T) {
	a := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
	entry := accounting.NewJournalEntry("user1", time.Now(), "Test")
	entry.AddLine(a.ID(), mustMoney(500, "VND"), accounting.Debit)
	entry.AddLine("other", mustMoney(500, "VND"), accounting.Credit)
	entry.Post()

	bal := a.Balance(entry.Lines())
	if !bal.Amount.Equal(decimal.NewFromInt(500)) {
		t.Errorf("want 500, got %s", bal.Amount)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd backend && go test ./internal/domain/accounting/... -v
```

Expected: compile error — `JournalEntry`, `EntryPosted` undefined.

- [ ] **Step 3: Implement events**

Create `backend/internal/domain/accounting/events.go`:

```go
package accounting

import "time"

type DomainEvent interface{ domainEvent() }

type EntryPosted struct {
	EntryID EntryID
	UserID  string
	Date    time.Time
}

func (EntryPosted) domainEvent() {}
```

- [ ] **Step 4: Implement JournalEntry**

Delete `backend/internal/domain/accounting/journal_line_stub.go`.

Create `backend/internal/domain/accounting/journal_entry.go`:

```go
package accounting

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type EntryID string

type JournalLine struct {
	id        string
	accountID AccountID
	money     Money
	side      Side
}

func (l JournalLine) ID() string          { return l.id }
func (l JournalLine) AccountID() AccountID { return l.accountID }
func (l JournalLine) Money() Money         { return l.money }
func (l JournalLine) Side() Side           { return l.side }

type JournalEntry struct {
	id          EntryID
	userID      string
	date        time.Time
	description string
	memo        string
	lines       []JournalLine
	events      []DomainEvent
}

func NewJournalEntry(userID string, date time.Time, description string) *JournalEntry {
	return &JournalEntry{
		id:          EntryID(uuid.New().String()),
		userID:      userID,
		date:        date,
		description: description,
	}
}

func ReconstitueEntry(id, userID string, date time.Time, desc, memo string) *JournalEntry {
	return &JournalEntry{
		id:          EntryID(id),
		userID:      userID,
		date:        date,
		description: desc,
		memo:        memo,
	}
}

func (e *JournalEntry) SetMemo(memo string) { e.memo = memo }

func (e *JournalEntry) AddLine(accountID AccountID, money Money, side Side) error {
	if money.Amount.IsZero() {
		return errors.New("line amount must be non-zero")
	}
	e.lines = append(e.lines, JournalLine{
		id:        uuid.New().String(),
		accountID: accountID,
		money:     money,
		side:      side,
	})
	return nil
}

func (e *JournalEntry) ReconstituteLine(id string, acctID AccountID, money Money, side Side) {
	e.lines = append(e.lines, JournalLine{id: id, accountID: acctID, money: money, side: side})
}

func (e *JournalEntry) Post() error {
	if len(e.lines) < 2 {
		return errors.New("entry requires at least 2 lines")
	}
	var debits, credits decimal.Decimal
	for _, l := range e.lines {
		if l.side == Debit {
			debits = debits.Add(l.money.Amount)
		} else {
			credits = credits.Add(l.money.Amount)
		}
	}
	if !debits.Equal(credits) {
		return fmt.Errorf("entry does not balance: debits %s ≠ credits %s", debits, credits)
	}
	e.events = append(e.events, EntryPosted{EntryID: e.id, UserID: e.userID, Date: e.date})
	return nil
}

func (e *JournalEntry) ID() EntryID          { return e.id }
func (e *JournalEntry) UserID() string        { return e.userID }
func (e *JournalEntry) Date() time.Time       { return e.date }
func (e *JournalEntry) Description() string   { return e.description }
func (e *JournalEntry) Memo() string          { return e.memo }
func (e *JournalEntry) Lines() []JournalLine  { return slices.Clone(e.lines) }
func (e *JournalEntry) Events() []DomainEvent { return slices.Clone(e.events) }
```

- [ ] **Step 5: Run tests — expect PASS**

```bash
cd backend && go test ./internal/domain/accounting/... -v
```

Expected: all tests PASS (NormalBalance + JournalEntry tests).

- [ ] **Step 6: Commit**

```bash
cd backend
git add internal/domain/accounting/
git commit -m "feat(accounting): JournalEntry aggregate with balance invariant and EntryPosted event"
```

---

## Task 4: NetWorthService

**Files:**
- Create: `backend/internal/domain/accounting/networth_service.go`
- Create: `backend/internal/domain/accounting/networth_service_test.go`

**Interfaces:**
- Consumes: `Account`, `JournalLine`, `Money` from Tasks 1–3
- Produces: `NetWorthService` struct, `(NetWorthService) Calculate(accounts []*Account, lines []JournalLine) Money`

- [ ] **Step 1: Write failing test**

Create `backend/internal/domain/accounting/networth_service_test.go`:

```go
package accounting_test

import (
	"testing"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/shopspring/decimal"
)

func TestNetWorthService_Calculate(t *testing.T) {
	cash    := accounting.NewAccount("u1", nil, "Cash",    accounting.Asset,     "VND", false, 0)
	visa    := accounting.NewAccount("u1", nil, "Visa",    accounting.Liability, "VND", false, 0)
	salary  := accounting.NewAccount("u1", nil, "Salary",  accounting.Income,    "VND", false, 0)
	food    := accounting.NewAccount("u1", nil, "Food",    accounting.Expense,   "VND", false, 0)

	// Salary received: debit Cash 10M, credit Salary 10M
	e1 := accounting.NewJournalEntry("u1", time.Now(), "Salary")
	e1.AddLine(cash.ID(),   mustMoney(10_000_000, "VND"), accounting.Debit)
	e1.AddLine(salary.ID(), mustMoney(10_000_000, "VND"), accounting.Credit)
	e1.Post()

	// Buy with Visa: debit Food 150k, credit Visa 150k
	e2 := accounting.NewJournalEntry("u1", time.Now(), "Coffee")
	e2.AddLine(food.ID(), mustMoney(150_000, "VND"), accounting.Debit)
	e2.AddLine(visa.ID(), mustMoney(150_000, "VND"), accounting.Credit)
	e2.Post()

	var lines []accounting.JournalLine
	lines = append(lines, e1.Lines()...)
	lines = append(lines, e2.Lines()...)

	svc := accounting.NetWorthService{}
	nw := svc.Calculate([]*accounting.Account{cash, visa, salary, food}, lines)

	// Cash = 10M (asset), Visa = 150k (liability)
	// Net worth = 10M - 150k = 9,850,000
	want := decimal.NewFromInt(9_850_000)
	if !nw.Amount.Equal(want) {
		t.Errorf("want %s, got %s", want, nw.Amount)
	}
}

func TestNetWorthService_SkipsGroupAccounts(t *testing.T) {
	group := accounting.NewAccount("u1", nil, "Assets", accounting.Asset, "VND", true, 0)
	leaf  := accounting.NewAccount("u1", nil, "Cash",   accounting.Asset, "VND", false, 0)

	e := accounting.NewJournalEntry("u1", time.Now(), "Test")
	e.AddLine(leaf.ID(),  mustMoney(100, "VND"), accounting.Debit)
	e.AddLine(group.ID(), mustMoney(100, "VND"), accounting.Credit) // unusual but tests skip logic

	svc := accounting.NetWorthService{}
	nw := svc.Calculate([]*accounting.Account{group, leaf}, e.Lines())

	// group skipped, leaf = 100 asset
	if !nw.Amount.Equal(decimal.NewFromInt(100)) {
		t.Errorf("want 100, got %s", nw.Amount)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd backend && go test ./internal/domain/accounting/... -v
```

Expected: compile error — `NetWorthService` undefined.

- [ ] **Step 3: Implement NetWorthService**

Create `backend/internal/domain/accounting/networth_service.go`:

```go
package accounting

import "github.com/shopspring/decimal"

type NetWorthService struct{}

func (NetWorthService) Calculate(accounts []*Account, lines []JournalLine) Money {
	total := decimal.Zero
	for _, a := range accounts {
		if a.IsGroup() || a.Archived() {
			continue
		}
		bal := a.Balance(lines)
		switch a.Type() {
		case Asset:
			total = total.Add(bal.Amount)
		case Liability:
			total = total.Sub(bal.Amount)
		}
	}
	return Money{Amount: total, Currency: "VND"}
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd backend && go test ./internal/domain/accounting/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend
git add internal/domain/accounting/networth_service.go internal/domain/accounting/networth_service_test.go
git commit -m "feat(accounting): NetWorthService.Calculate pure domain function"
```

---

## Task 5: Port interfaces

**Files:**
- Create: `backend/internal/port/repository/accounting.go`
- Create: `backend/internal/port/events/publisher.go`

**Interfaces:**
- Consumes: `Account`, `JournalEntry`, `DomainEvent` from Tasks 2–3
- Produces:
  - `repository.AccountRepo` interface
  - `repository.JournalRepo` interface
  - `events.Publisher` interface

- [ ] **Step 1: Create AccountRepo and JournalRepo interfaces**

Create `backend/internal/port/repository/accounting.go`:

```go
package repository

import (
	"context"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
)

type AccountRepo interface {
	Save(ctx context.Context, a *accounting.Account) error
	FindByUser(ctx context.Context, userID string) ([]*accounting.Account, error)
	FindByID(ctx context.Context, id accounting.AccountID) (*accounting.Account, error)
}

type JournalRepo interface {
	Save(ctx context.Context, e *accounting.JournalEntry) error
	FindByUser(ctx context.Context, userID string, from, to time.Time) ([]*accounting.JournalEntry, error)
}
```

- [ ] **Step 2: Create EventPublisher interface**

Create `backend/internal/port/events/publisher.go`:

```go
package events

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
)

type Publisher interface {
	Publish(ctx context.Context, event accounting.DomainEvent) error
}
```

- [ ] **Step 3: Verify compile**

```bash
cd backend && go build ./internal/port/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
cd backend
git add internal/port/repository/accounting.go internal/port/events/
git commit -m "feat(accounting): port interfaces for AccountRepo, JournalRepo, EventPublisher"
```

---

## Task 6: Application services

**Files:**
- Create: `backend/internal/service/accounting/commands.go`
- Create: `backend/internal/service/accounting/journal_service.go`
- Create: `backend/internal/service/accounting/account_service.go`
- Create: `backend/internal/service/accounting/networth_query.go`
- Create: `backend/internal/service/accounting/journal_service_test.go`
- Create: `backend/internal/service/accounting/account_service_test.go`

**Interfaces:**
- Consumes: domain aggregates (Tasks 2–4), port interfaces (Task 5)
- Produces:
  - `JournalService.RecordTransaction(ctx, RecordTransactionCmd) (accounting.EntryID, error)`
  - `AccountService.OpenAccount(ctx, OpenAccountCmd) (accounting.AccountID, error)`
  - `NetWorthQuery.Current(ctx, userID string) (accounting.Money, error)`

- [ ] **Step 1: Write failing tests**

Create `backend/internal/service/accounting/journal_service_test.go`:

```go
package accountingsvc_test

import (
	"context"
	"testing"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	"github.com/shopspring/decimal"
)

// --- fakes ---

type fakeJournalRepo struct {
	saved []*accounting.JournalEntry
}

func (r *fakeJournalRepo) Save(_ context.Context, e *accounting.JournalEntry) error {
	r.saved = append(r.saved, e)
	return nil
}

func (r *fakeJournalRepo) FindByUser(_ context.Context, _ string, _, _ time.Time) ([]*accounting.JournalEntry, error) {
	return r.saved, nil
}

type fakePublisher struct {
	published []accounting.DomainEvent
}

func (p *fakePublisher) Publish(_ context.Context, ev accounting.DomainEvent) error {
	p.published = append(p.published, ev)
	return nil
}

// --- tests ---

func TestJournalService_RecordTransaction_Balanced(t *testing.T) {
	repo := &fakeJournalRepo{}
	pub  := &fakePublisher{}
	svc  := accountingsvc.NewJournalService(repo, pub)

	cmd := accountingsvc.RecordTransactionCmd{
		UserID:      "user1",
		Date:        time.Now(),
		Description: "Coffee",
		Lines: []accountingsvc.LineCmd{
			{AccountID: "account-food", Amount: decimal.NewFromInt(150000), Currency: "VND", Side: accounting.Debit},
			{AccountID: "account-visa", Amount: decimal.NewFromInt(150000), Currency: "VND", Side: accounting.Credit},
		},
	}
	id, err := svc.RecordTransaction(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(id) == "" {
		t.Error("want non-empty entry ID")
	}
	if len(repo.saved) != 1 {
		t.Error("want 1 saved entry")
	}
	if len(pub.published) != 1 {
		t.Error("want 1 published event")
	}
}

func TestJournalService_RecordTransaction_UnbalancedReturnsError(t *testing.T) {
	repo := &fakeJournalRepo{}
	pub  := &fakePublisher{}
	svc  := accountingsvc.NewJournalService(repo, pub)

	cmd := accountingsvc.RecordTransactionCmd{
		UserID:      "user1",
		Date:        time.Now(),
		Description: "Bad",
		Lines: []accountingsvc.LineCmd{
			{AccountID: "a", Amount: decimal.NewFromInt(100), Currency: "VND", Side: accounting.Debit},
			{AccountID: "b", Amount: decimal.NewFromInt(50), Currency: "VND", Side: accounting.Credit},
		},
	}
	_, err := svc.RecordTransaction(context.Background(), cmd)
	if err == nil {
		t.Fatal("want error for unbalanced entry")
	}
	if len(repo.saved) != 0 {
		t.Error("unbalanced entry must not be saved")
	}
}
```

Create `backend/internal/service/accounting/account_service_test.go`:

```go
package accountingsvc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
)

type fakeAccountRepo struct {
	accounts map[accounting.AccountID]*accounting.Account
}

func newFakeAccountRepo() *fakeAccountRepo {
	return &fakeAccountRepo{accounts: map[accounting.AccountID]*accounting.Account{}}
}

func (r *fakeAccountRepo) Save(_ context.Context, a *accounting.Account) error {
	r.accounts[a.ID()] = a
	return nil
}

func (r *fakeAccountRepo) FindByUser(_ context.Context, userID string) ([]*accounting.Account, error) {
	var result []*accounting.Account
	for _, a := range r.accounts {
		if a.UserID() == userID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (r *fakeAccountRepo) FindByID(_ context.Context, id accounting.AccountID) (*accounting.Account, error) {
	a, ok := r.accounts[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return a, nil
}

func TestAccountService_OpenAccount_Root(t *testing.T) {
	repo := newFakeAccountRepo()
	svc  := accountingsvc.NewAccountService(repo)

	cmd := accountingsvc.OpenAccountCmd{
		UserID:   "user1",
		Name:     "Cash",
		Type:     accounting.Asset,
		Currency: "VND",
		IsGroup:  false,
	}
	id, err := svc.OpenAccount(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(id) == "" {
		t.Error("want non-empty ID")
	}
}

func TestAccountService_OpenAccount_ParentMustBeGroup(t *testing.T) {
	repo := newFakeAccountRepo()
	svc  := accountingsvc.NewAccountService(repo)

	// Create a leaf account (isGroup=false)
	leaf := accounting.NewAccount("user1", nil, "Leaf", accounting.Asset, "VND", false, 0)
	repo.Save(context.Background(), leaf)

	pid := string(leaf.ID())
	cmd := accountingsvc.OpenAccountCmd{
		UserID:   "user1",
		ParentID: &pid,
		Name:     "Child",
		Type:     accounting.Asset,
		Currency: "VND",
	}
	_, err := svc.OpenAccount(context.Background(), cmd)
	if err == nil {
		t.Fatal("want error: parent is not a group")
	}
}
```

- [ ] **Step 2: Run — expect FAIL (compile)**

```bash
cd backend && go test ./internal/service/accounting/... -v
```

Expected: compile error — package does not exist.

- [ ] **Step 3: Implement commands and services**

Create `backend/internal/service/accounting/commands.go`:

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

type OpenAccountCmd struct {
	UserID    string
	ParentID  *string
	Name      string
	Type      accounting.AccountType
	Currency  string
	IsGroup   bool
	SortOrder int
}
```

Create `backend/internal/service/accounting/journal_service.go`:

```go
package accountingsvc

import (
	"context"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/events"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type JournalService struct {
	journal repository.JournalRepo
	pub     events.Publisher
}

func NewJournalService(journal repository.JournalRepo, pub events.Publisher) *JournalService {
	return &JournalService{journal: journal, pub: pub}
}

func (s *JournalService) RecordTransaction(ctx context.Context, cmd RecordTransactionCmd) (accounting.EntryID, error) {
	entry := accounting.NewJournalEntry(cmd.UserID, cmd.Date, cmd.Description)
	entry.SetMemo(cmd.Memo)

	for _, l := range cmd.Lines {
		money, err := accounting.NewMoney(l.Amount, l.Currency)
		if err != nil {
			return "", err
		}
		if err := entry.AddLine(accounting.AccountID(l.AccountID), money, l.Side); err != nil {
			return "", err
		}
	}

	if err := entry.Post(); err != nil {
		return "", err
	}
	if err := s.journal.Save(ctx, entry); err != nil {
		return "", err
	}
	for _, ev := range entry.Events() {
		if err := s.pub.Publish(ctx, ev); err != nil {
			return "", err
		}
	}
	return entry.ID(), nil
}
```

Create `backend/internal/service/accounting/account_service.go`:

```go
package accountingsvc

import (
	"context"
	"errors"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/repository"
)

type AccountService struct {
	accounts repository.AccountRepo
}

func NewAccountService(accounts repository.AccountRepo) *AccountService {
	return &AccountService{accounts: accounts}
}

func (s *AccountService) OpenAccount(ctx context.Context, cmd OpenAccountCmd) (accounting.AccountID, error) {
	if cmd.ParentID != nil {
		parent, err := s.accounts.FindByID(ctx, accounting.AccountID(*cmd.ParentID))
		if err != nil {
			return "", err
		}
		if !parent.IsGroup() {
			return "", errors.New("parent account must be a group")
		}
	}
	a := accounting.NewAccount(cmd.UserID, cmd.ParentID, cmd.Name, cmd.Type, cmd.Currency, cmd.IsGroup, cmd.SortOrder)
	if err := s.accounts.Save(ctx, a); err != nil {
		return "", err
	}
	return a.ID(), nil
}
```

Create `backend/internal/service/accounting/networth_query.go`:

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

func (q *NetWorthQuery) Current(ctx context.Context, userID string) (accounting.Money, error) {
	accounts, err := q.accounts.FindByUser(ctx, userID)
	if err != nil {
		return accounting.Money{}, err
	}
	entries, err := q.journal.FindByUser(ctx, userID, time.Time{}, time.Now())
	if err != nil {
		return accounting.Money{}, err
	}
	var lines []accounting.JournalLine
	for _, e := range entries {
		lines = append(lines, e.Lines()...)
	}
	return accounting.NetWorthService{}.Calculate(accounts, lines), nil
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd backend && go test ./internal/service/accounting/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend
git add internal/service/accounting/
git commit -m "feat(accounting): application services — JournalService, AccountService, NetWorthQuery"
```

---

## Task 7: Database migration

**Files:**
- Create: `backend/internal/migrate/008_double_entry.sql`
- Create: `supabase/migrations/20260617000001_double_entry.sql`
- Modify: `backend/internal/migrate/migrate.go` — register new migration

**Interfaces:**
- Produces: `accounts`, `journal_entries`, `journal_lines` tables in Postgres

- [ ] **Step 1: Write migration SQL**

Create `backend/internal/migrate/008_double_entry.sql`:

```sql
CREATE TYPE account_type AS ENUM ('asset', 'liability', 'equity', 'income', 'expense');
CREATE TYPE journal_side AS ENUM ('debit', 'credit');

CREATE TABLE IF NOT EXISTS accounts (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid        NOT NULL,
  parent_id   uuid        REFERENCES accounts(id) ON DELETE RESTRICT,
  name        text        NOT NULL,
  type        account_type NOT NULL,
  currency    text        NOT NULL DEFAULT 'VND',
  is_group    boolean     NOT NULL DEFAULT false,
  archived    boolean     NOT NULL DEFAULT false,
  sort_order  int         NOT NULL DEFAULT 0,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS journal_entries (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid        NOT NULL,
  date        date        NOT NULL,
  description text        NOT NULL,
  memo        text        NOT NULL DEFAULT '',
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS journal_lines (
  id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  entry_id    uuid         NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
  account_id  uuid         NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
  amount      numeric(15,2) NOT NULL CHECK (amount > 0),
  currency    text         NOT NULL DEFAULT 'VND',
  side        journal_side NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_accounts_user ON accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_journal_entries_user_date ON journal_entries(user_id, date);
CREATE INDEX IF NOT EXISTS idx_journal_lines_entry ON journal_lines(entry_id);
CREATE INDEX IF NOT EXISTS idx_journal_lines_account ON journal_lines(account_id);
```

Create `supabase/migrations/20260617000001_double_entry.sql` with identical content.

- [ ] **Step 2: Register migration in migrate.go**

Read `backend/internal/migrate/migrate.go` to find the migration registration pattern, then add `008_double_entry.sql` following the same pattern as the existing migrations.

- [ ] **Step 3: Verify migration runs**

```bash
cd backend
SELF_HOSTED=true go run ./cmd/server &
sleep 2
kill %1
```

Expected: server starts, migrations log shows `008_double_entry` applied, no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/migrate/008_double_entry.sql backend/internal/migrate/migrate.go supabase/migrations/20260617000001_double_entry.sql
git commit -m "feat(accounting): DB migration — accounts, journal_entries, journal_lines tables"
```

---

## Task 8: Postgres AccountRepo

**Files:**
- Create: `backend/internal/infra/postgres/accounting_accounts.go`

**Interfaces:**
- Consumes: `repository.AccountRepo` interface (Task 5), `accounts` table (Task 7)
- Produces: `pgAccountRepo` implementing `repository.AccountRepo`; exported constructor `NewAccountRepo(db *pgxpool.Pool) repository.AccountRepo`

- [ ] **Step 1: Implement pgAccountRepo**

Create `backend/internal/infra/postgres/accounting_accounts.go`:

```go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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
	_, err := r.db.Exec(ctx, `
		INSERT INTO accounts (id, user_id, parent_id, name, type, currency, is_group, archived, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET
			name=$4, type=$5, currency=$6, is_group=$7, archived=$8, sort_order=$9`,
		string(a.ID()), a.UserID(), parentID, a.Name(),
		string(a.Type()), a.Currency(), a.IsGroup(), a.Archived(), a.SortOrder(),
	)
	return err
}

func (r *pgAccountRepo) FindByUser(ctx context.Context, userID string) ([]*accounting.Account, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, parent_id, name, type, currency, is_group, archived, sort_order
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
		SELECT id, user_id, parent_id, name, type, currency, is_group, archived, sort_order
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
		return nil, errors.New("account not found")
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
		)
		if err := rows.Scan(&id, &userID, &parentID, &name, &acctType, &currency, &isGroup, &archived, &sortOrder); err != nil {
			return nil, err
		}
		result = append(result, accounting.ReconstitueAccount(
			id, userID, parentID, name,
			accounting.AccountType(acctType), currency, isGroup, archived, sortOrder,
		))
	}
	return result, rows.Err()
}
```

- [ ] **Step 2: Verify compile**

```bash
cd backend && go build ./internal/infra/postgres/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd backend
git add internal/infra/postgres/accounting_accounts.go
git commit -m "feat(accounting): Postgres AccountRepo implementation"
```

---

## Task 9: Postgres JournalRepo + InProcessPublisher

**Files:**
- Create: `backend/internal/infra/postgres/accounting_journal.go`
- Create: `backend/internal/infra/events/publisher.go`

**Interfaces:**
- Consumes: `repository.JournalRepo` (Task 5), `journal_entries` + `journal_lines` tables (Task 7)
- Produces:
  - `pgJournalRepo` implementing `repository.JournalRepo`; `NewJournalRepo(db *pgxpool.Pool) repository.JournalRepo`
  - `InProcessPublisher` implementing `events.Publisher`; `NewInProcessPublisher() *InProcessPublisher`

- [ ] **Step 1: Implement pgJournalRepo**

Create `backend/internal/infra/postgres/accounting_journal.go`:

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

type pgJournalRepo struct{ db *pgxpool.Pool }

func NewJournalRepo(db *pgxpool.Pool) repository.JournalRepo {
	return &pgJournalRepo{db: db}
}

func (r *pgJournalRepo) Save(ctx context.Context, e *accounting.JournalEntry) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO journal_entries (id, user_id, date, description, memo)
		VALUES ($1,$2,$3,$4,$5)`,
		string(e.ID()), e.UserID(), e.Date(), e.Description(), e.Memo(),
	)
	if err != nil {
		return err
	}

	for _, l := range e.Lines() {
		_, err = tx.Exec(ctx, `
			INSERT INTO journal_lines (id, entry_id, account_id, amount, currency, side)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			l.ID(), string(e.ID()), string(l.AccountID()),
			l.Money().Amount, l.Money().Currency, string(l.Side()),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *pgJournalRepo) FindByUser(ctx context.Context, userID string, from, to time.Time) ([]*accounting.JournalEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.id, e.user_id, e.date, e.description, e.memo,
		       l.id, l.account_id, l.amount, l.currency, l.side
		FROM journal_entries e
		JOIN journal_lines l ON l.entry_id = e.id
		WHERE e.user_id = $1 AND ($2::date IS NULL OR e.date >= $2) AND ($3::date IS NULL OR e.date <= $3)
		ORDER BY e.date DESC, e.id, l.id`,
		userID, nullDate(from), nullDate(to),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return reconstituteEntries(rows)
}

func nullDate(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func reconstituteEntries(rows pgx.Rows) ([]*accounting.JournalEntry, error) {
	entries := map[string]*accounting.JournalEntry{}
	order   := []string{}

	for rows.Next() {
		var (
			eID, eUserID, eDesc, eMemo string
			eDate                      time.Time
			lID, lAcctID, lCurrency    string
			lAmount                    decimal.Decimal
			lSide                      string
		)
		if err := rows.Scan(&eID, &eUserID, &eDate, &eDesc, &eMemo,
			&lID, &lAcctID, &lAmount, &lCurrency, &lSide); err != nil {
			return nil, err
		}
		if _, exists := entries[eID]; !exists {
			entries[eID] = accounting.ReconstitueEntry(eID, eUserID, eDate, eDesc, eMemo)
			order = append(order, eID)
		}
		entries[eID].ReconstituteLine(
			lID,
			accounting.AccountID(lAcctID),
			accounting.Money{Amount: lAmount, Currency: lCurrency},
			accounting.Side(lSide),
		)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]*accounting.JournalEntry, len(order))
	for i, id := range order {
		result[i] = entries[id]
	}
	return result, nil
}
```

- [ ] **Step 2: Implement InProcessPublisher**

```bash
mkdir -p backend/internal/infra/events
```

Create `backend/internal/infra/events/publisher.go`:

```go
package infraevents

import (
	"context"
	"log"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/port/events"
)

type InProcessPublisher struct{}

func NewInProcessPublisher() events.Publisher {
	return &InProcessPublisher{}
}

func (p *InProcessPublisher) Publish(_ context.Context, ev accounting.DomainEvent) error {
	switch e := ev.(type) {
	case accounting.EntryPosted:
		log.Printf("accounting: entry posted userID=%s entryID=%s date=%s", e.UserID, e.EntryID, e.Date.Format("2006-01-02"))
	}
	return nil
}
```

- [ ] **Step 3: Verify compile**

```bash
cd backend && go build ./internal/infra/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
cd backend
git add internal/infra/postgres/accounting_journal.go internal/infra/events/
git commit -m "feat(accounting): Postgres JournalRepo and InProcessPublisher"
```

---

## Task 10: HTTP transport

**Files:**
- Create: `backend/internal/transport/http/accounting_accounts.go`
- Create: `backend/internal/transport/http/accounting_journal.go`
- Create: `backend/internal/transport/http/accounting_accounts_test.go`
- Create: `backend/internal/transport/http/accounting_journal_test.go`

**Interfaces:**
- Consumes: `AccountService`, `JournalService`, `NetWorthQuery` from Task 6
- Produces HTTP endpoints:
  - `POST /api/accounts` → OpenAccount
  - `GET  /api/accounts` → list accounts for user
  - `POST /api/journal/entries` → RecordTransaction
  - `GET  /api/journal/networth` → current net worth

- [ ] **Step 1: Implement AccountsHandler**

Create `backend/internal/transport/http/accounting_accounts.go`:

```go
package httphandler

import (
	"encoding/json"
	"net/http"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
)

type AccountsHandler struct {
	svc      *accountingsvc.AccountService
	acctRepo interface {
		FindByUser(ctx interface{ Deadline() (interface{}, bool); Done() <-chan struct{}; Err() error; Value(interface{}) interface{} }, userID string) ([]*accounting.Account, error)
	}
}
```

Wait — `AccountsHandler` needs both `AccountService` (for create) and `AccountRepo` (for list). To keep it clean, add a `ListAccounts` method to `AccountService` instead of injecting the repo directly into the handler.

Modify `backend/internal/service/accounting/account_service.go` — add:

```go
func (s *AccountService) ListAccounts(ctx context.Context, userID string) ([]*accounting.Account, error) {
	return s.accounts.FindByUser(ctx, userID)
}
```

Now create `backend/internal/transport/http/accounting_accounts.go`:

```go
package httphandler

import (
	"encoding/json"
	"net/http"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
)

type AccountsHandler struct {
	svc *accountingsvc.AccountService
}

func NewAccountsHandler(svc *accountingsvc.AccountService) *AccountsHandler {
	return &AccountsHandler{svc: svc}
}

func (h *AccountsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	accounts, err := h.svc.ListAccounts(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	type row struct {
		ID        string `json:"id"`
		ParentID  string `json:"parent_id,omitempty"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		Currency  string `json:"currency"`
		IsGroup   bool   `json:"is_group"`
		SortOrder int    `json:"sort_order"`
	}
	resp := make([]row, len(accounts))
	for i, a := range accounts {
		var pid string
		if a.ParentID() != nil {
			pid = string(*a.ParentID())
		}
		resp[i] = row{
			ID:        string(a.ID()),
			ParentID:  pid,
			Name:      a.Name(),
			Type:      string(a.Type()),
			Currency:  a.Currency(),
			IsGroup:   a.IsGroup(),
			SortOrder: a.SortOrder(),
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AccountsHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req struct {
		ParentID  *string `json:"parent_id"`
		Name      string  `json:"name"`
		Type      string  `json:"type"`
		Currency  string  `json:"currency"`
		IsGroup   bool    `json:"is_group"`
		SortOrder int     `json:"sort_order"`
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
	id, err := h.svc.OpenAccount(r.Context(), cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": string(id)})
}
```

- [ ] **Step 2: Implement JournalHandler**

Create `backend/internal/transport/http/accounting_journal.go`:

```go
package httphandler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
	"github.com/shopspring/decimal"
)

type JournalHandler struct {
	journal  *accountingsvc.JournalService
	networth *accountingsvc.NetWorthQuery
}

func NewJournalHandler(journal *accountingsvc.JournalService, networth *accountingsvc.NetWorthQuery) *JournalHandler {
	return &JournalHandler{journal: journal, networth: networth}
}

func (h *JournalHandler) RecordTransaction(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req struct {
		Date        string `json:"date"`
		Description string `json:"description"`
		Memo        string `json:"memo"`
		Lines       []struct {
			AccountID string          `json:"account_id"`
			Amount    decimal.Decimal `json:"amount"`
			Currency  string          `json:"currency"`
			Side      string          `json:"side"`
		} `json:"lines"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, "invalid date format, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	lines := make([]accountingsvc.LineCmd, len(req.Lines))
	for i, l := range req.Lines {
		cur := l.Currency
		if cur == "" {
			cur = "VND"
		}
		lines[i] = accountingsvc.LineCmd{
			AccountID: l.AccountID,
			Amount:    l.Amount,
			Currency:  cur,
			Side:      accounting.Side(l.Side),
		}
	}
	cmd := accountingsvc.RecordTransactionCmd{
		UserID:      userID,
		Date:        date,
		Description: req.Description,
		Memo:        req.Memo,
		Lines:       lines,
	}
	id, err := h.journal.RecordTransaction(r.Context(), cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": string(id)})
}

func (h *JournalHandler) NetWorth(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	nw, err := h.networth.Current(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"net_worth": nw.Amount,
		"currency":  nw.Currency,
	})
}
```

- [ ] **Step 3: Write handler tests**

Create `backend/internal/transport/http/accounting_accounts_test.go`:

```go
package httphandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
)

// reuse fakeAccountRepo from account_service_test — define locally here too
type testAccountRepo struct {
	accounts map[accounting.AccountID]*accounting.Account
}

func newTestAccountRepo() *testAccountRepo {
	return &testAccountRepo{accounts: map[accounting.AccountID]*accounting.Account{}}
}

func (r *testAccountRepo) Save(_ context.Context, a *accounting.Account) error {
	r.accounts[a.ID()] = a
	return nil
}

func (r *testAccountRepo) FindByUser(_ context.Context, _ string) ([]*accounting.Account, error) {
	var result []*accounting.Account
	for _, a := range r.accounts {
		result = append(result, a)
	}
	return result, nil
}

func (r *testAccountRepo) FindByID(_ context.Context, id accounting.AccountID) (*accounting.Account, error) {
	a, ok := r.accounts[id]
	if !ok {
		return nil, nil
	}
	return a, nil
}

func TestAccountsHandler_Create_Success(t *testing.T) {
	svc := accountingsvc.NewAccountService(newTestAccountRepo())
	h   := httphandler.NewAccountsHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "Cash", "type": "asset", "currency": "VND",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr  := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["id"] == "" {
		t.Error("want non-empty id in response")
	}
}

func TestAccountsHandler_Create_MissingName(t *testing.T) {
	svc := accountingsvc.NewAccountService(newTestAccountRepo())
	h   := httphandler.NewAccountsHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{"type": "asset"})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr  := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rr.Code)
	}
}

func TestAccountsHandler_List_Empty(t *testing.T) {
	svc := accountingsvc.NewAccountService(newTestAccountRepo())
	h   := httphandler.NewAccountsHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr  := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
}
```

Create `backend/internal/transport/http/accounting_journal_test.go`:

```go
package httphandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chiutuanbinh/mylifeos/backend/internal/domain/accounting"
	accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"
	httphandler "github.com/chiutuanbinh/mylifeos/backend/internal/transport/http"
)

type testJournalRepo struct {
	saved []*accounting.JournalEntry
}

func (r *testJournalRepo) Save(_ context.Context, e *accounting.JournalEntry) error {
	r.saved = append(r.saved, e)
	return nil
}

func (r *testJournalRepo) FindByUser(_ context.Context, _ string, _, _ time.Time) ([]*accounting.JournalEntry, error) {
	return r.saved, nil
}

type testPublisher struct{}

func (p *testPublisher) Publish(_ context.Context, _ accounting.DomainEvent) error { return nil }

func TestJournalHandler_RecordTransaction_Balanced(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepo()
	pub   := &testPublisher{}

	journalSvc  := accountingsvc.NewJournalService(jRepo, pub)
	nwQuery     := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h           := httphandler.NewJournalHandler(journalSvc, nwQuery)

	body, _ := json.Marshal(map[string]interface{}{
		"date":        "2026-07-01",
		"description": "Coffee",
		"lines": []map[string]interface{}{
			{"account_id": "acc-food", "amount": 150000, "side": "debit"},
			{"account_id": "acc-visa", "amount": 150000, "side": "credit"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/journal/entries", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr  := httptest.NewRecorder()

	h.RecordTransaction(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestJournalHandler_RecordTransaction_UnbalancedReturns422(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepo()
	pub   := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, pub)
	nwQuery    := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h          := httphandler.NewJournalHandler(journalSvc, nwQuery)

	body, _ := json.Marshal(map[string]interface{}{
		"date":        "2026-07-01",
		"description": "Bad",
		"lines": []map[string]interface{}{
			{"account_id": "a", "amount": 100, "side": "debit"},
			{"account_id": "b", "amount": 50, "side": "credit"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/journal/entries", bytes.NewReader(body))
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr  := httptest.NewRecorder()

	h.RecordTransaction(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", rr.Code)
	}
}

func TestJournalHandler_NetWorth_ReturnsJSON(t *testing.T) {
	jRepo := &testJournalRepo{}
	aRepo := newTestAccountRepo()
	pub   := &testPublisher{}

	journalSvc := accountingsvc.NewJournalService(jRepo, pub)
	nwQuery    := accountingsvc.NewNetWorthQuery(aRepo, jRepo)
	h          := httphandler.NewJournalHandler(journalSvc, nwQuery)

	req := httptest.NewRequest(http.MethodGet, "/api/journal/networth", nil)
	req = req.WithContext(withUserID(req.Context(), "user1"))
	rr  := httptest.NewRecorder()

	h.NetWorth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if _, ok := resp["net_worth"]; !ok {
		t.Error("want net_worth in response")
	}
}
```

Note: `withUserID` is likely already defined in the existing test files — check `backend/internal/transport/http/` for an existing helper. If not, add to a `testhelpers_test.go` file:

```go
// backend/internal/transport/http/testhelpers_test.go (only if withUserID not already defined)
package httphandler_test

import (
	"context"
	"github.com/chiutuanbinh/mylifeos/backend/internal/middleware"
)

func withUserID(ctx context.Context, userID string) context.Context {
	return middleware.WithUserID(ctx, userID)
}
```

Check `backend/internal/middleware/` for the correct function name to set userID in context before writing the helper.

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd backend && go test ./internal/transport/http/... -v -coverprofile=coverage.out
bash scripts/hooks/pre-commit
```

Expected: all new tests pass, coverage ≥80% on new files.

- [ ] **Step 5: Commit**

```bash
cd backend
git add internal/transport/http/accounting_accounts.go \
        internal/transport/http/accounting_journal.go \
        internal/transport/http/accounting_accounts_test.go \
        internal/transport/http/accounting_journal_test.go \
        internal/service/accounting/account_service.go
git commit -m "feat(accounting): HTTP handlers for accounts and journal entries"
```

---

## Task 11: Wire up main.go

**Files:**
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: all repos/services/handlers from Tasks 8–10
- Produces: new routes registered: `POST /api/accounts`, `GET /api/accounts`, `POST /api/journal/entries`, `GET /api/journal/networth`

- [ ] **Step 1: Wire new components in main.go**

In `backend/cmd/server/main.go`, after the existing repo declarations, add:

```go
// import additions needed at top of file:
// infraevents "github.com/chiutuanbinh/mylifeos/backend/internal/infra/events"
// accountingsvc "github.com/chiutuanbinh/mylifeos/backend/internal/service/accounting"

// After existing repos:
accountRepo  := postgres.NewAccountRepo(db)
journalRepo  := postgres.NewJournalRepo(db)
eventPub     := infraevents.NewInProcessPublisher()

accountSvc   := accountingsvc.NewAccountService(accountRepo)
journalSvc   := accountingsvc.NewJournalService(journalRepo, eventPub)
nwQuery      := accountingsvc.NewNetWorthQuery(accountRepo, journalRepo)

accountsHandler := httphandler.NewAccountsHandler(accountSvc)
journalHandler  := httphandler.NewJournalHandler(journalSvc, nwQuery)
```

In the router section, inside the `r.Group(func(r chi.Router) { ... })` block that has `middleware.Auth`, add:

```go
r.Get("/api/accounts",            accountsHandler.List)
r.Post("/api/accounts",           accountsHandler.Create)
r.Post("/api/journal/entries",    journalHandler.RecordTransaction)
r.Get("/api/journal/networth",    journalHandler.NetWorth)
```

- [ ] **Step 2: Build and run**

```bash
cd backend && go build ./cmd/server/
```

Expected: clean build, no errors.

- [ ] **Step 3: Smoke test**

```bash
# Start server (requires local postgres from docker compose)
SELF_HOSTED=true go run ./cmd/server &
SERVER_PID=$!
sleep 2

# Health check (existing endpoint)
curl -s http://localhost:8080/health

kill $SERVER_PID
```

Expected: health endpoint responds.

- [ ] **Step 4: Run full test suite + coverage gate**

```bash
cd backend
go test ./internal/transport/http/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic
bash scripts/hooks/pre-commit
```

Expected: all pass, all files ≥80%.

- [ ] **Step 5: Commit**

```bash
cd backend
git add cmd/server/main.go
git commit -m "feat(accounting): wire double-entry accounting into server — accounts + journal endpoints"
```

---

## Self-Review

**Spec coverage check:**

| Spec section | Covered by |
|---|---|
| `accounts` table (hierarchical) | Task 7 migration, Task 2 aggregate, Task 8 repo |
| `journal_entries` + `journal_lines` | Task 7 migration, Task 3 aggregate, Task 9 repo |
| Balance invariant enforced in domain | Task 3 `Post()` |
| Normal balance convention | Task 2 `NormalBalance()` |
| `Money` value object | Task 1 |
| Domain events `EntryPosted` | Task 3 |
| Repository interfaces in ports | Task 5 |
| Application services | Task 6 |
| `NetWorthService.Calculate` pure | Task 4 |
| `NetWorthQuery` orchestration | Task 6 |
| Postgres implementations | Tasks 8, 9 |
| Reconstitution bypass Post() | Task 3 `ReconstitueEntry` + `ReconstituteLine` |
| HTTP endpoints | Task 10 |
| Wire up | Task 11 |
| Cut-over migration SQL | Task 7 |
| Old transactions untouched | Never modified — ✓ |
| Coverage gate ≥80% | Verified in Tasks 10, 11 |

**Frontend** (account setup UI, smart-default entry form, live net worth display) is a separate plan.
