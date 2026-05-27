import api from '../lib/api'
import { ApiError } from '../types'
import type { AxiosError } from 'axios'
import type {
  SubmissionResponse,
  SubmissionListResponse,
  ConfigChangesResponse,
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

export async function getSubmission(aid: string): Promise<SubmissionResponse | null> {
  try {
    const res = await api.get<SubmissionResponse>(`/actividades/${aid}/submission`)
    return res.data
  } catch (err) {
    const axiosErr = err as AxiosError<ApiErrorBody>
    if (axiosErr.response?.status === 404) return null
    toApiError(err)
  }
}

export async function uploadSubmission(
  aid: string,
  file: File,
  onProgress?: (pct: number) => void,
): Promise<SubmissionResponse> {
  try {
    const formData = new FormData()
    formData.append('file', file)
    const res = await api.post<SubmissionResponse>(`/actividades/${aid}/submission`, formData, {
      onUploadProgress: (e) => onProgress?.(Math.round((e.loaded * 100) / (e.total ?? 1))),
    })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function replaceSubmission(
  aid: string,
  file: File,
  onProgress?: (pct: number) => void,
): Promise<SubmissionResponse> {
  try {
    const formData = new FormData()
    formData.append('file', file)
    const res = await api.put<SubmissionResponse>(`/actividades/${aid}/submission`, formData, {
      onUploadProgress: (e) => onProgress?.(Math.round((e.loaded * 100) / (e.total ?? 1))),
    })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function deleteSubmission(aid: string): Promise<void> {
  try {
    await api.delete(`/actividades/${aid}/submission`)
  } catch (err) {
    toApiError(err)
  }
}

export async function studentDownloadSubmission(aid: string, filename: string): Promise<void> {
  try {
    const res = await api.get(`/actividades/${aid}/submission/download`, {
      responseType: 'blob',
    })
    const url = URL.createObjectURL(new Blob([res.data]))
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  } catch (err) {
    toApiError(err)
  }
}

export async function listSubmissions(cid: string, aid: string): Promise<SubmissionListResponse> {
  try {
    const res = await api.get<SubmissionListResponse>(
      `/cursos/${cid}/actividades/${aid}/submissions`,
    )
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function downloadSubmission(sid: string, filename: string): Promise<void> {
  try {
    const res = await api.get(`/submissions/${sid}/download`, { responseType: 'blob' })
    const url = URL.createObjectURL(new Blob([res.data]))
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  } catch (err) {
    toApiError(err)
  }
}

export async function listConfigChanges(cid: string, aid: string): Promise<ConfigChangesResponse> {
  try {
    const res = await api.get<ConfigChangesResponse>(
      `/cursos/${cid}/actividades/${aid}/config-changes`,
    )
    return res.data
  } catch (err) {
    toApiError(err)
  }
}
