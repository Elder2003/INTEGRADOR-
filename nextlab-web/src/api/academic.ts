import api from '../lib/api'
import { ApiError } from '../types'
import type { AxiosError } from 'axios'
import type {
  Materia,
  Curso,
  Actividad,
  Grupo,
  RosterEntry,
  CodocenteGrant,
  CreateCursoRequest,
  UpdateCursoRequest,
  CreateActividadRequest,
  UpdateActividadRequest,
  CreateGrupoRequest,
  UpdateGrupoRequest,
} from '../types/academic'

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

// Materias — admin-only endpoint; returns [] on 403 for docentes
export async function getMaterias(): Promise<Materia[]> {
  try {
    const res = await api.get<Materia[]>('/admin/materias')
    return res.data
  } catch (err) {
    const axiosErr = err as AxiosError
    if (axiosErr.response?.status === 403) return []
    toApiError(err)
  }
}

// Cursos
export async function getCursos(): Promise<Curso[]> {
  try {
    const res = await api.get<Curso[]>('/cursos')
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function getCurso(id: string): Promise<Curso> {
  try {
    const res = await api.get<Curso>(`/cursos/${id}`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function createCurso(data: CreateCursoRequest): Promise<Curso> {
  try {
    const res = await api.post<Curso>('/cursos', data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function updateCurso(id: string, data: UpdateCursoRequest): Promise<Curso> {
  try {
    const res = await api.put<Curso>(`/cursos/${id}`, data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function deleteCurso(id: string): Promise<void> {
  try {
    await api.delete(`/cursos/${id}`)
  } catch (err) {
    toApiError(err)
  }
}

// Actividades
export async function getActividades(cursoId: string): Promise<Actividad[]> {
  try {
    const res = await api.get<Actividad[]>(`/cursos/${cursoId}/actividades`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function getActividad(id: string): Promise<Actividad> {
  try {
    const res = await api.get<Actividad>(`/actividades/${id}`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function createActividad(cursoId: string, data: CreateActividadRequest): Promise<Actividad> {
  try {
    const res = await api.post<Actividad>(`/cursos/${cursoId}/actividades`, data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function updateActividad(id: string, data: UpdateActividadRequest): Promise<Actividad> {
  try {
    const res = await api.put<Actividad>(`/actividades/${id}`, data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function deleteActividad(id: string): Promise<void> {
  try {
    await api.delete(`/actividades/${id}`)
  } catch (err) {
    toApiError(err)
  }
}

// Grupos
export async function getGrupos(cursoId: string): Promise<Grupo[]> {
  try {
    const res = await api.get<Grupo[]>(`/cursos/${cursoId}/grupos`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function createGrupo(cursoId: string, data: CreateGrupoRequest): Promise<Grupo> {
  try {
    const res = await api.post<Grupo>(`/cursos/${cursoId}/grupos`, data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function updateGrupo(cursoId: string, grupoId: string, data: UpdateGrupoRequest): Promise<Grupo> {
  try {
    const res = await api.put<Grupo>(`/cursos/${cursoId}/grupos/${grupoId}`, data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function deleteGrupo(cursoId: string, grupoId: string): Promise<void> {
  try {
    await api.delete(`/cursos/${cursoId}/grupos/${grupoId}`)
  } catch (err) {
    toApiError(err)
  }
}

// Student roster
export async function getRoster(cursoId: string): Promise<RosterEntry[]> {
  try {
    const res = await api.get<RosterEntry[]>(`/cursos/${cursoId}/students`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

// Group membership
export async function addStudentToGrupo(cursoId: string, grupoId: string, userId: string): Promise<void> {
  try {
    await api.post(`/cursos/${cursoId}/grupos/${grupoId}/students`, { user_id: userId })
  } catch (err) {
    toApiError(err)
  }
}

export async function removeStudentFromGrupo(cursoId: string, grupoId: string, userId: string): Promise<void> {
  try {
    await api.delete(`/cursos/${cursoId}/grupos/${grupoId}/students/${userId}`)
  } catch (err) {
    toApiError(err)
  }
}

// Co-docente grants
export async function grantAccess(actividadId: string, userId: string): Promise<CodocenteGrant> {
  try {
    const res = await api.post<CodocenteGrant>(`/actividades/${actividadId}/grants`, { user_id: userId })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function revokeAccess(actividadId: string, userId: string): Promise<void> {
  try {
    await api.delete(`/actividades/${actividadId}/grants/${userId}`)
  } catch (err) {
    toApiError(err)
  }
}
