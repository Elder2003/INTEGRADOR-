import api from '../lib/api'
import { ApiError } from '../types'
import type { AxiosError } from 'axios'
import type { SubmissionResponse, GitAnalysisReport, AIReport, DetectionReport } from '../types/academic'

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

export async function submitGitRepo(actividadId: string, repoUrl: string): Promise<SubmissionResponse> {
  try {
    const res = await api.post<SubmissionResponse>(`/actividades/${actividadId}/git-submission`, { repo_url: repoUrl })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function updateGitRepo(actividadId: string, repoUrl: string): Promise<SubmissionResponse> {
  try {
    const res = await api.patch<SubmissionResponse>(`/actividades/${actividadId}/git-submission`, { repo_url: repoUrl })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function getGitAnalysis(submissionId: string): Promise<GitAnalysisReport> {
  try {
    const res = await api.get<GitAnalysisReport>(`/submissions/${submissionId}/git-analysis`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function getAIReport(submissionId: string): Promise<AIReport> {
  try {
    const res = await api.get<AIReport>(`/submissions/${submissionId}/ai-report`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function getDetectionReport(submissionId: string): Promise<DetectionReport> {
  try {
    const res = await api.get<DetectionReport>(`/submissions/${submissionId}/detection-report`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}
