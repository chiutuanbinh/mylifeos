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
