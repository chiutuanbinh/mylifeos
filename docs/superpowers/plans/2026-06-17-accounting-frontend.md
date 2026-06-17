# Accounting Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a full accounting UI — chart of accounts management, journal entry recording, and live net worth display — wired to the new backend double-entry API.

**Architecture:** New `/accounting` page with two tabs (Accounts, Journal). Accounting API types and endpoint functions live in existing `src/api/` files. Live net worth widget replaces the old snapshot card in WealthPage and DashboardPage. No smart-inference in this phase — users supply both journal lines explicitly.

**Tech Stack:** React 19, TypeScript, Ant Design 6, TanStack Query v5, Vitest + Testing Library, `apiClient` (axios).

## Global Constraints

- All API calls go through `apiClient` from `src/api/client.ts` (Bearer token injected by interceptor)
- Base URL: `import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'`
- Currency display: `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}` for VND amounts
- Ant Design only for UI — no extra component libraries
- All monetary amounts sent to backend as strings (decimal) and displayed as VND integers
- TanStack Query keys: `['accounts']`, `['journal-entries']`, `['journal-networth']`
- `npm run lint && npm run build` must pass clean after each task
- Tests run with `npx vitest run` from `frontend/`
- No mock of `apiClient` — use `vi.mock('../api/endpoints')` at the module level in tests
- Account types: `'asset' | 'liability' | 'equity' | 'income' | 'expense'`
- Journal line sides: `'debit' | 'credit'`

---

### Task 1: API Types and Endpoint Functions

**Files:**
- Modify: `frontend/src/api/types.ts`
- Modify: `frontend/src/api/endpoints.ts`
- Test: `frontend/src/api/accounting.test.ts` (new)

**Interfaces:**
- Produces:
  - `Account` type
  - `JournalEntry` type
  - `JournalLine` type
  - `NetWorthResult` type
  - `getAccounts(): Promise<Account[]>`
  - `createAccount(data): Promise<Account>`
  - `createJournalEntry(data): Promise<{ id: string }>`
  - `getJournalNetWorth(): Promise<NetWorthResult>`

- [ ] **Step 1: Write failing tests**

Create `frontend/src/api/accounting.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getAccounts, createAccount, createJournalEntry, getJournalNetWorth } from './endpoints'
import { apiClient } from './client'
import type { Account, NetWorthResult, CreateJournalEntryRequest } from './types'

vi.mock('./client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
  },
}))

const mockGet = vi.mocked(apiClient.get)
const mockPost = vi.mocked(apiClient.post)

beforeEach(() => { vi.clearAllMocks() })

describe('getAccounts', () => {
  it('calls GET /accounts and returns data', async () => {
    const accounts: Account[] = [
      { id: 'a1', user_id: 'u1', parent_id: null, name: 'Checking', type: 'asset',
        currency: 'VND', is_group: false, archived: false, sort_order: 0 },
    ]
    mockGet.mockResolvedValueOnce({ data: accounts })
    const result = await getAccounts()
    expect(mockGet).toHaveBeenCalledWith('/accounts')
    expect(result).toEqual(accounts)
  })
})

describe('createAccount', () => {
  it('calls POST /accounts with payload', async () => {
    const payload = { name: 'Savings', type: 'asset' as const, currency: 'VND',
      is_group: false, sort_order: 1, parent_id: null }
    const created: Account = { id: 'a2', user_id: 'u1', ...payload, archived: false }
    mockPost.mockResolvedValueOnce({ data: created })
    const result = await createAccount(payload)
    expect(mockPost).toHaveBeenCalledWith('/accounts', payload)
    expect(result).toEqual(created)
  })
})

describe('createJournalEntry', () => {
  it('calls POST /journal/entries and returns id', async () => {
    const req: CreateJournalEntryRequest = {
      date: '2026-06-17', description: 'Test', memo: '',
      lines: [
        { account_id: 'a1', amount: '100000', currency: 'VND', side: 'debit' },
        { account_id: 'a2', amount: '100000', currency: 'VND', side: 'credit' },
      ],
    }
    mockPost.mockResolvedValueOnce({ data: { id: 'e1' } })
    const result = await createJournalEntry(req)
    expect(mockPost).toHaveBeenCalledWith('/journal/entries', req)
    expect(result).toEqual({ id: 'e1' })
  })
})

describe('getJournalNetWorth', () => {
  it('calls GET /journal/networth and returns result', async () => {
    const nw: NetWorthResult = { net_worth: '5000000', currency: 'VND' }
    mockGet.mockResolvedValueOnce({ data: nw })
    const result = await getJournalNetWorth()
    expect(mockGet).toHaveBeenCalledWith('/journal/networth')
    expect(result).toEqual(nw)
  })
})
```

