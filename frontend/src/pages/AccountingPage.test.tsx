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
    expect(screen.getAllByText('asset').length).toBeGreaterThan(0)
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
