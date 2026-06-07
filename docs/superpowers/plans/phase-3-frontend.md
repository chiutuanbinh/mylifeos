# MyLifeOS Phase 3: React Frontend

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Prerequisite:** Phase 1 and 2 complete. Backend running at `http://localhost:8080`.

**Goal:** All 8 module pages implemented with live API data, matching the Ant Design ERP design from the spec.

**Architecture:** Vite + React 18 + TypeScript. Zustand for auth state (JWT in memory). TanStack Query for server state. Axios API client with auth header injection. React Router v6 for navigation.

**Tech Stack:** React 18, TypeScript, Ant Design 5, Zustand, TanStack Query v5, Axios, React Router v6, Vitest

---

## File Map

```
frontend/src/
├── api/
│   ├── client.ts          # Task 1: axios instance + interceptor
│   ├── types.ts           # Task 1: TypeScript types matching Go models
│   └── endpoints.ts       # Task 1: typed API functions
├── store/
│   └── auth.ts            # Task 1: Zustand auth store
├── components/
│   ├── AppShell.tsx        # Task 2: sidebar + header layout
│   └── Sparkline.tsx       # Task 2: SVG sparkline
├── pages/
│   ├── LoginPage.tsx       # Task 3
│   ├── DashboardPage.tsx   # Task 4
│   ├── FinancePage.tsx     # Task 5
│   ├── HealthPage.tsx      # Task 6
│   ├── GoalsPage.tsx       # Task 7
│   ├── NotesPage.tsx       # Task 8
│   ├── CalendarPage.tsx    # Task 9
│   ├── InventoryPage.tsx   # Task 10
│   └── SettingsPage.tsx    # Task 11
├── main.tsx                # Task 1
├── App.tsx                 # Task 2
└── test-setup.ts           # (already exists from Phase 1)
```

---

### Task 1: API client, types, auth store

**Files:**
- Create: `frontend/src/api/types.ts`
- Create: `frontend/src/api/client.ts`
- Create: `frontend/src/api/endpoints.ts`
- Create: `frontend/src/store/auth.ts`
- Modify: `frontend/src/main.tsx`

- [ ] **Step 1: Create `frontend/src/api/types.ts`**

```typescript
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

export interface Habit {
  id: string
  user_id: string
  name: string
  icon: string
  created_at: string
}

export interface HabitLog {
  id: string
  habit_id: string
  user_id: string
  logged_date: string
  done: boolean
}

export interface KeyResult {
  id: string
  goal_id: string
  user_id: string
  description: string
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
}

export interface Asset {
  id: string
  user_id: string
  name: string
  category: string
  value: number
  purchased_at: string | null
  notes: string
}

export interface UserSettings {
  user_id: string
  notifications: Record<string, boolean>
  modules_enabled: Record<string, boolean>
}

export interface DashboardSummary {
  net_worth_trend: number[]
  habits_total: number
  habits_done_today: number
  goals_avg_progress: number
  budget_total: number
  budget_spent: number
  recent_transactions: Transaction[]
}
```

- [ ] **Step 2: Create `frontend/src/api/client.ts`**

```typescript
import axios from 'axios'
import { useAuthStore } from '../store/auth'

const BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

export const apiClient = axios.create({ baseURL: BASE_URL })

apiClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})
```

- [ ] **Step 3: Create `frontend/src/api/endpoints.ts`**

```typescript
import { apiClient } from './client'
import type {
  Transaction, Budget, Habit, HabitLog, Goal, KeyResult,
  Note, Event, Asset, UserSettings, DashboardSummary,
} from './types'

// Dashboard
export const getDashboardSummary = () =>
  apiClient.get<DashboardSummary>('/dashboard/summary').then(r => r.data)

// Transactions
export const getTransactions = (params?: { category?: string; from?: string; to?: string; limit?: number }) =>
  apiClient.get<Transaction[]>('/transactions', { params }).then(r => r.data)
export const createTransaction = (data: Omit<Transaction, 'id' | 'user_id' | 'created_at'>) =>
  apiClient.post<Transaction>('/transactions', data).then(r => r.data)
export const deleteTransaction = (id: string) =>
  apiClient.delete(`/transactions/${id}`)

// Budgets
export const getBudgets = () =>
  apiClient.get<Budget[]>('/budgets').then(r => r.data)
export const upsertBudget = (category: string, monthly_limit: number) =>
  apiClient.put<Budget>(`/budgets/${category}`, { monthly_limit }).then(r => r.data)

// Habits
export const getHabits = () =>
  apiClient.get<Habit[]>('/habits').then(r => r.data)
export const createHabit = (data: { name: string; icon: string }) =>
  apiClient.post<Habit>('/habits', data).then(r => r.data)
export const deleteHabit = (id: string) =>
  apiClient.delete(`/habits/${id}`)
export const getHabitLogs = (date?: string) =>
  apiClient.get<HabitLog[]>('/habits/logs', { params: { date } }).then(r => r.data)
export const toggleHabitLog = (habitId: string, date?: string) =>
  apiClient.post<HabitLog>(`/habits/${habitId}/log`, { date }).then(r => r.data)

// Goals
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

// Notes
export const getNotes = (params?: { search?: string; tags?: string; pinned?: boolean }) =>
  apiClient.get<Note[]>('/notes', { params }).then(r => r.data)
export const createNote = (data: Omit<Note, 'id' | 'user_id' | 'created_at' | 'updated_at'>) =>
  apiClient.post<Note>('/notes', data).then(r => r.data)
export const updateNote = (id: string, data: Partial<Note>) =>
  apiClient.patch<Note>(`/notes/${id}`, data).then(r => r.data)
export const deleteNote = (id: string) =>
  apiClient.delete(`/notes/${id}`)

// Events
export const getEvents = (params?: { from?: string; to?: string }) =>
  apiClient.get<Event[]>('/events', { params }).then(r => r.data)
export const createEvent = (data: Omit<Event, 'id' | 'user_id'>) =>
  apiClient.post<Event>('/events', data).then(r => r.data)
export const updateEvent = (id: string, data: Partial<Event>) =>
  apiClient.patch<Event>(`/events/${id}`, data).then(r => r.data)
export const deleteEvent = (id: string) =>
  apiClient.delete(`/events/${id}`)

// Assets
export const getAssets = () =>
  apiClient.get<Asset[]>('/assets').then(r => r.data)
export const createAsset = (data: Omit<Asset, 'id' | 'user_id'>) =>
  apiClient.post<Asset>('/assets', data).then(r => r.data)
export const updateAsset = (id: string, data: Partial<Asset>) =>
  apiClient.patch<Asset>(`/assets/${id}`, data).then(r => r.data)
export const deleteAsset = (id: string) =>
  apiClient.delete(`/assets/${id}`)

// Settings
export const getSettings = () =>
  apiClient.get<UserSettings>('/settings').then(r => r.data)
export const updateSettings = (data: Partial<UserSettings>) =>
  apiClient.put<UserSettings>('/settings', data).then(r => r.data)
```

