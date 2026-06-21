import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReportsTab } from './ReportsTab'

function wrap(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

describe('ReportsTab', () => {
  it('renders trial balance tab without crashing', () => {
    wrap(<ReportsTab accounts={[]} entries={[]} />)
    expect(screen.getByText('Trial Balance')).toBeTruthy()
  })

  it('renders Balance Sheet and P&L tabs', () => {
    wrap(<ReportsTab accounts={[]} entries={[]} />)
    expect(screen.getByText('Balance Sheet')).toBeTruthy()
    expect(screen.getByText('P&L')).toBeTruthy()
  })
})
