# Finance Page Merge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge `WealthPage` and `AccountingPage` into a single `FinancePage` at `/finance`, with tabs: Journal · Accounts · Budgets · Reports · Trends.

**Architecture:** Rename `AccountingPage.tsx` → `FinancePage.tsx` and absorb `BudgetsTab` and `TrendsTab` from `WealthPage.tsx`. Add a Ledger section (summary cards + transactions table) at the top of `ReportsTab`. Route `/wealth` and `/accounting` both redirect to `/finance`. Delete `WealthPage.tsx`.

**Tech Stack:** React, TypeScript, Ant Design, React Query, React Router v6, Vitest

## Global Constraints

- No backend changes — all existing API endpoints unchanged
- Tab URL param key stays `tab` (used by `useTabParam` hook)
- Nav label: "Finance", icon: `AccountBookOutlined` (already imported), route key: `/finance`
- `/wealth` and `/accounting` must redirect (not 404) after the merge
- Coverage gate: ≥80% per file in `transport/http` and `middleware` (backend — unaffected, but don't break it)

---

### Task 1: Add Ledger section to ReportsTab

Absorb the Transactions summary cards and table into `ReportsTab` as a "Ledger" section above existing report content. `ReportsTab` already receives `accounts` and `entries` props; Ledger needs `transactions` from a new prop.

**Files:**
- Modify: `frontend/src/pages/ReportsTab.tsx`
- Test: `frontend/src/pages/ReportsTab.test.tsx` (create new)

**Interfaces:**
- Produces: `ReportsTab({ accounts, entries, transactions })` — adds `transactions: Transaction[]` prop

- [ ] **Step 1: Write the failing test**

```tsx
// frontend/src/pages/ReportsTab.test.tsx
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReportsTab } from './ReportsTab'
import type { Transaction, Account, JournalEntry } from '../api/types'

function wrap(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

const sampleTxs: Transaction[] = [
  { id: 't1', user_id: 'u1', date: '2026-01-01', description: 'Salary', category: 'Income', amount: 5000000 },
  { id: 't2', user_id: 'u1', date: '2026-01-02', description: 'Lunch', category: 'Food', amount: -100000 },
]

describe('ReportsTab — Ledger section', () => {
  it('shows income, expenses, net cash summary cards', () => {
    wrap(<ReportsTab accounts={[]} entries={[]} transactions={sampleTxs} />)
    expect(screen.getByText('Income')).toBeTruthy()
    expect(screen.getByText('Expenses')).toBeTruthy()
    expect(screen.getByText('Net Cash')).toBeTruthy()
  })

  it('shows transaction rows in ledger table', async () => {
    wrap(<ReportsTab accounts={[]} entries={[]} transactions={sampleTxs} />)
    expect(await screen.findByText('Salary')).toBeTruthy()
    expect(await screen.findByText('Lunch')).toBeTruthy()
  })

  it('renders with empty transactions without crashing', () => {
    wrap(<ReportsTab accounts={[]} entries={[]} transactions={[]} />)
    expect(screen.getByText('Income')).toBeTruthy()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/pages/ReportsTab.test.tsx
```
Expected: FAIL — `transactions` prop not accepted, Ledger section not rendered

- [ ] **Step 3: Implement Ledger section in ReportsTab**

At top of `frontend/src/pages/ReportsTab.tsx`, add the `Transaction` type import and update the props type and component:

```tsx
// Add to imports at top
import { Row, Col, Card, Table, Tag, Spin } from 'antd'
import type { Transaction } from '../api/types'
import type { ColumnsType } from 'antd/es/table'
```

Update `ReportsTabProps` (find it near line 318):
```tsx
type ReportsTabProps = {
  accounts: Account[]
  entries: JournalEntry[]
  transactions: Transaction[]
}
```

Add constants before `ReportsTab` function (reuse from WealthPage):
```tsx
const fmtVND = (n: number) => `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}`

const CAT_COLORS: Record<string, string> = {
  Food: 'green', Income: 'blue', Entertainment: 'purple', Health: 'volcano',
  Tech: 'cyan', Auto: 'orange', Utilities: 'gold', Shopping: 'magenta',
}

const txColumns: ColumnsType<Transaction> = [
  { title: 'Date', dataIndex: 'date', width: 105 },
  { title: 'Description', dataIndex: 'description', ellipsis: true },
  { title: 'Category', dataIndex: 'category', width: 130, render: (c: string) => <Tag color={CAT_COLORS[c]}>{c}</Tag> },
  { title: 'Amount', dataIndex: 'amount', align: 'right', width: 150,
    render: (a: number) => (
      <span style={{ color: a > 0 ? '#52c41a' : '#ff4d4f', fontWeight: 600, whiteSpace: 'nowrap' }}>
        {a > 0 ? '+' : '-'}{fmtVND(a)}
      </span>
    )},
]
```

