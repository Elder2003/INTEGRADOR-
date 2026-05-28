import api from '../lib/api'
import { ApiError } from '../types'
import type { AxiosError } from 'axios'

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

export interface MateriaSuggestion {
  codigo_materia: string
  carrera: string
  semestre: number
  relevance_score: number
  excerpt: string
}

export interface SolveProblemResponse {
  problem: string
  suggestion: string
  materias: MateriaSuggestion[]
}

export async function solveProblem(problem: string): Promise<SolveProblemResponse> {
  try {
    const res = await api.post<SolveProblemResponse>('/rag/curricular/solve', { problem })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export interface CurriculumDocument {
  id: string
  carrera: string
  semestre: number
  codigo_materia: string
  original_filename: string
  is_rubric: boolean
  minio_path: string
  created_at: string
}

export interface IngestResult {
  document_id: string
  chunk_count: number
  status: string
}

export async function ingestCurricular(
  file: File,
  carrera: string,
  semestre: number,
  codigoMateria: string,
  isRubric: boolean,
): Promise<IngestResult> {
  try {
    const form = new FormData()
    form.append('file', file)
    form.append('carrera', carrera)
    form.append('semestre', String(semestre))
    form.append('codigo_materia', codigoMateria)
    if (isRubric) form.append('is_rubric', 'true')
    const res = await api.post<IngestResult>('/rag/curricular/ingest', form)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function listCurriculumDocs(): Promise<{ documents: CurriculumDocument[]; total: number }> {
  try {
    const res = await api.get<{ documents: CurriculumDocument[]; total: number }>('/admin/curriculum/documents', {
      params: { limit: 100 },
    })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function deleteCurriculumDoc(id: string): Promise<void> {
  try {
    await api.delete(`/admin/curriculum/documents/${id}`)
  } catch (err) {
    toApiError(err)
  }
}
