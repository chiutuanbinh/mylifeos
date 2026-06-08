import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Spin } from 'antd'
import { supabase, useAuthStore } from '../store/auth'

export function AuthCallbackPage() {
  const navigate = useNavigate()
  const setSession = useAuthStore(s => s.setSession)

  useEffect(() => {
    supabase?.auth.getSession().then(({ data }) => {
      const token = data.session?.access_token
      if (token) {
        setSession(token)
        navigate('/', { replace: true })
      } else {
        navigate('/login', { replace: true })
      }
    })
  }, [navigate, setSession])

  return (
    <div style={{ display: 'flex', height: '100vh', alignItems: 'center', justifyContent: 'center' }}>
      <Spin size="large" tip="Signing in..." />
    </div>
  )
}
