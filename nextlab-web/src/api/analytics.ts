import api from '../lib/api'
import { ApiError } from '../types'
import type { AxiosError } from 'axios'
import type { Alert, ActivityAnalytics, CourseAnalytics } from '../types/academic'

interface ApiErrorBody {
  error: string
  message: string
}

function toApiError(err: unknown): never {
  const axiosErr = err as AxiosError<ApiErrorBody>
  if (axiosErr.response) {
    const { error, message } = axiosErr.response.data ?? {}
    throw new ApiError(message ?? 'An error occurred', error ?? 'unknown_error')
  }
  if (axiosErr.request) {
    throw new ApiError('Unable to reach the server. Check your connection.', 'network_error')
  }
  throw new ApiError('An unexpected error occurred', 'unknown_error')
}

export async function listAlerts(): Promise<Alert[]> {
  try {
    const res = await api.get<Alert[]>('/docente/alerts')
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function dismissAlert(alertId: string): Promise<void> {
  try {
    await api.post(`/alerts/${alertId}/dismiss`)
  } catch (err) {
    toApiError(err)
  }
}

export async function getActivityAnalytics(aid: string): Promise<ActivityAnalytics> {
  try {
    const res = await api.get<ActivityAnalytics>(`/actividades/${aid}/analytics`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function getCourseAnalytics(cid: string): Promise<CourseAnalytics> {
  try {
    const res = await api.get<CourseAnalytics>(`/cursos/${cid}/analytics`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}
