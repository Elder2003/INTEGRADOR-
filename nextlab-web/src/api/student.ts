import api from '../lib/api'
import { ApiError } from '../types'
import type { AxiosError } from 'axios'
import type { StudentCursoEntry, StudentActividadEntry } from '../types/academic'

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

export async function getStudentCursos(): Promise<StudentCursoEntry[]> {
  try {
    const res = await api.get<StudentCursoEntry[]>('/me/cursos')
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function getStudentActividades(cursoId: string): Promise<StudentActividadEntry[]> {
  try {
    const res = await api.get<StudentActividadEntry[]>(`/me/cursos/${cursoId}/actividades`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}