Update `ReportsTab` function signature and add Ledger above the existing `<>` content:
```tsx
export function ReportsTab({ accounts, entries, transactions }: ReportsTabProps) {
  const [timeWindow, setTimeWindow] = useState<Window>('month')

  const balances = useMemo(() => {
    const { from, to } = windowBounds(timeWindow)
    return computeBalances(accounts, entries, from, to)
  }, [accounts, entries, timeWindow])

  const totalIncome = transactions.filter(t => t.amount > 0).reduce((s, t) => s + t.amount, 0)
  const totalExpenses = transactions.filter(t => t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0)

  const windowOptions = [
    { label: 'Today', value: 'today' },
    { label: 'This Month', value: 'month' },
    { label: 'This Quarter', value: 'quarter' },
    { label: 'This Year', value: 'year' },
    { label: 'All Time', value: 'all' },
  ]

  return (
    <>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        {[
          { label: 'Income', val: fmtVND(totalIncome), color: '#52c41a' },
          { label: 'Expenses', val: fmtVND(totalExpenses), color: '#ff4d4f' },
          { label: 'Net Cash', val: (totalIncome - totalExpenses >= 0 ? '' : '-') + fmtVND(totalIncome - totalExpenses), color: '#1677ff' },
        ].map((s, i) => (
          <Col xs={24} sm={8} key={i}>
            <Card size="small">
              <div style={{ fontSize: 12, color: '#999' }}>{s.label}</div>
              <div style={{ fontSize: 22, fontWeight: 700, color: s.color }}>{s.val}</div>
            </Card>
          </Col>
        ))}
      </Row>
      <Card size="small" title="Ledger" style={{ marginBottom: 16 }}>
        <Table dataSource={transactions} columns={txColumns} size="small" rowKey="id" pagination={{ pageSize: 20 }} scroll={{ x: true }} />
      </Card>
      <div style={{ marginBottom: 16 }}>
        <Segmented
          options={windowOptions}
          value={timeWindow}
          onChange={v => setTimeWindow(v as Window)}
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

- [ ] **Step 4: Run test to verify it passes**

```bash
cd frontend && npx vitest run src/pages/ReportsTab.test.tsx
```
Expected: PASS — 3 tests pass

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/ReportsTab.tsx frontend/src/pages/ReportsTab.test.tsx
git commit -m "feat(finance): add Ledger section to ReportsTab with transactions prop"
```

---

### Task 2: Build FinancePage from AccountingPage

Rename `AccountingPage.tsx` → `FinancePage.tsx`, export as `FinancePage`, absorb `BudgetsTab` and `TrendsTab` from `WealthPage`, add new tabs, pass `transactions` to `ReportsTab`.

**Files:**
- Create: `frontend/src/pages/FinancePage.tsx` (rename from `AccountingPage.tsx`)
- Modify: `frontend/src/pages/AccountingPage.tsx` → delete after rename
- Consumes: `ReportsTab({ accounts, entries, transactions })` from Task 1

**Interfaces:**
- Produces: `export function FinancePage()` at `frontend/src/pages/FinancePage.tsx`

- [ ] **Step 1: Copy AccountingPage.tsx to FinancePage.tsx**

```bash
cp frontend/src/pages/AccountingPage.tsx frontend/src/pages/FinancePage.tsx
```

- [ ] **Step 2: Add imports needed for Budgets and Trends tabs**

