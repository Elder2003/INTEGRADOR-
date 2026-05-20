# Data Model: M3 — Academic Management

## Migration: `000003_academic`

### ALTER TABLE users

```sql
-- Students authenticate via Google OAuth and have no password.
-- google_sub is the stable Google user identifier (Subject claim in the ID token).
ALTER TABLE users
  ALTER COLUMN password_hash DROP NOT NULL,
  ADD COLUMN IF NOT EXISTS google_sub TEXT UNIQUE;

CREATE INDEX IF NOT EXISTS idx_users_google_sub ON users(google_sub) WHERE google_sub IS NOT NULL;
```

---

### carreras

```sql
CREATE TABLE carreras (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  code       TEXT        NOT NULL UNIQUE,
  name       TEXT        NOT NULL,
  is_active  BOOLEAN     NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**Notes**:
- `code` is the short institutional identifier (e.g. `"IS"` for Ingeniería de Sistemas).
- Deletion is soft via `is_active = false`. Existing materias and cursos that reference a deactivated carrera are unaffected.

---

### materias

```sql
CREATE TABLE materias (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  carrera_id UUID        NOT NULL REFERENCES carreras(id),
  code       TEXT        NOT NULL UNIQUE,
  name       TEXT        NOT NULL,
  semester   INTEGER     NOT NULL CHECK (semester BETWEEN 1 AND 10),
  is_active  BOOLEAN     NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_materias_carrera_semester
  ON materias(carrera_id, semester)
  WHERE is_active = true;
```

**Notes**:
- `semester` is the academic semester number within the carrera (1–10).
- Used by `EnrollmentSync` to find all cursos a student should be enrolled in.

---

### student_profiles

```sql
CREATE TABLE student_profiles (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  email        TEXT        NOT NULL UNIQUE,
  carrera_id   UUID        NOT NULL REFERENCES carreras(id),
  semester     INTEGER     NOT NULL CHECK (semester BETWEEN 1 AND 10),
  student_code TEXT,
  is_active    BOOLEAN     NOT NULL DEFAULT true,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_student_profiles_email ON student_profiles(email);
```

**Notes**:
- Populated and maintained by the admin before students log in.
- `email` must match the student's `@unifranz.edu.bo` Google account exactly.
- `student_code` is the institutional student ID (optional; useful for cross-referencing).
- On student Google login, the system looks up this table by email to determine `carrera_id` and `semester` before creating/updating the `users` row and running `EnrollmentSync`.

---

### cursos

```sql
CREATE TABLE cursos (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  materia_id     UUID        NOT NULL REFERENCES materias(id),
  docente_id     UUID        NOT NULL REFERENCES users(id),
  semester_label TEXT        NOT NULL,  -- e.g. "2026-1", "2026-2"
  section        TEXT        NOT NULL,  -- e.g. "A", "B", "Noche"
  is_active      BOOLEAN     NOT NULL DEFAULT true,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (materia_id, docente_id, semester_label, section)
);

CREATE INDEX idx_cursos_docente
  ON cursos(docente_id)
  WHERE is_active = true;

CREATE INDEX idx_cursos_materia
  ON cursos(materia_id)
  WHERE is_active = true;
```

**Notes**:
- Unique constraint prevents a docente from creating duplicate cursos for the same materia, semester, and section.
- `is_active = false` is the soft-delete path; hard DELETE is blocked if the curso has associated submissions (enforced at the service layer via a check before DELETE).
- `semester_label` is a human-readable string chosen by the docente (e.g. `"2026-1"`); it is not a FK to an academic calendar table.

---

### grupos

```sql
CREATE TABLE grupos (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  curso_id   UUID        NOT NULL REFERENCES cursos(id) ON DELETE CASCADE,
  name       TEXT        NOT NULL,
  capacity   INTEGER,                  -- NULL = unlimited
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (curso_id, name)
);
```

**Notes**:
- Cascades delete from curso (safe: grupos themselves have no submissions; submissions belong to inscripciones/students).
- `capacity` is informational; the system does not block assignments when capacity is reached in M3.

---

### actividades

```sql
CREATE TYPE actividad_tipo AS ENUM (
  'practica',
  'tarea',
  'informe',
  'examen',
  'defensa_oral'
);

CREATE TABLE actividades (
  id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  curso_id        UUID          NOT NULL REFERENCES cursos(id),
  title           TEXT          NOT NULL,
  tipo            actividad_tipo NOT NULL,
  deadline        TIMESTAMPTZ   NOT NULL,
  instructions    TEXT          NOT NULL DEFAULT '',
  trace_enabled   BOOLEAN       NOT NULL
                    GENERATED ALWAYS AS (tipo IN ('examen', 'defensa_oral')) STORED,
  has_submissions BOOLEAN       NOT NULL DEFAULT false,
  created_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX idx_actividades_curso ON actividades(curso_id);
```

**Notes**:
- `trace_enabled` is a stored generated column — it cannot be set manually. It is `true` only for `examen` and `defensa_oral`. M5 uses this flag to gate trace capture.
- `has_submissions` is initialized to `false` by M3. M4 (file storage) sets it to `true` on the first submission. Once `true`, `title` and `tipo` cannot be changed and the actividad cannot be deleted.
- `deadline` in the past is allowed (docente may record a past event).
- `instructions` defaults to empty string (not NULL) for clean JSON serialization.

---

### inscripciones

```sql
CREATE TABLE inscripciones (
  id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id           UUID        NOT NULL REFERENCES users(id),
  curso_id          UUID        NOT NULL REFERENCES cursos(id),
  carrera_id        UUID        NOT NULL REFERENCES carreras(id),
  semester          INTEGER     NOT NULL,
  is_active         BOOLEAN     NOT NULL DEFAULT true,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at        TIMESTAMPTZ,
  revocation_reason TEXT,
  UNIQUE (user_id, curso_id)
);

CREATE INDEX idx_inscripciones_user
  ON inscripciones(user_id)
  WHERE is_active = true;

CREATE INDEX idx_inscripciones_curso
  ON inscripciones(curso_id)
  WHERE is_active = true;
```

**Notes**:
- Created and revoked automatically by `EnrollmentSync` on every student login.
- `carrera_id` and `semester` are snapshots at the time of enrollment — useful for audit if the student's profile is later corrected.
- `revocation_reason` values used by M3: `'semester_advance'` (student moved to next semester), `'profile_update'` (admin corrected profile).
- Never hard-deleted. Admin can read full history including revoked rows.

---

### grupo_asignaciones

```sql
CREATE TABLE grupo_asignaciones (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        NOT NULL REFERENCES users(id),
  grupo_id    UUID        NOT NULL REFERENCES grupos(id) ON DELETE CASCADE,
  curso_id    UUID        NOT NULL REFERENCES cursos(id),
  assigned_by UUID        NOT NULL REFERENCES users(id),
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, curso_id)   -- one grupo per student per curso
);

CREATE INDEX idx_grupo_asignaciones_grupo ON grupo_asignaciones(grupo_id);
CREATE INDEX idx_grupo_asignaciones_user  ON grupo_asignaciones(user_id);
```

**Notes**:
- `curso_id` is denormalized from `grupo.curso_id` to support the `UNIQUE(user_id, curso_id)` constraint, which enforces one-group-per-student-per-course at the DB level.
- `assigned_by` records which docente made the assignment (audit trail).
- Hard delete is acceptable here (removing a student from a group has no submission implications in M3; M4 may add a guard).

---

### codocente_grants

```sql
CREATE TABLE codocente_grants (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  actividad_id UUID        NOT NULL REFERENCES actividades(id),
  codocente_id UUID        NOT NULL REFERENCES users(id),
  granted_by   UUID        NOT NULL REFERENCES users(id),
  granted_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at   TIMESTAMPTZ,
  revoked_by   UUID        REFERENCES users(id),
  is_active    BOOLEAN     NOT NULL DEFAULT true,
  UNIQUE (actividad_id, codocente_id)
);

CREATE INDEX idx_codocente_grants_codocente
  ON codocente_grants(codocente_id)
  WHERE is_active = true;
```

**Notes**:
- Scoped per actividad. A co_docente who needs access to multiple actividades requires one grant per actividad.
- Revocation sets `revoked_at`, `revoked_by`, and `is_active = false`. Never hard-deleted.
- `is_active` is redundant with `revoked_at IS NULL` but simplifies query filters.

---

## Entity Relationship Summary

```
carreras ──< materias ──< cursos >── usuarios(docente)
                              │
                  ┌───────────┼──────────────────┐
                  │           │                  │
               grupos      actividades      inscripciones >── usuarios(student)
                  │           │
         grupo_asignaciones  codocente_grants >── usuarios(co_docente)
         >── usuarios(student)
```

## Down Migration

```sql
-- 000003_academic.down.sql
DROP TABLE IF EXISTS codocente_grants;
DROP TABLE IF EXISTS grupo_asignaciones;
DROP TABLE IF EXISTS inscripciones;
DROP TABLE IF EXISTS actividades;
DROP TYPE  IF EXISTS actividad_tipo;
DROP TABLE IF EXISTS grupos;
DROP TABLE IF EXISTS cursos;
DROP TABLE IF EXISTS student_profiles;
DROP TABLE IF EXISTS materias;
DROP TABLE IF EXISTS carreras;

ALTER TABLE users
  ALTER COLUMN password_hash SET NOT NULL,
  DROP COLUMN IF EXISTS google_sub;

DROP INDEX IF EXISTS idx_users_google_sub;
```
