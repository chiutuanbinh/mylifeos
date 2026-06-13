# PR2: Wealth Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace separate Finance and Inventory pages with a unified Wealth page (Tabs: Transactions | Budgets | Assets), update dashboard net worth display, and add asset edit functionality.

**Architecture:** Purely frontend. `WealthPage.tsx` replaces both `FinancePage.tsx` and `InventoryPage.tsx`. Route `/inventory` → `/wealth`; `/finance` removed. Nav updated. Dashboard card updated to show real net worth + real % change.

**Tech Stack:** React, TypeScript, Ant Design, React Query, Axios

**Prerequisite:** PR1 must be merged. Backend now returns `current_value`, `purchase_value`, `depreciation_rate` on assets, and `net_worth` + real `net_worth_trend` on dashboard summary.

---

## File Map

| Action | File |
|--------|------|
| Create | `frontend/src/pages/WealthPage.tsx` |
| Delete | `frontend/src/pages/InventoryPage.tsx` |
| Delete | `frontend/src/pages/FinancePage.tsx` |
| Modify | `frontend/src/api/types.ts` |
| Modify | `frontend/src/api/endpoints.ts` |
| Modify | `frontend/src/App.tsx` |
| Modify | `frontend/src/components/AppShell.tsx` |
| Modify | `frontend/src/pages/DashboardPage.tsx` |

---

## Task 1: Branch Setup

- [ ] **Create branch**

```bash
git checkout main && git pull
git checkout -b feat/wealth-page
```

---

## Task 2: Update API Types

**Files:**
- Modify: `frontend/src/api/types.ts`

- [ ] **Add `purchase_value`, `depreciation_rate`, `current_value` to `Asset`; add `net_worth` to `DashboardSummary`**

Replace the `Asset` interface:

```typescript
export interface Asset {
  id: string
  user_id: string
  name: string
  category: string
  value: number
  purchased_at: string | null
  notes: string
  purchase_value: number | null
  depreciation_rate: number
  current_value: number
}
```

Replace the `DashboardSummary` interface:

```typescript
export interface DashboardSummary {
  net_worth_trend: number[]
  net_worth: number
  habits_total: number
  habits_done_today: number
  goals_avg_progress: number
  budget_total: number
  budget_spent: number
  recent_transactions: Transaction[]
}
```

- [ ] **Commit**

```bash
git add frontend/src/api/types.ts
git commit -m "feat: update Asset and DashboardSummary types for wealth features"
```

---

## Task 3: Update API Endpoints

**Files:**
- Modify: `frontend/src/api/endpoints.ts`

- [ ] **Update `createAsset` and `updateAsset` to include new fields**

Replace `createAsset` and `updateAsset`:

```typescript
export const createAsset = (data: Omit<Asset, 'id' | 'user_id' | 'current_value'>) =>
  apiClient.post<Asset>('/assets', data).then(r => r.data)
export const updateAsset = (id: string, data: Partial<Omit<Asset, 'id' | 'user_id' | 'current_value'>>) =>
  apiClient.patch<Asset>(`/assets/${id}`, data).then(r => r.data)
```

- [ ] **Commit**

```bash
git add frontend/src/api/endpoints.ts
git commit -m "feat: update asset endpoint types for purchase_value and depreciation_rate"
```

---

## Task 4: Create WealthPage

**Files:**
- Create: `frontend/src/pages/WealthPage.tsx`

- [ ] **Write WealthPage with 3 tabs: Transactions, Budgets, Assets**

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Tabs, Row, Col, Card, Table, Tag, Button, Form, Input, Select,
  InputNumber, Modal, Progress, Spin, Tooltip, Drawer,
} from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import {
  getTransactions, createTransaction, deleteTransaction,
  getBudgets, upsertBudget,
  getAssets, createAsset, updateAsset, deleteAsset,
} from '../api/endpoints'
import type { Transaction, Asset } from '../api/types'

