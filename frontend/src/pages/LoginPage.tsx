import { useNavigate } from 'react-router-dom'
import { Form, Input, Button, Checkbox, Typography, message } from 'antd'
import { UserOutlined, LockOutlined, LoginOutlined } from '@ant-design/icons'
import { useAuthStore } from '../store/auth'

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
              <Checkbox defaultChecked style={{ fontSize: 13 }}>Remember me</Checkbox>
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
