import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import { useAuthStore } from './store/auth'
import { AppShell } from './components/AppShell'
import { LoginPage } from './pages/LoginPage'
import { AuthCallbackPage } from './pages/AuthCallbackPage'
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
          <Route path="/auth/callback" element={<AuthCallbackPage />} />
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