const CATEGORIES = ['Food', 'Income', 'Entertainment', 'Health', 'Tech', 'Auto', 'Utilities', 'Shopping']
const CAT_COLORS: Record<string, string> = {
  Food: 'green', Income: 'blue', Entertainment: 'purple', Health: 'volcano',
  Tech: 'cyan', Auto: 'orange', Utilities: 'gold', Shopping: 'magenta',
}

function TransactionsTab() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: txs = [], isLoading } = useQuery({ queryKey: ['transactions'], queryFn: () => getTransactions() })

  const addMutation = useMutation({
    mutationFn: createTransaction,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['transactions'] }); setAddOpen(false); form.resetFields() },
  })
  const deleteMutation = useMutation({
    mutationFn: deleteTransaction,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['transactions'] }),
  })

  const totalIncome = txs.filter(t => t.amount > 0).reduce((s, t) => s + t.amount, 0)
  const totalExpenses = txs.filter(t => t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0)

  const columns: ColumnsType<Transaction> = [
    { title: 'Date', dataIndex: 'date', width: 90 },
    { title: 'Description', dataIndex: 'description', ellipsis: true },
    { title: 'Category', dataIndex: 'category', width: 130, render: c => <Tag color={CAT_COLORS[c]}>{c}</Tag> },
    { title: 'Amount', dataIndex: 'amount', align: 'right', width: 100,
      render: a => <span style={{ color: a > 0 ? '#52c41a' : '#ff4d4f', fontWeight: 600 }}>{a > 0 ? '+' : '-'}${Math.abs(a).toFixed(2)}</span> },
    { title: '', width: 40, render: (_, row) => <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(row.id)} /> },
  ]

  return (
    <>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        {[
          { label: 'Income', val: `$${totalIncome.toFixed(2)}`, color: '#52c41a' },
          { label: 'Expenses', val: `$${totalExpenses.toFixed(2)}`, color: '#ff4d4f' },
          { label: 'Net Cash', val: `$${(totalIncome - totalExpenses).toFixed(2)}`, color: '#1677ff' },
        ].map((s, i) => (
          <Col span={8} key={i}>
            <Card size="small">
              <div style={{ fontSize: 12, color: '#999' }}>{s.label}</div>
              <div style={{ fontSize: 22, fontWeight: 700, color: s.color }}>{s.val}</div>
            </Card>
          </Col>
        ))}
      </Row>
      <Card size="small" title="Transactions" extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
        {isLoading ? <Spin /> : <Table dataSource={txs} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} />}
      </Card>
      <Modal title="Add Transaction" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="date" label="Date" rules={[{ required: true }]}><Input type="date" /></Form.Item>
          <Form.Item name="description" label="Description" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="category" label="Category" rules={[{ required: true }]}>
            <Select options={CATEGORIES.map(c => ({ value: c, label: c }))} />
          </Form.Item>
          <Form.Item name="amount" label="Amount (negative = expense)" rules={[{ required: true }]}>
            <InputNumber style={{ width: '100%' }} step={0.01} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </>
  )
}

function BudgetsTab() {
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: txs = [] } = useQuery({ queryKey: ['transactions'], queryFn: () => getTransactions() })
  const { data: budgets = [] } = useQuery({ queryKey: ['budgets'], queryFn: getBudgets })

  const upsertMutation = useMutation({
    mutationFn: ({ category, monthly_limit }: { category: string; monthly_limit: number }) =>
      upsertBudget(category, monthly_limit),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['budgets'] }); form.resetFields() },
  })

  return (
    <>
      {budgets.length > 0 && (
        <Card size="small" title="Budget Progress" style={{ marginBottom: 12 }}>
          <Row gutter={[12, 8]}>
            {budgets.map(b => {
              const spent = txs.filter(t => t.category === b.category && t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0)
              const pct = Math.min(Math.round(spent / b.monthly_limit * 100), 100)
              return (
                <Col span={8} key={b.id}>
                  <div style={{ fontSize: 12, marginBottom: 2 }}>{b.category} <span style={{ color: '#999' }}>${spent.toFixed(0)} / ${b.monthly_limit.toFixed(0)}</span></div>
                  <Progress percent={pct} size="small" strokeColor={pct > 90 ? '#ff4d4f' : '#1677ff'} />
                </Col>
              )
            })}
          </Row>
        </Card>
      )}
      <Card size="small" title="Set Budget Limit">
        <Form form={form} layout="inline" onFinish={values => upsertMutation.mutate(values)}>
          <Form.Item name="category" rules={[{ required: true }]}>
            <Select placeholder="Category" style={{ width: 160 }} options={CATEGORIES.map(c => ({ value: c, label: c }))} />
          </Form.Item>
          <Form.Item name="monthly_limit" rules={[{ required: true }]}>
            <InputNumber placeholder="Monthly limit $" min={0} step={1} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={upsertMutation.isPending}>Save</Button>
        </Form>
      </Card>
    </>
  )
}

