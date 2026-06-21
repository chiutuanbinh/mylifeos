# Accounting: Delete Account + Financial Reports — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add account deletion with safety guards, and a Financial Reports tab (Trial Balance, Balance Sheet, P&L) with time-window filtering.

**Architecture:** Backend adds `Delete` to `AccountRepo` interface + postgres impl + HTTP handler. Frontend adds delete button to AccountsTab and a new ReportsTab component with client-side aggregation from already-fetched data.

**Tech Stack:** Go (chi, pgx), React + TypeScript, Ant Design, React Query

## Global Constraints

- Go: all new handlers must maintain ≥80% test coverage per file in `transport/http` and `middleware`
- Frontend: `npm run lint && npm run build` must pass clean
- Currency: VND only (no multi-currency in reports)
- Auth: all backend handlers read userID from `middleware.GetUserID(r)`
- No new backend endpoints for reports — reuse existing data

---

## File Map

**Backend — new/modified:**
- `backend/internal/port/repository/accounting.go` — add `Delete` to `AccountRepo` interface
- `backend/internal/infra/postgres/accounting_accounts.go` — implement `Delete`
- `backend/internal/service/accounting/account_service.go` — add `DeleteAccount` method
- `backend/internal/transport/http/accounting_accounts.go` — add `Delete` handler
- `backend/internal/transport/http/accounting_accounts_test.go` — new tests for Delete
- `backend/cmd/server/main.go` — register `DELETE /accounts/{id}` route

**Frontend — new/modified:**
- `frontend/src/api/endpoints.ts` — add `deleteAccount`
- `frontend/src/api/types.ts` — no changes needed
- `frontend/src/pages/AccountingPage.tsx` — add delete button + ReportsTab
- `frontend/src/pages/ReportsTab.tsx` — new file: time window selector + Trial Balance + Balance Sheet + P&L

---

### Task 1: Add `Delete` to AccountRepo interface + postgres impl

**Files:**
- Modify: `backend/internal/port/repository/accounting.go`
- Modify: `backend/internal/infra/postgres/accounting_accounts.go`

**Interfaces:**
- Produces: `AccountRepo.Delete(ctx context.Context, id accounting.AccountID) error`

- [ ] **Step 1: Add method to interface**

In `backend/internal/port/repository/accounting.go`, add to `AccountRepo` interface:

```go
type AccountRepo interface {
    Save(ctx context.Context, a *accounting.Account) error
    FindByUser(ctx context.Context, userID string) ([]*accounting.Account, error)
    FindByID(ctx context.Context, id accounting.AccountID) (*accounting.Account, error)
    FindByNameAndType(ctx context.Context, userID, name string, t accounting.AccountType) (*accounting.Account, error)
    Delete(ctx context.Context, id accounting.AccountID) error
}
```

- [ ] **Step 2: Implement in postgres**

In `backend/internal/infra/postgres/accounting_accounts.go`, add after `FindByNameAndType`:

```go
func (r *pgAccountRepo) Delete(ctx context.Context, id accounting.AccountID) error {
    _, err := r.db.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, id)
    return err
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd backend && go build ./...
```
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add backend/internal/port/repository/accounting.go backend/internal/infra/postgres/accounting_accounts.go
git commit -m "feat(accounting): Add Delete to AccountRepo interface and postgres impl"
```

---

### Task 2: Add `DeleteAccount` to AccountService + test stubs

**Files:**
- Modify: `backend/internal/service/accounting/account_service.go`
- Modify: `backend/internal/transport/http/accounting_accounts_test.go` — add `Delete` to `testAccountRepo`

**Interfaces:**
- Consumes: `AccountRepo.Delete`, `AccountRepo.FindByUser`, `AccountRepo.FindByID`, `JournalRepo.FindByUser`
- Produces: `AccountService.DeleteAccount(ctx, userID, id string) error`

- [ ] **Step 1: Add `Delete` to `testAccountRepo` in test file**

In `backend/internal/transport/http/accounting_accounts_test.go`, add method to `testAccountRepo`:

```go
func (r *testAccountRepo) Delete(_ context.Context, id accounting.AccountID) error {
    delete(r.accounts, id)
    return nil
}
```

- [ ] **Step 2: Verify tests still compile and pass**

```bash
cd backend && go test ./internal/transport/http/... -count=1 -q
```
Expected: all existing tests pass

- [ ] **Step 3: Add `DeleteAccount` to account service**

In `backend/internal/service/accounting/account_service.go`, add:

```go
var (
    ErrAccountHasChildren     = errors.New("account has child accounts")
    ErrAccountHasJournalLines = errors.New("account has journal entries")
)

