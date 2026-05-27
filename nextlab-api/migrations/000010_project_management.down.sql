-- M10 down: drop all project management tables
-- Note: removing enum values requires recreating the type; this down migration
-- only drops the tables. The enum extension (investigacion, git_analysis) is
-- left in place on rollback to avoid dropping dependent objects.

DROP TABLE IF EXISTS defense_appointments;
DROP TABLE IF EXISTS git_analysis_jobs;
DROP TABLE IF EXISTS git_analyses;
DROP TABLE IF EXISTS git_submissions;
DROP TABLE IF EXISTS stage_submissions;
DROP TABLE IF EXISTS investigacion_stages;
