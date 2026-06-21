import { apiClient } from './client'
import type {
  Transaction, Budget, Goal, KeyResult, KRLog,
  Note, Event, Asset, Liability, UserSettings, DashboardSummary,
  NetWorthSnapshot, BenchmarkData, BankRate, NewsItem,
  Account, CreateAccountRequest, UpdateAccountRequest, CreateJournalEntryRequest, NetWorthResult, JournalEntry,
} from './types'

export const getDashboardSummary = () =>
  apiClient.get<DashboardSummary>('/dashboard/summary').then(r => r.data)

export const getTransactions = (params?: { category?: string; from?: string; to?: string; limit?: number }) =>
  apiClient.get<Transaction[]>('/transactions', { params }).then(r => r.data)
export const createTransaction = (data: Omit<Transaction, 'id' | 'user_id' | 'created_at'>) =>
  apiClient.post<Transaction>('/transactions', data).then(r => r.data)
export const deleteTransaction = (id: string) =>
  apiClient.delete(`/transactions/${id}`)

export const getBudgets = () =>
  apiClient.get<Budget[]>('/budgets').then(r => r.data)
export const upsertBudget = (category: string, monthly_limit: number) =>
  apiClient.put<Budget>(`/budgets/${category}`, { monthly_limit }).then(r => r.data)

export const getKRLogs = (date?: string) =>
  apiClient.get<KRLog[]>('/kr-logs', { params: { date } }).then(r => r.data)
export const getKRLogRange = (krId: string, from: string, to: string) =>
  apiClient.get<KRLog[]>(`/key-results/${krId}/logs`, { params: { from, to } }).then(r => r.data)
export const toggleKRLog = (krId: string, date?: string) =>
  apiClient.post<KRLog>(`/key-results/${krId}/log`, { date }).then(r => r.data)

export const getGoals = () =>
  apiClient.get<Goal[]>('/goals').then(r => r.data)
export const createGoal = (data: Omit<Goal, 'id' | 'user_id' | 'created_at' | 'key_results'>) =>
  apiClient.post<Goal>('/goals', data).then(r => r.data)
export const updateGoal = (id: string, data: Partial<Goal>) =>
  apiClient.patch<Goal>(`/goals/${id}`, data).then(r => r.data)
export const deleteGoal = (id: string) =>
  apiClient.delete(`/goals/${id}`)
export const addKeyResult = (goalId: string, description: string, recurring = false, reminderTime?: string) =>
  apiClient.post<KeyResult>(`/goals/${goalId}/key-results`, {
    description,
    recurring,
    reminder_time: reminderTime ?? null,
  }).then(r => r.data)
export const updateKeyResult = (goalId: string, krId: string, data: Partial<KeyResult>) =>
  apiClient.patch<KeyResult>(`/goals/${goalId}/key-results/${krId}`, data).then(r => r.data)
export const deleteKeyResult = (goalId: string, krId: string) =>
  apiClient.delete(`/goals/${goalId}/key-results/${krId}`)

export const getNotes = (params?: { search?: string; tags?: string; pinned?: boolean }) =>
  apiClient.get<Note[]>('/notes', { params }).then(r => r.data)
export const createNote = (data: Omit<Note, 'id' | 'user_id' | 'created_at' | 'updated_at'>) =>
  apiClient.post<Note>('/notes', data).then(r => r.data)
export const updateNote = (id: string, data: Partial<Note>) =>
  apiClient.patch<Note>(`/notes/${id}`, data).then(r => r.data)
export const deleteNote = (id: string) =>
  apiClient.delete(`/notes/${id}`)

export const getEvents = (params?: { from?: string; to?: string }) =>
  apiClient.get<Event[]>('/events', { params }).then(r => r.data)
export const createEvent = (data: Omit<Event, 'id' | 'user_id'>) =>
  apiClient.post<Event>('/events', data).then(r => r.data)
export const updateEvent = (id: string, data: Partial<Event>) =>
  apiClient.patch<Event>(`/events/${id}`, data).then(r => r.data)
export const deleteEvent = (id: string) =>
  apiClient.delete(`/events/${id}`)
export const syncGoogleCalendar = (providerToken: string, timeMin?: string, timeMax?: string) =>
  apiClient.post<{ synced: number; error?: string }>('/calendar/google/sync', {
    provider_token: providerToken,
    time_min: timeMin,
    time_max: timeMax,
  }).then(r => r.data)

export const getAssets = () =>
  apiClient.get<Asset[]>('/assets').then(r => r.data)
export const createAsset = (data: Omit<Asset, 'id' | 'user_id' | 'current_value'>) =>
  apiClient.post<Asset>('/assets', data).then(r => r.data)
export const updateAsset = (id: string, data: Partial<Omit<Asset, 'id' | 'user_id' | 'current_value'>>) =>
  apiClient.patch<Asset>(`/assets/${id}`, data).then(r => r.data)
export const deleteAsset = (id: string) =>
  apiClient.delete(`/assets/${id}`)

export const getSettings = () =>
  apiClient.get<UserSettings>('/settings').then(r => r.data)
export const updateSettings = (data: Partial<UserSettings>) =>
  apiClient.put<UserSettings>('/settings', data).then(r => r.data)

export const getNetWorthSnapshots = () =>
  apiClient.get<NetWorthSnapshot[]>('/net-worth-snapshots').then(r => r.data)

export const addNetWorthSnapshot = (data: { date: string; net_worth: number; note?: string }) =>
  apiClient.post<NetWorthSnapshot>('/net-worth-snapshots', data).then(r => r.data)

export const getBenchmarks = (sources: string[], from: string, to: string) =>
  apiClient.get<BenchmarkData[]>('/benchmarks', { params: { sources: sources.join(','), from, to } }).then(r => r.data)

export const getBankRates = () =>
  apiClient.get<BankRate[]>('/bank-rates').then(r => r.data)

export const getNews = () =>
  apiClient.get<NewsItem[]>('/news').then(r => r.data)

export const triggerScrape = () =>
  apiClient.post<{ status: string }>('/scrape').then(r => r.data)

export const getLiabilities = () =>
  apiClient.get<Liability[]>('/liabilities').then(r => r.data)
export const createLiability = (data: Omit<Liability, 'id' | 'user_id'>) =>
  apiClient.post<Liability>('/liabilities', data).then(r => r.data)
export const updateLiability = (id: string, data: Partial<Omit<Liability, 'id' | 'user_id'>>) =>
  apiClient.patch<Liability>(`/liabilities/${id}`, data).then(r => r.data)
export const deleteLiability = (id: string) =>
  apiClient.delete(`/liabilities/${id}`)

// Accounting
export const getAccounts = () =>
  apiClient.get<Account[]>('/accounts').then(r => r.data)

export const createAccount = (data: CreateAccountRequest) =>
  apiClient.post<{ id: string }>('/accounts', data).then(r => r.data)

export const updateAccount = (id: string, data: UpdateAccountRequest) =>
  apiClient.patch(`/accounts/${id}`, data)

export const createJournalEntry = (data: CreateJournalEntryRequest) =>
  apiClient.post<{ id: string }>('/journal/entries', data).then(r => r.data)

export const getJournalEntries = () =>
  apiClient.get<JournalEntry[]>('/journal/entries').then(r => r.data)

export const getJournalNetWorth = () =>
  apiClient.get<NetWorthResult>('/journal/networth').then(r => r.data)
