// MW2 — Academic domain types (API shapes)

export type ActividadTipo = 'practica' | 'tarea' | 'informe' | 'examen' | 'defensa_oral'

export interface Materia {
  id: string
  nombre: string
  codigo: string
  is_active: boolean
}

export interface Curso {
  id: string
  materia_id: string
  materia_nombre: string
  semester: string
  section: string
  docente_id: string
  student_count: number
  created_at: string
}

export interface Actividad {
  id: string
  curso_id: string
  title: string
  tipo: ActividadTipo
  deadline: string
  instructions: string
  late_submission_allowed: boolean
  has_submissions: boolean
  created_at: string
}

export interface Grupo {
  id: string
  curso_id: string
  nombre: string
  capacity: number | null
  member_count: number
  created_at: string
}

export interface RosterEntry {
  user_id: string
  email: string
  enrolled_at: string
  grupo_id: string | null
  grupo_nombre: string | null
}

export interface CodocenteGrant {
  id: string
  actividad_id: string
  user_id: string
  granted_at: string
}

export interface CreateCursoRequest {
  materia_id: string
  semester: string
  section: string
}

export interface UpdateCursoRequest {
  semester: string
  section: string
}

export interface CreateActividadRequest {
  title: string
  tipo: ActividadTipo
  deadline: string
  instructions: string
}

export interface UpdateActividadRequest {
  title?: string
  tipo?: ActividadTipo
  deadline?: string
  instructions?: string
  late_submission_allowed?: boolean
}

export interface CreateGrupoRequest {
  nombre: string
  capacity?: number
}

export interface UpdateGrupoRequest {
  nombre?: string
  capacity?: number
}
