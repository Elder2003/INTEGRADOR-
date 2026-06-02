ALTER TABLE worker_jobs DROP COLUMN IF EXISTS max_retries;
ALTER TABLE worker_jobs DROP COLUMN IF EXISTS dead_at;
ALTER TABLE worker_jobs DROP CONSTRAINT IF EXISTS worker_jobs_status_check;
ALTER TABLE worker_jobs ADD CONSTRAINT worker_jobs_status_check
    CHECK (status IN ('pending','queued','processing','done','failed'));

ALTER TABLE git_analysis_jobs DROP COLUMN IF EXISTS max_retries;
ALTER TABLE git_analysis_jobs DROP COLUMN IF EXISTS last_error;
ALTER TABLE git_analysis_jobs DROP COLUMN IF EXISTS dead_at;
ALTER TABLE git_analysis_jobs DROP CONSTRAINT IF EXISTS git_analysis_jobs_status_check;
ALTER TABLE git_analysis_jobs ADD CONSTRAINT git_analysis_jobs_status_check
    CHECK (status IN ('queued','processing','completed','failed'));
