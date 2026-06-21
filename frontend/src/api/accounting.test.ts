import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getAccounts, createAccount, createJournalEntry, getJournalNetWorth } from './endpoints'
import { apiClient } from './client'
import type { Account, NetWorthResult, CreateJournalEntryRequest } from './types'

vi.mock('./client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
  },
}))

const mockGet = vi.mocked(apiClient.get)
const mockPost = vi.mocked(apiClient.post)

beforeEach(() => { vi.clearAllMocks() })

describe('getAccounts', () => {
  it('calls GET /accounts and returns data', async () => {
    const accounts: Account[] = [
      { id: 'a1', user_id: 'u1', parent_id: null, name: 'Checking', type: 'asset',
        currency: 'VND', is_group: false, archived: false, sort_order: 0, balance: 0, asset_meta: null },
    ]
    mockGet.mockResolvedValueOnce({ data: accounts })
    const result = await getAccounts()
    expect(mockGet).toHaveBeenCalledWith('/accounts')
    expect(result).toEqual(accounts)
  })
})

describe('createAccount', () => {
  it('calls POST /accounts with payload', async () => {
    const payload = { name: 'Savings', type: 'asset' as const, currency: 'VND',
      is_group: false, sort_order: 1, parent_id: null }
    const created = { id: 'a2' }
    mockPost.mockResolvedValueOnce({ data: created })
    const result = await createAccount(payload)
    expect(mockPost).toHaveBeenCalledWith('/accounts', payload)
    expect(result).toEqual(created)
  })
})

describe('createJournalEntry', () => {
  it('calls POST /journal/entries and returns id', async () => {
    const req: CreateJournalEntryRequest = {
      date: '2026-06-17', description: 'Test', memo: '',
      lines: [
        { account_id: 'a1', amount: '100000', currency: 'VND', side: 'debit' },
        { account_id: 'a2', amount: '100000', currency: 'VND', side: 'credit' },
      ],
    }
    mockPost.mockResolvedValueOnce({ data: { id: 'e1' } })
    const result = await createJournalEntry(req)
    expect(mockPost).toHaveBeenCalledWith('/journal/entries', req)
    expect(result).toEqual({ id: 'e1' })
  })
})

describe('getJournalNetWorth', () => {
  it('calls GET /journal/networth and returns result', async () => {
    const nw: NetWorthResult = { net_worth: '5000000', currency: 'VND', net_income_ytd: '100000' }
    mockGet.mockResolvedValueOnce({ data: nw })
    const result = await getJournalNetWorth()
    expect(mockGet).toHaveBeenCalledWith('/journal/networth')
    expect(result).toEqual(nw)
  })
})
