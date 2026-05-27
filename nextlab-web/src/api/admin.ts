import api from '../lib/api'
import { ApiError } from '../types'
import type { AxiosError } from 'axios'
import type {
  Carrera,
  Materia,
  StudentProfile,
  CreateCarreraRequest,
  UpdateCarreraRequest,
  CreateMateriaRequest,
  UpdateMateriaRequest,
  CreateStudentProfileRequest,
  UpdateStudentProfileRequest,
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

// Carreras
export async function getCarreras(): Promise<Carrera[]> {
  try {
    const res = await api.get<Carrera[]>('/admin/carreras')
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function createCarrera(data: CreateCarreraRequest): Promise<Carrera> {
  try {
    const res = await api.post<Carrera>('/admin/carreras', data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function updateCarrera(id: string, data: UpdateCarreraRequest): Promise<Carrera> {
  try {
    const res = await api.put<Carrera>(`/admin/carreras/${id}`, data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function deleteCarrera(id: string): Promise<void> {
  try {
    await api.delete(`/admin/carreras/${id}`)
  } catch (err) {
    toApiError(err)
  }
}

// Materias
export async function getAdminMaterias(): Promise<Materia[]> {
  try {
    const res = await api.get<Materia[]>('/admin/materias')
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function createMateria(data: CreateMateriaRequest): Promise<Materia> {
  try {
    const res = await api.post<Materia>('/admin/materias', data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function updateMateria(id: string, data: UpdateMateriaRequest): Promise<Materia> {
  try {
    const res = await api.put<Materia>(`/admin/materias/${id}`, data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function deleteMateria(id: string): Promise<void> {
  try {
    await api.delete(`/admin/materias/${id}`)
  } catch (err) {
    toApiError(err)
  }
}

// Student Profiles
export async function getStudentProfiles(): Promise<StudentProfile[]> {
  try {
    const res = await api.get<StudentProfile[]>('/admin/student-profiles')
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function createStudentProfile(data: CreateStudentProfileRequest): Promise<StudentProfile> {
  try {
    const res = await api.post<StudentProfile>('/admin/student-profiles', data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function updateStudentProfile(id: string, data: UpdateStudentProfileRequest): Promise<StudentProfile> {
  try {
    const res = await api.put<StudentProfile>(`/admin/student-profiles/${id}`, data)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function deactivateStudentProfile(id: string): Promise<StudentProfile> {
  try {
    const res = await api.put<StudentProfile>(`/admin/student-profiles/${id}`, { is_active: false })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}
