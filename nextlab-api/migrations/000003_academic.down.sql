-- M3: Academic Management — rollback

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

DROP INDEX IF EXISTS idx_users_google_sub;
ALTER TABLE users
  ALTER COLUMN password_hash SET NOT NULL,
  DROP COLUMN IF EXISTS google_sub;