func (s *AccountService) DeleteAccount(ctx context.Context, userID, id string) error {
    acctID := accounting.AccountID(id)
    // verify ownership
    acct, err := s.accounts.FindByID(ctx, acctID)
    if err != nil {
        return err
    }
    if acct.UserID() != userID {
        return repository.ErrAccountNotFound
    }
    // check children
    all, err := s.accounts.FindByUser(ctx, userID)
    if err != nil {
        return err
    }
    for _, a := range all {
        if a.ParentID() != nil && *a.ParentID() == acctID {
            return ErrAccountHasChildren
        }
    }
    // check journal lines
    entries, err := s.journal.FindByUser(ctx, userID, time.Time{}, time.Now())
    if err != nil {
        return err
    }
    for _, e := range entries {
        for _, l := range e.Lines() {
            if l.AccountID() == acctID {
                return ErrAccountHasJournalLines
            }
        }
    }
    return s.accounts.Delete(ctx, acctID)
}
```

Make sure `"time"` and `"errors"` are imported. Check that `Account.ParentID()` exists — look for it with:

```bash
grep -n "ParentID\b" backend/internal/domain/accounting/account.go
```

If `ParentID()` is not a method, check how parent is exposed (may be `account.parentID` field accessed differently). Adapt accordingly.

- [ ] **Step 4: Verify it compiles**

```bash
cd backend && go build ./...
```
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/accounting/account_service.go backend/internal/transport/http/accounting_accounts_test.go
git commit -m "feat(accounting): DeleteAccount service method with child and journal-line guards"
```

---

### Task 3: HTTP handler for DELETE /accounts/{id} with tests

**Files:**
- Modify: `backend/internal/transport/http/accounting_accounts.go`
- Modify: `backend/internal/transport/http/accounting_accounts_test.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: `AccountService.DeleteAccount(ctx, userID, id string) error`
- Consumes: `accountingsvc.ErrAccountHasChildren`, `accountingsvc.ErrAccountHasJournalLines`
- Consumes: `repository.ErrAccountNotFound`

- [ ] **Step 1: Write failing tests**

In `backend/internal/transport/http/accounting_accounts_test.go`, add at end of file:

```go
func TestAccountsHandler_Delete_Success(t *testing.T) {
    repo := newTestAccountRepo()
    // create an account with no children and no journal lines
    acct := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
    repo.accounts[acct.ID()] = acct

    svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
    h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

    r := httptest.NewRequest(http.MethodDelete, "/accounts/"+string(acct.ID()), nil)
    r = r.WithContext(setUserID(r.Context(), "user1"))
    r = setChiURLParam(r, "id", string(acct.ID()))
    w := httptest.NewRecorder()
    h.Delete(w, r)

    if w.Code != http.StatusNoContent {
        t.Errorf("want 204, got %d: %s", w.Code, w.Body.String())
    }
}

