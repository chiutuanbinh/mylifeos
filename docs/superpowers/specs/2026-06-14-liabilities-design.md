# Liabilities Feature Design

**Date:** 2026-06-14

## Summary

Add a `liabilities` data model so users can track debts alongside assets. Net worth = Assets − Liabilities, following standard accounting principles.

## Data Model

```sql
CREATE TABLE liabilities (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id            TEXT NOT NULL,
  name               TEXT NOT NULL,
  category           TEXT NOT NULL,
  balance            FLOAT8 NOT NULL,     -- current outstanding balance, always positive
  original_principal FLOAT8,              -- optional: original loan amount
  interest_rate      FLOAT8,              -- optional: annual rate 0–1
  started_at         DATE,
  due_at             DATE,
  notes              TEXT NOT NULL DEFAULT '',
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Suggested categories: `Mortgage`, `Car Loan`, `Credit Card`, `Personal Loan`, `Student Loan`, `Other`

`balance` is always stored as a positive number. The liability meaning (subtraction from net worth) comes from the model type, not a sign flip.

## Backend

- `models.Liability` struct mirroring the table
- `LiabilityRepo` interface: `List`, `Create`, `Update`, `Delete`
- `pgLiabilityRepo` implementation in `internal/repo/liabilities.go`
- `LiabilityHandler` in `internal/handlers/liabilities.go`
- Validation: `name` required, `category` required, `balance >= 0`, `interest_rate` in [0,1] if provided
- Routes registered on router: `GET /api/liabilities`, `POST /api/liabilities`, `PUT /api/liabilities/{id}`, `DELETE /api/liabilities/{id}`
- Net worth snapshot handler updated: `net_worth = assets_value - liabilities_balance`

## Frontend

- New "Liabilities" tab in `WealthPage` alongside existing "Assets" tab
- Table columns: Name, Category, Balance, Interest Rate, Due Date, Notes, Actions
- Add/Edit/Delete UX mirrors Assets tab (modal for add, drawer for edit)
- Net worth summary widget updated to three stat cards: **Assets | Liabilities | Net Worth**
- Category dropdown: Mortgage, Car Loan, Credit Card, Personal Loan, Student Loan, Other

## Migration

- New Supabase migration: `supabase/migrations/<timestamp>_liabilities.sql`
- New backend embed migration: `backend/internal/migrate/<N>_liabilities.sql`

## Testing

- Unit tests for `LiabilityHandler` covering CRUD + validation edge cases
- Coverage must meet ≥80% per file gate