- [ ] **Step 4: Create `frontend/src/store/auth.ts`**

```typescript
import { create } from 'zustand'
import { createClient } from '@supabase/supabase-js'

const supabaseUrl = import.meta.env.VITE_SUPABASE_URL || ''
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY || ''

export const supabase = supabaseUrl
  ? createClient(supabaseUrl, supabaseAnonKey)
  : null

interface AuthState {
  token: string | null
  loading: boolean
  signIn: (email: string, password: string) => Promise<void>
  signOut: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  loading: false,

  signIn: async (email, password) => {
    set({ loading: true })
    try {
      if (supabase) {
        const { data, error } = await supabase.auth.signInWithPassword({ email, password })
        if (error) throw error
        set({ token: data.session?.access_token ?? null })
      } else {
        // Local dev: no Supabase — use a fake token; backend accepts any token in dev mode
        set({ token: 'local-dev-token' })
      }
    } finally {
      set({ loading: false })
    }
  },

  signOut: async () => {
    if (supabase) await supabase.auth.signOut()
    set({ token: null })
  },
}))
```

- [ ] **Step 5: Rewrite `frontend/src/main.tsx`**

```typescript
import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './App'
import 'antd/dist/reset.css'

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30_000, retry: 1 } },
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
)
```

- [ ] **Step 6: Test auth store**

Create `frontend/src/store/auth.test.ts`:

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from './auth'

describe('auth store', () => {
  beforeEach(() => {
    useAuthStore.setState({ token: null, loading: false })
  })

  it('sets token on local dev sign in', async () => {
    await useAuthStore.getState().signIn('test@test.com', 'password')
    expect(useAuthStore.getState().token).toBe('local-dev-token')
  })

  it('clears token on sign out', async () => {
    useAuthStore.setState({ token: 'some-token' })
    await useAuthStore.getState().signOut()
    expect(useAuthStore.getState().token).toBeNull()
  })
})
```

```bash
cd frontend && npm test -- --run
```

Expected: `2 passed`

- [ ] **Step 7: Commit**

```bash
cd .. && git add frontend/src/
git commit -m "feat: API client, TypeScript types, auth store"
```

---

### Task 2: AppShell + routing

**Files:**
- Create: `frontend/src/components/AppShell.tsx`
- Create: `frontend/src/components/Sparkline.tsx`
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Install React Router**

```bash
cd frontend && npm install react-router-dom && cd ..
```

- [ ] **Step 2: Create `frontend/src/components/Sparkline.tsx`**

```tsx
interface SparklineProps {
  data: number[]
  color?: string
  width?: number
  height?: number
}

