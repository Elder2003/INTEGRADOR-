-- M15: Dead letter queue fields for worker_jobs and git_analysis_jobs

-- worker_jobs DLQ
ALTER TABLE worker_jobs ADD COLUMN max_retries INT         NOT NULL DEFAULT 3;
ALTER TABLE worker_jobs ADD COLUMN dead_at     TIMESTAMPTZ;

ALTER TABLE worker_jobs DROP CONSTRAINT IF EXISTS worker_jobs_status_check;
ALTER TABLE worker_jobs ADD CONSTRAINT worker_jobs_status_check
    CHECK (status IN ('pending','queued','processing','done','failed','dead'));

-- git_analysis_jobs DLQ
ALTER TABLE git_analysis_jobs ADD COLUMN max_retries INT         NOT NULL DEFAULT 3;
ALTER TABLE git_analysis_jobs ADD COLUMN last_error  TEXT;
ALTER TABLE git_analysis_jobs ADD COLUMN dead_at     TIMESTAMPTZ;

ALTER TABLE git_analysis_jobs DROP CONSTRAINT IF EXISTS git_analysis_jobs_status_check;
ALTER TABLE git_analysis_jobs ADD CONSTRAINT git_analysis_jobs_status_check
    CHECK (status IN ('queued','processing','completed','failed','dead'));
