import { apiClient } from './client'
import type {
  Transaction, Budget, Habit, HabitLog, Goal, KeyResult,
  Note, Event, Asset, UserSettings, DashboardSummary,
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

export const getHabits = () =>
  apiClient.get<Habit[]>('/habits').then(r => r.data)
export const createHabit = (data: { name: string; icon: string }) =>
  apiClient.post<Habit>('/habits', data).then(r => r.data)
export const deleteHabit = (id: string) =>
  apiClient.delete(`/habits/${id}`)
export const updateHabit = (id: string, data: { name: string; icon: string }) =>
  apiClient.put<Habit>(`/habits/${id}`, data).then(r => r.data)
export const getHabitLogRange = (habitId: string, from: string, to: string) =>
  apiClient.get<HabitLog[]>(`/habits/${habitId}/logs`, { params: { from, to } }).then(r => r.data)
export const getHabitLogs = (date?: string) =>
  apiClient.get<HabitLog[]>('/habits/logs', { params: { date } }).then(r => r.data)
export const toggleHabitLog = (habitId: string, date?: string) =>
  apiClient.post<HabitLog>(`/habits/${habitId}/log`, { date }).then(r => r.data)

export const getGoals = () =>
  apiClient.get<Goal[]>('/goals').then(r => r.data)
export const createGoal = (data: Omit<Goal, 'id' | 'user_id' | 'created_at' | 'key_results'>) =>
  apiClient.post<Goal>('/goals', data).then(r => r.data)
export const updateGoal = (id: string, data: Partial<Goal>) =>
  apiClient.patch<Goal>(`/goals/${id}`, data).then(r => r.data)
export const deleteGoal = (id: string) =>
  apiClient.delete(`/goals/${id}`)
export const addKeyResult = (goalId: string, description: string) =>
  apiClient.post<KeyResult>(`/goals/${goalId}/key-results`, { description }).then(r => r.data)
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
