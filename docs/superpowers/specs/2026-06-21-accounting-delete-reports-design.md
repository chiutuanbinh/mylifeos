# Accounting: Delete Account + Financial Reports

Date: 2026-06-21

## Scope

1. Delete account with safety guards
2. Financial Reports tab (Trial Balance, Balance Sheet, P&L) with time window selector

---

## Feature 1: Delete Account

### Rules

- Block deletion if account has child accounts → 400 "account has child accounts"
- Block deletion if account has journal lines → 400 "account has journal entries"
- Hard delete from DB on success

### Backend

**New interface method:**
```go
// AccountRepo
Delete(ctx context.Context, id accounting.AccountID) error
```

**New route:** `DELETE /accounts/{id}`

**Handler logic:**
1. Load account by ID (404 if not found)
2. Check children: query accounts where parent_id = id → 400 if any
3. Check journal lines: query journal_lines where account_id = id → 400 if any
4. Delete account row

**Postgres impl:** single DELETE query; rely on `ON DELETE RESTRICT` FK as safety net.

### Frontend

- Trash icon button on each row in the accounts tree table (AccountsTab)
- Ant Design `Popconfirm` before firing
- On API error, show error message from backend response
- On success, invalidate `['accounts']` query

---

## Feature 2: Financial Reports Tab

### New tab

Add **"Reports"** tab to `AccountingPage` (after Assets tab).

### Time Window Selector

`Segmented` control at top of Reports tab:

| Label | Period |
|-------|--------|
| Today | current calendar day |
| This Month | first of current month → today |
| This Quarter | first of current quarter → today |
| This Year | Jan 1 → today |
| All Time | beginning of time → today |

### Sub-tabs

Three Ant Design sub-tabs inside Reports: **Trial Balance** | **Balance Sheet** | **P&L**

#### Trial Balance

- All leaf accounts grouped by type (Asset, Liability, Equity, Income, Expense)
- Columns: Account Name | Debit | Credit
- Period: selected time window (income/expense accounts period-only; balance sheet accounts cumulative through end of period)
- Footer: total debits | total credits (must balance)

#### Balance Sheet

- Cumulative as-of end of selected period (not period-only — asset/liability/equity balances are always cumulative)
- **Assets** section: grouped by parent, subtotals per group, grand total
- **Liabilities** section: same structure
- **Equity** section: same structure
- Footer: Assets = Liabilities + Equity check (show discrepancy in red if unbalanced)

#### P&L (Income Statement)

- Period-only (income/expense activity within selected window)
- **Income** section: grouped by parent, subtotals, grand total
- **Expense** section: same
- **Net Income** row: Total Income − Total Expense (green if positive, red if negative)

### Data

- No new backend endpoints
- Reuse `accounts` and `journal entries` already fetched in AccountingPage
- Pass both down to Reports tab as props
- All filtering, grouping, aggregation computed client-side

### Components

```
AccountingPage
  └── ReportsTab (props: accounts[], entries[])
        ├── time window selector (state: window)
        ├── TrialBalance (accounts, entries, window)
        ├── BalanceSheet (accounts, entries, window)
        └── ProfitAndLoss (accounts, entries, window)
```

Shared helper: `computeAccountBalance(accountId, entries, from, to, cumulative)` — returns `{ debit, credit, balance }`.

For group accounts: sum all leaf descendants.

---

## Out of Scope

- Exporting reports (PDF/CSV)
- Drill-down from report row to individual journal entries
- Per-account ledger view (individual account history)
