import axios from 'axios'
import { clearTokens, getRefreshToken } from './auth'
import type { LoginResponse } from '../types'

let accessTokenGetter: (() => string | null) | null = null

export function setAccessTokenGetter(fn: () => string | null): void {
  accessTokenGetter = fn
}

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL,
})

api.interceptors.request.use((config) => {
  const token = accessTokenGetter ? accessTokenGetter() : null
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let refreshPromise: Promise<LoginResponse> | null = null

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config

    if (
      error.response?.status !== 401 ||
      original._retry ||
      original.url?.includes('/auth/')
    ) {
      return Promise.reject(error)
    }

    original._retry = true

    if (!refreshPromise) {
      const rt = getRefreshToken()
      if (!rt) {
        clearTokens()
        window.location.href = '/login?session_expired=1'
        return Promise.reject(error)
      }

      refreshPromise = api
        .post<LoginResponse>('/auth/refresh', { refresh_token: rt })
        .then((res) => res.data)
        .catch((refreshError) => {
          clearTokens()
          window.location.href = '/login?session_expired=1'
          return Promise.reject(refreshError)
        })
        .finally(() => {
          refreshPromise = null
        })
    }

    try {
      const data = await refreshPromise
      if (accessTokenGetter) {
        original.headers.Authorization = `Bearer ${data.access_token}`
      }
      return api(original)
    } catch {
      return Promise.reject(error)
    }
  }
)

export default api
