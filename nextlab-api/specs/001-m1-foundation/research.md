# Research: M1 — Fundación del Proyecto

## Go Project Layout

**Decision**: Standard `cmd/` + `internal/` layout with one subdirectory per binary.

```
cmd/api/main.go      ← API binary
cmd/worker/main.go   ← Worker binary
internal/            ← All shared application code (unexportable outside module)
```

**Rationale**: The `internal/` package convention enforces module boundaries at
the compiler level — code in `internal/auth/` cannot be imported by an external
module. This is the Go idiomatic way to express domain isolation. The `cmd/`
convention is universally understood and clearly separates the two entry points.

**Alternatives considered**:
- Flat layout (`auth/`, `academic/` at root) — rejected; collides with Go tooling
  conventions and makes the two entry points ambiguous.
- `pkg/` layout — rejected; `pkg/` implies exportable library code, which is not
  the intent here. `internal/` is correct for application code.

---

## Migration Tool

**Decision**: `golang-migrate/migrate/v4` with the Go library for auto-apply at
startup and the standalone CLI for rollback.

**Rationale**: Explicitly referenced in the milestones document. Supports
PostgreSQL natively via `pgx`. Can be embedded in the API binary to run pending
migrations before `fiber.Listen()` is called. The same tool is invokable as a
standalone Docker image (`migrate/migrate`) for manual rollback and CI/CD.
Migration files are plain `.up.sql` / `.down.sql` pairs — no DSL to learn.

**Alternatives considered**:
- goose — similar capability, good Go embedding support, but not the stated
  preference in the milestones document.
- Atlas — more powerful (schema diffing), but too heavy for the current scope.

---

## Structured Logging

**Decision**: Go stdlib `log/slog` (available since Go 1.21) with `JSONHandler`.

**Rationale**: Zero external dependencies. The `slog.JSONHandler` produces
newline-delimited JSON with `time`, `level`, and `msg` built-in. Custom attributes
(`service`, `correlation_id`) are added via `slog.With(...)` for the logger
instance and via middleware for per-request fields. Sufficient performance for
≤1500 students.

Required fields per log entry (FR-007 + clarification):

| Field | Source |
|---|---|
| `time` | slog built-in |
| `level` | slog built-in |
| `msg` | slog built-in |
| `service` | set at logger init: `"api"` or `"worker"` |
| `correlation_id` | set by HTTP middleware per request; `"system"` for startup logs |

**Alternatives considered**:
- Uber zap — faster, but the throughput gain is irrelevant at this scale and
  adds an external dependency.
- zerolog — minimal allocations, but slog is now the idiomatic stdlib choice.

---

## Health Check Pattern

**Decision**: Parallel dependency pings with a shared 400ms timeout, aggregated
into a single JSON response; HTTP 503 if any dependency is down.

**Rationale**: Parallel pings keep total latency under the 500ms SC-002 target
even if one dependency is slow. The 400ms internal timeout leaves 100ms margin
for Fiber's handler overhead. Each dependency is reported individually so
monitoring tools can distinguish partial from full degradation.

Response shape:
```json
{
  "status": "ok | degraded | down",
  "dependencies": {
    "database": { "status": "ok", "latency_ms": 3 },
    "cache":    { "status": "ok", "latency_ms": 1 },
    "storage":  { "status": "down", "error": "dial tcp: connection refused" }
  }
}
```

HTTP 200 when `status` is `"ok"`. HTTP 503 for `"degraded"` or `"down"`.

**Alternatives considered**:
- Sequential pings — rejected; worst-case latency = sum of all timeouts.
- Single aggregate status without per-dependency detail — rejected; not useful
  for diagnosing which service is down.

---

## Docker Compose Service Ordering

**Decision**: `depends_on` with `condition: service_healthy` for the API service.
Each dependency declares a `healthcheck` using its native probe.

| Service | Health probe |
|---|---|
| postgres | `pg_isready -U $POSTGRES_USER` |
| redis | `redis-cli ping` |
| minio | `curl -f http://localhost:9000/minio/health/live` |

**Rationale**: Docker Compose `condition: service_healthy` blocks the API
container from starting until all three probes pass. This satisfies SC-001
(60s to ready) without adding sleep hacks in the entrypoint.

The worker uses `depends_on` on the API service with `condition: service_started`
(not `service_healthy`) — the worker doesn't need the API to be fully ready,
only that the database is up (covered transitively via the API's healthcheck
dependency chain).

---

## Configuration Management

**Decision**: Single `Config` struct in `internal/config/config.go` populated
via `os.Getenv`, validated at startup (missing required vars → fatal log + exit 1).

```go
type Config struct {
    DatabaseURL   string // required
    RedisAddr     string // required
    MinioEndpoint string // required
    MinioAccess   string // required
    MinioSecret   string // required
    APIPort       string // default: "8080"
    LogLevel      string // default: "info"
}
```

**Rationale**: No external dependency. Validated at process startup so the error
surfaces immediately with the missing variable name (FR-006 edge case). The struct
is passed through dependency injection — no global state, no `init()` magic.

**Alternatives considered**:
- viper — supports file + env + remote config, but over-engineered for env-only
  configuration.
- godotenv — useful for loading `.env` files in development; can be added as a
  dev-only convenience but is not required since Docker Compose handles env injection.

---

## pgvector Setup

**Decision**: First migration enables the extension:
```sql
-- 000001_init.up.sql
CREATE EXTENSION IF NOT EXISTS vector;

-- 000001_init.down.sql
DROP EXTENSION IF EXISTS vector;
```

**Rationale**: `IF NOT EXISTS` makes the migration idempotent. Enabling pgvector
in M1 means all future milestones (M7 RAG tables, etc.) can add `vector(1536)`
columns without coordinating an extension migration later.

**Note**: The `pgvector/pgvector:pg16` Docker image includes the extension
pre-installed. No additional installation steps are required.

---

## Multi-Stage Docker Build

**Decision**: Single `Dockerfile` with two build stages; both binaries built
in the same builder stage and copied to the runtime image.

```dockerfile
# Stage 1: builder
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download            # cached layer
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/api    ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/worker ./cmd/worker

# Stage 2: runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/bin/api    .
COPY --from=builder /build/bin/worker .
```

**Rationale**: Both binaries share the same `go mod download` cached layer,
keeping build times fast. `CGO_ENABLED=0` produces a fully static binary that
runs on the minimal Alpine runtime image. `ca-certificates` is needed for HTTPS
calls to MinIO and future external services.

**Alternatives considered**:
- Separate Dockerfiles per binary — rejected; doubles build time and diverges
  dependency management.
- Distroless base — valid alternative to Alpine, but Alpine's `apk` is useful
  for debugging in early milestones.
