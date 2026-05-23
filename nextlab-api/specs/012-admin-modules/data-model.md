# Data Model: Admin Modules (M12)

## Migration 000012_admin_modules.up.sql

```sql
-- Add full_name to users (all existing rows get empty string default)
ALTER TABLE users ADD COLUMN full_name TEXT NOT NULL DEFAULT '';

-- Format template library (admin-managed)
CREATE TABLE format_templates (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL UNIQUE,
    description TEXT        NOT NULL DEFAULT '',
    rules       JSONB       NOT NULL DEFAULT '{}',
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_by  UUID        NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Tracks which template was applied to an actividad (snapshot at apply time)
CREATE TABLE actividad_format_template (
    actividad_id        UUID        PRIMARY KEY REFERENCES actividades(id) ON DELETE CASCADE,
    format_template_id  UUID        NOT NULL REFERENCES format_templates(id),
    applied_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    applied_by          UUID        NOT NULL REFERENCES users(id)
);

-- Extra curricular RAG scopes for interdisciplinary activities
CREATE TABLE actividad_interdisciplinary_scopes (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actividad_id    UUID        NOT NULL REFERENCES actividades(id) ON DELETE CASCADE,
    carrera         TEXT        NOT NULL,
    semestre        SMALLINT    NOT NULL CHECK (semestre BETWEEN 1 AND 10),
    codigo_materia  TEXT        NOT NULL,
    UNIQUE (actividad_id, carrera, semestre, codigo_materia)
);

CREATE INDEX ais_actividad_idx ON actividad_interdisciplinary_scopes(actividad_id);
```

## Go Structs

### internal/admin package

```go
type DocenteAccount struct {
    ID        uuid.UUID  `json:"id"`
    Email     string     `json:"email"`
    FullName  string     `json:"full_name"`
    IsActive  bool       `json:"is_active"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}

type CreateDocenteRequest struct {
    Email    string `json:"email"`
    FullName string `json:"full_name"`
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
    Name        *string                 `json:"name,omitempty"`
    Description *string                 `json:"description,omitempty"`
    Rules       *format.FormatRulesConfig `json:"rules,omitempty"`
    IsActive    *bool                   `json:"is_active,omitempty"`
}

type AppliedTemplate struct {
    FormatTemplateID uuid.UUID  `json:"format_template_id"`
    Name             string     `json:"name"`
    Rules            format.FormatRulesConfig `json:"rules"`
    AppliedAt        time.Time  `json:"applied_at"`
}

type CurricularDocument struct {
    ID               uuid.UUID  `json:"id"`
    Carrera          string     `json:"carrera"`
    Semestre         int        `json:"semestre"`
    CodigoMateria    string     `json:"codigo_materia"`
    OriginalFilename string     `json:"original_filename"`
    IsRubric         bool       `json:"is_rubric"`
    MinioPath        string     `json:"minio_path"`
    CreatedAt        time.Time  `json:"created_at"`
}

type InterdisciplinaryScope struct {
    ID            uuid.UUID `json:"id"`
    ActividadID   uuid.UUID `json:"actividad_id"`
    Carrera       string    `json:"carrera"`
    Semestre      int       `json:"semestre"`
    CodigoMateria string    `json:"codigo_materia"`
}
```

## Relationships

- `format_templates` → no FK deps on actividades (templates are independent)
- `actividad_format_template.actividad_id` → `actividades(id)` — one-to-one (each actividad can have at most one applied template)
- `actividad_format_template.format_template_id` → `format_templates(id)` — prevents deletion of in-use templates
- `actividad_interdisciplinary_scopes.actividad_id` → `actividades(id)` — one actividad can have many extra scopes
- `users.full_name` — plain column addition, no FK
