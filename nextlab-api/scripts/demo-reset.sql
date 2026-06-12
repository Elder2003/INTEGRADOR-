-- ═══════════════════════════════════════════════════════
-- DEMO RESET — limpia todo con TRUNCATE CASCADE
-- ═══════════════════════════════════════════════════════

BEGIN;

-- Limpiar todas las tablas transaccionales de una vez
TRUNCATE TABLE
  grade_decisions,
  audit_events,
  appeal_decisions,
  appeals,
  analysis_results,
  format_reports,
  trace_events,
  submissions,
  exam_sessions,
  defense_appointments,
  git_analyses,
  git_submissions,
  stage_submissions,
  investigacion_stages,
  codocente_grants,
  actividad_format_template,
  actividad_config_changes,
  actividad_interdisciplinary_scopes,
  rag_curricular_chunks,
  rag_curricular,
  rag_docente_chunks,
  rag_docente,
  alerts,
  ai_request_logs,
  worker_jobs,
  refresh_tokens,
  student_behavior_baseline,
  student_notifications,
  password_reset_tokens,
  webhook_configs,
  webhook_delivery_log,
  format_templates,
  ml_models,
  actividades,
  inscripciones,
  grupo_asignaciones,
  grupos,
  cursos
CASCADE;

-- Borrar usuarios que NO son los 4 semilla
DELETE FROM student_profiles
WHERE email NOT IN ('estudiante@nextlab.dev');

DELETE FROM users
WHERE id NOT IN (
  '11110000-0000-0000-0000-000000000001',
  '11110000-0000-0000-0000-000000000002',
  '11110000-0000-0000-0000-000000000003',
  '11110000-0000-0000-0000-000000000004'
);

-- Limpiar materias y carreras basura
TRUNCATE TABLE materias CASCADE;
TRUNCATE TABLE carreras CASCADE;

-- Nombres visibles para la demo
UPDATE users SET full_name = 'Prof. Carlos Mendoza' WHERE id = '11110000-0000-0000-0000-000000000002';
UPDATE users SET full_name = 'Ana Lopez'            WHERE id = '11110000-0000-0000-0000-000000000004';
UPDATE users SET full_name = 'Admin Sistema'        WHERE id = '11110000-0000-0000-0000-000000000001';
UPDATE users SET full_name = 'Prof. Laura Quispe'   WHERE id = '11110000-0000-0000-0000-000000000003';

-- Carrera
INSERT INTO carreras (id, code, name)
VALUES ('22220000-0000-0000-0000-000000000001', 'INF', 'Ingenieria en Informatica');

-- Materia
INSERT INTO materias (id, carrera_id, code, name, semester)
VALUES (
  '33330000-0000-0000-0000-000000000001',
  '22220000-0000-0000-0000-000000000001',
  'INF-301', 'Programacion Avanzada', 3
);

-- Curso de demo
INSERT INTO cursos (id, materia_id, semester_label, section, docente_id)
VALUES (
  '44440000-0000-0000-0000-000000000001',
  '33330000-0000-0000-0000-000000000001',
  '2026-I', 'A',
  '11110000-0000-0000-0000-000000000002'
);

-- Perfil del estudiante
INSERT INTO student_profiles (email, carrera_id, semester, student_code)
VALUES ('estudiante@nextlab.dev', '22220000-0000-0000-0000-000000000001', 3, 'INF-2024-001')
ON CONFLICT (email) DO NOTHING;

-- Inscripcion del estudiante
INSERT INTO inscripciones (curso_id, user_id, carrera_id, semester)
VALUES (
  '44440000-0000-0000-0000-000000000001',
  '11110000-0000-0000-0000-000000000004',
  '22220000-0000-0000-0000-000000000001',
  3
);

-- Actividad EXAMEN para la demo (deadline en 2 horas)
INSERT INTO actividades (id, curso_id, title, tipo, deadline, instructions, late_submission_allowed)
VALUES (
  '55550000-0000-0000-0000-000000000001',
  '44440000-0000-0000-0000-000000000001',
  'Examen Parcial 1 - Algoritmos y Complejidad',
  'examen',
  NOW() + INTERVAL '2 hours',
  'Responde las siguientes preguntas de forma clara y justificada:

1. Explica la diferencia entre complejidad O(n log n) y O(n^2). Da un ejemplo de algoritmo para cada caso.

2. Implementa en pseudocodigo el algoritmo QuickSort e indica su complejidad en el caso promedio y peor caso.

3. Cuando preferirias usar MergeSort sobre QuickSort? Justifica tu respuesta.

4. Dado un arreglo de 1000 elementos ya ordenado, cual de los siguientes algoritmos seria mas eficiente: BubbleSort, MergeSort o BinarySearch? Explica por que.',
  false
);

COMMIT;

-- Verificacion final
SELECT email, role, full_name FROM users ORDER BY role, email;
SELECT a.title, a.tipo, a.deadline FROM actividades a;
SELECT 'RESET OK' as resultado;