- [ ] **Step 2: Run test — verify it fails**

```bash
cd frontend && npx vitest run src/api/accounting.test.ts
```

Expected: FAIL — `getAccounts`, `createAccount`, etc. not exported from endpoints.

- [ ] **Step 3: Add types to `src/api/types.ts`**

Append after the last existing interface:

```typescript
export interface Account {
  id: string
  user_id: string
  parent_id: string | null
  name: string
  type: 'asset' | 'liability' | 'equity' | 'income' | 'expense'
  currency: string
  is_group: boolean
  archived: boolean
  sort_order: number
}

export interface JournalLine {
  id: string
  entry_id: string
  account_id: string
  amount: string
  currency: string
  side: 'debit' | 'credit'
}

export interface JournalEntry {
  id: string
  user_id: string
  date: string
  description: string
  memo: string
  lines: JournalLine[]
}

export interface CreateAccountRequest {
  name: string
  type: 'asset' | 'liability' | 'equity' | 'income' | 'expense'
  currency: string
  is_group: boolean
  sort_order: number
  parent_id: string | null
}

export interface CreateJournalEntryRequest {
  date: string
  description: string
  memo: string
  lines: {
    account_id: string
    amount: string
    currency: string
    side: 'debit' | 'credit'
  }[]
}

export interface NetWorthResult {
  net_worth: string
  currency: string
}
```

- [ ] **Step 4: Add endpoint functions to `src/api/endpoints.ts`**

Add these imports to the existing import block at the top of `endpoints.ts`:

```typescript
import type {
  // existing imports...
  Account, CreateAccountRequest, CreateJournalEntryRequest, NetWorthResult,
} from './types'
```

Then append at the end of the file:

```typescript
// Accounting
export const getAccounts = () =>
  apiClient.get<Account[]>('/accounts').then(r => r.data)

export const createAccount = (data: CreateAccountRequest) =>
  apiClient.post<Account>('/accounts', data).then(r => r.data)

export const createJournalEntry = (data: CreateJournalEntryRequest) =>
  apiClient.post<{ id: string }>('/journal/entries', data).then(r => r.data)

export const getJournalNetWorth = () =>
  apiClient.get<NetWorthResult>('/journal/networth').then(r => r.data)
```

- [ ] **Step 5: Run test — verify it passes**

```bash
cd frontend && npx vitest run src/api/accounting.test.ts
```

Expected: PASS — 4 tests pass.

- [ ] **Step 6: Lint + build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: clean (no errors, no new warnings).

- [ ] **Step 7: Commit**

```bash
git add frontend/src/api/types.ts frontend/src/api/endpoints.ts frontend/src/api/accounting.test.ts
git commit -m "feat(accounting-ui): add API types and endpoint functions"
```

---

### Task 2: Chart of Accounts Tab

**Files:**
- Create: `frontend/src/pages/AccountingPage.tsx`
- Create: `frontend/src/pages/AccountingPage.test.tsx`
- Modify: `frontend/src/App.tsx` (add route)
- Modify: `frontend/src/components/AppShell.tsx` (add nav entry)

**Interfaces:**
- Consumes: `getAccounts()`, `createAccount()`, `Account`, `CreateAccountRequest` from Task 1
- Produces: `<AccountingPage />` — exported from `AccountingPage.tsx`; renders Tabs with "Accounts" tab

- [ ] **Step 1: Write failing test**

Create `frontend/src/pages/AccountingPage.test.tsx`:

