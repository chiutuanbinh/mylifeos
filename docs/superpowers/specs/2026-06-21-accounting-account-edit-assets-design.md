# Accounting: Account Edit, Opening Balance, Physical Assets, Net Income

Date: 2026-06-21

## Scope

Four features delivered together:

1. Edit existing accounts (name, type, parent, sort order)
2. Opening balance when creating an account
3. Physical asset metadata on accounting accounts (replaces `wealth.Asset`)
4. Net income (YTD) display card

---

## Background

The accounting domain uses double-entry (Option B — running balances, no period closing). Income/expense accounts accumulate forever; net income is a derived query for a date range. Net worth = Assets − Liabilities.

The existing `wealth.Asset` entity (WealthPage) is a separate, simpler model with depreciation fields. This spec retires it in favour of extending accounting `Account` with optional asset metadata.

---

## 1. Account Edit

### Domain

Add mutation methods to `Account`:

```go
func (a *Account) Rename(name string) { a.name = name }
func (a *Account) ChangeType(t AccountType) { a.acctType = t }
func (a *Account) Reparent(parentID *AccountID) { a.parentID = parentID }
func (a *Account) Reorder(n int) { a.sortOrder = n }
```

### Service

```go
type UpdateAccountCmd struct {
    ID        string
    UserID    string
    Name      string
    Type      AccountType
    ParentID  *string
    SortOrder int
}

func (s *AccountService) UpdateAccount(ctx context.Context, cmd UpdateAccountCmd) error
```

Logic: load by ID → verify ownership → validate parent (if set: must exist, belong to user, be a group) → mutate → `repo.Save`.

`repo.Save` already does `ON CONFLICT (id) DO UPDATE`, so no new repo method needed.

### HTTP

`PATCH /accounts/{id}` — authenticated. Request body:

```json
{
  "name": "string",
  "type": "asset|liability|equity|income|expense",
  "parent_id": "uuid|null",
  "sort_order": 0
}
```

Returns `204 No Content` on success.

### Frontend

- Add `EditOutlined` action column to accounts table in `AccountsTab`
- Click opens modal pre-filled with current account values (same form fields as Create)
- Calls new `updateAccount(id, patch)` API function → `PATCH /accounts/{id}`
- On success: invalidate `['accounts']` query

---

## 2. Opening Balance

When creating a new account, the user may specify an optional opening balance amount. The backend auto-posts a balancing journal entry against the "Opening Balance" equity account.

### Backend

Add to `OpenAccountCmd`:

```go
OpeningBalance *decimal.Decimal
```

In `OpenAccount` service method, after saving the new account:

```go
if cmd.OpeningBalance != nil && cmd.OpeningBalance.IsPositive() {
    // find Opening Balance equity account for this user
    ob, err := s.accounts.FindByNameAndType(ctx, cmd.UserID, "Opening Balance", accounting.Equity)
    if err != nil { return "", errors.New("Opening Balance equity account not found") }
    // post journal entry: DR new account / CR Opening Balance
    entry := accounting.NewJournalEntry(cmd.UserID, time.Now(), "Opening balance", "", []LineCmd{
        {AccountID: newID, Amount: *cmd.OpeningBalance, Currency: cmd.Currency, Side: accounting.Debit},
        {AccountID: ob.ID(), Amount: *cmd.OpeningBalance, Currency: cmd.Currency, Side: accounting.Credit},
    })
    s.journal.Save(ctx, entry)
}
```

New repo method required: `FindByNameAndType(ctx, userID, name, type) (*Account, error)`. Returns a typed error `ErrNotFound` when no match — caller surfaces this as `422 Unprocessable Entity` with message "Opening Balance equity account not found; run account setup first".

### Frontend

Add optional `InputNumber` "Opening Balance (VND)" to Create Account modal. Sent as `opening_balance` in `POST /accounts` body. Only shown for non-group accounts.

---

## 3. Physical Asset Metadata

### Approach

