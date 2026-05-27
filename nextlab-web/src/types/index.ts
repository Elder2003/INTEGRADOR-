// --- Auth types (MW1) ---

export type Role = 'estudiante' | 'docente' | 'co_docente' | 'admin'

export interface User {
  id: string
  email: string
  role: Role
}

export interface AuthState {
  user: User | null
  accessToken: string | null
  isLoading: boolean
}

export interface JWTClaims {
  sub: string
  email: string
  role: Role
  jti: string
  iat: number
  exp: number
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  access_token: string
  token_type: string
  expires_in: number
  refresh_token: string
}

export interface RefreshRequest {
  refresh_token: string
}

export interface LogoutRequest {
  refresh_token: string
}

export class ApiError extends Error {
  code: string
  retryAfter?: number

  constructor(message: string, code: string, retryAfter?: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.retryAfter = retryAfter
  }
}

// --- Domain types (Bloque 4 prototype — will be replaced per milestone) ---

export type ActividadTipo = 'examen' | 'tarea' | 'informe' | 'practica' | 'proyecto'
export type ActividadEstado = 'activa' | 'proxima' | 'entregada' | 'calificada'
export type EtapaEstado = 'entregada' | 'activa' | 'pendiente'

export interface Etapa {
  id: string
  nombre: string
  estado: EtapaEstado
  fechaLimite: string
}

export interface Actividad {
  id: string
  tipo: ActividadTipo
  titulo: string
  materia: string
  docente: string
  fechaLimite: string | null
  estado: ActividadEstado
  auditoria: boolean
  etapas?: Etapa[]
}

export interface Proyecto extends Actividad {
  descripcion?: string
  defensa?: { fecha: string; id: string }
}

export interface CalificacionReciente {
  id: string
  actividad: string
  materia: string
  nota: number
  notaMaxima: number
  fecha: string
  tieneAlerta: boolean
}

export interface Pregunta {
  id: string
  numero: number
  puntos: number
  modalidad: 'texto' | 'codigo'
  lenguaje?: string
  enunciado: string
  placeholder: string
  maxCaracteres: number
}

export interface Examen {
  id: string
  titulo: string
  materia: string
  docente: string
  duracionMinutos: number
  preguntas: Pregunta[]
}

export interface Calificacion {
  id: string
  actividad: string
  tipo: ActividadTipo
  materia: string
  nota: number
  notaMaxima: number
  fecha: string
  tieneAlerta: boolean
  mensajeDocente: string | null
  puedeApelar: boolean
}

export interface Defensa {
  id: string
  proyecto: string
  docente: string
  preguntaActual: string
  historial: Array<{ pregunta: string; respuesta: string; timestamp: string }>
}

export interface FeedbackIA {
  pregunta: number
  feedback: string
  tipo: 'sugerencia' | 'corrección' | 'validación'
}

export interface Usuario {
  id: string
  nombre: string
  carrera: string
  semestre: string
}