```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AccountingPage } from './AccountingPage'
import * as endpoints from '../api/endpoints'
import type { Account } from '../api/types'

vi.mock('../api/endpoints')

const mockGetAccounts = vi.mocked(endpoints.getAccounts)
const mockCreateAccount = vi.mocked(endpoints.createAccount)

function wrap(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

const sampleAccounts: Account[] = [
  { id: 'a1', user_id: 'u1', parent_id: null, name: 'Cash', type: 'asset',
    currency: 'VND', is_group: false, archived: false, sort_order: 0 },
  { id: 'a2', user_id: 'u1', parent_id: null, name: 'Assets', type: 'asset',
    currency: 'VND', is_group: true, archived: false, sort_order: 0 },
]

beforeEach(() => { vi.clearAllMocks() })

describe('AccountingPage — Accounts tab', () => {
  it('renders account list', async () => {
    mockGetAccounts.mockResolvedValueOnce(sampleAccounts)
    wrap(<AccountingPage />)
    await waitFor(() => expect(screen.getByText('Cash')).toBeInTheDocument())
    expect(screen.getByText('asset')).toBeInTheDocument()
  })

  it('opens create modal on Add button click', async () => {
    mockGetAccounts.mockResolvedValueOnce([])
    wrap(<AccountingPage />)
    await waitFor(() => screen.getByRole('button', { name: /add account/i }))
    fireEvent.click(screen.getByRole('button', { name: /add account/i }))
    expect(screen.getByText(/new account/i)).toBeInTheDocument()
  })

  it('calls createAccount on form submit', async () => {
    mockGetAccounts.mockResolvedValue([])
    mockCreateAccount.mockResolvedValueOnce(
      { id: 'a3', user_id: 'u1', parent_id: null, name: 'Savings',
        type: 'asset', currency: 'VND', is_group: false, archived: false, sort_order: 1 }
    )
    wrap(<AccountingPage />)
    await waitFor(() => screen.getByRole('button', { name: /add account/i }))
    fireEvent.click(screen.getByRole('button', { name: /add account/i }))
    fireEvent.change(screen.getByLabelText(/name/i), { target: { value: 'Savings' } })
    fireEvent.click(screen.getByRole('button', { name: /^save$/i }))
    await waitFor(() => expect(mockCreateAccount).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Savings', type: 'asset', currency: 'VND' })
    ))
  })
})
```

- [ ] **Step 2: Run test — verify it fails**

```bash
cd frontend && npx vitest run src/pages/AccountingPage.test.tsx
```

Expected: FAIL — `AccountingPage` not found.

- [ ] **Step 3: Implement `AccountingPage.tsx`**

Create `frontend/src/pages/AccountingPage.tsx`:

