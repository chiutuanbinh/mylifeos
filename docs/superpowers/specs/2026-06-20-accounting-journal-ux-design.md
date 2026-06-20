# Accounting Journal UX — Design Spec
_2026-06-20_

## Goal

1. Disable legacy transaction inputs in the Wealth tab, routing users to the Accounting journal.
2. Rework the journal entry form to be usable by non-accountants while preserving double-entry accuracy.

---

## Part 1 — Disable Legacy Inputs

### Scope
- **In scope**: TransactionsTab, AssetsTab, LiabilitiesTab Add buttons
- **Out of scope**: BudgetsTab (budget limits have no journal equivalent; keep as-is)

### Change
Remove the "Add" button from TransactionsTab, AssetsTab, and LiabilitiesTab.

Replace each with a small inline notice below the card title:

> "To record new entries, use **Accounting → Journal**."

Tables remain fully visible and read-only. Delete buttons on existing rows stay (they call existing API mutations — no change needed there unless decided later).

### No redirects needed
Simple removal + notice is sufficient. No router navigation logic required.

---

## Part 2 — Smarter Journal Entry Form

### Problems with current form
- 3 stacked fields per line (Account, Amount, Side) inside individual cards — bulky
- "Debit / Credit" dropdown is opaque to non-accountants
- No balance feedback until submit fails
- Second line not auto-suggested when first line is filled

### New form layout

**Header fields** (unchanged):
- Date (required)
- Description (required)
- Memo (optional)

**Lines section** — compact table layout, one row per line:

| Account | Amount (VND) | DR | CR |
|---|---|---|---|
| `[searchable select]` | `[number input]` | `○` | `○` |

- Account option label format: `Cash (asset · DR+)` — shows type and normal balance side
  - Normal DR+: asset, expense
  - Normal CR+: liability, equity, income
- When user selects account on a line, the DR/CR radio auto-selects the normal side for that account type
- When first line's account is selected, auto-add a second empty line with the opposite side pre-selected
- "Add Line" button appends a new row (side defaults to whatever keeps running balance equal)

**Balance indicator** (below lines, above submit):

```
DR ₫1,000,000   CR ₫1,000,000   ✓ Balanced
```

- Green + checkmark when DR total == CR total
- Red + "₫X unbalanced" warning when totals differ
- Submit button disabled when unbalanced (prevents posting invalid entry)

### Account selector hint logic

```
type normalSide(type):
  asset | expense   → 'debit'
  liability | equity | income → 'credit'
```

Label suffix:
- `· DR+` means debiting this account increases its balance
- `· CR+` means crediting this account increases its balance

### Validation
- All existing required-field validation unchanged
- Add: form-level check that sum(debit amounts) === sum(credit amounts) before enabling submit

---

## Files Affected

- `frontend/src/pages/WealthPage.tsx` — remove Add buttons from TransactionsTab, AssetsTab, LiabilitiesTab; add redirect notice
- `frontend/src/pages/AccountingPage.tsx` — replace JournalTab form with new compact table layout + balance indicator

---

## Out of Scope

- Journal history list (already marked "coming soon")
- Editing or deleting journal entries
- Multi-currency balance checking
- Asset/Liability journal entry templates (e.g. "Buy Asset" wizard)
