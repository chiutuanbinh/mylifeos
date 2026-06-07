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
