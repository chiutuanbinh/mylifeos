# Accounting Onboarding Wizard

**Date:** 2026-06-20  
**Status:** Approved

## Problem

`AccountingPage` opens to an empty accounts table with no guidance. New users have no starting point.

## Solution

Frontend-only onboarding wizard that auto-opens when `accounts.length === 0`. User confirms (or unchecks) a pre-selected set of default accounts, clicks "Set Up", and the wizard creates them via existing `POST /accounts` API.

## Default Accounts

Groups (always created, not shown as checkboxes):

| Name | Type | is_group |
|------|------|----------|
| Assets | asset | true |
| Liabilities | liability | true |
| Equity | equity | true |
| Income | income | true |
| Expenses | expense | true |

Leaf accounts (shown as pre-checked checkboxes):

| Name | Type | Parent Group | Currency |
|------|------|-------------|----------|
| Cash | asset | Assets | VND |
| Bank Account | asset | Assets | VND |
| Credit Card | liability | Liabilities | VND |
| Opening Balance | equity | Equity | VND |
| Salary | income | Income | VND |
| Living Expenses | expense | Expenses | VND |

## Component Design

### `SetupWizard` (new component in `AccountingPage.tsx`)

Triggered by: `accounts.length === 0 && !isLoading` in `AccountsTab`.

**Modal contents:**
- Heading: "Set up your accounts"
- Subtext: "We'll create a starter chart of accounts. Uncheck any you don't need."
- Checkbox list of leaf accounts (all pre-checked)
- "Set Up" button — starts creation sequence
- "Skip" link — dismisses modal, shows empty table

**Creation sequence:**
1. Create all 5 group accounts in parallel (`Promise.all`)
2. Map returned IDs to parent references
3. Create selected leaf accounts in parallel with correct `parent_id`
4. `invalidateQueries(['accounts'])` → modal closes

**Error handling:** any failure shows inline error inside modal with "Retry" option. Partial state left in place (idempotent names prevent dupes if user retries, though API doesn't deduplicate — retry creates extras; acceptable for MVP).

**Skip behavior:** sets local state `skipped=true`, persisted only for the session (no localStorage). Revisiting the page with zero accounts re-shows wizard.

## Out of Scope

- Deduplication / idempotency on retry
- Persisting "skipped" across sessions
- Multi-currency defaults
- Backend seed endpoint
