import { useState, useMemo } from 'react'
import { Tabs, Segmented, Table, Typography } from 'antd'
import type { Account, JournalEntry } from '../api/types'

const { Text } = Typography

const fmtVND = (n: number) =>
  n === 0 ? '—' : `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}`

function AmtCell({ v, strong }: { v: number; strong?: boolean }) {
  const neg = v < 0
  const style = neg ? { color: '#ff4d4f' } : undefined
  const text = neg ? `-${fmtVND(v)}` : fmtVND(v)
  return strong ? <Text strong style={style}>{text}</Text> : <Text style={style}>{text}</Text>
}

function topLevel(balances: Map<string, AccountBalance>, type: string) {
  return [...balances.values()].filter(b => b.type === type && (!b.parentId || balances.get(b.parentId)?.type !== type))
}

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
      parentId: a.parent_id,
      isGroup: a.is_group,
      debit: 0,
      credit: 0,
      balance: 0,
    })
  }

  // accumulate leaf entries
  for (const entry of entries) {
    const entryDate = new Date(entry.date)
    const isCumulative = (type: string) => CUMULATIVE_TYPES.has(type)

    for (const line of entry.lines) {
      const acct = result.get(line.account_id)
      if (!acct) continue
      // for period-only accounts: filter by window; for cumulative: always include up to `to`
      if (isCumulative(acct.type)) {
        if (entryDate > to) continue
      } else {
        if (entryDate < from || entryDate > to) continue
      }
      const amt = parseFloat(line.amount)
      if (line.side === 'debit') acct.debit += amt
      else acct.credit += amt
    }
  }

  // compute balance per leaf (debit-normal: asset, expense → debit-credit; credit-normal: liability, equity, income → credit-debit)
  for (const [, a] of result) {
    if (a.isGroup) continue
    if (a.type === 'asset' || a.type === 'expense') {
      a.balance = a.debit - a.credit
    } else {
      a.balance = a.credit - a.debit
    }
  }

  // aggregate group accounts via topological sort (single pass, handles arbitrary depth)
  const visited = new Set<string>()
  function aggregate(id: string): void {
    if (visited.has(id)) return
    visited.add(id)
    const node = result.get(id)
    if (!node || !node.isGroup) return
    for (const child of result.values()) {
      if (child.parentId === id) {
        aggregate(child.id)
        node.debit += child.debit
        node.credit += child.credit
        node.balance += child.balance
      }
    }
  }
  for (const [id, a] of result) {
    if (a.isGroup) aggregate(id)
  }

  return result
}

// ── Sub-components ────────────────────────────────────────────────────────

