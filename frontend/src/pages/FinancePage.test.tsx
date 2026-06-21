import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { FinancePage } from './FinancePage'
import * as endpoints from '../api/endpoints'
import type { Account } from '../api/types'

vi.mock('../api/endpoints')

const mockGetAccounts = vi.mocked(endpoints.getAccounts)
const mockCreateAccount = vi.mocked(endpoints.createAccount)
const mockGetJournalEntries = vi.mocked(endpoints.getJournalEntries)
const mockGetJournalNetWorth = vi.mocked(endpoints.getJournalNetWorth)

function wrap(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<MemoryRouter><QueryClientProvider client={qc}>{ui}</QueryClientProvider></MemoryRouter>)
}

const sampleAccounts: Account[] = [
  { id: 'a1', user_id: 'u1', parent_id: null, name: 'Cash', type: 'asset',
    currency: 'VND', is_group: false, archived: false, sort_order: 0, balance: 0, asset_meta: null },
  { id: 'a2', user_id: 'u1', parent_id: null, name: 'Assets', type: 'asset',
    currency: 'VND', is_group: true, archived: false, sort_order: 0, balance: 0, asset_meta: null },
]

beforeEach(() => {
  vi.clearAllMocks()
  mockGetJournalEntries.mockResolvedValue([])
  mockGetJournalNetWorth.mockResolvedValue({ net_worth: '0', currency: 'VND', net_income_ytd: '0' })
  vi.mocked(endpoints.getTransactions).mockResolvedValue([])
  vi.mocked(endpoints.getBudgets).mockResolvedValue([])
  vi.mocked(endpoints.getNetWorthSnapshots).mockResolvedValue([])
  vi.mocked(endpoints.getBankRates).mockResolvedValue([])
  vi.mocked(endpoints.getNews).mockResolvedValue([])
  vi.mocked(endpoints.getBenchmarks).mockResolvedValue([])
})

async function switchToAccountsTab() {
  fireEvent.click(screen.getByRole('tab', { name: /accounts/i }))
}

describe('FinancePage — Accounts tab', () => {
  it('renders account list', async () => {
    mockGetAccounts.mockResolvedValue(sampleAccounts)
    wrap(<FinancePage />)
    await switchToAccountsTab()
    await waitFor(() => expect(screen.getByText('Cash')).toBeInTheDocument())
    expect(screen.getAllByText('asset').length).toBeGreaterThan(0)
  })

  it('opens create modal on Add button click', async () => {
    mockGetAccounts.mockResolvedValue([])
    wrap(<FinancePage />)
    await switchToAccountsTab()
    await waitFor(() => screen.getByRole('button', { name: /add account/i }))
    fireEvent.click(screen.getByRole('button', { name: /add account/i }))
    expect(screen.getByText(/new account/i)).toBeInTheDocument()
  })

  it('calls createAccount on form submit', async () => {
    mockGetAccounts.mockResolvedValue([])
    mockCreateAccount.mockResolvedValueOnce({ id: 'a3' })
    wrap(<FinancePage />)
    await switchToAccountsTab()
    await waitFor(() => screen.getByRole('button', { name: /add account/i }))
    fireEvent.click(screen.getByRole('button', { name: /add account/i }))
    fireEvent.change(screen.getByLabelText(/name/i), { target: { value: 'Savings' } })
    fireEvent.click(screen.getByRole('button', { name: /^save$/i }))
    await waitFor(() => expect(mockCreateAccount).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Savings', type: 'asset', currency: 'VND' })
    ))
  })
})

describe('FinancePage — SetupWizard', () => {
  it('shows setup wizard when accounts list is empty', async () => {
    mockGetAccounts.mockResolvedValue([])
    wrap(<FinancePage />)
    await switchToAccountsTab()
    await waitFor(() =>
      expect(screen.getByText(/set up your accounts/i)).toBeInTheDocument()
    )
  })

  it('does not show wizard when accounts exist', async () => {
    mockGetAccounts.mockResolvedValue(sampleAccounts)
    wrap(<FinancePage />)
    await switchToAccountsTab()
    await waitFor(() => screen.getByText('Cash'))
    expect(screen.queryByText(/set up your accounts/i)).not.toBeInTheDocument()
  })

  it('dismisses wizard on Skip', async () => {
    mockGetAccounts.mockResolvedValue([])
    wrap(<FinancePage />)
    await switchToAccountsTab()
    await waitFor(() => screen.getByText(/set up your accounts/i))
    fireEvent.click(screen.getByRole('button', { name: /skip/i }))
    await waitFor(() =>
      expect(screen.queryByText(/set up your accounts/i)).not.toBeInTheDocument()
    )
  })

  it('renders all default leaf accounts as checked checkboxes', async () => {
    mockGetAccounts.mockResolvedValue([])
    wrap(<FinancePage />)
    await switchToAccountsTab()
    await waitFor(() => screen.getByText(/set up your accounts/i))
    expect(screen.getByLabelText(/cash/i)).toBeChecked()
    expect(screen.getByLabelText(/bank account/i)).toBeChecked()
    expect(screen.getByLabelText(/credit card/i)).toBeChecked()
    expect(screen.getByLabelText(/opening balance/i)).toBeChecked()
    expect(screen.getByLabelText(/salary/i)).toBeChecked()
    expect(screen.getByLabelText(/living expenses/i)).toBeChecked()
  })
})

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
