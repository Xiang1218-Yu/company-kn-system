import axios, { AxiosError } from 'axios'
import type { APIEnvelope } from '../types'

// api is the single axios instance. The request interceptor attaches the JWT
// from localStorage so every call is authenticated without per-call plumbing.
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('kn_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// The response interceptor unwraps the API envelope: on success it returns
// `data.data` directly so callers get the payload; on failure it throws an
// Error carrying the server's message. A 401 clears the stored token and
// redirects to login so expired sessions don't leave the UI in a broken state.
api.interceptors.response.use(
  (response) => {
    const env = response.data as APIEnvelope<unknown>
    if (env && typeof env === 'object' && 'success' in env) {
      if (env.success) {
        return env.data as never
      }
      return Promise.reject(new Error(env.error?.message ?? '请求失败'))
    }
    return response.data
  },
  (error: AxiosError<APIEnvelope<unknown>>) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('kn_token')
      localStorage.removeItem('kn_user')
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    const msg = error.response?.data?.error?.message ?? error.message ?? '网络错误'
    return Promise.reject(new Error(msg))
  },
)

export default api
