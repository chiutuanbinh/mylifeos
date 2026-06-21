export interface Transaction {
  id: string
  user_id: string
  date: string
  description: string
  category: string
  amount: number
  created_at: string
}

export interface Budget {
  id: string
  user_id: string
  category: string
  monthly_limit: number
  created_at: string
}

export interface KeyResult {
  id: string
  goal_id: string
  user_id: string
  description: string
  done: boolean
  recurring: boolean
  reminder_time?: string | null
}

export interface KRLog {
  id: string
  kr_id: string
  user_id: string
  logged_date: string
  done: boolean
}

export interface Goal {
  id: string
  user_id: string
  name: string
  description: string
  target_date: string | null
  progress: number
  color: string
  status: 'active' | 'completed' | 'archived'
  created_at: string
  key_results: KeyResult[]
}

export interface Note {
  id: string
  user_id: string
  title: string
  content: string
  tags: string[]
  pinned: boolean
  created_at: string
  updated_at: string
}

export interface Event {
  id: string
  user_id: string
  title: string
  start_at: string
  end_at: string
  color: string
  all_day: boolean
  google_event_id?: string
}

export interface Asset {
  id: string
  user_id: string
  name: string
  category: string
  value: number
  purchased_at: string | null
  notes: string
  purchase_value: number | null
  depreciation_rate: number
  current_value: number
}

export interface Liability {
  id: string
  user_id: string
  name: string
  category: string
  balance: number
  original_principal: number | null
  interest_rate: number | null
  started_at: string | null
  due_at: string | null
  notes: string
}

export interface UserSettings {
  user_id: string
  notifications: Record<string, boolean>
  modules_enabled: Record<string, boolean>
}

export interface DashboardSummary {
  net_worth_trend: number[]
  net_worth: number
  habits_total: number
  habits_done_today: number
  goals_avg_progress: number
  budget_total: number
  budget_spent: number
  recent_transactions: Transaction[]
}

export interface NetWorthSnapshot {
  id: string
  user_id: string
  snapshot_date: string
  assets_value: number
  cash_position: number
  net_worth: number
  note: string
}

export interface BenchmarkData {
  id: string
  source: string
  date: string
  value: number
}

export interface BankRate {
  bank: string
  saving_12m: number
  lending: number
  fetched_date: string
}

export interface NewsItem {
  id: string
  source: string
  published_at: string
  title: string
  url: string
}

export interface AssetMeta {
  purchase_value: string | null
  purchased_at: string | null
  depreciation_rate: string | null
  notes: string
}

export interface Account {
  id: string
  user_id: string
  parent_id: string | null
  name: string
  type: 'asset' | 'liability' | 'equity' | 'income' | 'expense'
  currency: string
  is_group: boolean
  archived: boolean
  sort_order: number
  balance: number
  asset_meta: AssetMeta | null
}

export interface JournalLine {
  id: string
  entry_id: string
  account_id: string
  amount: string
  currency: string
  side: 'debit' | 'credit'
}

export interface JournalEntry {
  id: string
  user_id: string
  date: string
  description: string
  memo: string
  lines: JournalLine[]
}

export interface CreateAccountRequest {
  name: string
  type: 'asset' | 'liability' | 'equity' | 'income' | 'expense'
  currency: string
  is_group: boolean
  sort_order: number
  parent_id: string | null
}

export interface UpdateAccountRequest {
  name: string
  type: 'asset' | 'liability' | 'equity' | 'income' | 'expense'
  parent_id?: string | null
  sort_order: number
  asset_meta?: {
    purchase_value?: number | null
    purchased_at?: string | null
    depreciation_rate?: number | null
    notes?: string
  } | null
}

export interface CreateJournalEntryRequest {
  date: string
  description: string
  memo: string
  lines: {
    account_id: string
    amount: string
    currency: string
    side: 'debit' | 'credit'
  }[]
}

export interface NetWorthResult {
  net_worth: string
  currency: string
  net_income_ytd: string
}
