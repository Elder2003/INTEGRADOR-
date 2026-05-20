-- M3: Academic Management

-- Extend users for Google OAuth (students have no password)
ALTER TABLE users
  ALTER COLUMN password_hash DROP NOT NULL,
  ADD COLUMN IF NOT EXISTS google_sub TEXT UNIQUE;

CREATE INDEX IF NOT EXISTS idx_users_google_sub ON users(google_sub) WHERE google_sub IS NOT NULL;

-- Degree program catalog
CREATE TABLE carreras (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  code       TEXT        NOT NULL UNIQUE,
  name       TEXT        NOT NULL,
  is_active  BOOLEAN     NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Subject catalog
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

-- Admin-provisioned student enrollment data (email → carrera + semester)
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

-- Course instances (one per materia per docente per semester+section)
CREATE TABLE cursos (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  materia_id     UUID        NOT NULL REFERENCES materias(id),
  docente_id     UUID        NOT NULL REFERENCES users(id),
  semester_label TEXT        NOT NULL,
  section        TEXT        NOT NULL,
  is_active      BOOLEAN     NOT NULL DEFAULT true,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (materia_id, docente_id, semester_label, section)
);

CREATE INDEX idx_cursos_docente ON cursos(docente_id) WHERE is_active = true;
CREATE INDEX idx_cursos_materia ON cursos(materia_id) WHERE is_active = true;

-- Student groups within a curso
CREATE TABLE grupos (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  curso_id   UUID        NOT NULL REFERENCES cursos(id) ON DELETE CASCADE,
  name       TEXT        NOT NULL,
  capacity   INTEGER,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (curso_id, name)
);

-- Activity type enum
CREATE TYPE actividad_tipo AS ENUM (
  'practica',
  'tarea',
  'informe',
  'examen',
  'defensa_oral'
);

-- Activities and evaluations within a curso
CREATE TABLE actividades (
  id              UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
  curso_id        UUID           NOT NULL REFERENCES cursos(id),
  title           TEXT           NOT NULL,
  tipo            actividad_tipo NOT NULL,
  deadline        TIMESTAMPTZ    NOT NULL,
  instructions    TEXT           NOT NULL DEFAULT '',
  trace_enabled   BOOLEAN        NOT NULL GENERATED ALWAYS AS (tipo IN ('examen', 'defensa_oral')) STORED,
  has_submissions BOOLEAN        NOT NULL DEFAULT false,
  created_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX idx_actividades_curso ON actividades(curso_id);

-- System-generated course memberships (auto-created on student login)
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

CREATE INDEX idx_inscripciones_user  ON inscripciones(user_id)  WHERE is_active = true;
CREATE INDEX idx_inscripciones_curso ON inscripciones(curso_id) WHERE is_active = true;

-- Docente-managed group assignments (one grupo per student per curso)
CREATE TABLE grupo_asignaciones (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        NOT NULL REFERENCES users(id),
  grupo_id    UUID        NOT NULL REFERENCES grupos(id) ON DELETE CASCADE,
  curso_id    UUID        NOT NULL REFERENCES cursos(id),
  assigned_by UUID        NOT NULL REFERENCES users(id),
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, curso_id)
);

CREATE INDEX idx_grupo_asignaciones_grupo ON grupo_asignaciones(grupo_id);
CREATE INDEX idx_grupo_asignaciones_user  ON grupo_asignaciones(user_id);

-- Per-activity co-teaching grants
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
