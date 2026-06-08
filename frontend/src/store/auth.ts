import { create } from 'zustand'
import { createClient } from '@supabase/supabase-js'

const supabaseUrl = import.meta.env.VITE_SUPABASE_URL || ''
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY || ''

export const supabase = supabaseUrl
  ? createClient(supabaseUrl, supabaseAnonKey)
  : null

const PROVIDER_TOKEN_KEY = 'gcal_provider_token'

export function getStoredProviderToken(): string | null {
  try {
    const raw = localStorage.getItem(PROVIDER_TOKEN_KEY)
    if (!raw) return null
    const { token, expiresAt } = JSON.parse(raw)
    if (Date.now() > expiresAt) { localStorage.removeItem(PROVIDER_TOKEN_KEY); return null }
    return token
  } catch { return null }
}

// Restore session from Supabase on page load, keep in sync on token refresh.
if (supabase) {
  supabase.auth.getSession().then(({ data }) => {
    const token = data.session?.access_token
    if (token) useAuthStore.setState({ token })
  })
  supabase.auth.onAuthStateChange((event, session) => {
    useAuthStore.setState({ token: session?.access_token ?? null })
    if (event === 'SIGNED_IN' && session?.provider_token) {
      // Google access tokens last ~1 hour
      localStorage.setItem(PROVIDER_TOKEN_KEY, JSON.stringify({
        token: session.provider_token,
        expiresAt: Date.now() + 55 * 60 * 1000,
      }))
    }
    if (event === 'SIGNED_OUT') {
      localStorage.removeItem(PROVIDER_TOKEN_KEY)
    }
  })
}

interface AuthState {
  token: string | null
  loading: boolean
  signIn: (email: string, password: string) => Promise<void>
  signInWithGoogle: () => Promise<void>
  setSession: (token: string) => void
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
        set({ token: 'local-dev-token' })
      }
    } finally {
      set({ loading: false })
    }
  },

  signInWithGoogle: async () => {
    if (!supabase) {
      set({ token: 'local-dev-token' })
      return
    }
    const { error } = await supabase.auth.signInWithOAuth({
      provider: 'google',
      options: {
        redirectTo: `${window.location.origin}/auth/callback`,
        scopes: 'email profile https://www.googleapis.com/auth/calendar.readonly',
      },
    })
    if (error) throw error
  },

  setSession: (token) => set({ token }),

  signOut: async () => {
    if (supabase) await supabase.auth.signOut()
    set({ token: null })
  },
}))
