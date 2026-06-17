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
    const spinner = document.querySelector('[aria-busy="true"]')
    expect(spinner).toBeInTheDocument()
  })
})
