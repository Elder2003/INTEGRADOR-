# Data Model: M1 — Fundación del Proyecto

## Overview

M1 creates no domain tables. The database schema for this milestone consists
only of infrastructure managed by `golang-migrate` and the pgvector extension.
All domain tables (users, courses, sessions, etc.) are added in M2 and later.

---

## Database Objects Created in M1

### Extension: pgvector

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

- Enables the `vector` data type for future use in `rag_curricular` and
  `rag_docente` tables (M7).
- Idempotent — safe to run on a database that already has it enabled.

### Table: schema_migrations (managed by golang-migrate)

Automatically created by `golang-migrate` on first run. Not defined manually.

| Column | Type | Description |
|---|---|---|
| `version` | `bigint` | Migration version number (e.g., 1) |
| `dirty` | `boolean` | True if the last migration did not complete cleanly |

---

## Application-Layer Entities (not persisted)

These entities exist as Go structs and JSON responses. They are not database tables.

### Config

Populated from environment variables at process startup. Shared across all
internal packages via dependency injection.

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `DatabaseURL` | string | yes | — | PostgreSQL connection string |
| `RedisAddr` | string | yes | — | Redis host:port |
| `MinioEndpoint` | string | yes | — | MinIO host:port |
| `MinioAccessKey` | string | yes | — | MinIO access key |
| `MinioSecretKey` | string | yes | — | MinIO secret key |
| `MinioBucketSubmissions` | string | no | `submissions` | Bucket for student files |
| `MinioBucketRAG` | string | no | `rag-content` | Bucket for RAG material |
| `APIPort` | string | no | `8080` | Port the API listens on |
| `LogLevel` | string | no | `info` | Minimum log level |
| `AppEnv` | string | no | `development` | `development` \| `production` |

**Validation rule**: If any required field resolves to an empty string, the
process logs the missing variable name at `error` level and exits with code 1.

---

### HealthReport (JSON response body for `GET /health`)

```json
{
  "status": "ok",
  "dependencies": {
    "database": {
      "status": "ok",
      "latency_ms": 3
    },
    "cache": {
      "status": "ok",
      "latency_ms": 1
    },
    "storage": {
      "status": "down",
      "error": "dial tcp [::1]:9000: connect: connection refused"
    }
  }
}
```

| Field | Type | Values |
|---|---|---|
| `status` | string | `"ok"` — all deps healthy; `"degraded"` — some down; `"down"` — all down |
| `dependencies.<name>.status` | string | `"ok"` \| `"degraded"` \| `"down"` |
| `dependencies.<name>.latency_ms` | int | Present when status is `"ok"` |
| `dependencies.<name>.error` | string | Present when status is `"down"` |

**HTTP status rule**: 200 when top-level `status` is `"ok"`. 503 otherwise.

---

### LogEntry (structured log line — not persisted)

Every log line written by the API or worker conforms to this shape:

```json
{
  "time": "2026-05-17T10:30:00.000Z",
  "level": "INFO",
  "msg": "server started",
  "service": "api",
  "correlation_id": "system"
}
```

| Field | Type | Notes |
|---|---|---|
| `time` | RFC3339 string | Set by `slog` automatically |
| `level` | string | `DEBUG` \| `INFO` \| `WARN` \| `ERROR` |
| `msg` | string | Human-readable message |
| `service` | string | `"api"` or `"worker"` — set at logger initialisation |
| `correlation_id` | string | Request UUID injected by middleware; `"system"` for startup/background logs |

---

## Migration Files

```
migrations/
├── 000001_init.up.sql
└── 000001_init.down.sql
```

### 000001_init.up.sql
```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

### 000001_init.down.sql
```sql
DROP EXTENSION IF EXISTS vector;
```

---

## Domain Stub Directories

The following `internal/` module directories are created as empty stubs in M1.
They contain only a placeholder `.go` file declaring the package. No tables,
no logic.

| Directory | Populated in |
|---|---|
| `internal/auth/` | M2 |
| `internal/academic/` | M3 |
| `internal/storage/` | M4 |
| `internal/tracing/` | M5 |
| `internal/ai/` | M6 |
| `internal/rag/` | M7 |
| `internal/detection/` | M8 |
| `internal/projects/` | M10 |
| `internal/alerts/` | M11 |
| `internal/analytics/` | M11 |
| `internal/admin/` | M12 |