function TrialBalance({ balances, accounts }: { balances: Map<string, AccountBalance>; accounts: Account[] }) {
  const rows = accounts.filter(a => !a.is_group).map(a => balances.get(a.id)!).filter(Boolean)
  const totalDebit = rows.reduce((s, r) => s + r.debit, 0)
  const totalCredit = rows.reduce((s, r) => s + r.credit, 0)

  return (
    <Table
      dataSource={rows}
      rowKey="id"
      columns={[
        { title: 'Account', dataIndex: 'name' },
        { title: 'Type', dataIndex: 'type', width: 100 },
        { title: 'Debit',  dataIndex: 'debit',  width: 160, align: 'right' as const, render: (v: number) => <Text>{fmtVND(v)}</Text> },
        { title: 'Credit', dataIndex: 'credit', width: 160, align: 'right' as const, render: (v: number) => <Text>{fmtVND(v)}</Text> },
      ]}
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

function BalanceSection({ title, type, balances, accounts }: {
  title: string; type: string; balances: Map<string, AccountBalance>; accounts: Account[]
}) {
  const relevant = accounts.filter(a => a.type === type)
  const rows = relevant.map(a => balances.get(a.id)!).filter(Boolean)
  const total = rows.reduce((s, r) => {
    const parentAb = r.parentId ? balances.get(r.parentId) : undefined
    if (!parentAb || parentAb.type !== type) return s + r.balance
    return s
  }, 0)

  return (
    <Table
      dataSource={rows}
      rowKey="id"
      columns={[
        {
          title, dataIndex: 'name',
          render: (name: string, row: AccountBalance) => (
            <span style={{ paddingLeft: row.isGroup ? 0 : 16, fontWeight: row.isGroup ? 600 : 400 }}>{name}</span>
          ),
        },
        {
          title: 'Balance', dataIndex: 'balance', width: 180, align: 'right' as const,
          render: (v: number, row: AccountBalance) => <AmtCell v={v} strong={row.isGroup} />,
        },
      ]}
      size="small"
      pagination={false}
      style={{ marginBottom: 16 }}
      summary={() => (
        <Table.Summary.Row>
          <Table.Summary.Cell index={0}><Text strong>Total {title}</Text></Table.Summary.Cell>
          <Table.Summary.Cell index={1} align="right"><AmtCell v={total} strong /></Table.Summary.Cell>
        </Table.Summary.Row>
      )}
    />
  )
}

function BalanceSheet({ balances, bsBalances, accounts }: {
  balances: Map<string, AccountBalance>
  bsBalances: Map<string, AccountBalance>
  accounts: Account[]
}) {
  const assetTotal = topLevel(balances, 'asset').reduce((s, b) => s + b.balance, 0)
  const liabTotal = topLevel(balances, 'liability').reduce((s, b) => s + b.balance, 0)
  const equityAccountTotal = topLevel(balances, 'equity').reduce((s, b) => s + b.balance, 0)

  // Retained earnings = all-time net income cumulative to period end
  const allTimeIncome = topLevel(bsBalances, 'income').reduce((s, b) => s + b.balance, 0)
  const allTimeExpense = topLevel(bsBalances, 'expense').reduce((s, b) => s + b.balance, 0)
  const retainedEarnings = allTimeIncome - allTimeExpense

  const equityTotal = equityAccountTotal + retainedEarnings
  const balanced = Math.abs(assetTotal - liabTotal - equityTotal) < 1

  return (
    <>
      <BalanceSection title="Assets" type="asset" balances={balances} accounts={accounts} />
      <BalanceSection title="Liabilities" type="liability" balances={balances} accounts={accounts} />
      <BalanceSection title="Equity" type="equity" balances={balances} accounts={accounts} />
      <Table
        dataSource={[{ id: '__retained__', name: 'Retained Earnings', balance: retainedEarnings }]}
        rowKey="id"
        columns={[
          { title: 'Equity', dataIndex: 'name' },
          { title: 'Balance', dataIndex: 'balance', width: 180, align: 'right' as const, render: (v: number) => <AmtCell v={v} /> },
        ]}
        size="small"
        pagination={false}
        style={{ marginBottom: 16 }}
        summary={() => (
          <Table.Summary.Row>
            <Table.Summary.Cell index={0}><Text strong>Total Equity</Text></Table.Summary.Cell>
            <Table.Summary.Cell index={1} align="right"><AmtCell v={equityTotal} strong /></Table.Summary.Cell>
          </Table.Summary.Row>
        )}
      />
      <div style={{ textAlign: 'right', padding: '8px 0', color: balanced ? '#52c41a' : 'red' }}>
        {balanced
          ? `✓ Balanced: Assets ${fmtVND(assetTotal)} = Liabilities + Equity ${fmtVND(liabTotal + equityTotal)}`
          : `⚠ Unbalanced: Assets ${fmtVND(assetTotal)} ≠ Liabilities + Equity ${fmtVND(liabTotal + equityTotal)}`}
      </div>
    </>
  )
}

function PnLSection({ title, type, balances, accounts }: {
  title: string; type: string; balances: Map<string, AccountBalance>; accounts: Account[]
}) {
  const relevant = accounts.filter(a => a.type === type)
  const rows = relevant.map(a => balances.get(a.id)!).filter(Boolean)
  const total = rows.reduce((s, r) => {
    const parentAb = r.parentId ? balances.get(r.parentId) : undefined
    if (!parentAb || parentAb.type !== type) return s + r.balance
    return s
  }, 0)

  return (
    <Table
      dataSource={rows}
      rowKey="id"
      columns={[
        {
          title, dataIndex: 'name',
          render: (name: string, row: AccountBalance) => (
            <span style={{ paddingLeft: row.isGroup ? 0 : 16, fontWeight: row.isGroup ? 600 : 400 }}>{name}</span>
          ),
        },
        {
          title: 'Amount', dataIndex: 'balance', width: 180, align: 'right' as const,
          render: (v: number, row: AccountBalance) => <AmtCell v={v} strong={row.isGroup} />,
        },
      ]}
      size="small"
      pagination={false}
      style={{ marginBottom: 16 }}
      summary={() => (
        <Table.Summary.Row>
          <Table.Summary.Cell index={0}><Text strong>Total {title}</Text></Table.Summary.Cell>
          <Table.Summary.Cell index={1} align="right"><AmtCell v={total} strong /></Table.Summary.Cell>
        </Table.Summary.Row>
      )}
    />
  )
}

function ProfitAndLoss({ balances, accounts }: { balances: Map<string, AccountBalance>; accounts: Account[] }) {
  const fmtVND = (n: number) => `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}`
  const income = accounts.filter(a => a.type === 'income' && !a.is_group).reduce((s, a) => s + (balances.get(a.id)?.balance ?? 0), 0)
  const expenses = accounts.filter(a => a.type === 'expense' && !a.is_group).reduce((s, a) => s + (balances.get(a.id)?.balance ?? 0), 0)
  const net = income - expenses
  return (
    <>
      <PnLSection title="Income" type="income" balances={balances} accounts={accounts} />
      <PnLSection title="Expenses" type="expense" balances={balances} accounts={accounts} />
      <div style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 8px', background: net >= 0 ? '#f6ffed' : '#fff2f0', borderRadius: 4, fontWeight: 700, fontSize: 13 }}>
        <span>Net Income</span>
        <span style={{ fontFamily: 'monospace', color: net >= 0 ? '#52c41a' : '#ff4d4f' }}>{net < 0 ? '-' : ''}{fmtVND(net)}</span>
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

  // All-time income/expense for retained earnings on balance sheet (not period-filtered)
  const bsBalances = useMemo(() => {
    const { to } = windowBounds(timeWindow)
    return computeBalances(accounts, entries, new Date(0), to)
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
          { key: 'bs', label: 'Balance Sheet', children: <BalanceSheet balances={balances} bsBalances={bsBalances} accounts={accounts} /> },
          { key: 'pl', label: 'P&L', children: <ProfitAndLoss balances={balances} accounts={accounts} /> },
        ]}
      />
    </>
  )
}