At the top of `FinancePage.tsx`, the existing imports include React Query and Ant Design. Add the missing ones. Find the import block and extend it:

Existing antd import line — add `Progress`, `Drawer`, `Alert`, `Tooltip`:
```tsx
import {
  Tabs, Card, Table, Tag, Button, Form, Input, Select, Switch,
  InputNumber, Modal, Spin, Badge, Checkbox, Radio, Collapse, Row, Col,
  Popconfirm, message, Typography, Divider, Progress, Alert,
} from 'antd'
```

Add to icon imports:
```tsx
import { PlusOutlined, FolderOutlined, FileOutlined, EditOutlined, DeleteOutlined,
  QuestionCircleOutlined, LineChartOutlined } from '@ant-design/icons'
```

Add API endpoint imports (append to existing `getAccounts, createAccount...` import line):
```tsx
import {
  getAccounts, createAccount, updateAccount, deleteAccount,
  createJournalEntry, getJournalEntries, getJournalNetWorth,
  getTransactions, deleteTransaction,
  getBudgets, upsertBudget,
  getNetWorthSnapshots, addNetWorthSnapshot,
  getBenchmarks, getBankRates, getNews, triggerScrape,
} from '../api/endpoints'
```

Add type imports:
```tsx
import type {
  Account, CreateAccountRequest, UpdateAccountRequest,
  CreateJournalEntryRequest, JournalEntry,
  Transaction, BankRate, NewsItem,
} from '../api/types'
```

Add component imports after `ReportsTab` import:
```tsx
import { NetWorthChart } from '../components/NetWorthChart'
import { LiveNetWorthCard } from './LiveNetWorthCard'
```

- [ ] **Step 3: Add shared constants after existing ones**

After the existing `normalSide` function and `TYPE_COLORS` constant in `FinancePage.tsx`, add:

```tsx
const CATEGORIES = ['Food', 'Income', 'Entertainment', 'Health', 'Tech', 'Auto', 'Utilities', 'Shopping']

const CAT_COLORS_WEALTH: Record<string, string> = {
  Food: 'green', Income: 'blue', Entertainment: 'purple', Health: 'volcano',
  Tech: 'cyan', Auto: 'orange', Utilities: 'gold', Shopping: 'magenta',
}

const BANK_DISPLAY: Record<string, string> = {
  vcb: 'Vietcombank', tcb: 'Techcombank', mbbank: 'MB Bank',
  acb: 'ACB', vpbank: 'VPBank', bidv: 'BIDV', agribank: 'Agribank',
}
```

- [ ] **Step 4: Copy BudgetsTab function into FinancePage.tsx**

Paste this function before the `SetupWizard` function in `FinancePage.tsx`:

```tsx
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

  const fmtVND = (n: number) => `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}`

  return (
    <>
      {budgets.length > 0 && (
        <Card size="small" title="Budget Progress" style={{ marginBottom: 12 }}>
          <Row gutter={[12, 8]}>
            {budgets.map(b => {
              const spent = txs.filter(t => t.category === b.category && t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0)
              const pct = Math.min(Math.round(spent / b.monthly_limit * 100), 100)
              return (
                <Col xs={24} sm={8} key={b.id}>
                  <div style={{ fontSize: 12, marginBottom: 2 }}>{b.category} <span style={{ color: '#999' }}>{fmtVND(spent)} / {fmtVND(b.monthly_limit)}</span></div>
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
            <InputNumber placeholder="Monthly limit ₫" min={0} step={1} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={upsertMutation.isPending}>Save</Button>
        </Form>
      </Card>
    </>
  )
}
```

- [ ] **Step 5: Copy TrendsTab function into FinancePage.tsx**

Paste this function after `BudgetsTab` and before `SetupWizard`:

```tsx
function TrendsTab() {
  const [backfillOpen, setBackfillOpen] = useState(false)
  const [scraping, setScraping] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const fmtVND = (n: number) => `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}`

  const handleScrape = async () => {
    setScraping(true)
    try {
      await triggerScrape()
      setTimeout(() => {
        qc.invalidateQueries({ queryKey: ['benchmarks'] })
        qc.invalidateQueries({ queryKey: ['bank-rates'] })
        qc.invalidateQueries({ queryKey: ['news'] })
        setScraping(false)
      }, 5000)
    } catch {
      setScraping(false)
    }
  }

  const now = new Date()
  const yearAgo = new Date(now.getFullYear() - 1, now.getMonth(), now.getDate()).toISOString().split('T')[0]
  const todayStr = now.toISOString().split('T')[0]

  const { data: snapshots = [] } = useQuery({ queryKey: ['net-worth-snapshots'], queryFn: getNetWorthSnapshots })
  const { data: benchmarks = [] } = useQuery({
    queryKey: ['benchmarks', yearAgo, todayStr],
    queryFn: () => getBenchmarks(['vn_index', 'sjc_gold', 'gso_cpi'], yearAgo, todayStr),
  })
  const { data: bankRatesRaw } = useQuery({ queryKey: ['bank-rates'], queryFn: getBankRates })
  const bankRates: BankRate[] = bankRatesRaw ?? []
  const { data: newsRaw } = useQuery({ queryKey: ['news'], queryFn: getNews })
  const news: NewsItem[] = newsRaw ?? []

  const addMutation = useMutation({
    mutationFn: addNetWorthSnapshot,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['net-worth-snapshots'] }); setBackfillOpen(false); form.resetFields() },
  })

  const latest = snapshots[snapshots.length - 1]
  const thirtyDaysAgo = new Date(now)
  thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)
  const cutoff30 = thirtyDaysAgo.toISOString().split('T')[0]
  const snap30 = snapshots.filter(s => s.snapshot_date <= cutoff30).slice(-1)[0]

  const pctChange = (curr: number, prev?: number) =>
    prev && prev !== 0 ? ((curr - prev) / prev * 100).toFixed(1) : null

  const latestBenchmark = (source: string) => {
    const pts = benchmarks.filter(b => b.source === source).sort((a, b) => a.date.localeCompare(b.date))
    return { latest: pts[pts.length - 1], oldest: pts[0] }
  }

  const vnidx = latestBenchmark('vn_index')
  const gold = latestBenchmark('sjc_gold')

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={8}><LiveNetWorthCard /></Col>
        <Col xs={12} sm={6}>
          <Card size="small">
            <div style={{ fontSize: 11, color: '#999' }}>Net Worth (30d)</div>
            <div style={{ fontSize: 18, fontWeight: 700, color: '#1677ff' }}>
              {latest ? fmtVND(latest.net_worth) : '—'}
            </div>
            {snap30 && latest && (
              <div style={{ fontSize: 11, color: Number(pctChange(latest.net_worth, snap30.net_worth)) >= 0 ? '#52c41a' : '#ff4d4f' }}>
                {pctChange(latest.net_worth, snap30.net_worth)}% vs 30d ago
              </div>
            )}
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small">
            <div style={{ fontSize: 11, color: '#999' }}>VN-Index (1Y)</div>
            <div style={{ fontSize: 18, fontWeight: 700 }}>{vnidx.latest ? vnidx.latest.value.toFixed(0) : '—'}</div>
            {vnidx.oldest && vnidx.latest && (
              <div style={{ fontSize: 11, color: Number(pctChange(vnidx.latest.value, vnidx.oldest.value)) >= 0 ? '#52c41a' : '#ff4d4f' }}>
                {pctChange(vnidx.latest.value, vnidx.oldest.value)}% vs 1Y ago
              </div>
            )}
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small">
            <div style={{ fontSize: 11, color: '#999' }}>SJC Gold (1Y)</div>
            <div style={{ fontSize: 18, fontWeight: 700 }}>{gold.latest ? `${(gold.latest.value / 1e6).toFixed(1)}M/lượng` : '—'}</div>
            {gold.oldest && gold.latest && (
              <div style={{ fontSize: 11, color: Number(pctChange(gold.latest.value, gold.oldest.value)) >= 0 ? '#52c41a' : '#ff4d4f' }}>
                {pctChange(gold.latest.value, gold.oldest.value)}% vs 1Y ago
              </div>
            )}
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ display: 'flex', flexDirection: 'column', gap: 8, alignItems: 'flex-start', justifyContent: 'center' }}>
            <Button size="small" icon={<PlusOutlined />} onClick={() => setBackfillOpen(true)}>Add past data point</Button>
            <Button size="small" loading={scraping} onClick={handleScrape}>Refresh market data</Button>
          </Card>
        </Col>
      </Row>

      <Card size="small" title="Net Worth Trend vs Benchmarks (% change from start)" style={{ marginBottom: 16 }}>
        <NetWorthChart snapshots={snapshots} benchmarks={benchmarks} />
      </Card>

      <Card size="small" title="Bank Interest Rates" style={{ marginBottom: 16 }}>
        {bankRates.length === 0 ? (
          <div style={{ color: '#bbb', fontSize: 12 }}>Rates fetched daily. Check back tomorrow.</div>
        ) : (
          <>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
              <thead>
                <tr style={{ borderBottom: '1px solid #f0f0f0' }}>
                  <th style={{ padding: '6px 8px', textAlign: 'left', color: '#999', fontWeight: 500 }}>Bank</th>
                  <th style={{ padding: '6px 8px', textAlign: 'right', color: '#999', fontWeight: 500 }}>Saving 12m</th>
                  <th style={{ padding: '6px 8px', textAlign: 'right', color: '#999', fontWeight: 500 }}>Lending</th>
                </tr>
              </thead>
              <tbody>
                {bankRates.map((r: BankRate) => (
                  <tr key={r.bank} style={{ borderBottom: '1px solid #f5f5f5' }}>
                    <td style={{ padding: '6px 8px' }}>{BANK_DISPLAY[r.bank] ?? r.bank}</td>
                    <td style={{ padding: '6px 8px', textAlign: 'right', color: '#52c41a' }}>{r.saving_12m}%</td>
                    <td style={{ padding: '6px 8px', textAlign: 'right', color: '#ff4d4f' }}>{r.lending}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {bankRates[0] && <div style={{ fontSize: 11, color: '#bbb', marginTop: 6 }}>Updated: {bankRates[0].fetched_date}</div>}
          </>
        )}
      </Card>

      <Card size="small" title="Finance News (vneconomy.vn)">
        {news.length === 0 ? (
          <div style={{ color: '#bbb', fontSize: 12 }}>News fetched daily.</div>
        ) : (
          news.slice(0, 10).map((n: NewsItem) => (
            <div key={n.id} style={{ padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
              <a href={n.url} target="_blank" rel="noopener noreferrer" style={{ fontSize: 13, color: '#1677ff', textDecoration: 'none' }}>{n.title}</a>
              <div style={{ fontSize: 11, color: '#bbb', marginTop: 2 }}>
                {new Date(n.published_at).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })}
              </div>
            </div>
          ))
        )}
      </Card>

      <Modal title="Add Past Net Worth" open={backfillOpen} onCancel={() => setBackfillOpen(false)} footer={null}>
        <Form form={form} layout="vertical"
          onFinish={values => addMutation.mutate({ date: values.date, net_worth: values.net_worth, note: values.note })}>
          <Form.Item name="date" label="Date" rules={[{ required: true }]}><Input type="date" /></Form.Item>
          <Form.Item name="net_worth" label="Net Worth (₫)" rules={[{ required: true }]}>
            <InputNumber style={{ width: '100%' }} min={0} step={1000000} />
          </Form.Item>
          <Form.Item name="note" label="Note (optional)"><Input /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Step 6: Update the export function name and Tabs in FinancePage.tsx**

Find `export function AccountingPage()` (line ~740) and rename to `export function FinancePage()`.

Find the `useQuery` calls inside `FinancePage` for `getTransactions` — add one if not present. The function already queries `getAccounts` and `getJournalEntries`. Add:

```tsx
const { data: transactions = [] } = useQuery({ queryKey: ['transactions'], queryFn: () => getTransactions() })
```

Find the `<Tabs ... items={[...]}` block (near end of file) and replace with:

```tsx
<Tabs
  activeKey={activeTab}
  onChange={setActiveTab}
  items={[
    { key: 'journal',   label: 'Journal',   children: <JournalTab /> },
    { key: 'accounts',  label: 'Accounts',  children: <AccountsTab /> },
    { key: 'budgets',   label: 'Budgets',   children: <BudgetsTab /> },
    { key: 'reports',   label: 'Reports',   children: <ReportsTab accounts={accounts} entries={entries} transactions={transactions} /> },
    { key: 'trends',    label: <><LineChartOutlined /> Trends</>, children: <TrendsTab /> },
  ]}
