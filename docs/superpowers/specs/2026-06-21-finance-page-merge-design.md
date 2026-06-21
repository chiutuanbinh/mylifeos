# Finance Page Merge Design

**Date:** 2026-06-21
**Status:** Approved

## Problem

Two separate nav entries — Wealth and Accounting — represent the same conceptual domain (personal finance). Users must switch between tabs to get a complete financial picture. Assets/Liabilities exist in both pages with overlapping but distinct semantics.

## Decision

Merge into a single **Finance** page at `/finance`. Accounting becomes the source of truth; Wealth views (Budgets, Trends, Transactions history) become tabs within Finance. The Wealth Assets/Liabilities tabs are retired — chart-of-accounts asset/liability accounts replace them. Asset balances are derived purely from journal entries (no manual overrides).

## Route

- `/finance` — new canonical route
- `/wealth` → redirect to `/finance`
- `/accounting` → redirect to `/finance`

## Tab Structure

| # | Tab | Source | Notes |
|---|-----|--------|-------|
| 1 | Journal | `AccountingPage.JournalTab` | Unchanged |
| 2 | Accounts | `AccountingPage.AccountsTab` | Unchanged, includes SetupWizard |
| 3 | Budgets | `WealthPage.BudgetsTab` | Moved verbatim |
| 4 | Reports | `AccountingPage.ReportsTab` + Ledger section | Gains Ledger at top |
| 5 | Trends | `WealthPage.TrendsTab` | Moved verbatim |

## Reports Tab — Ledger Section

Added above existing P&L/balance sheet content:
- Summary cards: Income / Expenses / Net Cash (from `getTransactions()`)
- Transactions table (read-only, same as old `WealthPage.TransactionsTab`)
- Existing Reports content follows

## Data Flow

- Single source of truth: journal entries → account balances → all derived views
- No backend changes required
- `getTransactions()`, `getBudgets()`, `getNetWorthSnapshots()`, `getBenchmarks()`, `getBankRates()`, `getNews()`, `triggerScrape()` — all endpoints unchanged
- Net worth snapshots reflect accounting balances, not manual asset edits

## Files Changed

| Action | File |
|--------|------|
| Rename + extend | `AccountingPage.tsx` → `FinancePage.tsx` |
| Delete | `WealthPage.tsx` |
| Update | `App.tsx` — add `/finance` route, redirects for `/wealth` and `/accounting` |
| Update | `AppShell.tsx` — replace two nav items with one "Finance" at `/finance` |
| Rename + extend | `AccountingPage.test.tsx` → `FinancePage.test.tsx` |

## Migration Order

1. Copy `BudgetsTab` and `TrendsTab` from `WealthPage.tsx` into `AccountingPage.tsx`
2. Add Ledger section (summary cards + transactions table) into `ReportsTab`
3. Add `budgets` and `trends` entries to `Tabs` items array
4. Rename file to `FinancePage.tsx`, update all imports
5. `App.tsx`: add `/finance` route, add redirects from `/wealth` and `/accounting`
6. `AppShell.tsx`: replace Wealth + Accounting nav items with Finance
7. Delete `WealthPage.tsx`
8. Rename and extend test file

## Deleted Content

- `WealthPage.TransactionsTab` — content absorbed into Reports › Ledger
- `WealthPage.AssetsTab` — replaced by Accounting's Accounts tab
- `WealthPage.LiabilitiesTab` — replaced by Accounting's Accounts tab
- `LiveNetWorthCard` — audit whether Trends tab uses it; delete if not

## Testing

- Rename `AccountingPage.test.tsx` → `FinancePage.test.tsx`
- Add smoke tests for Budgets and Trends tabs rendering
- Verify redirects from `/wealth` and `/accounting` work
- Run full lint + build before PR
