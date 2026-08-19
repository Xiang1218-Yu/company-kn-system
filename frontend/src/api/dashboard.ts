import api from './client'
import type { Dashboard } from '../types'

// Dashboard API. Manager/admin-only on the backend.
export const dashboardApi = {
  load: () => api.get<unknown, Dashboard>('/dashboard'),
}