export function Sparkline({ data, color = '#1677ff', width = 100, height = 28 }: SparklineProps) {
  if (!data || data.length < 2) return null
  const max = Math.max(...data)
  const min = Math.min(...data)
  const range = max - min || 1
  const pts = data.map((v, i) => [
    (i / (data.length - 1)) * width,
    height - ((v - min) / range) * (height - 4) - 2,
  ])
  const d = pts.map((p, i) => `${i === 0 ? 'M' : 'L'}${p[0].toFixed(1)},${p[1].toFixed(1)}`).join(' ')
  return (
    <svg width={width} height={height} style={{ display: 'block' }}>
      <path d={d} fill="none" stroke={color} strokeWidth="1.5" strokeLinejoin="round" strokeLinecap="round" />
    </svg>
  )
}
```

- [ ] **Step 3: Create `frontend/src/components/AppShell.tsx`**

```tsx
import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, Avatar, Badge, Space, Button, Breadcrumb, Typography } from 'antd'
import {
  DashboardOutlined, DollarOutlined, HeartOutlined, TrophyOutlined,
  FileTextOutlined, CalendarOutlined, AppstoreOutlined, SettingOutlined,
  BellOutlined, MenuFoldOutlined, MenuUnfoldOutlined, LogoutOutlined, SyncOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../store/auth'

const NAV = [
  { key: '/',          icon: <DashboardOutlined />, label: 'Dashboard' },
  { type: 'divider' as const },
  { key: '/finance',   icon: <DollarOutlined />,   label: 'Finance' },
  { key: '/health',    icon: <HeartOutlined />,     label: 'Health & Habits' },
  { key: '/goals',     icon: <TrophyOutlined />,    label: 'Goals & OKRs' },
  { key: '/notes',     icon: <FileTextOutlined />,  label: 'Notes' },
  { key: '/calendar',  icon: <CalendarOutlined />,  label: 'Calendar' },
  { key: '/inventory', icon: <AppstoreOutlined />,  label: 'Inventory' },
  { type: 'divider' as const },
  { key: '/settings',  icon: <SettingOutlined />,   label: 'Settings' },
]

const TITLES: Record<string, string> = {
  '/': 'Dashboard', '/finance': 'Finance & Budget', '/health': 'Health & Habits',
  '/goals': 'Goals & OKRs', '/notes': 'Notes & Knowledge', '/calendar': 'Calendar & Schedule',
  '/inventory': 'Inventory & Assets', '/settings': 'Settings',
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const [collapsed, setCollapsed] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const signOut = useAuthStore(s => s.signOut)

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Layout.Sider
        width={220} collapsedWidth={56} collapsed={collapsed}
        style={{ background: '#fff', boxShadow: '2px 0 8px rgba(0,0,0,0.07)', position: 'fixed', height: '100vh', zIndex: 100 }}
      >
        <div style={{ height: 48, display: 'flex', alignItems: 'center', padding: '0 16px', gap: 10, borderBottom: '1px solid #f0f0f0', overflow: 'hidden' }}>
          <div style={{ width: 28, height: 28, borderRadius: 6, flexShrink: 0, background: 'linear-gradient(135deg, #1677ff, #0952c6)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: 14, fontWeight: 800 }}>M</div>
          {!collapsed && <span style={{ fontSize: 15, fontWeight: 700, color: '#111' }}>MyLifeOS</span>}
        </div>
        <div style={{ height: 'calc(100vh - 48px - 52px)', overflowY: 'auto', overflowX: 'hidden' }}>
          <Menu
            mode="inline"
            selectedKeys={[location.pathname]}
            items={NAV}
            onClick={({ key }) => navigate(key)}
            style={{ border: 'none', paddingTop: 4 }}
          />
        </div>
        <div style={{ position: 'absolute', bottom: 0, left: 0, right: 0, height: 52, display: 'flex', alignItems: 'center', padding: '0 16px', gap: 10, borderTop: '1px solid #f0f0f0', background: '#fff', overflow: 'hidden' }}>
          <Avatar size={28} style={{ background: '#1677ff', flexShrink: 0, fontSize: 11, fontWeight: 700 }}>Me</Avatar>
          {!collapsed && <div style={{ flex: 1, minWidth: 0 }}><div style={{ fontSize: 12, fontWeight: 600, color: '#111' }}>My Account</div><div style={{ fontSize: 11, color: '#aaa' }}>Personal Plan</div></div>}
        </div>
      </Layout.Sider>

      <Layout style={{ marginLeft: collapsed ? 56 : 220, transition: 'margin-left 0.2s' }}>
        <Layout.Header style={{ background: '#fff', padding: '0 16px', height: 48, lineHeight: '48px', borderBottom: '1px solid #f0f0f0', position: 'sticky', top: 0, zIndex: 99, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Space size={10}>
            <Button type="text" size="small" icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />} onClick={() => setCollapsed(c => !c)} />
            <Breadcrumb items={[
              { title: <Typography.Text type="secondary" style={{ fontSize: 12 }}>MyLifeOS</Typography.Text> },
              { title: <span style={{ fontSize: 12, fontWeight: 500 }}>{TITLES[location.pathname] || '—'}</span> },
            ]} />
          </Space>
          <Space size={6}>
            <Badge count={0} size="small">
              <Button type="text" size="small" icon={<BellOutlined />} />
            </Badge>
            <Button type="text" size="small" icon={<LogoutOutlined />} onClick={() => signOut()} />
          </Space>
        </Layout.Header>
        <Layout.Content style={{ padding: 16, background: '#f0f2f5', minHeight: 'calc(100vh - 48px)' }}>
          {children}
        </Layout.Content>
      </Layout>
    </Layout>
  )
}
```

- [ ] **Step 4: Rewrite `frontend/src/App.tsx`**

```tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import { useAuthStore } from './store/auth'
import { AppShell } from './components/AppShell'
import { LoginPage } from './pages/LoginPage'
import { DashboardPage } from './pages/DashboardPage'
import { FinancePage } from './pages/FinancePage'
import { HealthPage } from './pages/HealthPage'
import { GoalsPage } from './pages/GoalsPage'
import { NotesPage } from './pages/NotesPage'
import { CalendarPage } from './pages/CalendarPage'
import { InventoryPage } from './pages/InventoryPage'
import { SettingsPage } from './pages/SettingsPage'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const token = useAuthStore(s => s.token)
  return token ? <>{children}</> : <Navigate to="/login" replace />
}

export default function App() {
  return (
    <ConfigProvider theme={{ token: { colorPrimary: '#1677ff', borderRadius: 4, fontSize: 13 } }}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/*" element={
            <PrivateRoute>
              <AppShell>
                <Routes>
                  <Route path="/"          element={<DashboardPage />} />
                  <Route path="/finance"   element={<FinancePage />} />
                  <Route path="/health"    element={<HealthPage />} />
                  <Route path="/goals"     element={<GoalsPage />} />
                  <Route path="/notes"     element={<NotesPage />} />
                  <Route path="/calendar"  element={<CalendarPage />} />
                  <Route path="/inventory" element={<InventoryPage />} />
                  <Route path="/settings"  element={<SettingsPage />} />
                </Routes>
              </AppShell>
            </PrivateRoute>
          } />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  )
}
```

- [ ] **Step 5: Stub all page files so App.tsx compiles**

```bash
for page in LoginPage DashboardPage FinancePage HealthPage GoalsPage NotesPage CalendarPage InventoryPage SettingsPage; do
  echo "export function ${page}() { return <div>${page}</div> }" > frontend/src/pages/${page}.tsx
done
```

- [ ] **Step 6: Build check**

```bash
cd frontend && npm run build
```

Expected: no TypeScript errors, dist/ produced.

- [ ] **Step 7: Commit**

```bash
cd .. && git add frontend/src/
git commit -m "feat: AppShell, routing, page stubs"
```

---

### Task 3: Login page

**Files:**
- Modify: `frontend/src/pages/LoginPage.tsx`

- [ ] **Step 1: Implement `frontend/src/pages/LoginPage.tsx`**

```tsx
import { useNavigate } from 'react-router-dom'
import { Form, Input, Button, Checkbox, Typography } from 'antd'
import { UserOutlined, LockOutlined, LoginOutlined } from '@ant-design/icons'
import { useAuthStore } from '../store/auth'
import { message } from 'antd'

export function LoginPage() {
  const { signIn, loading } = useAuthStore()
  const navigate = useNavigate()

  const onFinish = async ({ email, password }: { email: string; password: string }) => {
    try {
      await signIn(email, password)
      navigate('/')
    } catch {
      message.error('Invalid credentials')
    }
  }

  return (
    <div style={{ display: 'flex', height: '100vh' }}>
      {/* Brand panel */}
      <div style={{ width: 400, flexShrink: 0, background: 'linear-gradient(158deg, #04172e 0%, #083d8a 50%, #1677ff 100%)', display: 'flex', flexDirection: 'column', justifyContent: 'center', padding: '0 52px', color: '#fff' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 52 }}>
          <div style={{ width: 40, height: 40, borderRadius: 9, flexShrink: 0, background: 'rgba(255,255,255,0.15)', border: '1.5px solid rgba(255,255,255,0.3)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 20, fontWeight: 800 }}>M</div>
          <span style={{ fontSize: 21, fontWeight: 700, letterSpacing: -0.5 }}>MyLifeOS</span>
        </div>
        <div style={{ fontSize: 28, fontWeight: 700, lineHeight: 1.28, marginBottom: 14 }}>Your life,<br />organized.</div>
        <div style={{ fontSize: 13, opacity: 0.65, lineHeight: 1.85, marginBottom: 48 }}>Finance · Health · Goals · Notes<br />Calendar · Inventory — all in one place.</div>
        {['Finance & Budget Tracking', 'Health & Habit Monitoring', 'Goals & OKR Management', 'Personal Asset Inventory'].map((f, i) => (
          <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
            <div style={{ width: 18, height: 18, borderRadius: '50%', flexShrink: 0, background: 'rgba(255,255,255,0.2)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10 }}>✓</div>
            <span style={{ fontSize: 13, opacity: 0.88 }}>{f}</span>
          </div>
        ))}
      </div>

      {/* Login form */}
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#fafafa' }}>
        <div style={{ width: 340 }}>
          <Typography.Title level={3} style={{ marginBottom: 4, fontWeight: 700 }}>Sign in</Typography.Title>
          <Typography.Text type="secondary" style={{ fontSize: 14, display: 'block', marginBottom: 30 }}>Welcome back to your personal ERP</Typography.Text>
          <Form layout="vertical" size="large" onFinish={onFinish} initialValues={{ email: 'me@mylifeos.app', password: 'password' }}>
            <Form.Item label="Email" name="email" rules={[{ required: true, type: 'email' }]}>
              <Input prefix={<UserOutlined style={{ color: '#ccc' }} />} />
            </Form.Item>
            <Form.Item label="Password" name="password" rules={[{ required: true }]}>
              <Input.Password prefix={<LockOutlined style={{ color: '#ccc' }} />} />
            </Form.Item>
            <Form.Item style={{ marginBottom: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Checkbox defaultChecked style={{ fontSize: 13 }}>Remember me</Checkbox>
              </div>
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading} block icon={<LoginOutlined />}>Sign In</Button>
            </Form.Item>
          </Form>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/LoginPage.tsx
git commit -m "feat: login page with Supabase auth"
```

---

### Task 4: Dashboard page

**Files:**
- Modify: `frontend/src/pages/DashboardPage.tsx`

- [ ] **Step 1: Implement `frontend/src/pages/DashboardPage.tsx`**

```tsx
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Row, Col, Card, Progress, Table, Tag, Spin } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getDashboardSummary } from '../api/endpoints'
import { Sparkline } from '../components/Sparkline'
import type { Transaction } from '../api/types'

const CAT_COLORS: Record<string, string> = {
  Food: 'green', Income: 'blue', Entertainment: 'purple', Health: 'volcano',
  Tech: 'cyan', Auto: 'orange', Utilities: 'gold', Shopping: 'magenta',
}

const txColumns: ColumnsType<Transaction> = [
  { title: 'Date',        dataIndex: 'date',        width: 72,  render: v => <span style={{ color: '#bbb', fontSize: 12 }}>{v}</span> },
  { title: 'Description', dataIndex: 'description', ellipsis: true, render: v => <span style={{ fontSize: 12 }}>{v}</span> },
  { title: 'Category',    dataIndex: 'category',    width: 120, render: c => <Tag color={CAT_COLORS[c]} style={{ fontSize: 11, margin: 0 }}>{c}</Tag> },
  { title: 'Amount',      dataIndex: 'amount',      align: 'right', width: 92,
    render: a => <span style={{ color: a > 0 ? '#52c41a' : '#ff4d4f', fontFamily: 'monospace', fontSize: 12, fontWeight: 600 }}>{a > 0 ? '+' : '-'}${Math.abs(a).toFixed(2)}</span> },
]

export function DashboardPage() {
  const navigate = useNavigate()
  const { data, isLoading } = useQuery({ queryKey: ['dashboard'], queryFn: getDashboardSummary })

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />
  if (!data) return null

  const habitPct = data.habits_total ? Math.round(data.habits_done_today / data.habits_total * 100) : 0
  const budgetPct = data.budget_total ? Math.round(data.budget_spent / data.budget_total * 100) : 0

  const stats = [
    { label: 'Net Worth',      val: `$${data.net_worth_trend[data.net_worth_trend.length - 1]?.toLocaleString() ?? '—'}`, sub: '↑ +2.1% this month', subC: '#52c41a', spark: data.net_worth_trend, sparkC: '#52c41a', nav: '/finance' },
    { label: "Today's Habits", val: `${data.habits_done_today} / ${data.habits_total}`, sub: `${habitPct}% complete`, subC: '#1677ff', pct: habitPct, nav: '/health' },
    { label: 'Goals (avg)',    val: `${data.goals_avg_progress}%`, sub: 'active OKRs', subC: '#722ed1', pct: data.goals_avg_progress, pctC: '#722ed1', nav: '/goals' },
    { label: 'Monthly Budget', val: `$${data.budget_total.toLocaleString()}`, sub: `$${data.budget_spent.toLocaleString()} spent · ${budgetPct}%`, subC: '#fa8c16', pct: budgetPct, pctC: '#fa8c16', nav: '/finance' },
  ]

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        {stats.map((s, i) => (
          <Col span={6} key={i}>
            <Card size="small" hoverable style={{ cursor: 'pointer' }} onClick={() => navigate(s.nav)}>
              <div style={{ fontSize: 12, color: '#999', marginBottom: 4 }}>{s.label}</div>
              <div style={{ fontSize: 22, fontWeight: 700, marginBottom: 4 }}>{s.val}</div>
              {s.spark && <Sparkline data={s.spark} color={s.sparkC} width={100} height={28} />}
              {s.pct !== undefined && <Progress percent={s.pct} size="small" showInfo={false} strokeColor={s.pctC ?? '#1677ff'} style={{ margin: '4px 0 2px' }} />}
              <div style={{ fontSize: 12, color: s.subC }}>{s.sub}</div>
            </Card>
          </Col>
        ))}
      </Row>
      <Row gutter={[12, 12]}>
        <Col span={24}>
          <Card size="small" title={<span style={{ fontSize: 13 }}>Recent Transactions</span>} extra={<a onClick={() => navigate('/finance')} style={{ fontSize: 12 }}>View all →</a>}>
            <Table dataSource={data.recent_transactions} columns={txColumns} size="small" pagination={false} rowKey="id" />
          </Card>
        </Col>
      </Row>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/DashboardPage.tsx
git commit -m "feat: dashboard page with live API data"
```

---

### Task 5: Finance page

**Files:**
- Modify: `frontend/src/pages/FinancePage.tsx`

- [ ] **Step 1: Implement `frontend/src/pages/FinancePage.tsx`**

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Table, Tag, Button, Form, Input, Select, InputNumber, Modal, Progress, Spin } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { getTransactions, createTransaction, deleteTransaction, getBudgets } from '../api/endpoints'
import type { Transaction } from '../api/types'

const CATEGORIES = ['Food', 'Income', 'Entertainment', 'Health', 'Tech', 'Auto', 'Utilities', 'Shopping']
const CAT_COLORS: Record<string, string> = { Food: 'green', Income: 'blue', Entertainment: 'purple', Health: 'volcano', Tech: 'cyan', Auto: 'orange', Utilities: 'gold', Shopping: 'magenta' }

export function FinancePage() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: txs = [], isLoading } = useQuery({ queryKey: ['transactions'], queryFn: () => getTransactions() })
  const { data: budgets = [] } = useQuery({ queryKey: ['budgets'], queryFn: getBudgets })

  const addMutation = useMutation({
    mutationFn: createTransaction,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['transactions'] }); setAddOpen(false); form.resetFields() },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteTransaction,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['transactions'] }),
  })

  const columns: ColumnsType<Transaction> = [
    { title: 'Date', dataIndex: 'date', width: 90 },
    { title: 'Description', dataIndex: 'description', ellipsis: true },
    { title: 'Category', dataIndex: 'category', width: 130, render: c => <Tag color={CAT_COLORS[c]}>{c}</Tag> },
    { title: 'Amount', dataIndex: 'amount', align: 'right', width: 100,
      render: a => <span style={{ color: a > 0 ? '#52c41a' : '#ff4d4f', fontWeight: 600 }}>{a > 0 ? '+' : '-'}${Math.abs(a).toFixed(2)}</span> },
    { title: '', width: 40, render: (_, row) => <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(row.id)} /> },
  ]

  const totalIncome = txs.filter(t => t.amount > 0).reduce((s, t) => s + t.amount, 0)
  const totalExpenses = txs.filter(t => t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0)

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        {[
          { label: 'Income', val: `$${totalIncome.toFixed(2)}`, color: '#52c41a' },
          { label: 'Expenses', val: `$${totalExpenses.toFixed(2)}`, color: '#ff4d4f' },
          { label: 'Net', val: `$${(totalIncome - totalExpenses).toFixed(2)}`, color: '#1677ff' },
        ].map((s, i) => (
          <Col span={8} key={i}>
            <Card size="small">
              <div style={{ fontSize: 12, color: '#999' }}>{s.label}</div>
              <div style={{ fontSize: 22, fontWeight: 700, color: s.color }}>{s.val}</div>
            </Card>
          </Col>
        ))}
      </Row>

      {budgets.length > 0 && (
        <Card size="small" title="Budgets" style={{ marginBottom: 12 }}>
          <Row gutter={[12, 8]}>
            {budgets.map(b => {
              const spent = txs.filter(t => t.category === b.category && t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0)
              const pct = Math.min(Math.round(spent / b.monthly_limit * 100), 100)
              return (
                <Col span={8} key={b.id}>
                  <div style={{ fontSize: 12, marginBottom: 2 }}>{b.category} <span style={{ color: '#999' }}>${spent.toFixed(0)} / ${b.monthly_limit.toFixed(0)}</span></div>
                  <Progress percent={pct} size="small" strokeColor={pct > 90 ? '#ff4d4f' : '#1677ff'} />
                </Col>
              )
            })}
          </Row>
        </Card>
      )}

      <Card size="small"
        title="Transactions"
        extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}
      >
        {isLoading ? <Spin /> : <Table dataSource={txs} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} />}
      </Card>

      <Modal title="Add Transaction" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="date" label="Date" rules={[{ required: true }]}><Input type="date" /></Form.Item>
          <Form.Item name="description" label="Description" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="category" label="Category" rules={[{ required: true }]}>
            <Select options={CATEGORIES.map(c => ({ value: c, label: c }))} />
          </Form.Item>
          <Form.Item name="amount" label="Amount (negative = expense)" rules={[{ required: true }]}><InputNumber style={{ width: '100%' }} step={0.01} /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/FinancePage.tsx
git commit -m "feat: finance page with transactions CRUD and budget progress"
```

---

### Task 6: Health & Habits page

**Files:**
- Modify: `frontend/src/pages/HealthPage.tsx`

- [ ] **Step 1: Implement `frontend/src/pages/HealthPage.tsx`**

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, Row, Col, Button, Modal, Form, Input, Spin, Tooltip } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { getHabits, createHabit, deleteHabit, getHabitLogs, toggleHabitLog } from '../api/endpoints'

const today = new Date().toISOString().split('T')[0]

export function HealthPage() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: habits = [], isLoading } = useQuery({ queryKey: ['habits'], queryFn: getHabits })
  const { data: logs = [] } = useQuery({ queryKey: ['habit-logs', today], queryFn: () => getHabitLogs(today) })

  const addMutation = useMutation({
    mutationFn: createHabit,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['habits'] }); setAddOpen(false); form.resetFields() },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteHabit,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['habits'] }),
  })

  const toggleMutation = useMutation({
    mutationFn: ({ habitId }: { habitId: string }) => toggleHabitLog(habitId, today),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['habit-logs', today] }),
  })

  const doneSet = new Set(logs.filter(l => l.done).map(l => l.habit_id))
  const donePct = habits.length ? Math.round(doneSet.size / habits.length * 100) : 0

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  return (
    <div>
      <Row gutter={[12, 12]}>
        <Col span={12}>
          <Card size="small" title={`Today's Habits — ${donePct}% done`}
            extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}
          >
            {habits.map(h => {
              const done = doneSet.has(h.id)
              return (
                <div key={h.id} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
                  <div
                    onClick={() => toggleMutation.mutate({ habitId: h.id })}
                    style={{ width: 22, height: 22, borderRadius: '50%', cursor: 'pointer', flexShrink: 0, background: done ? '#52c41a' : '#f0f0f0', border: done ? 'none' : '1.5px solid #d9d9d9', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 11, color: '#fff' }}
                  >{done ? '✓' : ''}</div>
                  <span style={{ fontSize: 13, flex: 1, textDecoration: done ? 'line-through' : 'none', color: done ? '#bbb' : '#222' }}>{h.icon} {h.name}</span>
                  <Tooltip title="Delete habit">
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(h.id)} />
                  </Tooltip>
                </div>
              )
            })}
            {habits.length === 0 && <div style={{ color: '#bbb', textAlign: 'center', padding: 20 }}>No habits yet. Add your first!</div>}
          </Card>
        </Col>
      </Row>

      <Modal title="Add Habit" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="name" label="Habit name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="icon" label="Icon (emoji)" initialValue="✓"><Input /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/HealthPage.tsx
