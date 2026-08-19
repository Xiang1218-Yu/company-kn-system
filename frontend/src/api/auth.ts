import api from './client'
import type { AuthResponse, User } from '../types'

// Auth API. Each function is one endpoint; keeping them grouped by resource
// makes the backend contract legible at a glance.
export const authApi = {
  register: (body: { email: string; password: string; name: string; role?: string }) =>
    api.post<unknown, AuthResponse>('/auth/register', body),
  login: (body: { email: string; password: string }) =>
    api.post<unknown, AuthResponse>('/auth/login', body),
  me: () => api.get<unknown, User>('/auth/me'),
}
