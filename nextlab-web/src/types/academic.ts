// MW2 — Academic domain types (API shapes)

export type ActividadTipo = 'practica' | 'tarea' | 'informe' | 'examen' | 'defensa_oral'

export interface Materia {
  id: string
  code: string
  name: string
  carrera_id: string
  semester: number
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface Carrera {
  id: string
  code: string
  name: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface StudentProfile {
  id: string
  email: string
  carrera_id: string
  semester: number
  student_code: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateCarreraRequest {
  code: string
  name: string
}

export interface UpdateCarreraRequest {
  name?: string
  is_active?: boolean
}

export interface CreateMateriaRequest {
  carrera_id: string
  code: string
  name: string
  semester: number
}

export interface UpdateMateriaRequest {
  name?: string
  semester?: number
  is_active?: boolean
}

export interface CreateStudentProfileRequest {
  email: string
  carrera_id: string
  semester: number
  student_code?: string
}

export interface UpdateStudentProfileRequest {
  semester?: number
  student_code?: string | null
  is_active?: boolean
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
  git_enabled: boolean
  ai_enabled: boolean
  detection_enabled: boolean
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

export interface SubmissionResponse {
  id: string
  actividad_id: string
  session_id: string | null
  original_filename: string
  file_size_bytes: number
  mime_type: string
  submitted_at: string
  is_late: boolean
  edit_window_expires_at: string | null
  is_locked: boolean
  last_replaced_at: string | null
  grade: GradeDecision | null
  stage: InvestigaciónStage | null
  exam_answer: string | null
  repo_url: string | null
}

export interface StudentInfo {
  id: string
  name: string
  email: string
}

export interface SubmissionListItem {
  id: string
  student: StudentInfo
  original_filename: string
  file_size_bytes: number
  mime_type: string
  submitted_at: string
  is_late: boolean
  is_locked: boolean
  last_replaced_at: string | null
  edit_window_expires_at: string | null
  session_id: string | null
  grade: GradeDecision | null
  stage: InvestigaciónStage | null
  repo_url: string | null
}

export interface SubmissionListResponse {
  actividad_id: string
  submissions: SubmissionListItem[]
  total: number
}

export interface ConfigChangeActor {
  id: string
  name: string
}

export interface ConfigChangeResponse {
  id: string
  changed_by: ConfigChangeActor
  field_name: string
  previous_value: string
  new_value: string
  changed_at: string
}

export interface ConfigChangesResponse {
  actividad_id: string
  changes: ConfigChangeResponse[]
}

export interface GrupoRef {
  id: string
  name: string
}

export interface StudentCursoEntry {
  id: string
  materia_name: string
  docente_email: string
  semester_label: string
  section: string
  grupo: GrupoRef | null
}

export interface StudentActividadEntry {
  id: string
  title: string
  tipo: ActividadTipo
  deadline: string
  instructions: string
  trace_enabled: boolean
  git_enabled: boolean
  ai_enabled: boolean
  detection_enabled: boolean
  posted_at: string
  last_edited_at: string
}

export interface SessionResponse {
  id: string
  actividad_id: string
  student_id: string
  started_at: string
  ended_at: string | null
}

export interface TraceLogEntry {
  id: string
  session_id: string
  type: 'keystroke' | 'copy' | 'paste' | 'focus_in' | 'focus_out'
  timestamp: string
  content: string | null
}

export interface TraceLogResponse {
  session_id: string
  events: TraceLogEntry[]
  total: number
}

export interface ActiveSessionEntry {
  session_id: string
  student: StudentInfo
  started_at: string
  event_count: number
}

export interface ActiveSessionsResponse {
  actividad_id: string
  sessions: ActiveSessionEntry[]
}

// MW7 — Grading types

export interface GradeDecision {
  id: string
  submission_id: string
  score: number
  feedback: string
  graded_at: string
  graded_by: string
}

export type InvestigaciónStage = 'propuesta' | 'desarrollo' | 'revisión' | 'aprobado'

export interface DefensaSlot {
  submission_id: string
  scheduled_at: string
  location: string | null
}

export interface DefensaSlotEntry {
  submission: SubmissionListItem
  slot: DefensaSlot | null
}

export interface DefensaSlotsResponse {
  actividad_id: string
  entries: DefensaSlotEntry[]
}

// MW8 — Git & AI Results types

export interface ActividadFlags {
  git_enabled: boolean
  ai_enabled: boolean
  detection_enabled: boolean
}

export interface GitFileEntry {
  path: string
  additions: number
  deletions: number
}

export interface GitAnalysisReport {
  status: 'pending' | 'done' | 'error'
  error_message?: string | null
  commit_count?: number
  authors?: string[]
  date_from?: string
  date_to?: string
  files?: GitFileEntry[]
}

export interface AIReportSection {
  title: string
  content: string
}

export interface AIReport {
  status: 'pending' | 'done' | 'error'
  error_message?: string | null
  sections?: AIReportSection[]
}

export interface FlaggedSegment {
  source: string
  match_percentage: number
  content?: string
}

export interface DetectionReport {
  status: 'pending' | 'done' | 'error'
  error_message?: string | null
  similarity_score?: number
  flagged_segments?: FlaggedSegment[]
}

// MW9 — Alerts & Analytics types

export interface Alert {
  id: string
  type: string
  severity: string
  description: string
  actividad_id: string | null
  curso_id: string | null
  actividad_title: string
  created_at: string
  is_dismissed: boolean
}

export interface GradeDistributionBucket {
  range_label: string
  count: number
}

export interface StageBreakdown {
  propuesta: number
  desarrollo: number
  revisión: number
  aprobado: number
}

export interface ActivityAnalytics {
  total_submissions: number
  late_submissions: number
  grade_distribution: GradeDistributionBucket[]
  stage_breakdown: StageBreakdown | null
}

export interface ActivitySummary {
  actividad_id: string
  activity_name: string
  submissions: number
  average_grade: number | null
  late_count: number
}

export interface CourseAnalytics {
  activity_summaries: ActivitySummary[]
  overall_grade_distribution: GradeDistributionBucket[]
}

// MW10 — Appeals types (M13 contract)

export type AppealStatus = 'pending' | 'escalated' | 'resolved' | 'admin_resolved'

export interface Appeal {
  id: string
  session_id: string
  filed_by: string
  type: string
  status: AppealStatus
  reason: string
  created_at: string
}

export interface AppealDecision {
  id: string
  appeal_id: string
  decided_by: string
  role: string
  decision: 'upheld' | 'rejected' | 'overridden'
  justification: string
  decided_at: string
}

export interface FileAppealRequest {
  session_id: string
  type: string
  reason: string
}

export interface ReviewRequest {
  decision: 'upheld' | 'rejected'
  justification: string
}
