-- M17: Verde Largo Plazo — rollback

DROP INDEX IF EXISTS uq_trace_offline;
ALTER TABLE trace_events DROP COLUMN IF EXISTS received_offline;
ALTER TABLE trace_events DROP COLUMN IF EXISTS client_timestamp;

DROP UNIQUE INDEX IF EXISTS uq_ml_models_active;
DROP TABLE IF EXISTS ml_models;

DROP INDEX IF EXISTS idx_webhook_log_status;
DROP INDEX IF EXISTS idx_webhook_log_webhook_id;
DROP TABLE IF EXISTS webhook_delivery_log;
DROP TABLE IF EXISTS webhook_configs;

DROP TABLE IF EXISTS student_behavior_baseline;

DROP INDEX IF EXISTS idx_notifications_student_all;
DROP INDEX IF EXISTS idx_notifications_student_unread;
DROP TABLE IF EXISTS student_notifications;
