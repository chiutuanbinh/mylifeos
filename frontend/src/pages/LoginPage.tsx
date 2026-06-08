import { useNavigate } from 'react-router-dom'
import { Form, Input, Button, Checkbox, Typography, message, Divider } from 'antd'
import { UserOutlined, LockOutlined, LoginOutlined } from '@ant-design/icons'
import { useAuthStore } from '../store/auth'

function GoogleIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" style={{ display: 'block' }}>
      <path fill="#4285F4" d="M17.64 9.2c0-.637-.057-1.251-.164-1.84H9v3.481h4.844c-.209 1.125-.843 2.078-1.796 2.717v2.258h2.908c1.702-1.567 2.684-3.875 2.684-6.615z"/>
      <path fill="#34A853" d="M9 18c2.43 0 4.467-.806 5.956-2.184l-2.908-2.258c-.806.54-1.837.86-3.048.86-2.344 0-4.328-1.584-5.036-3.711H.957v2.332A8.997 8.997 0 0 0 9 18z"/>
      <path fill="#FBBC05" d="M3.964 10.707A5.41 5.41 0 0 1 3.682 9c0-.593.102-1.17.282-1.707V4.961H.957A8.996 8.996 0 0 0 0 9c0 1.452.348 2.827.957 4.039l3.007-2.332z"/>
      <path fill="#EA4335" d="M9 3.58c1.321 0 2.508.454 3.44 1.345l2.582-2.58C13.463.891 11.426 0 9 0A8.997 8.997 0 0 0 .957 4.961L3.964 7.293C4.672 5.163 6.656 3.58 9 3.58z"/>
    </svg>
  )
}

export function LoginPage() {
  const { signIn, signInWithGoogle, loading } = useAuthStore()
  const navigate = useNavigate()

  const onFinish = async ({ email, password }: { email: string; password: string }) => {
    try {
      await signIn(email, password)
      navigate('/')
    } catch {
      message.error('Invalid credentials')
    }
  }

  const onGoogle = async () => {
    try {
      await signInWithGoogle()
      // redirect happens automatically via OAuth flow
    } catch {
      message.error('Google sign-in failed')
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

          <Button
            block
            size="large"
            icon={<GoogleIcon />}
            onClick={onGoogle}
            style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, marginBottom: 4, fontWeight: 500, borderColor: '#d9d9d9' }}
          >
            Continue with Google
          </Button>

          <Divider style={{ margin: '20px 0', color: '#bbb', fontSize: 12 }}>or sign in with email</Divider>

          <Form layout="vertical" size="large" onFinish={onFinish}>
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
