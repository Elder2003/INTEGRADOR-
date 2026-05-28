-- NextLab test seed — run once after migrations
-- Password for all accounts: test1234
-- Safe to re-run (ON CONFLICT DO NOTHING throughout)

-- ─── Users ────────────────────────────────────────────────────────────────────
-- bcrypt hash of "test1234" at cost 12
DO $$
DECLARE
  pw TEXT := '$2a$12$pacRbXnEKqulywfzXgSABujKGGBiW/0EJJsNBQM096oSLRrw9YxW6';
BEGIN
  INSERT INTO users (id, email, password_hash, role)
  VALUES
    ('11110000-0000-0000-0000-000000000001', 'admin@nextlab.dev',      pw, 'admin'),
    ('11110000-0000-0000-0000-000000000002', 'docente@nextlab.dev',    pw, 'docente'),
    ('11110000-0000-0000-0000-000000000003', 'codocente@nextlab.dev',  pw, 'co_docente'),
    ('11110000-0000-0000-0000-000000000004', 'estudiante@nextlab.dev', pw, 'estudiante')
  ON CONFLICT (email) DO NOTHING;
END $$;

-- ─── Sample carrera ───────────────────────────────────────────────────────────
INSERT INTO carreras (id, code, name)
VALUES ('22220000-0000-0000-0000-000000000001', 'INF', 'Ingeniería en Informática')
ON CONFLICT (code) DO NOTHING;

-- ─── Student profile (required for student dashboard) ─────────────────────────
INSERT INTO student_profiles (email, carrera_id, semester, student_code)
VALUES (
  'estudiante@nextlab.dev',
  '22220000-0000-0000-0000-000000000001',
  3,
  'INF-2024-001'
)
ON CONFLICT (email) DO NOTHING;

-- ─── Sample materia ───────────────────────────────────────────────────────────
INSERT INTO materias (id, carrera_id, code, name, semester)
VALUES (
  '33330000-0000-0000-0000-000000000001',
  '22220000-0000-0000-0000-000000000001',
  'INF-301',
  'Programación Avanzada',
  3
)
ON CONFLICT (code) DO NOTHING;

-- ─── Sample curso (taught by test docente) ────────────────────────────────────
INSERT INTO cursos (id, materia_id, semester_label, section, docente_id)
VALUES (
  '44440000-0000-0000-0000-000000000001',
  '33330000-0000-0000-0000-000000000001',
  '2024-II',
  'A',
  '11110000-0000-0000-0000-000000000002'
)
ON CONFLICT (materia_id, docente_id, semester_label, section) DO NOTHING;

-- ─── Enroll student in sample curso ──────────────────────────────────────────
INSERT INTO inscripciones (curso_id, user_id, carrera_id, semester)
VALUES (
  '44440000-0000-0000-0000-000000000001',
  '11110000-0000-0000-0000-000000000004',
  '22220000-0000-0000-0000-000000000001',
  3
)
ON CONFLICT (user_id, curso_id) DO NOTHING;

-- ─── Grant co-docente access to the sample curso actividades ─────────────────
-- (actividades are granted per-actividad, so this just registers the user;
--  the docente can grant access from the UI after adding an actividad)

-- ─── Sample actividad ─────────────────────────────────────────────────────────
INSERT INTO actividades (id, curso_id, title, tipo, deadline, instructions, late_submission_allowed)
VALUES (
  '55550000-0000-0000-0000-000000000001',
  '44440000-0000-0000-0000-000000000001',
  'Tarea 1 — Algoritmos de ordenamiento',
  'tarea',
  NOW() + INTERVAL '7 days',
  'Implementa al menos dos algoritmos de ordenamiento (ej. QuickSort y MergeSort) en el lenguaje de tu preferencia. Incluye análisis de complejidad.',
  true
)
ON CONFLICT DO NOTHING;
