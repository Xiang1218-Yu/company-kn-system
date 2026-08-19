import { create } from 'zustand'
import type { User } from '../types'
import { authApi } from '../api/auth'

// AuthStore holds the current user and the login/register actions. The token
// itself lives in localStorage (see api/client) so it survives reloads; this
// store holds the derived user object and loading flags for the UI.
interface AuthState {
  user: User | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, name: string, role?: string) => Promise<void>
  logout: () => void
  restore: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set) => ({
  user: (() => {
    // Seed from localStorage so the UI doesn't flash the login screen before the
    // restore() call confirms the session is still valid.
    const raw = localStorage.getItem('kn_user')
    return raw ? (JSON.parse(raw) as User) : null
  })(),
  loading: false,

  async login(email, password) {
    set({ loading: true })
    try {
      const res = await authApi.login({ email, password })
      persist(res.user, res.token)
      set({ user: res.user })
    } finally {
      set({ loading: false })
    }
  },

  async register(email, password, name, role) {
    set({ loading: true })
    try {
      const res = await authApi.register({ email, password, name, role })
      persist(res.user, res.token)
      set({ user: res.user })
    } finally {
      set({ loading: false })
    }
  },

  logout() {
    localStorage.removeItem('kn_token')
    localStorage.removeItem('kn_user')
    set({ user: null })
  },

  async restore() {
    if (!localStorage.getItem('kn_token')) {
      set({ user: null })
      return
    }
    try {
      const user = await authApi.me()
      localStorage.setItem('kn_user', JSON.stringify(user))
      set({ user })
    } catch {
      localStorage.removeItem('kn_token')
      localStorage.removeItem('kn_user')
      set({ user: null })
    }
  },
}))

function persist(user: User, token: string) {
  localStorage.setItem('kn_token', token)
  localStorage.setItem('kn_user', JSON.stringify(user))
}
