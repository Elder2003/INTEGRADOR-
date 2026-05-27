import api from '../lib/api'
import { ApiError } from '../types'
import type { AxiosError } from 'axios'
import type { Appeal } from '../types/academic'

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

export async function getAppeal(submissionId: string): Promise<Appeal | null> {
  try {
    const res = await api.get<Appeal>(`/submissions/${submissionId}/appeal`)
    return res.data
  } catch (err) {
    const axiosErr = err as AxiosError
    if (axiosErr.response?.status === 404) return null
    toApiError(err)
  }
}

export async function createAppeal(submissionId: string, student_message: string): Promise<Appeal> {
  try {
    const res = await api.post<Appeal>(`/submissions/${submissionId}/appeal`, { student_message })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function resolveAppeal(
  submissionId: string,
  decision: 'accept' | 'reject',
  response_note: string,
): Promise<Appeal> {
  try {
    const res = await api.post<Appeal>(`/submissions/${submissionId}/appeal/resolve`, {
      decision,
      response_note,
    })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}