```typescript
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Tabs, Card, Table, Tag, Button, Form, Input, Select, Switch,
  InputNumber, Modal, Spin, Badge,
} from 'antd'
import { PlusOutlined, FolderOutlined, FileOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { getAccounts, createAccount, createJournalEntry, getJournalNetWorth } from '../api/endpoints'
import type { Account, CreateAccountRequest, CreateJournalEntryRequest } from '../api/types'

const TYPE_COLORS: Record<string, string> = {
  asset: 'green', liability: 'red', equity: 'blue', income: 'cyan', expense: 'orange',
}

const fmtVND = (s: string) => `₫${Math.round(Math.abs(parseFloat(s))).toLocaleString('vi-VN')}`

function AccountsTab() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: accounts = [], isLoading } = useQuery({
    queryKey: ['accounts'],
    queryFn: getAccounts,
  })

  const createMutation = useMutation({
    mutationFn: createAccount,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['accounts'] })
      setAddOpen(false)
      form.resetFields()
    },
  })

  const groupAccounts = accounts.filter(a => a.is_group)

  const columns: ColumnsType<Account> = [
    {
      title: 'Name', dataIndex: 'name',
      render: (name, row) => (
        <span>
          {row.is_group ? <FolderOutlined style={{ marginRight: 6, color: '#faad14' }} /> : <FileOutlined style={{ marginRight: 6, color: '#8c8c8c' }} />}
          {name}
          {row.archived && <Badge count="archived" style={{ marginLeft: 8, backgroundColor: '#d9d9d9', color: '#595959', fontSize: 10 }} />}
        </span>
      ),
    },
    {
      title: 'Type', dataIndex: 'type', width: 110,
      render: t => <Tag color={TYPE_COLORS[t]}>{t}</Tag>,
    },
    { title: 'Currency', dataIndex: 'currency', width: 90 },
    {
      title: 'Parent', dataIndex: 'parent_id', width: 160,
      render: pid => accounts.find(a => a.id === pid)?.name ?? '—',
    },
  ]

  return (
    <>
      <Card
        size="small"
        title="Chart of Accounts"
        extra={
          <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>
            Add Account
          </Button>
        }
      >
        {isLoading ? <Spin /> : (
          <Table
            dataSource={accounts}
            columns={columns}
            size="small"
            rowKey="id"
            pagination={false}
            scroll={{ x: true }}
          />
        )}
      </Card>

      <Modal
        title="New Account"
        open={addOpen}
        onCancel={() => { setAddOpen(false); form.resetFields() }}
        footer={null}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ type: 'asset', currency: 'VND', is_group: false, sort_order: 0 }}
          onFinish={(values: CreateAccountRequest) => createMutation.mutate(values)}
        >
          <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Required' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="Type" rules={[{ required: true }]}>
            <Select options={['asset','liability','equity','income','expense'].map(t => ({ value: t, label: t }))} />
          </Form.Item>
          <Form.Item name="currency" label="Currency">
            <Input disabled />
          </Form.Item>
          <Form.Item name="parent_id" label="Parent Group">
            <Select
              allowClear
              placeholder="None (root)"
              options={groupAccounts.map(a => ({ value: a.id, label: a.name }))}
            />
          </Form.Item>
          <Form.Item name="is_group" label="Is Group?" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="sort_order" label="Sort Order">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={createMutation.isPending} block>
              Save
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

function JournalTab() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: accounts = [] } = useQuery({ queryKey: ['accounts'], queryFn: getAccounts })
  const { data: nw } = useQuery({ queryKey: ['journal-networth'], queryFn: getJournalNetWorth })

  const leafAccounts = accounts.filter(a => !a.is_group)

  const recordMutation = useMutation({
    mutationFn: (values: { date: string; description: string; memo: string; lines: { account_id: string; amount: number; side: 'debit' | 'credit' }[] }) => {
      const req: CreateJournalEntryRequest = {
        date: values.date,
        description: values.description,
        memo: values.memo ?? '',
        lines: values.lines.map(l => ({
          account_id: l.account_id,
          amount: String(l.amount),
          currency: 'VND',
          side: l.side,
        })),
      }
      return createJournalEntry(req)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['journal-networth'] })
      setAddOpen(false)
      form.resetFields()
    },
  })

  return (
    <>
      {nw && (
        <Card size="small" style={{ marginBottom: 12 }}>
          <div style={{ fontSize: 12, color: '#999' }}>Live Net Worth</div>
          <div style={{ fontSize: 28, fontWeight: 700, color: '#1677ff' }}>
            {fmtVND(nw.net_worth)}
          </div>
        </Card>
      )}
      <Card
        size="small"
        title="Journal"
        extra={
          <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>
            Record Entry
          </Button>
        }
      >
        <div style={{ color: '#999', padding: '24px 0', textAlign: 'center' }}>
          Journal history coming soon
        </div>
      </Card>

      <Modal
        title="Record Journal Entry"
        open={addOpen}
        onCancel={() => { setAddOpen(false); form.resetFields() }}
        footer={null}
        width={560}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ lines: [{ side: 'debit' }, { side: 'credit' }] }}
          onFinish={values => recordMutation.mutate(values)}
        >
          <Form.Item name="date" label="Date" rules={[{ required: true }]}>
            <Input type="date" />
          </Form.Item>
          <Form.Item name="description" label="Description" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="memo" label="Memo">
            <Input />
          </Form.Item>

          <Form.List name="lines">
            {(fields, { add, remove }) => (
              <>
                {fields.map((field, idx) => (
                  <Card key={field.key} size="small" style={{ marginBottom: 8 }}
                    title={`Line ${idx + 1}`}
                    extra={fields.length > 2 && (
                      <Button type="text" size="small" danger onClick={() => remove(field.name)}>Remove</Button>
                    )}
                  >
                    <Form.Item name={[field.name, 'account_id']} label="Account" rules={[{ required: true }]}>
                      <Select
                        showSearch
                        optionFilterProp="label"
                        options={leafAccounts.map(a => ({ value: a.id, label: `${a.name} (${a.type})` }))}
                      />
                    </Form.Item>
                    <Form.Item name={[field.name, 'amount']} label="Amount (VND)" rules={[{ required: true }]}>
                      <InputNumber min={1} style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={[field.name, 'side']} label="Side" rules={[{ required: true }]}>
                      <Select options={[{ value: 'debit', label: 'Debit' }, { value: 'credit', label: 'Credit' }]} />
                    </Form.Item>
                  </Card>
                ))}
                <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add({ side: 'debit' })}>
                  Add Line
                </Button>
              </>
            )}
          </Form.List>

          <Form.Item style={{ marginTop: 16 }}>
            <Button type="primary" htmlType="submit" loading={recordMutation.isPending} block>
              Post Entry
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

export function AccountingPage() {
  return (
    <Tabs
      defaultActiveKey="accounts"
      items={[
        { key: 'accounts', label: 'Accounts', children: <AccountsTab /> },
        { key: 'journal', label: 'Journal', children: <JournalTab /> },
      ]}
    />
  )
}
```