/>
```

- [ ] **Step 7: Verify TypeScript compiles**

```bash
cd frontend && npx tsc --noEmit
```
Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add frontend/src/pages/FinancePage.tsx
git commit -m "feat(finance): create FinancePage with Journal/Accounts/Budgets/Reports/Trends tabs"
```

---

### Task 3: Update routing and nav

Wire `/finance` route, add redirects for `/wealth` and `/accounting`, update AppShell nav.

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/AppShell.tsx`

**Interfaces:**
- Consumes: `FinancePage` from `./pages/FinancePage`

- [ ] **Step 1: Update App.tsx**

Open `frontend/src/App.tsx`. Replace:
```tsx
import { WealthPage } from './pages/WealthPage'
// ...
import { AccountingPage } from './pages/AccountingPage'
```
With:
```tsx
import { FinancePage } from './pages/FinancePage'
```

Replace the two route lines:
```tsx
<Route path="/wealth"    element={<WealthPage />} />
<Route path="/accounting" element={<AccountingPage />} />
```
With:
```tsx
<Route path="/finance"    element={<FinancePage />} />
<Route path="/wealth"     element={<Navigate to="/finance" replace />} />
<Route path="/accounting" element={<Navigate to="/finance" replace />} />
```

`Navigate` is already imported from `react-router-dom` at line 1.

- [ ] **Step 2: Update AppShell.tsx**

Open `frontend/src/components/AppShell.tsx`.

Find the `NAV` array. Replace the two entries:
```tsx
{ key: '/wealth',   icon: <DollarOutlined />,   label: 'Wealth' },
{ key: '/accounting', icon: <AccountBookOutlined />, label: 'Accounting' },
```
With:
```tsx
{ key: '/finance', icon: <AccountBookOutlined />, label: 'Finance' },
```

Find the `BOTTOM_NAV` array. Replace:
```tsx
{ key: '/wealth',     icon: <DollarOutlined />,   label: 'Wealth' },
{ key: '/accounting', icon: <AccountBookOutlined />, label: 'Accounting' },
```
With:
```tsx
{ key: '/finance', icon: <AccountBookOutlined />, label: 'Finance' },
```

Find the `PAGE_TITLES` map. Replace:
```tsx
'/wealth': 'Wealth',
'/accounting': 'Accounting',
```
With:
```tsx
'/finance': 'Finance',
```

Remove `DollarOutlined` from the icon import if it's no longer used elsewhere (check with grep first):
```bash
grep -n "DollarOutlined" frontend/src/components/AppShell.tsx
```
If only in the two removed lines, remove it from the import.

- [ ] **Step 3: Verify build**

```bash
cd frontend && npm run build
```
Expected: clean build, no errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/App.tsx frontend/src/components/AppShell.tsx
git commit -m "feat(finance): route /finance, redirect /wealth and /accounting"
```

