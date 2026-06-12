import api from '../lib/api'
import { ApiError } from '../types'
import type { AxiosError } from 'axios'
import type {
  SessionResponse,
  TraceLogResponse,
  ActiveSessionsResponse,
} from '../types/academic'

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

export async function createSession(aid: string): Promise<SessionResponse> {
  try {
    const res = await api.post<SessionResponse>(`/actividades/${aid}/session`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function endSession(sid: string): Promise<void> {
  try {
    await api.post(`/sessions/${sid}/end`)
  } catch (err) {
    toApiError(err)
  }
}

export async function getTraceLog(submissionId: string): Promise<TraceLogResponse> {
  try {
    const res = await api.get<TraceLogResponse>(`/submissions/${submissionId}/trace`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function listActiveSessions(cid: string, aid: string): Promise<ActiveSessionsResponse> {
  try {
    const res = await api.get<ActiveSessionsResponse>(
      `/cursos/${cid}/actividades/${aid}/sessions/active`,
    )
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function getLiveEvents(sid: string, since?: string): Promise<TraceLogResponse> {
  try {
    const url = since
      ? `/sessions/${sid}/events?since=${encodeURIComponent(since)}`
      : `/sessions/${sid}/events`
    const res = await api.get<TraceLogResponse>(url)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}
