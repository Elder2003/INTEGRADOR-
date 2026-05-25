import api from '../lib/api'
import { ApiError, type LoginResponse } from '../types'
import type { AxiosError } from 'axios'

interface ApiErrorBody {
  error: string
  message: string
  retry_after?: number
}

function toApiError(err: unknown): never {
  const axiosErr = err as AxiosError<ApiErrorBody>
  if (axiosErr.response) {
    const { error, message, retry_after } = axiosErr.response.data ?? {}
    throw new ApiError(message ?? 'An error occurred', error ?? 'unknown_error', retry_after)
  }
  if (axiosErr.request) {
    throw new ApiError('Unable to reach the server. Check your connection.', 'network_error')
  }
  throw new ApiError('An unexpected error occurred', 'unknown_error')
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  try {
    const res = await api.post<LoginResponse>('/auth/login', { email, password })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function refreshToken(rt: string): Promise<LoginResponse> {
  try {
    const res = await api.post<LoginResponse>('/auth/refresh', { refresh_token: rt })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function logout(refreshTokenValue: string): Promise<void> {
  try {
    await api.post('/auth/logout', { refresh_token: refreshTokenValue })
  } catch (err) {
    toApiError(err)
  }
}