func TestAccountsHandler_Delete_HasChildren(t *testing.T) {
    repo := newTestAccountRepo()
    parent := accounting.NewAccount("user1", nil, "Assets", accounting.Asset, "VND", true, 0)
    child := accounting.NewAccount("user1", func() *accounting.AccountID { id := parent.ID(); return &id }(), "Cash", accounting.Asset, "VND", false, 0)
    repo.accounts[parent.ID()] = parent
    repo.accounts[child.ID()] = child

    svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
    h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

    r := httptest.NewRequest(http.MethodDelete, "/accounts/"+string(parent.ID()), nil)
    r = r.WithContext(setUserID(r.Context(), "user1"))
    r = setChiURLParam(r, "id", string(parent.ID()))
    w := httptest.NewRecorder()
    h.Delete(w, r)

    if w.Code != http.StatusBadRequest {
        t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
    }
}

func TestAccountsHandler_Delete_HasJournalLines(t *testing.T) {
    repo := newTestAccountRepo()
    acct := accounting.NewAccount("user1", nil, "Cash", accounting.Asset, "VND", false, 0)
    repo.accounts[acct.ID()] = acct

    // journal repo with a line referencing this account
    jr := &testJournalRepo{}
    entry := accounting.NewJournalEntry("user1", time.Now(), "test")
    _ = entry.AddLine(acct.ID(), accounting.Money{Amount: decimal.NewFromInt(100), Currency: "VND"}, accounting.Debit)
    _ = entry.Post()
    jr.entries = append(jr.entries, entry)

    svc := accountingsvc.NewAccountService(repo, jr)
    h := httphandler.NewAccountsHandler(svc, jr)

    r := httptest.NewRequest(http.MethodDelete, "/accounts/"+string(acct.ID()), nil)
    r = r.WithContext(setUserID(r.Context(), "user1"))
    r = setChiURLParam(r, "id", string(acct.ID()))
    w := httptest.NewRecorder()
    h.Delete(w, r)

    if w.Code != http.StatusBadRequest {
        t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
    }
}

func TestAccountsHandler_Delete_NotFound(t *testing.T) {
    repo := newTestAccountRepo()
    svc := accountingsvc.NewAccountService(repo, &testJournalRepo{})
    h := httphandler.NewAccountsHandler(svc, &testJournalRepo{})

    r := httptest.NewRequest(http.MethodDelete, "/accounts/nonexistent", nil)
    r = r.WithContext(setUserID(r.Context(), "user1"))
    r = setChiURLParam(r, "id", "nonexistent")
    w := httptest.NewRecorder()
    h.Delete(w, r)

    if w.Code != http.StatusNotFound {
        t.Errorf("want 404, got %d: %s", w.Code, w.Body.String())
    }
}
```

Note: Check how `testJournalRepo` stores entries — look for its struct definition and `entries` field. If it uses a different field name, adapt the test above.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend && go test ./internal/transport/http/... -run TestAccountsHandler_Delete -v 2>&1 | head -20
```
Expected: compile error "h.Delete undefined"

- [ ] **Step 3: Implement the handler**

In `backend/internal/transport/http/accounting_accounts.go`, add:

```go
func (h *AccountsHandler) Delete(w http.ResponseWriter, r *http.Request) {
    userID := middleware.GetUserID(r)
    id := chi.URLParam(r, "id")
    err := h.svc.DeleteAccount(r.Context(), userID, id)
    if err == nil {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    switch {
    case errors.Is(err, repository.ErrAccountNotFound):
        http.Error(w, `{"error":"account not found"}`, http.StatusNotFound)
    case errors.Is(err, accountingsvc.ErrAccountHasChildren):
        http.Error(w, `{"error":"account has child accounts"}`, http.StatusBadRequest)
    case errors.Is(err, accountingsvc.ErrAccountHasJournalLines):
        http.Error(w, `{"error":"account has journal entries"}`, http.StatusBadRequest)
    default:
        http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
    }
}
```

Make sure `"errors"` is imported at the top of the file.

- [ ] **Step 4: Register route in main.go**

In `backend/cmd/server/main.go`, in the accounts route group, add:

