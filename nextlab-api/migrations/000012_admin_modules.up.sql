-- M12: Admin Modules — Curriculum, format templates, interdisciplinary

ALTER TABLE users ADD COLUMN IF NOT EXISTS full_name TEXT NOT NULL DEFAULT '';

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

CREATE TABLE actividad_format_template (
    actividad_id        UUID        PRIMARY KEY REFERENCES actividades(id) ON DELETE CASCADE,
    format_template_id  UUID        NOT NULL REFERENCES format_templates(id),
    applied_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    applied_by          UUID        NOT NULL REFERENCES users(id)
);

CREATE TABLE actividad_interdisciplinary_scopes (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actividad_id    UUID        NOT NULL REFERENCES actividades(id) ON DELETE CASCADE,
    carrera         TEXT        NOT NULL,
    semestre        SMALLINT    NOT NULL CHECK (semestre BETWEEN 1 AND 10),
    codigo_materia  TEXT        NOT NULL,
    UNIQUE (actividad_id, carrera, semestre, codigo_materia)
);

CREATE INDEX ais_actividad_idx ON actividad_interdisciplinary_scopes(actividad_id);
