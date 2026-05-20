package academic

import (
	"time"

	"github.com/google/uuid"
)

// ─── Domain entities ─────────────────────────────────────────────────────────

type Carrera struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Materia struct {
	ID        uuid.UUID `json:"id"`
	CarreraID uuid.UUID `json:"carrera_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Semester  int       `json:"semester"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StudentProfile struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	CarreraID   uuid.UUID `json:"carrera_id"`
	Semester    int       `json:"semester"`
	StudentCode *string   `json:"student_code,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Curso struct {
	ID            uuid.UUID `json:"id"`
	MateriaID     uuid.UUID `json:"materia_id"`
	MateriaName   string    `json:"materia_name,omitempty"`
	DocenteID     uuid.UUID `json:"docente_id"`
	DocenteEmail  string    `json:"docente_email,omitempty"`
	SemesterLabel string    `json:"semester_label"`
	Section       string    `json:"section"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Grupo struct {
	ID        uuid.UUID `json:"id"`
	CursoID   uuid.UUID `json:"curso_id"`
	Name      string    `json:"name"`
	Capacity  *int      `json:"capacity,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Actividad struct {
	ID             uuid.UUID `json:"id"`
	CursoID        uuid.UUID `json:"curso_id"`
	Title          string    `json:"title"`
	Tipo           string    `json:"tipo"`
	Deadline       time.Time `json:"deadline"`
	Instructions   string    `json:"instructions"`
	TraceEnabled   bool      `json:"trace_enabled"`
	HasSubmissions bool      `json:"has_submissions,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Inscripcion struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	CursoID          uuid.UUID  `json:"curso_id"`
	CarreraID        uuid.UUID  `json:"carrera_id"`
	Semester         int        `json:"semester"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RevocationReason *string    `json:"revocation_reason,omitempty"`
}

type GrupoAsignacion struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	GrupoID    uuid.UUID `json:"grupo_id"`
	CursoID    uuid.UUID `json:"curso_id"`
	AssignedBy uuid.UUID `json:"assigned_by"`
	AssignedAt time.Time `json:"assigned_at"`
}

type CodocenteGrant struct {
	ID          uuid.UUID  `json:"id"`
	ActividadID uuid.UUID  `json:"actividad_id"`
	CodocenteID uuid.UUID  `json:"codocente_id"`
	GrantedBy   uuid.UUID  `json:"granted_by"`
	GrantedAt   time.Time  `json:"granted_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	RevokedBy   *uuid.UUID `json:"revoked_by,omitempty"`
	IsActive    bool       `json:"is_active"`
}

// ─── Request types ────────────────────────────────────────────────────────────

type CreateCarreraRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type UpdateCarreraRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

type CreateMateriaRequest struct {
	CarreraID uuid.UUID `json:"carrera_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Semester  int       `json:"semester"`
}

type UpdateMateriaRequest struct {
	Name     *string `json:"name"`
	Semester *int    `json:"semester"`
	IsActive *bool   `json:"is_active"`
}

type CreateStudentProfileRequest struct {
	Email       string    `json:"email"`
	CarreraID   uuid.UUID `json:"carrera_id"`
	Semester    int       `json:"semester"`
	StudentCode *string   `json:"student_code"`
}

type UpdateStudentProfileRequest struct {
	CarreraID   *uuid.UUID `json:"carrera_id"`
	Semester    *int       `json:"semester"`
	StudentCode *string    `json:"student_code"`
	IsActive    *bool      `json:"is_active"`
}

type CreateCursoRequest struct {
	MateriaID     uuid.UUID `json:"materia_id"`
	SemesterLabel string    `json:"semester_label"`
	Section       string    `json:"section"`
}

type UpdateCursoRequest struct {
	SemesterLabel *string `json:"semester_label"`
	Section       *string `json:"section"`
}

type CreateGrupoRequest struct {
	Name     string `json:"name"`
	Capacity *int   `json:"capacity"`
}

type UpdateGrupoRequest struct {
	Name     *string `json:"name"`
	Capacity *int    `json:"capacity"`
}

type AssignStudentRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

type CreateActividadRequest struct {
	Title        string    `json:"title"`
	Tipo         string    `json:"tipo"`
	Deadline     time.Time `json:"deadline"`
	Instructions string    `json:"instructions"`
}

type UpdateActividadRequest struct {
	Title        *string    `json:"title"`
	Tipo         *string    `json:"tipo"`
	Deadline     *time.Time `json:"deadline"`
	Instructions *string    `json:"instructions"`
}

type CreateGrantRequest struct {
	CodocenteID uuid.UUID `json:"codocente_id"`
}

// ─── Response types ───────────────────────────────────────────────────────────

type RosterEntry struct {
	UserID     uuid.UUID  `json:"user_id"`
	Email      string     `json:"email"`
	EnrolledAt time.Time  `json:"enrolled_at"`
	Grupo      *GrupoRef  `json:"grupo"`
}

type GrupoRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type StudentCursoEntry struct {
	ID            uuid.UUID `json:"id"`
	MateriaName   string    `json:"materia_name"`
	DocenteEmail  string    `json:"docente_email"`
	SemesterLabel string    `json:"semester_label"`
	Section       string    `json:"section"`
	Grupo         *GrupoRef `json:"grupo"`
}

type StudentActividadEntry struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	Tipo         string    `json:"tipo"`
	Deadline     time.Time `json:"deadline"`
	Instructions string    `json:"instructions"`
	TraceEnabled bool      `json:"trace_enabled"`
	PostedAt     time.Time `json:"posted_at"`
	LastEditedAt time.Time `json:"last_edited_at"`
}

// ─── Errors ───────────────────────────────────────────────────────────────────

type ErrorResponse struct {
	Error         string `json:"error"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
}