```go
r.Delete("/accounts/{id}", accountsHandler.Delete)
```

Place it after `r.Patch("/accounts/{id}", accountsHandler.Update)`.

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd backend && go test ./internal/transport/http/... -run TestAccountsHandler_Delete -v
```
Expected: all 4 Delete tests pass

- [ ] **Step 6: Run full coverage check**

```bash
cd backend && go test ./internal/transport/http/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic && bash ../scripts/hooks/pre-commit
```
Expected: ✓ Coverage OK, all files ≥80%

- [ ] **Step 7: Commit**

```bash
git add backend/internal/transport/http/accounting_accounts.go backend/internal/transport/http/accounting_accounts_test.go backend/cmd/server/main.go
git commit -m "feat(accounting): DELETE /accounts/{id} with child and journal-line guards"
```

---

### Task 4: Frontend — deleteAccount API + delete button in AccountsTab

**Files:**
- Modify: `frontend/src/api/endpoints.ts`
- Modify: `frontend/src/pages/AccountingPage.tsx`

**Interfaces:**
- Produces: `deleteAccount(id: string): Promise<void>`

- [ ] **Step 1: Add `deleteAccount` to endpoints**

In `frontend/src/api/endpoints.ts`, add (near other accounting endpoints):

```typescript
export const deleteAccount = (id: string) =>
  apiClient.delete(`/accounts/${id}`)
