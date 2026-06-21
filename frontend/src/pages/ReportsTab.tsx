import { useState, useMemo } from 'react'
import { Tabs, Segmented, Typography } from 'antd'
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
  const leafAccounts = accounts.filter(a => !a.is_group)
  const fmtVND = (n: number) => `₫${Math.round(n).toLocaleString('vi-VN')}`
  const totalDr = leafAccounts.reduce((s, a) => s + (balances.get(a.id)?.debit ?? 0), 0)
  const totalCr = leafAccounts.reduce((s, a) => s + (balances.get(a.id)?.credit ?? 0), 0)
  return (
    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
      <thead>
        <tr style={{ borderBottom: '1px solid #f0f0f0' }}>
          <th style={{ textAlign: 'left', padding: '4px 8px', color: '#999', fontWeight: 500 }}>Account</th>
          <th style={{ textAlign: 'right', padding: '4px 8px', color: '#999', fontWeight: 500 }}>Debit</th>
          <th style={{ textAlign: 'right', padding: '4px 8px', color: '#999', fontWeight: 500 }}>Credit</th>
        </tr>
      </thead>
      <tbody>
        {leafAccounts.map(a => {
          const b = balances.get(a.id)
          if (!b || (b.debit === 0 && b.credit === 0)) return null
          return (
            <tr key={a.id} style={{ borderBottom: '1px solid #fafafa' }}>
              <td style={{ padding: '3px 8px' }}>{a.name}</td>
              <td style={{ padding: '3px 8px', textAlign: 'right', fontFamily: 'monospace' }}>{b.debit > 0 ? fmtVND(b.debit) : ''}</td>
              <td style={{ padding: '3px 8px', textAlign: 'right', fontFamily: 'monospace' }}>{b.credit > 0 ? fmtVND(b.credit) : ''}</td>
            </tr>
          )
        })}
      </tbody>
      <tfoot>
        <tr style={{ borderTop: '2px solid #d9d9d9', fontWeight: 700 }}>
          <td style={{ padding: '4px 8px' }}>Total</td>
          <td style={{ padding: '4px 8px', textAlign: 'right', fontFamily: 'monospace' }}>{fmtVND(totalDr)}</td>
          <td style={{ padding: '4px 8px', textAlign: 'right', fontFamily: 'monospace' }}>{fmtVND(totalCr)}</td>
        </tr>
        <tr>
          <td colSpan={3} style={{ padding: '2px 8px', fontSize: 11, color: Math.abs(totalDr - totalCr) < 0.01 ? '#52c41a' : '#ff4d4f' }}>
            {Math.abs(totalDr - totalCr) < 0.01 ? '✓ Balanced' : `⚠ Out of balance by ₫${Math.round(Math.abs(totalDr - totalCr)).toLocaleString('vi-VN')}`}
          </td>
        </tr>
      </tfoot>
    </table>
  )
}

function BalanceSection({ title, type, balances, accounts }: {
  title: string; type: string; balances: Map<string, AccountBalance>; accounts: Account[]
}) {
  const relevant = accounts.filter(a => a.type === type)
  const fmtVND = (n: number) => `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}`
  const total = relevant.filter(a => !a.is_group).reduce((s, a) => s + (balances.get(a.id)?.balance ?? 0), 0)

  const renderAccount = (a: Account, depth = 0): React.ReactNode => {
    const b = balances.get(a.id)
    const children = accounts.filter(c => c.parent_id === a.id && c.type === type)
    return (
      <div key={a.id}>
        <div style={{ display: 'flex', justifyContent: 'space-between', padding: '2px 8px', paddingLeft: 8 + depth * 16, fontSize: 12, fontWeight: a.is_group ? 600 : 400, color: a.is_group ? '#222' : '#444' }}>
          <Text ellipsis style={{ maxWidth: '70%', fontSize: 12 }}>{a.name}</Text>
          {b && b.balance !== 0 && <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>{fmtVND(b.balance)}</span>}
        </div>
        {children.map(c => renderAccount(c, depth + 1))}
      </div>
    )
  }

  const rootAccounts = relevant.filter(a => a.parent_id === null || !relevant.find(p => p.id === a.parent_id))
  return (
    <div style={{ marginBottom: 12 }}>
      <div style={{ fontWeight: 700, fontSize: 13, padding: '4px 8px', background: '#f5f5f5', borderRadius: 4, marginBottom: 4 }}>{title}</div>
      {rootAccounts.map(a => renderAccount(a))}
      <div style={{ display: 'flex', justifyContent: 'space-between', padding: '4px 8px', borderTop: '1px solid #f0f0f0', fontWeight: 600, fontSize: 12 }}>
        <span>Total {title}</span>
        <span style={{ fontFamily: 'monospace' }}>{fmtVND(total)}</span>
      </div>
    </div>
  )
}

function BalanceSheet({ balances, bsBalances, accounts }: {
  balances: Map<string, AccountBalance>
  bsBalances: Map<string, AccountBalance>
  accounts: Account[]
}) {
  return (
    <>
      <BalanceSection title="Assets" type="asset" balances={balances} accounts={accounts} />
      <BalanceSection title="Liabilities" type="liability" balances={balances} accounts={accounts} />
      <BalanceSection title="Equity" type="equity" balances={bsBalances} accounts={accounts} />
    </>
  )
}

function PnLSection({ title, type, balances, accounts }: {
  title: string; type: string; balances: Map<string, AccountBalance>; accounts: Account[]
}) {
  const relevant = accounts.filter(a => a.type === type)
  const fmtVND = (n: number) => `₫${Math.round(Math.abs(n)).toLocaleString('vi-VN')}`
  const total = relevant.filter(a => !a.is_group).reduce((s, a) => s + (balances.get(a.id)?.balance ?? 0), 0)

  const renderAccount = (a: Account, depth = 0): React.ReactNode => {
    const b = balances.get(a.id)
    const children = accounts.filter(c => c.parent_id === a.id && c.type === type)
    return (
      <div key={a.id}>
        <div style={{ display: 'flex', justifyContent: 'space-between', padding: '2px 8px', paddingLeft: 8 + depth * 16, fontSize: 12, fontWeight: a.is_group ? 600 : 400, color: a.is_group ? '#222' : '#444' }}>
          <Text ellipsis style={{ maxWidth: '70%', fontSize: 12 }}>{a.name}</Text>
          {b && b.balance !== 0 && <span style={{ fontFamily: 'monospace', whiteSpace: 'nowrap' }}>{fmtVND(b.balance)}</span>}
        </div>
        {children.map(c => renderAccount(c, depth + 1))}
      </div>
    )
  }

  const rootAccounts = relevant.filter(a => a.parent_id === null || !relevant.find(p => p.id === a.parent_id))
  return (
    <div style={{ marginBottom: 12 }}>
      <div style={{ fontWeight: 700, fontSize: 13, padding: '4px 8px', background: '#f5f5f5', borderRadius: 4, marginBottom: 4 }}>{title}</div>
      {rootAccounts.map(a => renderAccount(a))}
      <div style={{ display: 'flex', justifyContent: 'space-between', padding: '4px 8px', borderTop: '1px solid #f0f0f0', fontWeight: 600, fontSize: 12 }}>
        <span>Total {title}</span>
        <span style={{ fontFamily: 'monospace' }}>{fmtVND(total)}</span>
      </div>
    </div>
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