- [ ] **Step 4: Run test — verify it passes**

```bash
cd frontend && npx vitest run src/pages/AccountingPage.test.tsx
```

Expected: PASS — 3 tests pass.

- [ ] **Step 5: Add route in `App.tsx`**

Add import alongside existing page imports:
```typescript
import { AccountingPage } from './pages/AccountingPage'
```

Add route inside the inner `<Routes>` block after the `/wealth` route:
```typescript
<Route path="/accounting" element={<AccountingPage />} />
```

- [ ] **Step 6: Add nav entry in `AppShell.tsx`**

Add `AccountOutlined` to the icon imports:
```typescript
import {
  DashboardOutlined, DollarOutlined, TrophyOutlined,
  CalendarOutlined, SettingOutlined, AccountBookOutlined,
  BellOutlined, MenuFoldOutlined, MenuUnfoldOutlined, LogoutOutlined,
} from '@ant-design/icons'
```

Add to the `NAV` array after the `/wealth` entry:
```typescript
{ key: '/accounting', icon: <AccountBookOutlined />, label: 'Accounting' },
```

Add to `BOTTOM_NAV` array (replacing one of the less-used items or adding fifth):
```typescript
{ key: '/accounting', icon: <AccountBookOutlined />, label: 'Accounting' },
```

Add to `TITLES`:
```typescript
'/accounting': 'Accounting',
```

- [ ] **Step 7: Lint + build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/pages/AccountingPage.tsx frontend/src/pages/AccountingPage.test.tsx \
        frontend/src/App.tsx frontend/src/components/AppShell.tsx
git commit -m "feat(accounting-ui): add AccountingPage with chart of accounts and journal entry form"
```

---

### Task 3: Live Net Worth in WealthPage and DashboardPage

**Files:**
- Modify: `frontend/src/pages/WealthPage.tsx`
- Modify: `frontend/src/pages/DashboardPage.tsx`
- Test: `frontend/src/pages/LiveNetWorth.test.tsx` (new)

**Interfaces:**
- Consumes: `getJournalNetWorth()`, `NetWorthResult` from Task 1
- Produces: `<LiveNetWorthCard />` — exported component, renders current live net worth from journal API

- [ ] **Step 1: Write failing test**

Create `frontend/src/pages/LiveNetWorth.test.tsx`:

```typescript
import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { LiveNetWorthCard } from './LiveNetWorthCard'
import * as endpoints from '../api/endpoints'

vi.mock('../api/endpoints')
const mockGetJournalNetWorth = vi.mocked(endpoints.getJournalNetWorth)

