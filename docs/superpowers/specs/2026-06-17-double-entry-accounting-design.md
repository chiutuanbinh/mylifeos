# Double-Entry Accounting System Design

**Date:** 2026-06-17  
**Status:** Approved

## Overview

Replace the flat `transactions` table with a proper double-entry bookkeeping system. Every transaction moves value between two accounts. Net worth = Assets − Liabilities, always live, never a manual snapshot.

Old `transactions` table stays read-only for historical spend reports. New system starts from a cut-over date (opening balances).

---

## 1. Data Model

### `accounts` — hierarchical chart of accounts

```sql
accounts (
  id          uuid PK
  user_id     uuid
  parent_id   uuid NULL FK → accounts(id)   -- NULL = root node
  name        text                           -- "Techcombank Checking"
  type        enum(asset, liability, equity, income, expense)
  currency    text DEFAULT 'VND'
  is_group    bool                           -- true = folder, no direct balance
  archived    bool DEFAULT false
  sort_order  int
)
```

### `journal_entries` — transaction header

```sql
journal_entries (
  id          uuid PK
  user_id     uuid
  date        date
  description text
  memo        text NULL
  created_at  timestamptz
)
```

### `journal_lines` — debit/credit legs

```sql
journal_lines (
  id          uuid PK
  entry_id    uuid FK → journal_entries(id)
  account_id  uuid FK → accounts(id)
  amount      numeric(12,2)   -- always positive
  currency    text
  side        enum(debit, credit)
  -- invariant: sum(debit amounts) = sum(credit amounts) per entry_id
)
```

### Net worth query

```sql
SELECT
  SUM(CASE
    WHEN a.type = 'asset'     AND l.side = 'debit'  THEN  l.amount
    WHEN a.type = 'asset'     AND l.side = 'credit' THEN -l.amount
    WHEN a.type = 'liability' AND l.side = 'credit' THEN -l.amount
    WHEN a.type = 'liability' AND l.side = 'debit'  THEN  l.amount
    ELSE 0
  END) AS net_worth
FROM journal_lines l
JOIN accounts a ON l.account_id = a.id
WHERE a.user_id = $1 AND a.type IN ('asset','liability');
```

---

## 2. Domain Layer (DDD)

Pure Go, no imports from infra. Storage is a plugin.

### Value Objects

- `AccountID string` — typed ID, not raw string
- `EntryID string` — typed ID
- `Money{Amount decimal.Decimal, Currency string}` — immutable, equality by value, constructed via `NewMoney()` which rejects negative amounts, rejects cross-currency arithmetic

### Aggregate Root: JournalEntry

Owns its lines. Enforces balance invariant through methods, not external validators.

- `NewJournalEntry(userID, date, description)` — creates unposted entry
- `AddLine(accountID, money, side)` — appends a line
- `Post()` — validates ≥2 lines, debits == credits; emits `EntryPosted` domain event; only valid posted entries are saved
- `Lines()` — returns defensive copy
- `Events()` — returns domain events to dispatch after save

Reconstitution path (`ReconstitueEntry` + `ReconstituteLine`) bypasses `Post()` for loading from persistence.

### Aggregate Root: Account

- `NormalBalance() Side` — asset/expense = debit; liability/equity/income = credit
- `Balance(lines []JournalLine) Money` — pure calculation, no I/O

### Domain Events

```go
type EntryPosted struct {
    EntryID EntryID
    UserID  string
    Date    time.Time
}
```

### Domain Service: NetWorthService

```go
func (NetWorthService) Calculate(accounts []Account, lines []JournalLine) Money
```

Sums asset balances, subtracts liability balances. Pure function.

### Repository Interfaces (defined in domain, implemented in infra)

```go
type AccountRepository interface {
    Save(ctx, *Account) error
    FindByUser(ctx, userID string) ([]*Account, error)
    FindByID(ctx, AccountID) (*Account, error)
}

type JournalRepository interface {
    Save(ctx, *JournalEntry) error
    FindByUser(ctx, userID string, from, to time.Time) ([]*JournalEntry, error)
}
```

---

## 3. Application Layer