git commit -m "feat: health & habits page with toggle persistence"
```

---

### Task 7: Goals page

**Files:**
- Modify: `frontend/src/pages/GoalsPage.tsx`

- [ ] **Step 1: Implement `frontend/src/pages/GoalsPage.tsx`**

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Progress, Button, Modal, Form, Input, InputNumber, Checkbox, Spin } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { getGoals, createGoal, deleteGoal, addKeyResult, updateKeyResult } from '../api/endpoints'

export function GoalsPage() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: goals = [], isLoading } = useQuery({ queryKey: ['goals'], queryFn: getGoals })

  const addMutation = useMutation({
    mutationFn: createGoal,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['goals'] }); setAddOpen(false); form.resetFields() },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteGoal,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['goals'] }),
  })

  const toggleKrMutation = useMutation({
    mutationFn: ({ goalId, krId, done }: { goalId: string; krId: string; done: boolean }) =>
      updateKeyResult(goalId, krId, { done }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['goals'] }),
  })

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add Goal</Button>
      </div>
      <Row gutter={[12, 12]}>
        {goals.map(g => (
          <Col span={8} key={g.id}>
            <Card size="small"
              title={<span style={{ fontSize: 13, fontWeight: 600 }}>{g.name}</span>}
              extra={<Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(g.id)} />}
              style={{ borderTop: `3px solid ${g.color}` }}
            >
              <Progress percent={g.progress} strokeColor={g.color} size="small" style={{ marginBottom: 10 }} />
              {g.description && <div style={{ fontSize: 12, color: '#888', marginBottom: 8 }}>{g.description}</div>}
              {g.key_results.map(kr => (
                <div key={kr.id} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                  <Checkbox
                    checked={kr.done}
                    onChange={e => toggleKrMutation.mutate({ goalId: g.id, krId: kr.id, done: e.target.checked })}
                  />
                  <span style={{ fontSize: 12, textDecoration: kr.done ? 'line-through' : 'none', color: kr.done ? '#bbb' : '#222' }}>{kr.description}</span>
                </div>
              ))}
            </Card>
          </Col>
        ))}
        {goals.length === 0 && <Col span={24}><div style={{ color: '#bbb', textAlign: 'center', padding: 40 }}>No goals yet. Add your first!</div></Col>}
      </Row>

      <Modal title="Add Goal" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate({ ...values, key_results: [] })}>
          <Form.Item name="name" label="Goal name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="description" label="Description"><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="progress" label="Initial progress %" initialValue={0}><InputNumber min={0} max={100} style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="color" label="Color" initialValue="#1677ff"><Input type="color" /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/GoalsPage.tsx
git commit -m "feat: goals page with key results checkboxes"
```