Extend `Account` with optional asset metadata. An accounting account of type `asset` may carry physical asset details. Current value = accounting balance (journal entries are source of truth). Depreciation is posted manually by the user as a journal entry:

```
DR  Depreciation Expense   (amount)
CR  Asset Account          (amount)
```

No automatic depreciation calculation.

### DB Migration

```sql
ALTER TABLE accounts
  ADD COLUMN purchase_value    NUMERIC,
  ADD COLUMN purchased_at      DATE,
  ADD COLUMN depreciation_rate NUMERIC,  -- annual fraction, e.g. 0.15
  ADD COLUMN asset_notes       TEXT;
```

### Domain

```go
type AssetMeta struct {
    PurchaseValue    *decimal.Decimal
    PurchasedAt      *time.Time
    DepreciationRate *decimal.Decimal
    Notes            string
}

// On Account:
func (a *Account) AssetMeta() *AssetMeta { return a.assetMeta }
func (a *Account) AttachAssetMeta(m *AssetMeta) { a.assetMeta = m }
```

`NewAccount` and `ReconstituteAccount` updated to accept/scan asset metadata. `repo.Save` and scan updated to include 4 new columns.

### HTTP

- `POST /accounts` and `PATCH /accounts/{id}` accept optional `asset_meta` object:

```json
{
  "asset_meta": {
    "purchase_value": 500000000,
    "purchased_at": "2023-01-15",
    "depreciation_rate": 0.15,
    "notes": "Toyota Innova"
  }
}
```

- `GET /accounts` response includes `asset_meta` (null if not set)

### Frontend

- In Create/Edit modal: when `type === 'asset'` and `is_group === false`, show collapsible "Asset Details" section with purchase value, purchase date, depreciation rate, notes
- New "Assets" tab in `AccountingPage` — filters `accounts` where `asset_meta != null`. Columns: Name, Purchase Value, Purchase Date, Depreciation Rate, Current Balance (from accounting), Notes
- WealthPage Assets tab: deprecated. Existing `wealth.Asset` endpoints remain and the tab still renders existing records, but shows a banner: "Assets are now tracked in Accounting → Assets tab." Full removal in a follow-up.

---

## 4. Net Income (YTD) Display

### Backend

Extend `GET /api/accounting/networth` response:

```json
{
  "net_worth": "1500000000",
  "net_income_ytd": "45000000"
}
```

`net_income_ytd` = (sum of credit lines on income accounts) − (sum of debit lines on expense accounts), filtered to current calendar year.

Existing `NetWorthQuery` service extended with this calculation.

### Frontend

Add "Net Income (YTD)" card next to Net Worth card in Journal tab:
- Green text if positive
- Red text if negative (net loss)

---

## Invariants

- Parent account must be a group; non-group accounts cannot be parents
- Opening balance only allowed for non-group accounts (any type)
- Asset metadata allowed on any non-group account; frontend only shows the section for type=asset for UX clarity
- Changing account type is allowed but caller is responsible for understanding balance implications
- `wealth.Asset` endpoints are not deleted in this change — deprecation only

---

## Files Affected

**Backend:**
- `domain/accounting/account.go` — mutation methods, AssetMeta, updated constructors
- `service/accounting/commands.go` — UpdateAccountCmd, OpeningBalance field
- `service/accounting/account_service.go` — UpdateAccount, opening balance logic
- `service/accounting/networth_query.go` — net_income_ytd
- `port/repository/accounting.go` — FindByNameAndType, updated AccountRepo interface
- `infra/postgres/accounting_accounts.go` — asset meta columns in Save/scan
- `transport/http/accounting_accounts.go` — PATCH handler, asset_meta in responses
- New migration: `supabase/migrations/YYYYMMDDHHMMSS_account_asset_meta.sql`

**Frontend:**
- `api/endpoints.ts` — updateAccount, updated types
- `api/types.ts` — Account type extended with asset_meta
- `pages/AccountingPage.tsx` — edit modal, opening balance field, Assets tab, net income card
