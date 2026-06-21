import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, Avatar, Badge, Space, Button, Breadcrumb, Typography, Grid } from 'antd'
import {
  DashboardOutlined, TrophyOutlined,
  CalendarOutlined, SettingOutlined, AccountBookOutlined,
  BellOutlined, MenuFoldOutlined, MenuUnfoldOutlined, LogoutOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../store/auth'

const NAV = [
  { key: '/',         icon: <DashboardOutlined />, label: 'Dashboard' },
  { type: 'divider' as const },
  { key: '/finance', icon: <AccountBookOutlined />, label: 'Finance' },
  { key: '/objectives', icon: <TrophyOutlined />, label: 'Objectives' },
  { key: '/calendar', icon: <CalendarOutlined />, label: 'Calendar' },
  { type: 'divider' as const },
  { key: '/settings', icon: <SettingOutlined />,  label: 'Settings' },
]

const BOTTOM_NAV = [
  { key: '/',           icon: <DashboardOutlined />, label: 'Home' },
  { key: '/finance',    icon: <AccountBookOutlined />, label: 'Finance' },
  { key: '/objectives', icon: <TrophyOutlined />,   label: 'Goals' },
  { key: '/calendar',   icon: <CalendarOutlined />, label: 'Calendar' },
  { key: '/settings',   icon: <SettingOutlined />,  label: 'Settings' },
]

const TITLES: Record<string, string> = {
  '/': 'Dashboard',
  '/finance': 'Finance',
  '/objectives': 'Objectives',
  '/calendar': 'Calendar & Schedule',
  '/settings': 'Settings',
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const [collapsed, setCollapsed] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const signOut = useAuthStore(s => s.signOut)
  const { xs } = Grid.useBreakpoint()
  const isMobile = !!xs

  if (isMobile) {
    return (
      <Layout style={{ minHeight: '100vh' }}>
        <Layout.Header style={{ background: '#fff', padding: '0 12px', height: 48, lineHeight: '48px', borderBottom: '1px solid #f0f0f0', position: 'sticky', top: 0, zIndex: 99, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <div style={{ width: 24, height: 24, borderRadius: 5, background: 'linear-gradient(135deg, #1677ff, #0952c6)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: 12, fontWeight: 800 }}>M</div>
            <span style={{ fontSize: 14, fontWeight: 700, color: '#111' }}>{TITLES[location.pathname] || 'MyLifeOS'}</span>
          </div>
          <Space size={4}>
            <Badge count={0} size="small">
              <Button type="text" size="small" icon={<BellOutlined />} />
            </Badge>
            <Button type="text" size="small" icon={<LogoutOutlined />} onClick={() => signOut()} />
          </Space>
        </Layout.Header>
        <Layout.Content style={{ padding: 12, background: '#f0f2f5', minHeight: 'calc(100vh - 48px - 56px)', paddingBottom: 68 }}>
          {children}
        </Layout.Content>
        <div style={{ position: 'fixed', bottom: 0, left: 0, right: 0, height: 56, background: '#fff', borderTop: '1px solid #f0f0f0', display: 'flex', zIndex: 100 }}>
          {BOTTOM_NAV.map(item => {
            const active = location.pathname === item.key
            return (
              <button
                key={item.key}
                onClick={() => navigate(item.key)}
                style={{ flex: 1, border: 'none', background: 'none', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 2, cursor: 'pointer', color: active ? '#1677ff' : '#999', fontSize: 10, fontWeight: active ? 600 : 400, padding: 0 }}
              >
                <span style={{ fontSize: 18 }}>{item.icon}</span>
                {item.label}
              </button>
            )
          })}
        </div>
      </Layout>
    )
  }

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