```

- [ ] **Step 2: Import `deleteAccount` and `DeleteOutlined` in AccountingPage**

In `frontend/src/pages/AccountingPage.tsx`:

Top import line for antd icons — add `DeleteOutlined`:
```typescript
import { PlusOutlined, FolderOutlined, FileOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
```

API imports line — add `deleteAccount`:
```typescript
import { getAccounts, createAccount, updateAccount, deleteAccount, createJournalEntry, getJournalEntries, getJournalNetWorth } from '../api/endpoints'
```

Also add `Popconfirm, message` to the antd import:
```typescript
import {
  Tabs, Card, Table, Tag, Button, Form, Input, Select, Switch,
  InputNumber, Modal, Spin, Badge, Checkbox, Radio, Collapse, Row, Col,
  Popconfirm, message,
} from 'antd'
```

- [ ] **Step 3: Add delete mutation inside `AccountsTab`**

Inside `function AccountsTab()`, after the `editMutation` block, add:

```typescript
const deleteMutation = useMutation({
  mutationFn: (id: string) => deleteAccount(id),
  onSuccess: () => {
    qc.invalidateQueries({ queryKey: ['accounts'] })
    message.success('Account deleted')
  },
  onError: (err: unknown) => {
    const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Failed to delete account'
    message.error(msg)
  },
})
```

- [ ] **Step 4: Add delete button column to `columns`**

In `AccountsTab`, find the `columns` array. After the existing edit button column (`title: ''`), add another column:

```typescript
{
  title: '',
  width: 48,
  render: (_: unknown, row: AccountTreeNode) => (
    <Popconfirm
      title="Delete account?"
      description="This cannot be undone."
      onConfirm={() => deleteMutation.mutate(row.id)}
      okText="Delete"
      okButtonProps={{ danger: true }}
    >
      <Button
        type="text"
        size="small"
        danger
        icon={<DeleteOutlined />}
      />
    </Popconfirm>
  ),
},
```

- [ ] **Step 5: Lint and build**

```bash
cd frontend && npm run lint && npm run build
```
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add frontend/src/api/endpoints.ts frontend/src/pages/AccountingPage.tsx
git commit -m "feat(accounting): delete account button with confirmation"
```

---

### Task 5: Frontend — ReportsTab component

**Files:**
- Create: `frontend/src/pages/ReportsTab.tsx`
- Modify: `frontend/src/pages/AccountingPage.tsx`

**Interfaces:**
- Consumes: `Account` type from `../api/types`, `JournalEntry` type from `../api/types`
- Consumes: `getAccounts`, `getJournalEntries` (already available in AccountingPage)

- [ ] **Step 1: Create `frontend/src/pages/ReportsTab.tsx`**

```typescript
import { useState, useMemo } from 'react'
import { Tabs, Segmented, Table, Typography } from 'antd'
import type { Account, JournalEntry } from '../api/types'

const { Text } = Typography

// ── helpers ──────────────────────────────────────────────────────────────

type Window = 'today' | 'month' | 'quarter' | 'year' | 'all'

function windowBounds(w: Window): { from: Date; to: Date } {
  const now = new Date()
  const to = new Date(now)
  to.setHours(23, 59, 59, 999)
  let from: Date
  switch (w) {
    case 'today':
      from = new Date(now); from.setHours(0, 0, 0, 0); break
    case 'month':
      from = new Date(now.getFullYear(), now.getMonth(), 1); break
    case 'quarter': {
      const q = Math.floor(now.getMonth() / 3)
      from = new Date(now.getFullYear(), q * 3, 1); break
    }
    case 'year':
      from = new Date(now.getFullYear(), 0, 1); break
    case 'all':
      from = new Date(0); break
  }
  return { from, to }
}

// Balance-sheet account types are cumulative (all time), not period-only
const CUMULATIVE_TYPES = new Set(['asset', 'liability', 'equity'])

interface AccountBalance {
  id: string
  name: string
  type: string
  parentId: string | null
  isGroup: boolean
  debit: number
  credit: number
  balance: number // debit-normal types: debit-credit; credit-normal: credit-debit
}

function computeBalances(
  accounts: Account[],
  entries: JournalEntry[],
  from: Date,
  to: Date,
): Map<string, AccountBalance> {
  const result = new Map<string, AccountBalance>()

  // init all accounts
  for (const a of accounts) {
    result.set(a.id, {
      id: a.id,
      name: a.name,
      type: a.type,
      parentId: a.parent_id ?? null,
      isGroup: a.is_group,
      debit: 0,
      credit: 0,
      balance: 0,
    })
  }

  for (const entry of entries) {
    const entryDate = new Date(entry.date)
    for (const line of entry.lines) {
      const ab = result.get(line.account_id)
      if (!ab) continue
      const isCumulative = CUMULATIVE_TYPES.has(ab.type)
      // for period-only accounts: filter by window; for cumulative: always include up to `to`
      const inWindow = isCumulative
        ? entryDate <= to
        : entryDate >= from && entryDate <= to
      if (!inWindow) continue
      const amt = parseFloat(line.amount)
      if (line.side === 'debit') ab.debit += amt
      else ab.credit += amt
    }
  }

  // compute balance (debit-normal for asset/expense, credit-normal for liability/equity/income)
  const DEBIT_NORMAL = new Set(['asset', 'expense'])
  for (const ab of result.values()) {
    ab.balance = DEBIT_NORMAL.has(ab.type)
      ? ab.debit - ab.credit
      : ab.credit - ab.debit
  }

  // aggregate group accounts (sum children balances)
  // do multiple passes until stable (handles nested groups)
  for (let pass = 0; pass < 10; pass++) {
    for (const ab of result.values()) {
      if (!ab.isGroup) continue
      let d = 0, c = 0
      for (const child of result.values()) {
        if (child.parentId === ab.id) { d += child.debit; c += child.credit }
      }
      ab.debit = d; ab.credit = c
      ab.balance = DEBIT_NORMAL.has(ab.type) ? d - c : c - d
    }
  }

  return result
}

const fmtVND = (n: number) =>
  n === 0 ? '—' : `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}`

// ── Trial Balance ─────────────────────────────────────────────────────────

function TrialBalance({ balances, accounts }: { balances: Map<string, AccountBalance>; accounts: Account[] }) {
  const leafAccounts = accounts.filter(a => !a.is_group)
  const rows = leafAccounts.map(a => balances.get(a.id)!).filter(Boolean)
  const totalDebit = rows.reduce((s, r) => s + r.debit, 0)
  const totalCredit = rows.reduce((s, r) => s + r.credit, 0)

  const columns = [
    { title: 'Account', dataIndex: 'name' },
    { title: 'Type', dataIndex: 'type', width: 100 },
    {
      title: 'Debit', dataIndex: 'debit', width: 160, align: 'right' as const,
      render: (v: number) => <Text>{fmtVND(v)}</Text>,
    },
    {
      title: 'Credit', dataIndex: 'credit', width: 160, align: 'right' as const,
      render: (v: number) => <Text>{fmtVND(v)}</Text>,
    },
  ]

  return (
    <Table
      dataSource={rows}
      rowKey="id"
      columns={columns}
      size="small"
      pagination={false}
      summary={() => (
        <Table.Summary.Row>
          <Table.Summary.Cell index={0} colSpan={2}><Text strong>Total</Text></Table.Summary.Cell>
          <Table.Summary.Cell index={2} align="right"><Text strong>{fmtVND(totalDebit)}</Text></Table.Summary.Cell>
          <Table.Summary.Cell index={3} align="right">
            <Text strong style={{ color: Math.abs(totalDebit - totalCredit) > 0.01 ? 'red' : undefined }}>
              {fmtVND(totalCredit)}
            </Text>
          </Table.Summary.Cell>
        </Table.Summary.Row>
      )}
    />
  )
}

// ── Balance Sheet ──────────────────────────────────────────────────────────

function BalanceSection({ title, type, balances, accounts }: {
  title: string; type: string; balances: Map<string, AccountBalance>; accounts: Account[]
}) {
  const relevant = accounts.filter(a => a.type === type)
  const rows = relevant.map(a => balances.get(a.id)!).filter(Boolean)
  const total = rows.filter(r => !r.isGroup || !relevant.some(a => a.parent_id === r.id /* direct children exist */))
    // use leaf totals aggregated in groups — just sum top-level groups + ungrouped leaves
    .reduce((s, r) => {
      // only count top-level items (no parent or parent is different type)
      const parentAb = r.parentId ? balances.get(r.parentId) : undefined
      if (!parentAb || parentAb.type !== type) return s + r.balance
      return s
    }, 0)

  const columns = [
    {
      title, dataIndex: 'name',
      render: (name: string, row: AccountBalance) => (
        <span style={{ paddingLeft: row.isGroup ? 0 : 16, fontWeight: row.isGroup ? 600 : 400 }}>{name}</span>
      ),
    },
    {
      title: 'Balance', dataIndex: 'balance', width: 180, align: 'right' as const,
      render: (v: number, row: AccountBalance) => (
        <Text strong={row.isGroup}>{fmtVND(v)}</Text>
      ),
    },
  ]

  return (
    <Table
      dataSource={rows}
      rowKey="id"
      columns={columns}
      size="small"
      pagination={false}
      style={{ marginBottom: 16 }}
      summary={() => (
        <Table.Summary.Row>
          <Table.Summary.Cell index={0}><Text strong>Total {title}</Text></Table.Summary.Cell>
          <Table.Summary.Cell index={1} align="right"><Text strong>{fmtVND(total)}</Text></Table.Summary.Cell>
        </Table.Summary.Row>
      )}
    />
  )
}

function BalanceSheet({ balances, accounts }: { balances: Map<string, AccountBalance>; accounts: Account[] }) {
  const assetTotal = [...balances.values()].filter(b => b.type === 'asset' && (!b.parentId || balances.get(b.parentId)?.type !== 'asset')).reduce((s, b) => s + b.balance, 0)
  const liabTotal = [...balances.values()].filter(b => b.type === 'liability' && (!b.parentId || balances.get(b.parentId)?.type !== 'liability')).reduce((s, b) => s + b.balance, 0)
  const equityTotal = [...balances.values()].filter(b => b.type === 'equity' && (!b.parentId || balances.get(b.parentId)?.type !== 'equity')).reduce((s, b) => s + b.balance, 0)
  const balanced = Math.abs(assetTotal - liabTotal - equityTotal) < 1

  return (
    <>
      <BalanceSection title="Assets" type="asset" balances={balances} accounts={accounts} />
      <BalanceSection title="Liabilities" type="liability" balances={balances} accounts={accounts} />
      <BalanceSection title="Equity" type="equity" balances={balances} accounts={accounts} />
      <div style={{ textAlign: 'right', padding: '8px 0', color: balanced ? '#52c41a' : 'red' }}>
        {balanced
          ? `✓ Balanced: Assets ${fmtVND(assetTotal)} = Liabilities + Equity ${fmtVND(liabTotal + equityTotal)}`
          : `⚠ Unbalanced: Assets ${fmtVND(assetTotal)} ≠ Liabilities + Equity ${fmtVND(liabTotal + equityTotal)}`}
      </div>
    </>
  )
}

// ── P&L ───────────────────────────────────────────────────────────────────

function PnLSection({ title, type, balances, accounts }: {
  title: string; type: string; balances: Map<string, AccountBalance>; accounts: Account[]
}) {
  const relevant = accounts.filter(a => a.type === type)
  const rows = relevant.map(a => balances.get(a.id)!).filter(Boolean)
  const total = rows.filter(r => {
    const parentAb = r.parentId ? balances.get(r.parentId) : undefined
    return !parentAb || parentAb.type !== type
  }).reduce((s, r) => s + r.balance, 0)

  const columns = [
    {
      title, dataIndex: 'name',
      render: (name: string, row: AccountBalance) => (
        <span style={{ paddingLeft: row.isGroup ? 0 : 16, fontWeight: row.isGroup ? 600 : 400 }}>{name}</span>
      ),
    },
    {
      title: 'Amount', dataIndex: 'balance', width: 180, align: 'right' as const,
      render: (v: number, row: AccountBalance) => <Text strong={row.isGroup}>{fmtVND(v)}</Text>,
    },
  ]

  return (
    <Table
      dataSource={rows}
      rowKey="id"
      columns={columns}
      size="small"
      pagination={false}
      style={{ marginBottom: 16 }}
      summary={() => (
        <Table.Summary.Row>
          <Table.Summary.Cell index={0}><Text strong>Total {title}</Text></Table.Summary.Cell>
          <Table.Summary.Cell index={1} align="right"><Text strong>{fmtVND(total)}</Text></Table.Summary.Cell>
        </Table.Summary.Row>
      )}
    />
  )
}

function ProfitAndLoss({ balances, accounts }: { balances: Map<string, AccountBalance>; accounts: Account[] }) {
  const incomeTotal = [...balances.values()].filter(b => b.type === 'income' && (!b.parentId || balances.get(b.parentId)?.type !== 'income')).reduce((s, b) => s + b.balance, 0)
  const expenseTotal = [...balances.values()].filter(b => b.type === 'expense' && (!b.parentId || balances.get(b.parentId)?.type !== 'expense')).reduce((s, b) => s + b.balance, 0)
  const netIncome = incomeTotal - expenseTotal

  return (
    <>
      <PnLSection title="Income" type="income" balances={balances} accounts={accounts} />
      <PnLSection title="Expenses" type="expense" balances={balances} accounts={accounts} />
      <div style={{
        textAlign: 'right', padding: '12px 0', fontSize: 18, fontWeight: 700,
        color: netIncome >= 0 ? '#52c41a' : '#ff4d4f',
      }}>
        Net Income: {netIncome < 0 ? '-' : ''}{fmtVND(netIncome)}
      </div>
    </>
  )
}

// ── Main component ────────────────────────────────────────────────────────

interface ReportsTabProps {
  accounts: Account[]
  entries: JournalEntry[]
}

export function ReportsTab({ accounts, entries }: ReportsTabProps) {
  const [window, setWindow] = useState<Window>('month')

  const balances = useMemo(() => {
    const { from, to } = windowBounds(window)
    return computeBalances(accounts, entries, from, to)
  }, [accounts, entries, window])

  const windowOptions = [
    { label: 'Today', value: 'today' },
    { label: 'This Month', value: 'month' },
    { label: 'This Quarter', value: 'quarter' },
    { label: 'This Year', value: 'year' },
    { label: 'All Time', value: 'all' },
  ]

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Segmented
          options={windowOptions}
          value={window}
          onChange={v => setWindow(v as Window)}
        />
      </div>
      <Tabs
        items={[
          { key: 'trial', label: 'Trial Balance', children: <TrialBalance balances={balances} accounts={accounts} /> },
          { key: 'bs', label: 'Balance Sheet', children: <BalanceSheet balances={balances} accounts={accounts} /> },
          { key: 'pl', label: 'P&L', children: <ProfitAndLoss balances={balances} accounts={accounts} /> },
        ]}
      />
    </>
  )
}
```

