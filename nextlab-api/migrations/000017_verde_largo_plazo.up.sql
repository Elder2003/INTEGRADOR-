-- M17: Verde Largo Plazo

-- student_notifications
CREATE TABLE student_notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_student_unread
    ON student_notifications (student_id, created_at DESC)
    WHERE read_at IS NULL;

CREATE INDEX idx_notifications_student_all
    ON student_notifications (student_id, created_at DESC);

-- student_behavior_baseline
CREATE TABLE student_behavior_baseline (
    student_id              UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    avg_tab_blurs_per_hour  FLOAT8 NOT NULL DEFAULT 0,
    stddev_tab_blurs        FLOAT8 NOT NULL DEFAULT 0,
    avg_clipboard_per_hour  FLOAT8 NOT NULL DEFAULT 0,
    stddev_clipboard        FLOAT8 NOT NULL DEFAULT 0,
    last_10_tab_blurs       FLOAT8[] NOT NULL DEFAULT '{}',
    last_10_clipboard       FLOAT8[] NOT NULL DEFAULT '{}',
    total_sessions_analyzed INT NOT NULL DEFAULT 0,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- webhook_configs
CREATE TABLE webhook_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url         TEXT NOT NULL,
    secret_hmac TEXT NOT NULL,
    events      TEXT[] NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_by  UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- webhook_delivery_log
CREATE TABLE webhook_delivery_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id      UUID NOT NULL REFERENCES webhook_configs(id) ON DELETE CASCADE,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    status_code     INT,
    attempt_count   INT NOT NULL DEFAULT 1,
    status          TEXT NOT NULL DEFAULT 'pending',
    last_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_log_webhook_id
    ON webhook_delivery_log (webhook_id, created_at DESC);

CREATE INDEX idx_webhook_log_status
    ON webhook_delivery_log (status)
    WHERE status != 'success';

-- ml_models
CREATE TABLE ml_models (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version                 INT NOT NULL,
    trained_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    training_session_count  INT NOT NULL,
    file_path               TEXT NOT NULL,
    feature_names           TEXT[] NOT NULL,
    is_active               BOOLEAN NOT NULL DEFAULT FALSE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_ml_models_active
    ON ml_models (is_active)
    WHERE is_active = TRUE;

-- trace_events: offline batch support
ALTER TABLE trace_events
    ADD COLUMN IF NOT EXISTS client_timestamp TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS received_offline  BOOLEAN NOT NULL DEFAULT FALSE;

CREATE UNIQUE INDEX uq_trace_offline
    ON trace_events (session_id, client_timestamp, event_type)
    WHERE received_offline = TRUE;