function AssetsTab() {
  const [addOpen, setAddOpen] = useState(false)
  const [editAsset, setEditAsset] = useState<Asset | null>(null)
  const [addForm] = Form.useForm()
  const [editForm] = Form.useForm()
  const qc = useQueryClient()

  const { data: assets = [], isLoading } = useQuery({ queryKey: ['assets'], queryFn: getAssets })

  const addMutation = useMutation({
    mutationFn: (values: any) => createAsset({
      ...values,
      value: values.purchase_value ?? 0,
      notes: values.notes || '',
      depreciation_rate: (values.depreciation_rate_pct ?? 0) / 100,
      purchase_value: values.purchase_value ?? null,
      purchased_at: values.purchased_at || null,
    }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['assets'] }); setAddOpen(false); addForm.resetFields() },
  })

  const editMutation = useMutation({
    mutationFn: ({ id, values }: { id: string; values: any }) => updateAsset(id, {
      ...values,
      value: values.purchase_value ?? 0,
      notes: values.notes || '',
      depreciation_rate: (values.depreciation_rate_pct ?? 0) / 100,
      purchase_value: values.purchase_value ?? null,
      purchased_at: values.purchased_at || null,
    }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['assets'] }); setEditAsset(null) },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteAsset,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['assets'] }),
  })

  const grandTotal = assets.reduce((s, a) => s + a.current_value, 0)
  const categories = [...new Set(assets.map(a => a.category))]

  const columns: ColumnsType<Asset> = [
    { title: 'Name', dataIndex: 'name', ellipsis: true },
    { title: 'Category', dataIndex: 'category', width: 120 },
    {
      title: 'Current Value', dataIndex: 'current_value', width: 140, align: 'right',
      render: (cv, row) => (
        <Tooltip title={row.purchase_value ? `Purchased at $${row.purchase_value.toLocaleString()}, depreciating ${(row.depreciation_rate * 100).toFixed(0)}%/yr` : undefined}>
          <span>${cv.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
        </Tooltip>
      ),
    },
    { title: 'Bought', dataIndex: 'purchased_at', width: 110, render: v => v ?? '—' },
    {
      title: '', width: 72,
      render: (_, row) => (
        <>
          <Button type="text" size="small" icon={<EditOutlined />} onClick={() => {
            setEditAsset(row)
            editForm.setFieldsValue({
              ...row,
              depreciation_rate_pct: Math.round(row.depreciation_rate * 100),
            })
          }} />
          <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(row.id)} />
        </>
      ),
    },
  ]

  const assetForm = (form: any, onFinish: (v: any) => void, loading: boolean) => (
    <Form form={form} layout="vertical" onFinish={onFinish}>
      <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Name is required' }]}><Input /></Form.Item>
      <Form.Item name="category" label="Category" rules={[{ required: true, message: 'Category is required' }]}><Input /></Form.Item>
      <Form.Item name="purchase_value" label="Purchase Value ($)" rules={[{ required: true, message: 'Purchase value is required' }, { type: 'number', min: 0 }]}>
        <InputNumber min={0} step={0.01} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="depreciation_rate_pct" label="Depreciation Rate (% per year)" initialValue={0} rules={[{ type: 'number', min: 0, max: 100 }]}>
        <InputNumber min={0} max={100} step={1} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="purchased_at" label="Purchase Date"><Input type="date" /></Form.Item>
      <Form.Item name="notes" label="Notes"><Input.TextArea rows={2} /></Form.Item>
      <Button type="primary" htmlType="submit" loading={loading} block>Save</Button>
    </Form>
  )

  return (
    <>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col span={6}>
          <Card size="small">
            <div style={{ fontSize: 12, color: '#999' }}>Total Assets</div>
            <div style={{ fontSize: 22, fontWeight: 700, color: '#52c41a' }}>${grandTotal.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</div>
          </Card>
        </Col>
        {categories.slice(0, 3).map(cat => {
          const total = assets.filter(a => a.category === cat).reduce((s, a) => s + a.current_value, 0)
          return (
            <Col span={6} key={cat}>
              <Card size="small">
                <div style={{ fontSize: 12, color: '#999' }}>{cat}</div>
                <div style={{ fontSize: 18, fontWeight: 600 }}>${total.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</div>
              </Card>
            </Col>
          )
        })}
      </Row>

      <Card size="small" title="Assets" extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
        {isLoading ? <Spin /> : <Table dataSource={assets} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} />}
      </Card>

      <Modal title="Add Asset" open={addOpen} onCancel={() => { setAddOpen(false); addForm.resetFields() }} footer={null}>
        {assetForm(addForm, values => addMutation.mutate(values), addMutation.isPending)}
      </Modal>

      <Drawer
        title="Edit Asset"
        open={editAsset !== null}
        onClose={() => setEditAsset(null)}
        width={400}
        footer={null}
      >
        {editAsset && assetForm(editForm, values => editMutation.mutate({ id: editAsset.id, values }), editMutation.isPending)}
      </Drawer>
    </>
  )
}