- [ ] **Step 2: Wire ReportsTab into AccountingPage**

In `frontend/src/pages/AccountingPage.tsx`:

Add import at top:
```typescript
import { ReportsTab } from './ReportsTab'
```

Replace the `AccountingPage` export at the bottom of the file:

```typescript
export function AccountingPage() {
  const { data: accounts = [] } = useQuery({ queryKey: ['accounts'], queryFn: getAccounts })
  const { data: entries = [] } = useQuery({ queryKey: ['journal-entries'], queryFn: getJournalEntries })

  return (
    <Tabs
      defaultActiveKey="journal"
      items={[
        { key: 'journal', label: 'Journal', children: <JournalTab /> },
        { key: 'accounts', label: 'Accounts', children: <AccountsTab /> },
        { key: 'assets', label: 'Assets', children: <AssetsTab /> },
        { key: 'reports', label: 'Reports', children: <ReportsTab accounts={accounts} entries={entries} /> },
      ]}
    />
  )
}
```

Note: `AccountsTab` and `JournalTab` each independently fetch `accounts` and `entries` via React Query — the shared fetches in `AccountingPage` will be deduplicated by React Query's cache automatically.

- [ ] **Step 3: Lint and build**

```bash
cd frontend && npm run lint && npm run build
```
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/ReportsTab.tsx frontend/src/pages/AccountingPage.tsx
git commit -m "feat(accounting): Reports tab with Trial Balance, Balance Sheet, P&L"
```

---

## Self-Review

**Spec coverage:**
- ✅ DELETE /accounts/{id} with child guard → Task 3
- ✅ DELETE /accounts/{id} with journal-line guard → Task 3
- ✅ Delete button + Popconfirm in AccountsTab → Task 4
- ✅ Reports tab added to AccountingPage → Task 5
- ✅ Time window: Today/Month/Quarter/Year/All → Task 5 `windowBounds`
- ✅ Trial Balance: leaf accounts, debit/credit columns, footer totals → Task 5 `TrialBalance`
- ✅ Balance Sheet: cumulative, grouped by parent, Assets=Liabilities+Equity check → Task 5 `BalanceSheet`
- ✅ P&L: period-only, income/expense sections, net income row → Task 5 `ProfitAndLoss`
- ✅ No new backend endpoints — client-side aggregation → Task 5

**Type consistency:**
- `AccountBalance` defined once, used by all three report components ✅
- `Window` type matches `windowBounds` param and `Segmented` value ✅
- `ReportsTabProps` matches usage in AccountingPage ✅

**Coverage:** Task 3 adds 4 new test cases for the Delete handler. The handler adds ~20 lines to `accounting_accounts.go`; existing file is at 84.4% coverage — new handler tests should keep it above 80%.
