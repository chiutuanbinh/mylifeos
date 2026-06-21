import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReportsTab } from './ReportsTab'
import type { Transaction } from '../api/types'

function wrap(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

const sampleTxs: Transaction[] = [
  { id: 't1', user_id: 'u1', date: '2026-01-01', description: 'Salary', category: 'Food', amount: 5000000 },
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