---

### Task 4: Rename test file and add Budgets/Trends smoke tests, then delete WealthPage

**Files:**
- Create: `frontend/src/pages/FinancePage.test.tsx` (from `AccountingPage.test.tsx`)
- Delete: `frontend/src/pages/AccountingPage.test.tsx`
- Delete: `frontend/src/pages/WealthPage.tsx`
- Delete: `frontend/src/pages/AccountingPage.tsx`

- [ ] **Step 1: Copy test file**

```bash
cp frontend/src/pages/AccountingPage.test.tsx frontend/src/pages/FinancePage.test.tsx
```

- [ ] **Step 2: Update imports and references in FinancePage.test.tsx**

Replace:
```tsx
import { AccountingPage } from './AccountingPage'
```
With:
```tsx
import { FinancePage } from './FinancePage'
```

Replace all `<AccountingPage />` with `<FinancePage />`.

Replace all `describe('AccountingPage` with `describe('FinancePage`.

- [ ] **Step 3: Add Budgets tab smoke test**

Append this describe block to `FinancePage.test.tsx`:

```tsx
describe('FinancePage — Budgets tab', () => {
  beforeEach(() => {
    vi.mocked(endpoints.getTransactions).mockResolvedValue([])
    vi.mocked(endpoints.getBudgets).mockResolvedValue([])
    mockGetAccounts.mockResolvedValue([])
    mockGetJournalEntries.mockResolvedValue([])
    mockGetJournalNetWorth.mockResolvedValue({ net_worth: '0', currency: 'VND', net_income_ytd: '0' })
  })

  it('renders Budgets tab without crashing', async () => {
    // mock all endpoints used by FinancePage
    vi.mocked(endpoints.getTransactions).mockResolvedValue([])
    wrap(<FinancePage />)
    fireEvent.click(screen.getByRole('tab', { name: /budgets/i }))
    expect(await screen.findByText('Set Budget Limit')).toBeTruthy()
  })
})
```

