import api from '../lib/api'
import { ApiError } from '../types'
import type { AxiosError } from 'axios'
import type {
  GradeDecision,
  InvestigaciónStage,
  SubmissionResponse,
  DefensaSlotsResponse,
  DefensaSlot,
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

export async function createGrade(
  sid: string,
  score: number,
  feedback: string,
): Promise<GradeDecision> {
  try {
    const res = await api.post<GradeDecision>(`/submissions/${sid}/grade`, { score, feedback })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function updateGrade(
  sid: string,
  score: number,
  feedback: string,
): Promise<GradeDecision> {
  try {
    const res = await api.patch<GradeDecision>(`/submissions/${sid}/grade`, { score, feedback })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function submitExamAnswer(
  aid: string,
  answer: string,
): Promise<SubmissionResponse> {
  try {
    const res = await api.post<SubmissionResponse>(`/actividades/${aid}/submission/exam`, { answer })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function updateStage(
  sid: string,
  stage: InvestigaciónStage,
): Promise<{ stage: InvestigaciónStage }> {
  try {
    const res = await api.patch<{ stage: InvestigaciónStage }>(`/submissions/${sid}/stage`, { stage })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function listDefensaSlots(
  cid: string,
  aid: string,
): Promise<DefensaSlotsResponse> {
  try {
    const res = await api.get<DefensaSlotsResponse>(
      `/cursos/${cid}/actividades/${aid}/defensa-slots`,
    )
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function assignDefensaSlot(
  sid: string,
  scheduledAt: string,
  location: string | null,
): Promise<DefensaSlot> {
  try {
    const res = await api.put<DefensaSlot>(`/submissions/${sid}/defensa-slot`, {
      scheduled_at: scheduledAt,
      location,
    })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function getMyDefensaSlot(aid: string): Promise<DefensaSlot | null> {
  try {
    const res = await api.get<DefensaSlot | null>(`/actividades/${aid}/defensa-slot/me`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}
