CREATE TABLE alerts (
    id           BIGSERIAL    PRIMARY KEY,
    docente_id   UUID         NOT NULL REFERENCES users(id),
    actividad_id UUID         REFERENCES actividades(id) ON DELETE SET NULL,
    student_id   UUID         REFERENCES users(id),
    type         TEXT         NOT NULL,
    payload      JSONB        NOT NULL DEFAULT '{}',
    advisory     BOOLEAN      NOT NULL DEFAULT false,
    is_read      BOOLEAN      NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX alerts_docente_id_idx  ON alerts (docente_id, id);
CREATE INDEX alerts_actividad_id_idx ON alerts (actividad_id);