---

### Task 8: Notes page

**Files:**
- Modify: `frontend/src/pages/NotesPage.tsx`

- [ ] **Step 1: Implement `frontend/src/pages/NotesPage.tsx`**

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Input, Button, Modal, Form, Tag, Switch, Spin } from 'antd'
import { PlusOutlined, DeleteOutlined, PushpinOutlined } from '@ant-design/icons'
import { getNotes, createNote, deleteNote, updateNote } from '../api/endpoints'

export function NotesPage() {
  const [search, setSearch] = useState('')
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: notes = [], isLoading } = useQuery({
    queryKey: ['notes', search],
    queryFn: () => getNotes({ search }),
  })

  const addMutation = useMutation({
    mutationFn: (values: any) => createNote({ ...values, tags: values.tags ? values.tags.split(',').map((t: string) => t.trim()) : [] }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['notes'] }); setAddOpen(false); form.resetFields() },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteNote,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notes'] }),
  })

  const pinMutation = useMutation({
    mutationFn: ({ id, pinned }: { id: string; pinned: boolean }) => updateNote(id, { pinned }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notes'] }),
  })

  return (
    <div>
      <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
        <Input.Search placeholder="Search notes..." value={search} onChange={e => setSearch(e.target.value)} style={{ maxWidth: 320 }} allowClear />
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add Note</Button>
      </div>

      {isLoading ? <Spin size="large" style={{ display: 'block', margin: '80px auto' }} /> : (
        <Row gutter={[12, 12]}>
          {notes.map(n => (
            <Col span={8} key={n.id}>
              <Card size="small"
                title={<span style={{ fontSize: 13 }}>{n.pinned && <PushpinOutlined style={{ color: '#faad14', marginRight: 6 }} />}{n.title}</span>}
                extra={
                  <div style={{ display: 'flex', gap: 4 }}>
                    <Button type="text" size="small" icon={<PushpinOutlined />} onClick={() => pinMutation.mutate({ id: n.id, pinned: !n.pinned })} />
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(n.id)} />
                  </div>
                }
              >
                <div style={{ fontSize: 12, color: '#555', marginBottom: 8, maxHeight: 60, overflow: 'hidden' }}>{n.content}</div>
                <div>{n.tags.map(t => <Tag key={t} style={{ fontSize: 11 }}>{t}</Tag>)}</div>
              </Card>
            </Col>
          ))}
          {notes.length === 0 && <Col span={24}><div style={{ color: '#bbb', textAlign: 'center', padding: 40 }}>No notes yet.</div></Col>}
        </Row>
      )}

      <Modal title="Add Note" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="title" label="Title" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="content" label="Content"><Input.TextArea rows={4} /></Form.Item>
          <Form.Item name="tags" label="Tags (comma-separated)"><Input /></Form.Item>
          <Form.Item name="pinned" label="Pinned" valuePropName="checked" initialValue={false}><Switch /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/NotesPage.tsx
git commit -m "feat: notes page with search and pin"
```

---

### Task 9: Calendar page

**Files:**
- Modify: `frontend/src/pages/CalendarPage.tsx`

- [ ] **Step 1: Implement `frontend/src/pages/CalendarPage.tsx`**

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Button, Modal, Form, Input, Switch, Spin } from 'antd'
import { PlusOutlined, DeleteOutlined, LeftOutlined, RightOutlined } from '@ant-design/icons'
import { getEvents, createEvent, deleteEvent } from '../api/endpoints'

export function CalendarPage() {
  const today = new Date()
  const [year, setYear] = useState(today.getFullYear())
  const [month, setMonth] = useState(today.getMonth()) // 0-indexed
  const [selectedDay, setSelectedDay] = useState(today.getDate())
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const fromDate = new Date(year, month, 1).toISOString()
  const toDate = new Date(year, month + 1, 0, 23, 59, 59).toISOString()

  const { data: events = [], isLoading } = useQuery({
    queryKey: ['events', year, month],
    queryFn: () => getEvents({ from: fromDate, to: toDate }),
  })

  const addMutation = useMutation({
    mutationFn: (values: any) => createEvent({
      title: values.title,
      start_at: new Date(`${values.date}T${values.start_time || '09:00'}`).toISOString(),
      end_at: new Date(`${values.date}T${values.end_time || '10:00'}`).toISOString(),
      color: values.color || '#1677ff',
      all_day: values.all_day || false,
    }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['events'] }); setAddOpen(false); form.resetFields() },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteEvent,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['events'] }),
  })

  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const firstDayOfWeek = new Date(year, month, 1).getDay()
  const monthName = new Date(year, month).toLocaleString('default', { month: 'long' })

  const dayEvents = events.filter(e => {
    const d = new Date(e.start_at)
    return d.getDate() === selectedDay && d.getMonth() === month
  })

  const eventsByDay = events.reduce<Record<number, typeof events>>((acc, e) => {
    const d = new Date(e.start_at).getDate()
    acc[d] = acc[d] || []
    acc[d].push(e)
    return acc
  }, {})

  const prevMonth = () => { if (month === 0) { setMonth(11); setYear(y => y - 1) } else setMonth(m => m - 1) }
  const nextMonth = () => { if (month === 11) { setMonth(0); setYear(y => y + 1) } else setMonth(m => m + 1) }

  return (
    <div>
      <Row gutter={[12, 12]}>
        <Col span={16}>
          <Card size="small" title={
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <Button type="text" size="small" icon={<LeftOutlined />} onClick={prevMonth} />
              <span style={{ fontSize: 14, fontWeight: 600 }}>{monthName} {year}</span>
              <Button type="text" size="small" icon={<RightOutlined />} onClick={nextMonth} />
            </div>
          } extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
            {isLoading ? <Spin /> : (
              <div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 1, marginBottom: 4 }}>
                  {['Sun','Mon','Tue','Wed','Thu','Fri','Sat'].map(d => (
                    <div key={d} style={{ textAlign: 'center', fontSize: 11, color: '#999', padding: '4px 0' }}>{d}</div>
                  ))}
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 1 }}>
                  {Array.from({ length: firstDayOfWeek }).map((_, i) => <div key={`e${i}`} />)}
                  {Array.from({ length: daysInMonth }, (_, i) => i + 1).map(day => {
                    const isToday = day === today.getDate() && month === today.getMonth() && year === today.getFullYear()
                    const isSelected = day === selectedDay
                    const hasEvents = !!eventsByDay[day]?.length
                    return (
                      <div key={day} onClick={() => setSelectedDay(day)} style={{ textAlign: 'center', padding: '6px 2px', cursor: 'pointer', borderRadius: 4, background: isSelected ? '#1677ff' : isToday ? '#e6f4ff' : 'transparent', color: isSelected ? '#fff' : isToday ? '#1677ff' : '#222', fontSize: 13, position: 'relative' }}>
                        {day}
                        {hasEvents && <div style={{ width: 4, height: 4, borderRadius: '50%', background: isSelected ? '#fff' : '#1677ff', margin: '2px auto 0' }} />}
                      </div>
                    )
                  })}
                </div>
              </div>
            )}
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title={<span style={{ fontSize: 13 }}>Events — {monthName} {selectedDay}</span>}>
            {dayEvents.length === 0 && <div style={{ color: '#bbb', textAlign: 'center', padding: 20, fontSize: 12 }}>No events this day.</div>}
            {dayEvents.map(e => (
              <div key={e.id} style={{ display: 'flex', gap: 8, padding: '6px 0', borderBottom: '1px solid #f5f5f5', alignItems: 'flex-start' }}>
                <div style={{ width: 3, height: 36, background: e.color, borderRadius: 2, flexShrink: 0, marginTop: 2 }} />
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 12, fontWeight: 500 }}>{e.title}</div>
                  <div style={{ fontSize: 11, color: '#bbb' }}>{e.all_day ? 'All day' : new Date(e.start_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</div>
                </div>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(e.id)} />
              </div>
            ))}
          </Card>
        </Col>
      </Row>

      <Modal title="Add Event" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate(values)}>
          <Form.Item name="title" label="Title" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="date" label="Date" rules={[{ required: true }]}><Input type="date" /></Form.Item>
          <Form.Item name="all_day" label="All day" valuePropName="checked" initialValue={false}><Switch /></Form.Item>
          <Form.Item name="start_time" label="Start time" initialValue="09:00"><Input type="time" /></Form.Item>
          <Form.Item name="end_time" label="End time" initialValue="10:00"><Input type="time" /></Form.Item>
          <Form.Item name="color" label="Color" initialValue="#1677ff"><Input type="color" /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/CalendarPage.tsx
git commit -m "feat: calendar page with month grid and event list"
```

---

### Task 10: Inventory page

**Files:**
- Modify: `frontend/src/pages/InventoryPage.tsx`

- [ ] **Step 1: Implement `frontend/src/pages/InventoryPage.tsx`**

```tsx
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Row, Col, Card, Table, Button, Modal, Form, Input, InputNumber, Spin } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { getAssets, createAsset, deleteAsset } from '../api/endpoints'
import type { Asset } from '../api/types'

export function InventoryPage() {
  const [addOpen, setAddOpen] = useState(false)
  const [form] = Form.useForm()
  const qc = useQueryClient()

  const { data: assets = [], isLoading } = useQuery({ queryKey: ['assets'], queryFn: getAssets })

  const addMutation = useMutation({
    mutationFn: createAsset,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['assets'] }); setAddOpen(false); form.resetFields() },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteAsset,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['assets'] }),
  })

  const categories = [...new Set(assets.map(a => a.category))]
  const categoryTotals = categories.map(cat => ({
    category: cat,
    total: assets.filter(a => a.category === cat).reduce((s, a) => s + a.value, 0),
    count: assets.filter(a => a.category === cat).length,
  }))
  const grandTotal = assets.reduce((s, a) => s + a.value, 0)

  const columns: ColumnsType<Asset> = [
    { title: 'Name',     dataIndex: 'name',        ellipsis: true },
    { title: 'Category', dataIndex: 'category',    width: 120 },
    { title: 'Value',    dataIndex: 'value',        width: 120, align: 'right', render: v => `$${v.toLocaleString()}` },
    { title: 'Bought',   dataIndex: 'purchased_at', width: 110, render: v => v ?? '—' },
    { title: '',         width: 40, render: (_, row) => <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => deleteMutation.mutate(row.id)} /> },
  ]

  return (
    <div>
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col span={6}>
          <Card size="small">
            <div style={{ fontSize: 12, color: '#999' }}>Total Assets</div>
            <div style={{ fontSize: 22, fontWeight: 700, color: '#52c41a' }}>${grandTotal.toLocaleString()}</div>
          </Card>
        </Col>
        {categoryTotals.map(ct => (
          <Col span={6} key={ct.category}>
            <Card size="small">
              <div style={{ fontSize: 12, color: '#999' }}>{ct.category} ({ct.count})</div>
              <div style={{ fontSize: 18, fontWeight: 600 }}>${ct.total.toLocaleString()}</div>
            </Card>
          </Col>
        ))}
      </Row>

      <Card size="small" title="Assets" extra={<Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>Add</Button>}>
        {isLoading ? <Spin /> : <Table dataSource={assets} columns={columns} size="small" rowKey="id" pagination={{ pageSize: 20 }} />}
      </Card>

      <Modal title="Add Asset" open={addOpen} onCancel={() => setAddOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={values => addMutation.mutate({ ...values, notes: values.notes || '' })}>
          <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="category" label="Category" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="value" label="Value ($)" rules={[{ required: true }]}><InputNumber min={0} step={0.01} style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="purchased_at" label="Purchase date"><Input type="date" /></Form.Item>
          <Form.Item name="notes" label="Notes"><Input.TextArea rows={2} /></Form.Item>
          <Button type="primary" htmlType="submit" loading={addMutation.isPending} block>Save</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/InventoryPage.tsx
git commit -m "feat: inventory page with category totals"
```

---

### Task 11: Settings page

**Files:**
- Modify: `frontend/src/pages/SettingsPage.tsx`

- [ ] **Step 1: Implement `frontend/src/pages/SettingsPage.tsx`**

```tsx
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, Switch, Form, Button, Row, Col, Spin, message } from 'antd'
import { getSettings, updateSettings } from '../api/endpoints'

const MODULES = ['finance', 'health', 'goals', 'notes', 'calendar', 'inventory']
const NOTIF_KEYS = ['email', 'push']

export function SettingsPage() {
  const qc = useQueryClient()
  const { data: settings, isLoading } = useQuery({ queryKey: ['settings'], queryFn: getSettings })

  const updateMutation = useMutation({
    mutationFn: updateSettings,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['settings'] }); message.success('Settings saved') },
  })

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />
  if (!settings) return null

  return (
    <Row gutter={[12, 12]}>
      <Col span={12}>
        <Card size="small" title="Notifications">
          {NOTIF_KEYS.map(key => (
            <div key={key} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
              <span style={{ fontSize: 13, textTransform: 'capitalize' }}>{key} notifications</span>
              <Switch
                checked={!!settings.notifications[key]}
                onChange={checked => updateMutation.mutate({ notifications: { ...settings.notifications, [key]: checked } })}
              />
            </div>
          ))}
        </Card>
      </Col>
      <Col span={12}>
        <Card size="small" title="Modules">
          {MODULES.map(mod => (
            <div key={mod} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
              <span style={{ fontSize: 13, textTransform: 'capitalize' }}>{mod}</span>
              <Switch
                checked={!!settings.modules_enabled[mod]}
                onChange={checked => updateMutation.mutate({ modules_enabled: { ...settings.modules_enabled, [mod]: checked } })}
              />
            </div>
          ))}
        </Card>
      </Col>
    </Row>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/SettingsPage.tsx
git commit -m "feat: settings page with module toggles and notifications"
```

---

### Phase 3 Complete

Build check:

```bash
cd frontend && npm run build
```

Expected: `dist/` produced, zero TypeScript errors.

End-to-end smoke test with backend running:

```bash
docker compose up --build -d
```

Open `http://localhost:5173` — should show login page. Sign in with any credentials (local dev mode). Verify all 8 pages load and CRUD operations work.
