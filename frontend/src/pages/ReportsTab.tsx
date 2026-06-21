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

  // aggregate group accounts via topological sort (single pass, handles arbitrary depth)
  // Build parent→children map for topological ordering
  const children = new Map<string, string[]>()
  for (const ab of result.values()) {
    if (ab.parentId) {
      const siblings = children.get(ab.parentId) ?? []
      siblings.push(ab.id)
      children.set(ab.parentId, siblings)
    }
  }

  // Topological sort: leaves first, roots last (so we accumulate bottom-up)
  const sorted: AccountBalance[] = []
  const visited = new Set<string>()
  const visit = (id: string) => {
    if (visited.has(id)) return
    visited.add(id)
    for (const childId of children.get(id) ?? []) visit(childId)
    const ab = result.get(id)
    if (ab) sorted.push(ab)
  }
  for (const ab of result.values()) visit(ab.id)

  // Single bottom-up pass: each group sums its direct children
  for (const ab of sorted) {
    if (!ab.isGroup) continue
    let d = 0, c = 0
    for (const childId of children.get(ab.id) ?? []) {
      const child = result.get(childId)
      if (child) { d += child.debit; c += child.credit }
    }
    ab.debit = d; ab.credit = c
    ab.balance = DEBIT_NORMAL.has(ab.type) ? d - c : c - d
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
  const [timeWindow, setTimeWindow] = useState<Window>('month')

  const balances = useMemo(() => {
    const { from, to } = windowBounds(timeWindow)
    return computeBalances(accounts, entries, from, to)
  }, [accounts, entries, timeWindow])

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