Add to the `vi.mock('../api/endpoints')` mocks at top of file — ensure these are mocked:
```tsx
vi.mocked(endpoints.getTransactions).mockResolvedValue([])
vi.mocked(endpoints.getBudgets).mockResolvedValue([])
vi.mocked(endpoints.getNetWorthSnapshots).mockResolvedValue([])
vi.mocked(endpoints.getBankRates).mockResolvedValue([])
vi.mocked(endpoints.getNews).mockResolvedValue([])
vi.mocked(endpoints.getBenchmarks).mockResolvedValue([])
```

Add these to the `beforeEach` block.

- [ ] **Step 4: Run tests**

```bash
cd frontend && npx vitest run src/pages/FinancePage.test.tsx
```
Expected: all tests pass (same as before + new Budgets smoke test)

- [ ] **Step 5: Delete old files**

```bash
rm frontend/src/pages/WealthPage.tsx
rm frontend/src/pages/AccountingPage.tsx
rm frontend/src/pages/AccountingPage.test.tsx
```

- [ ] **Step 6: Run full test suite and lint**

```bash
cd frontend && npx vitest run && npm run lint
```
Expected: all tests pass, no lint errors

- [ ] **Step 7: Commit**

```bash
git add -A frontend/src/pages/
git commit -m "feat(finance): rename test, delete WealthPage and AccountingPage"
```

---

### Task 5: Final verification and PR

- [ ] **Step 1: Run full frontend checks**

```bash
cd frontend && npm run lint && npm run build
```
Expected: clean

- [ ] **Step 2: Run backend tests (must not regress)**

```bash
cd backend && go test ./internal/transport/http/... ./internal/middleware/... -coverprofile=coverage.out -covermode=atomic
bash scripts/hooks/pre-commit
```
Expected: ✓ Coverage OK

- [ ] **Step 3: Run integration smoke tests**

```bash
bash scripts/integration-test.sh
```
Expected: passes — `/finance` loads, no JS crashes

- [ ] **Step 4: Create PR**

```bash
git push -u origin feat/finance-page-merge
gh pr create --title "feat(finance): merge Wealth and Accounting into Finance page" --body "$(cat <<'EOF'
## Summary
- Merges `/wealth` and `/accounting` routes into a single `/finance` page
- FinancePage tabs: Journal · Accounts · Budgets · Reports · Trends
- Reports tab gains Ledger section (transaction summary + table)
- `/wealth` and `/accounting` redirect to `/finance`
- Deletes WealthPage.tsx; accounting is now sole source of truth

## Test plan
- [ ] Visit `/wealth` — redirects to `/finance`
- [ ] Visit `/accounting` — redirects to `/finance`
- [ ] Finance nav item appears once in sidebar
- [ ] All 5 tabs load without JS errors
- [ ] Budgets tab shows budget progress and set-limit form
- [ ] Reports tab shows Income/Expenses/Net Cash cards + Ledger table above P&L/Balance Sheet
- [ ] Trends tab shows net worth chart, bank rates, news

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
gh pr merge --auto --squash
```
