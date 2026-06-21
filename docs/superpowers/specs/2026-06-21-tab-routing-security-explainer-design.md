# Design: Tab Routing, Security Hardening, Accounting Explainer

Date: 2026-06-21

## 1. Tab URL Sync

### Goal
Preserve active tab across refresh and enable shareable URLs for any tabbed page.

### Approach
Custom hook `useTabParam(defaultKey: string)` wraps `useSearchParams` from react-router-dom. Returns `[activeKey, setActiveKey]` — drop-in for `useState`. On change, updates `?tab=<key>` in URL without full navigation. On mount, reads `?tab` from URL; falls back to `defaultKey` if absent or invalid.

### Usage
```tsx
// Before
const [active, setActive] = useState('journal')

// After
const [active, setActive] = useTabParam('journal')
```

### Applied to
- `AccountingPage` — tabs: journal, accounts, assets, reports
- Pattern available for any future tabbed page via the hook

### File
- `frontend/src/hooks/useTabParam.ts` (new)
- `frontend/src/pages/AccountingPage.tsx` (switch to hook)

---

## 2. Security Hardening (Defense-in-Depth)

### Current state
All HTTP handlers extract `userID` from JWT via `middleware.GetUserID`. Service layer checks `a.UserID() != cmd.UserID` before any mutation. No authorization bypass exists at the service level.

### Gap
Two SQL queries lack `user_id` filter:
- `FindByID`: `SELECT ... FROM accounts WHERE id = $1` — returns account regardless of owner; service checks afterward
- `Delete`: `DELETE FROM accounts WHERE id = $1` — deletes by ID only; safe because service verifies ownership first, but DB-layer bypass (direct SQL, future migration) could skip this

### Fix
Add `AND user_id = $2` to both queries. No behavior change for normal flow; adds DB-level enforcement as second line of defense.

```sql
-- FindByID
SELECT ... FROM accounts WHERE id = $1 AND user_id = $2

-- Delete
DELETE FROM accounts WHERE id = $1 AND user_id = $2
```

Update `repository.AccountRepo` interface and `postgres/accounting_accounts.go` to pass `userID` to both methods. Update callers in service layer.

### Files
- `backend/internal/infra/postgres/accounting_accounts.go`
- `backend/internal/port/repository/accounting.go`
- `backend/internal/service/accounting/account_service.go`

---

## 3. Accounting Explainer (How It Works)

### Goal
Help users understand double-entry accounting without leaving the app.

### Approach
`HowItWorksModal` — a React modal component triggered by a `?` icon button in the AccountingPage header. Static content only, no backend.

### Content outline
1. **What is double-entry accounting** — every transaction affects two accounts; debits = credits always
2. **Debits and credits intuition** — assets/expenses increase with debit; liabilities/equity/income increase with credit
3. **How this app uses it** — accounts chart, journal entries, how balance sheet / P&L are derived
4. **External references** with links:
   - Wikipedia: Double-entry bookkeeping
   - Investopedia: Double Entry
   - AccountingCoach: Debits and Credits

### UI
- `?` (QuestionCircleOutlined) icon button in AccountingPage title/header area
- Ant Design `Modal` with `footer={null}`, width ~600px
- Sections separated by `Divider`, external links open in new tab

### Files
- `frontend/src/pages/AccountingPage.tsx` — add button + modal inline (component is small enough)

---

## Out of scope
- Server-side rendering of tab state
- Persisting last-visited tab to user settings
- Full security pen-test (covered by service-layer ownership checks)
- Accounting tutorial with interactive exercises
