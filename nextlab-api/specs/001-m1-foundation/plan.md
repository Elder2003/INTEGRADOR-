# Implementation Plan: M1 — Fundación del Proyecto

**Branch**: `001-m1-foundation` | **Date**: 2026-05-17 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-m1-foundation/spec.md`

---

## Summary

Establish a fully containerised Go project with two binaries (API + worker)
compiled from a single codebase. The API auto-applies database migrations at
startup (pgvector extension enabled), exposes `GET /health` with per-dependency
status, and uses structured JSON logging with a defined field set. The codebase
is organised into 11 domain module stubs so all subsequent milestones can add
code without touching shared files.

**Technical approach**: Standard Go `cmd/` + `internal/` layout · golang-migrate
embedded in the API binary · `log/slog` JSON handler · parallel health pings with
400ms timeout · multi-stage Docker build · Docker Compose `depends_on` with
healthchecks.

---

## Technical Context

**Language/Version**: Go 1.22

**Primary Dependencies**:
- `gofiber/fiber/v2` — HTTP framework (WebSocket + SSE native; locked by constitution)
- `golang-migrate/migrate/v4` + `golang-migrate/migrate/v4/database/pgx/v5`
- `jackc/pgx/v5` — PostgreSQL driver
- `redis/go-redis/v9` — Redis client
- `minio/minio-go/v7` — MinIO client
- `log/slog` (stdlib) — structured JSON logging

**Storage**: PostgreSQL 16 + pgvector | Redis 7 | MinIO RELEASE.2024+

**Testing**: `go test` stdlib | `testcontainers-go` for integration tests

**Target Platform**: Linux/AMD64 (Hetzner CX42, Docker Compose)

**Project Type**: Web service (REST API) + background worker — two binaries,
one codebase

**Performance Goals**:
- `GET /health` ≤ 500ms end-to-end (SC-002)
- All services ready ≤ 60s after `docker compose up` with cached images (SC-001)

**Constraints**:
- Multi-stage Docker build — runtime image contains only compiled binaries
- All configuration via environment variables — nothing hardcoded (FR-006)
- Structured JSON logs with `time`, `level`, `msg`, `service`, `correlation_id`
  in every entry (FR-007)

**Scale/Scope**: ~800–1500 active students, 2 carreras (Unifranz initial deployment)

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Immutable Evidence | ✅ N/A | No trace events in M1; append-only tables added in M5 |
| II. Async Analysis Isolation | ✅ | Worker binary scaffolded; Redis connection established; job logic in M5 |
| III. Human Authority Over AI | ✅ N/A | No AI calls in M1 |
| IV. Strict Role-Based Data Isolation | ✅ | No auth yet (M2); `internal/` module boundaries enforce structural isolation |
| V. Dual RAG Independence | ✅ N/A | RAG tables added in M7 |
| VI. Non-Invasive Auditing | ✅ N/A | No audit capture in M1 |
| VII. Submission Always Accepted | ✅ N/A | No submission endpoints in M1 |

**Result: PASS** — No violations.

---

## Project Structure

### Documentation (this feature)

```text
specs/001-m1-foundation/
├── plan.md              ← this file
├── research.md          ← Phase 0 output
├── data-model.md        ← Phase 1 output
├── quickstart.md        ← Phase 1 output
├── contracts/
│   └── health.md        ← Phase 1 output
└── tasks.md             ← Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
nextlab-api/
├── cmd/
│   ├── api/
│   │   └── main.go          # Fiber setup, migration runner, server start
│   └── worker/
│       └── main.go          # Redis Streams consumer skeleton
├── internal/
│   ├── config/
│   │   └── config.go        # Typed env-var struct, validation, fatal on missing
│   ├── health/
│   │   └── handler.go       # GET /health — parallel pings, JSON response
│   ├── auth/                # Stub (package auth) — M2
│   ├── academic/            # Stub (package academic) — M3
│   ├── storage/             # Stub (package storage) — M4
│   ├── tracing/             # Stub (package tracing) — M5
│   ├── ai/                  # Stub (package ai) — M6
│   ├── rag/                 # Stub (package rag) — M7
│   ├── detection/           # Stub (package detection) — M8
│   ├── projects/            # Stub (package projects) — M10
│   ├── alerts/              # Stub (package alerts) — M11
│   ├── analytics/           # Stub (package analytics) — M11
│   └── admin/               # Stub (package admin) — M12
├── migrations/
│   ├── 000001_init.up.sql   # CREATE EXTENSION IF NOT EXISTS vector
│   └── 000001_init.down.sql # DROP EXTENSION IF EXISTS vector
├── docker/
│   └── Dockerfile           # Multi-stage builder → alpine runtime
├── docker-compose.yml       # All VPS-1 services + migrate one-off service
├── .env.example             # All required vars with safe local defaults
├── go.mod
└── go.sum
```

**Structure Decision**: Standard Go `cmd/` + `internal/` layout. `internal/`
enforces package boundaries at the compiler level. Eleven stub module directories
ensure SC-004 (adding a new module requires zero changes to existing files).

---

## Phase 0: Research Summary

See [research.md](research.md) for full decision log. Key decisions:

| Topic | Decision |
|---|---|
| Project layout | `cmd/` + `internal/` standard Go conventions |
| Migration tool | `golang-migrate/migrate/v4` embedded in API binary |
| Logger | Go stdlib `log/slog` with `JSONHandler` |
| Health check | Parallel pings, 400ms shared timeout, 200/503 aggregate |
| Docker ordering | `depends_on: condition: service_healthy` per dependency |
| Configuration | Typed struct from `os.Getenv`, validated at startup |
| pgvector | Enabled in migration 000001 via `CREATE EXTENSION IF NOT EXISTS vector` |
| Docker build | Single multi-stage Dockerfile, both binaries built in one stage |

---

## Phase 1: Design Summary

### Data Model

See [data-model.md](data-model.md). M1 creates no domain tables. Schema objects:
- `vector` extension (via migration 000001)
- `schema_migrations` table (auto-managed by golang-migrate)

Application-layer entities (Go structs, not DB tables): `Config`, `HealthReport`,
`LogEntry`.

### API Contract

See [contracts/health.md](contracts/health.md).

Single endpoint: `GET /health` — parallel dependency pings, JSON response,
HTTP 200 all-ok / 503 any-down.

### Quickstart

See [quickstart.md](quickstart.md).

Three commands from a clean clone to a green health check:
```bash
cp .env.example .env
docker compose up --build
curl http://localhost:8080/health
```

---

## Key Implementation Notes

### Migration auto-apply in API startup

`cmd/api/main.go` runs `migrate.Up()` before calling `fiber.Listen()`.
If migration fails → log error + `os.Exit(1)`.

```
startup sequence:
  1. Load & validate Config (exit 1 on missing required vars)
  2. Init slog JSON logger with service="api"
  3. Connect to PostgreSQL (with retry, max 10s)
  4. Run migrations (exit 1 on failure)
  5. Connect to Redis, MinIO
  6. Register routes (health only in M1)
  7. fiber.Listen(":PORT")
```

### Worker startup sequence

```
startup sequence:
  1. Load & validate Config
  2. Init slog JSON logger with service="worker"
  3. Connect to Redis
  4. Log "worker started" + block (no jobs to consume until M5)
```

### Correlation ID middleware (scaffolded, used from M2)

An HTTP middleware in `cmd/api/main.go` generates a UUID per request, stores it
in the Fiber context, and adds it to the request-scoped logger. In M1 all
correlation IDs for startup logs use the literal `"system"`.

### Docker Compose migrate service

A one-off `migrate` service using the `migrate/migrate` Docker image handles
the standalone rollback command:

```yaml
migrate:
  image: migrate/migrate
  profiles: [tools]
  volumes:
    - ./migrations:/migrations
  command: ["-path=/migrations", "-database=${DATABASE_URL}", "down", "1"]
```

Running `docker compose run --rm migrate down 1` invokes rollback without
needing the Go tool installed locally.

---

## Complexity Tracking

> No constitution violations — section intentionally empty.
