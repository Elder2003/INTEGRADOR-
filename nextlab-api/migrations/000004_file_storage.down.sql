DROP TABLE IF EXISTS submissions;
DROP TABLE IF EXISTS actividad_config_changes;
ALTER TABLE actividades DROP COLUMN IF EXISTS late_submission_allowed;