function wrap(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

beforeEach(() => { vi.clearAllMocks() })

describe('LiveNetWorthCard', () => {
  it('displays net worth from journal API', async () => {
    mockGetJournalNetWorth.mockResolvedValueOnce({ net_worth: '123456789', currency: 'VND' })
    wrap(<LiveNetWorthCard />)
    await waitFor(() => expect(screen.getByText(/123\.456\.789/)).toBeInTheDocument())
  })

  it('shows zero when net worth is 0', async () => {
    mockGetJournalNetWorth.mockResolvedValueOnce({ net_worth: '0', currency: 'VND' })
    wrap(<LiveNetWorthCard />)
    await waitFor(() => expect(screen.getByText(/₫0/)).toBeInTheDocument())
  })

  it('shows loading state initially', () => {
    mockGetJournalNetWorth.mockImplementation(() => new Promise(() => {}))
    wrap(<LiveNetWorthCard />)
    expect(screen.getByRole('img', { hidden: true })).toBeInTheDocument() // Ant Spin
  })
})
```

- [ ] **Step 2: Run test — verify it fails**

```bash
cd frontend && npx vitest run src/pages/LiveNetWorth.test.tsx
```

Expected: FAIL — `LiveNetWorthCard` not found.

- [ ] **Step 3: Create `LiveNetWorthCard.tsx`**

Create `frontend/src/pages/LiveNetWorthCard.tsx`:

```typescript
import { useQuery } from '@tanstack/react-query'
import { Card, Spin } from 'antd'
import { getJournalNetWorth } from '../api/endpoints'

const fmtVND = (s: string) => {
  const n = parseFloat(s)
  return `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}`
}

export function LiveNetWorthCard() {
  const { data, isLoading } = useQuery({
    queryKey: ['journal-networth'],
    queryFn: getJournalNetWorth,
    refetchInterval: 30_000,
  })

  return (
    <Card size="small">
      <div style={{ fontSize: 12, color: '#999' }}>Live Net Worth</div>
      {isLoading || !data
        ? <Spin />
        : <div style={{ fontSize: 28, fontWeight: 700, color: '#1677ff' }}>
            {fmtVND(data.net_worth)}
          </div>
      }
    </Card>
  )
}
```

- [ ] **Step 4: Run test — verify it passes**

```bash
cd frontend && npx vitest run src/pages/LiveNetWorth.test.tsx
```

Expected: PASS — 3 tests pass.

- [ ] **Step 5: Integrate into WealthPage**

In `frontend/src/pages/WealthPage.tsx`, find the existing net worth snapshot card (the one that calls `getNetWorthSnapshots` or displays net worth) and add `<LiveNetWorthCard />` above it.

Add import at top:
```typescript
import { LiveNetWorthCard } from './LiveNetWorthCard'
```

Locate the existing `NetWorthTab` function (or the section that renders the net worth trend). Add `<LiveNetWorthCard />` as the first element inside the tab content, before the snapshot chart:

```typescript
// At the top of the NetWorthTab JSX return, before the existing chart/cards:
<Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
  <Col xs={24} sm={8}>
    <LiveNetWorthCard />
  </Col>
</Row>
```

- [ ] **Step 6: Integrate into DashboardPage**

In `frontend/src/pages/DashboardPage.tsx`, find where the existing net worth number is displayed (likely from `getDashboardSummary`). Add the live card alongside it.

Add import:
```typescript
import { LiveNetWorthCard } from './LiveNetWorthCard'
```

Find the net worth summary card render and add `<LiveNetWorthCard />` in the summary cards row. If DashboardPage has a `<Col>` grid of summary cards, add one more `<Col xs={24} sm={8}><LiveNetWorthCard /></Col>`.

- [ ] **Step 7: Lint + build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/pages/LiveNetWorthCard.tsx frontend/src/pages/LiveNetWorth.test.tsx \
        frontend/src/pages/WealthPage.tsx frontend/src/pages/DashboardPage.tsx
git commit -m "feat(accounting-ui): add live net worth card wired to journal API"
```

---

## Self-Review

### Spec coverage

| Requirement | Task |
|---|---|
| API types for Account, JournalEntry, NetWorthResult | Task 1 |
| Endpoint functions for all 4 accounting routes | Task 1 |
| Chart of accounts list with type badges | Task 2 |
| Create account form (name, type, parent, is_group) | Task 2 |
| Journal entry form with dynamic lines (debit/credit) | Task 2 |
| `/accounting` route and nav entry | Task 2 |
| Live net worth card from `/journal/networth` | Task 3 |
| Live net worth visible in WealthPage | Task 3 |
| Live net worth visible in DashboardPage | Task 3 |

### Gaps

- **Journal history list** — the Journal tab shows "coming soon" placeholder. Listing past entries requires a `GET /journal/entries` backend endpoint (currently only `POST` exists). Not implementing — no backend endpoint to wire to.
- **Smart-default entry** (infer both legs from one account + category) — Phase 2, requires backend inference or frontend category→account mapping setup that doesn't exist yet.

### Placeholder scan

No TBDs. Step 5 and Step 6 in Task 3 say "find the existing..." — those are locate-then-modify instructions, not placeholders. The implementer reads the file to find the exact location.

### Type consistency

- `Account.type` — `'asset' | 'liability' | 'equity' | 'income' | 'expense'` — consistent across types.ts, AccountingPage.tsx, test
- `CreateJournalEntryRequest.lines[].amount` — `string` — consistent (backend expects decimal string)
- `NetWorthResult.net_worth` — `string` — consistent; `fmtVND` parses it with `parseFloat`
- `getJournalNetWorth` query key — `['journal-networth']` — same in AccountingPage.tsx, LiveNetWorthCard.tsx, test
