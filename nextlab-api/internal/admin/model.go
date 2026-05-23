package admin

import (
	"time"

	"github.com/Elder2003/nextlab-api/internal/format"
	"github.com/google/uuid"
)

type DocenteAccount struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateDocenteRequest struct {
	Email            string  `json:"email"`
	FullName         string  `json:"full_name"`
	TestPassword     *string `json:"test_password,omitempty"`
}

type UpdateDocenteRequest struct {
	FullName *string `json:"full_name,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

type FormatTemplate struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Rules       format.FormatRulesConfig `json:"rules"`
	IsActive    bool                   `json:"is_active"`
	CreatedBy   uuid.UUID              `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type CreateFormatTemplateRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Rules       format.FormatRulesConfig `json:"rules"`
}

type UpdateFormatTemplateRequest struct {
	Name        *string                  `json:"name,omitempty"`
	Description *string                  `json:"description,omitempty"`
	Rules       *format.FormatRulesConfig `json:"rules,omitempty"`
	IsActive    *bool                    `json:"is_active,omitempty"`
}

type AppliedTemplate struct {
	FormatTemplateID uuid.UUID              `json:"format_template_id"`
	Name             string                 `json:"name"`
	Rules            format.FormatRulesConfig `json:"rules"`
	AppliedAt        time.Time              `json:"applied_at"`
}

type CurricularDocument struct {
	ID               uuid.UUID `json:"id"`
	Carrera          string    `json:"carrera"`
	Semestre         int       `json:"semestre"`
	CodigoMateria    string    `json:"codigo_materia"`
	OriginalFilename string    `json:"original_filename"`
	IsRubric         bool      `json:"is_rubric"`
	MinioPath        string    `json:"minio_path"`
	CreatedAt        time.Time `json:"created_at"`
}

type InterdisciplinaryScope struct {
	ID            uuid.UUID `json:"id"`
	ActividadID   uuid.UUID `json:"actividad_id"`
	Carrera       string    `json:"carrera"`
	Semestre      int       `json:"semestre"`
	CodigoMateria string    `json:"codigo_materia"`
}

type InterdisciplinaryScopeInput struct {
	Carrera       string `json:"carrera"`
	Semestre      int    `json:"semestre"`
	CodigoMateria string `json:"codigo_materia"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