Orchestrates domain and infra. No business logic.

### Commands

- `RecordTransactionCmd{UserID, Date, Description, Memo, Lines[]LineCmd}`
- `OpenAccountCmd{UserID, ParentID, Name, Type, Currency, IsGroup, SortOrder}`

### Services

**JournalService.RecordTransaction:**
1. Build `JournalEntry` via domain constructor
2. Add lines via `AddLine()`
3. Call `Post()` — domain validates balance
4. Save via `JournalRepository`
5. Publish domain events via `EventPublisher`

**AccountService.OpenAccount:**
1. Validate parent exists and is a group account
2. Create `Account` aggregate
3. Save via `AccountRepository`

**NetWorthQuery.Current:**
1. Load all accounts for user
2. Load all journal entries for user
3. Flatten lines
4. Call `NetWorthService.Calculate()`

### EventPublisher interface

```go
type EventPublisher interface {
    Publish(ctx context.Context, event domain.DomainEvent) error
}
```

---

## 4. Infrastructure Layer

### Postgres repos

- `pgJournalRepo.Save` — single DB transaction: insert entry row + all line rows
- `pgJournalRepo.FindByUser` — JOIN query, reconstruct aggregates via `reconstituteEntries()`
- `pgAccountRepo` — standard CRUD

### InProcessPublisher

Handles `EntryPosted`: invalidates net worth cache for user. Swap for queue later when mobile capture pipeline is added.

### HTTP transport

Thin handlers: decode JSON → build command → call app service → encode response. Domain/app errors (balance, validation) → 422. Infra errors → 500.

---

## 5. Migration & Cut-over Strategy

### Approach: Clean cut-over date (no old data migration)

Old `transactions` table stays, read-only, for historical spend category reports.

### Steps

1. **Deploy new schema** — `accounts`, `journal_entries`, `journal_lines` alongside existing tables
2. **User sets up chart of accounts** — one-time UI, system seeds defaults:
   ```
   Assets
     Current Assets / Investments / Fixed Assets
   Liabilities
     Credit Cards / Loans
   Equity
     Opening Balance              ← system-managed
   Income
     Salary / Other Income
   Expenses
     [seeded from existing transaction categories]
   ```
3. **User records opening balances** — single journal entry on cut-over date:
   - User enters current balance for each asset and liability account
   - System auto-calculates the `Equity > Opening Balance` credit leg
4. **All new transactions** use journal entries from cut-over forward
5. **Net worth** computed from journal lines only — old `assets`/`liabilities` tables retired

---

## 6. Transaction Entry UX (Smart Defaults — v1)

User specifies one account + category. System infers both journal legs:

| User input | Generated entry |
|---|---|
| MB Visa + Food & Dining, 150k | DEBIT Expenses>Food&Dining / CREDIT Liabilities>MB Visa |
| Cash + Food & Dining, 150k | DEBIT Expenses>Food&Dining / CREDIT Assets>Cash |
| Transfer: Techcombank → VPS, 10M | DEBIT Assets>VPS / CREDIT Assets>Techcombank |
| Salary received to Techcombank, 20M | DEBIT Assets>Techcombank / CREDIT Income>Salary |
| Sell gold → cash, 5M | DEBIT Assets>Cash / CREDIT Assets>Gold |

v2: payee templates — "The Coffee House" always maps to Food & Dining from MB Visa.

---

## 7. Mobile Capture (Future)

```
Bank push/SMS: "MB Card -150,000 VND The Coffee House"
→ parser: extract account=MB Visa, amount=150k, payee
→ draft JournalEntry with smart default
→ push notification: "Confirm: Food & Dining 150k from MB Visa?"
→ user confirms → RecordTransaction command fires → entry posted
```

EventPublisher is the integration point — swap InProcessPublisher for a queue that feeds the mobile pipeline.

---

## Layer Diagram

```
transport/http
    ↓ commands
application services
    ↓ domain calls          ↑ repo interfaces
domain (pure)           infra/postgres
    ↓ domain events         infra/events (publisher)
                                ↓
                        cache invalidation / mobile queue
```
