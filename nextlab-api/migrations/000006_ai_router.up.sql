-- Migration: 000006_ai_router
-- Append-only tables for AI router. No UPDATE or DELETE ever (Constitution Principle I).

CREATE TABLE IF NOT EXISTS router_configs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    mode        TEXT        NOT NULL CHECK (mode IN ('local', 'auto', 'cloud')),
    prev_mode   TEXT        CHECK (prev_mode IN ('local', 'auto', 'cloud')),
    changed_by  UUID        REFERENCES users(id),
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_router_configs_changed_at
    ON router_configs(changed_at DESC);

-- ai_request_logs: metadata only — no prompt_content, no completion_content (FR-011).
CREATE TABLE IF NOT EXISTS ai_request_logs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id      UUID        NOT NULL,
    mode            TEXT        NOT NULL CHECK (mode IN ('local', 'auto', 'cloud')),
    backend_used    TEXT        NOT NULL CHECK (backend_used IN ('local', 'cloud')),
    fallback_used   BOOLEAN     NOT NULL DEFAULT false,
    fallback_reason TEXT,
    latency_ms      INT         NOT NULL,
    status          TEXT        NOT NULL CHECK (status IN ('success', 'failed')),
    error_detail    TEXT,
    logged_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ai_request_logs_logged_at
    ON ai_request_logs(logged_at DESC);

CREATE INDEX IF NOT EXISTS idx_ai_request_logs_request_id
    ON ai_request_logs(request_id);