export function WealthPage() {
  return (
    <Tabs
      defaultActiveKey="transactions"
      items={[
        { key: 'transactions', label: 'Transactions', children: <TransactionsTab /> },
        { key: 'budgets',      label: 'Budgets',      children: <BudgetsTab /> },
        { key: 'assets',       label: 'Assets',       children: <AssetsTab /> },
      ]}
    />
  )
}
```

- [ ] **Commit**

```bash
git add frontend/src/pages/WealthPage.tsx
git commit -m "feat: add WealthPage with Transactions/Budgets/Assets tabs and asset edit"
```

---

## Task 5: Update Routes and Nav

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/AppShell.tsx`

- [ ] **Update App.tsx** — replace `/finance` and `/inventory` routes with `/wealth`

In `App.tsx`, remove the `FinancePage` and `InventoryPage` imports and routes. Add `WealthPage`:

```tsx
import { WealthPage } from './pages/WealthPage'
// Remove: import { FinancePage } from './pages/FinancePage'
// Remove: import { InventoryPage } from './pages/InventoryPage'
```

Replace the route entries:

```tsx
<Route path="/wealth" element={<WealthPage />} />
// Remove: <Route path="/finance" element={<FinancePage />} />
// Remove: <Route path="/inventory" element={<InventoryPage />} />
```

- [ ] **Update AppShell.tsx** — change nav items

Replace the NAV array entries for finance and inventory:

```tsx
const NAV = [
  { key: '/',       icon: <DashboardOutlined />, label: 'Dashboard' },
  { type: 'divider' as const },
  { key: '/wealth', icon: <DollarOutlined />,    label: 'Wealth' },         // replaces /finance and /inventory
  { key: '/health', icon: <HeartOutlined />,     label: 'Health & Habits' },
  { key: '/goals',  icon: <TrophyOutlined />,    label: 'Goals & OKRs' },
  { key: '/notes',  icon: <FileTextOutlined />,  label: 'Notes' },
  { key: '/calendar', icon: <CalendarOutlined />, label: 'Calendar' },
  { type: 'divider' as const },
  { key: '/settings', icon: <SettingOutlined />, label: 'Settings' },
]
```

Update the `TITLES` record:

```tsx
const TITLES: Record<string, string> = {
  '/': 'Dashboard',
  '/wealth': 'Wealth',
  '/health': 'Health & Habits',
  '/goals': 'Goals & OKRs',
  '/notes': 'Notes & Knowledge',
  '/calendar': 'Calendar & Schedule',
  '/settings': 'Settings',
}
```

Remove `AppstoreOutlined` from imports (no longer used).

- [ ] **Delete old pages**

```bash
rm frontend/src/pages/FinancePage.tsx frontend/src/pages/InventoryPage.tsx
```

- [ ] **Commit**

```bash
git add frontend/src/App.tsx frontend/src/components/AppShell.tsx
git rm frontend/src/pages/FinancePage.tsx frontend/src/pages/InventoryPage.tsx
git commit -m "feat: replace /finance and /inventory routes with /wealth"
```

---

## Task 6: Update Dashboard Page

**Files:**
- Modify: `frontend/src/pages/DashboardPage.tsx`

- [ ] **Use real net_worth and compute real % change from trend**

Replace the stats array in `DashboardPage`:

```tsx
const trend = data.net_worth_trend
const netWorth = data.net_worth
const prevNetWorth = trend.length >= 2 ? trend[trend.length - 2] : null
const netWorthChange = prevNetWorth && prevNetWorth !== 0
  ? ((netWorth - prevNetWorth) / Math.abs(prevNetWorth) * 100).toFixed(1)
  : null

const stats = [
  {
    label: 'Net Worth',
    val: `$${netWorth.toLocaleString(undefined, { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`,
    sub: netWorthChange !== null ? `${Number(netWorthChange) >= 0 ? '↑' : '↓'} ${Math.abs(Number(netWorthChange))}% vs last snapshot` : 'No history yet',
    subC: netWorthChange !== null && Number(netWorthChange) >= 0 ? '#52c41a' : '#ff4d4f',
    spark: trend,
    sparkC: '#52c41a',
    nav: '/wealth',
  },
  { label: "Today's Habits", val: `${data.habits_done_today} / ${data.habits_total}`, sub: `${habitPct}% complete`, subC: '#1677ff', pct: habitPct, nav: '/health' },
  { label: 'Goals (avg)',    val: `${data.goals_avg_progress}%`, sub: 'active OKRs', subC: '#722ed1', pct: data.goals_avg_progress, pctC: '#722ed1', nav: '/goals' },
  { label: 'Monthly Budget', val: `$${data.budget_total.toLocaleString()}`, sub: `$${data.budget_spent.toLocaleString()} spent · ${budgetPct}%`, subC: '#fa8c16', pct: budgetPct, pctC: '#fa8c16', nav: '/wealth' },
]
```

Also update the "View all →" link in the Recent Transactions card:

```tsx
extra={<a onClick={() => navigate('/wealth')} style={{ fontSize: 12 }}>View all →</a>}
```

- [ ] **Commit**

```bash
git add frontend/src/pages/DashboardPage.tsx
git commit -m "feat: dashboard uses real net worth and computed % change"
```

---

## Task 7: Build Check + PR

- [ ] **Run lint and build**

```bash
cd frontend && npm run lint && npm run build
```

Expected: no errors, build succeeds.

- [ ] **Create PR**

```bash
git push -u origin feat/wealth-page
gh pr create --title "feat: unified Wealth page (Transactions + Budgets + Assets) with asset edit" --body "$(cat <<'EOF'
## Summary
- New `/wealth` route replaces separate `/finance` and `/inventory`
- Tabbed layout: Transactions | Budgets | Assets
- Asset edit via Drawer (name, category, purchase value, depreciation rate, date, notes)
- Dashboard net worth shows real value + real % change vs last snapshot
- Nav updated: single "Wealth" entry

## Test plan
- [ ] `npm run lint && npm run build` passes
- [ ] Navigate to /wealth — all 3 tabs render
- [ ] Add, edit, and delete an asset
- [ ] Add and delete a transaction
- [ ] Set a budget and see progress bar
- [ ] Dashboard net worth card shows real value and non-hardcoded trend

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
gh pr merge --auto --squash
```
