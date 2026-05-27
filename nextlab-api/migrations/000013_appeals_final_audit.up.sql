CREATE TABLE appeals (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID        NOT NULL REFERENCES exam_sessions(id),
    filed_by   UUID        NOT NULL REFERENCES users(id),
    type       TEXT        NOT NULL CHECK (type IN ('analysis_dispute', 'grade_dispute')),
    status     TEXT        NOT NULL DEFAULT 'pending'
                           CHECK (status IN ('pending', 'escalated', 'resolved', 'admin_resolved')),
    reason     TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (session_id)
);

CREATE INDEX appeals_filed_by_idx ON appeals(filed_by);
CREATE INDEX appeals_status_idx   ON appeals(status);

CREATE TABLE appeal_decisions (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    appeal_id     UUID        NOT NULL REFERENCES appeals(id),
    decided_by    UUID        NOT NULL REFERENCES users(id),
    role          TEXT        NOT NULL CHECK (role IN ('docente', 'admin')),
    decision      TEXT        NOT NULL CHECK (decision IN ('upheld', 'rejected', 'overridden')),
    justification TEXT        NOT NULL,
    decided_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX appeal_decisions_appeal_idx ON appeal_decisions(appeal_id);

CREATE TABLE audit_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    appeal_id   UUID        NOT NULL REFERENCES appeals(id),
    sequence    INT         NOT NULL,
    event_type  TEXT        NOT NULL,
    payload     JSONB       NOT NULL DEFAULT '{}',
    prev_hash   TEXT,
    event_hash  TEXT        NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (appeal_id, sequence)
);

CREATE INDEX audit_events_appeal_idx ON audit_events(appeal_id, sequence);
