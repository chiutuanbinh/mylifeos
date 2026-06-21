# Goals & Budgets: Editable + Journal Integration

**Date:** 2026-06-21

## Summary

Three focused changes:
1. Budgets become fully editable (edit modal + delete per row)
2. Journal entries can be tagged with one or more goals
3. Journal Record Entry modal shows live budget remaining as context

## Data Model

New migration adds a join table linking journal entries to goals:

```sql
CREATE TABLE journal_entry_goals (
  entry_id  TEXT NOT NULL REFERENCES accounting_journal_entries(id) ON DELETE CASCADE,
  goal_id   TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  user_id   TEXT NOT NULL,
  PRIMARY KEY (entry_id, goal_id)
);
```

No changes to `Budget` or `Goal` structs.

`JournalEntry` domain struct gains `GoalIDs []string` (populated on read).

## Backend

### New endpoint
- `DELETE /finance/budgets/:category` — deletes budget for authenticated user + category. Follows existing pattern in `transactions.go`.

### Journal service
- `RecordTransactionCmd` gains `GoalIDs []string`
- `JournalService.RecordTransaction` calls `s.journal.SaveGoalLinks(ctx, entryID, userID, goalIDs)` after saving entry
- `JournalRepo` interface gains:
  - `SaveGoalLinks(ctx, entryID, userID string, goalIDs []string) error`
  - `FindByUser` updated to populate `GoalIDs` on returned entries (LEFT JOIN or batch query)

### HTTP handler
- `accounting_journal.go` decodes `goal_ids []string` from request body, passes to `RecordTransactionCmd`

## Frontend

### BudgetsTab (WealthPage)
- Replace "Set Budget Limit" upsert card with a proper management UI:
  - Table: Category | Monthly Limit | Actions
  - Edit action: opens modal with `InputNumber` for monthly_limit, submits `upsertBudget`
  - Delete action: calls `DELETE /finance/budgets/:category`, invalidates `['budgets']`
  - "Add Budget" button above table opens same modal with empty category select

### Journal — Record Entry modal (AccountingPage)
- Add `goal_ids` field: `Select mode="multiple"`, options from `['goals']` query (goal name + color tag)
- Add budget context panel below lines: shows current-month remaining per budget category
  - Data source: existing `['budgets']` + `['transactions']` — no new API calls
  - Renders as compact tag row: `Food: 2.3M remaining` (green if > 20% left, red if ≤ 20%)

### Journal list
- Add "Goals" column: renders goal name tags for entries with `goal_ids`

## API Contract

`POST /accounting/journal` request body gains:
```json
{ "goal_ids": ["goal-uuid-1", "goal-uuid-2"] }
```
Optional — omit or empty array means no goal links.

`GET /accounting/journal` response entries gain:
```json
{ "goal_ids": ["goal-uuid-1"] }
```

## Error Handling
- Invalid goal IDs in `goal_ids` are silently ignored (goal may have been deleted)
- Budget delete on non-existent category returns 404
- Goal tag save failure does not roll back the journal entry (best-effort)

## Out of Scope
- Goal financial targets (e.g., "save 50M VND") — not in this spec
- Filtering journal entries by goal
- Budget period other than monthly
