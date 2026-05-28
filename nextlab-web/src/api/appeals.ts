import api from '../lib/api'
import { ApiError } from '../types'
import type { AxiosError } from 'axios'
import type { Appeal, AppealDecision } from '../types/academic'

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

interface AppealsListResponse {
  appeals: Appeal[]
  total: number
}

export interface ReviewResponse {
  appeal: Appeal
  decision: AppealDecision | null
}

export async function getAppeal(sessionId: string): Promise<Appeal | null> {
  try {
    const res = await api.get<AppealsListResponse>('/appeals', { params: { session_id: sessionId } })
    return res.data.appeals[0] ?? null
  } catch (err) {
    toApiError(err)
  }
}

export async function createAppeal(sessionId: string, reason: string): Promise<Appeal> {
  try {
    const res = await api.post<Appeal>('/appeals', {
      session_id: sessionId,
      type: 'grade_dispute',
      reason,
    })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function reviewAppeal(
  appealId: string,
  decision: 'upheld' | 'rejected',
  justification: string,
): Promise<ReviewResponse> {
  try {
    const res = await api.put<ReviewResponse>(`/appeals/${appealId}/review`, {
      decision,
      justification,
    })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function listActivityAppeals(actividadId: string): Promise<Appeal[]> {
  try {
    const res = await api.get<AppealsListResponse>(`/actividades/${actividadId}/appeals`)
    return res.data.appeals
  } catch (err) {
    toApiError(err)
  }
}
