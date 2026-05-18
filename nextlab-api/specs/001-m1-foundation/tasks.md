---
description: "Task list for M1 — Fundación del Proyecto"
---

# Tasks: M1 — Fundación del Proyecto

**Input**: Design documents from `specs/001-m1-foundation/`

**Prerequisites**: plan.md ✅ | spec.md ✅ | research.md ✅ | data-model.md ✅ | contracts/ ✅

**Tests**: Not requested — tasks contain no test generation steps.

**Organization**: Tasks grouped by user story to enable independent implementation
and testing of each story.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- File paths are relative to the repository root (`nextlab-api/`)

---

## Phase 1: Setup

**Purpose**: Initialize the Go module and create the full directory skeleton.
No logic here — just scaffolding.

- [ ] T001 Initialize Go module in repo root: `go mod init github.com/[org]/nextlab-api` (replace [org] with actual org/username)
- [ ] T002 Create full directory structure: `cmd/api/`, `cmd/worker/`, `internal/config/`, `internal/health/`, `internal/auth/`, `internal/academic/`, `internal/storage/`, `internal/tracing/`, `internal/ai/`, `internal/rag/`, `internal/detection/`, `internal/projects/`, `internal/alerts/`, `internal/analytics/`, `internal/admin/`, `internal/server/`, `migrations/`, `docker/`
- [ ] T003 [P] Add all Go dependencies to `go.mod`: `gofiber/fiber/v2`, `golang-migrate/migrate/v4`, `jackc/pgx/v5`, `redis/go-redis/v9`, `minio/minio-go/v7`; run `go mod tidy` to generate `go.sum`
- [ ] T004 [P] Create `.env.example` with all required environment variables and safe local defaults (DATABASE_URL, REDIS_ADDR, MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, API_PORT=8080, LOG_LEVEL=info, APP_ENV=development)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that ALL user stories depend on. No user story
implementation can begin until this phase is complete.

**⚠️ CRITICAL**: Phases 3–5 cannot start until this phase is complete.

- [ ] T005 Create `internal/config/config.go`: define `Config` struct with all fields from data-model.md, implement `Load() (*Config, error)` that reads from `os.Getenv`, validate all required fields (DATABASE_URL, REDIS_ADDR, MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY) — log field name at error level and return error if any are empty
- [ ] T006 [P] Create `migrations/000001_init.up.sql`: single statement `CREATE EXTENSION IF NOT EXISTS vector;`
- [ ] T007 [P] Create `migrations/000001_init.down.sql`: single statement `DROP EXTENSION IF EXISTS vector;`
- [ ] T008 Create `docker/Dockerfile`: multi-stage build — Stage 1 (`golang:1.22-alpine` AS builder): copy `go.mod`/`go.sum`, `go mod download`, copy source, build both binaries (`CGO_ENABLED=0 GOOS=linux go build -o bin/api ./cmd/api` and `./cmd/worker`); Stage 2 (`alpine:3.19`): install `ca-certificates tzdata`, copy both binaries from builder
- [ ] T009 Create `docker-compose.yml`: five services — `postgres` (image: `pgvector/pgvector:pg16`, healthcheck: `pg_isready`), `redis` (image: `redis:7-alpine`, healthcheck: `redis-cli ping`), `minio` (healthcheck: `curl -f localhost:9000/minio/health/live`), `api` (build: `docker/Dockerfile`, command: `./api`, depends_on all three with `condition: service_healthy`), `worker` (same image, command: `./worker`, depends_on `api` with `condition: service_started`); add `migrate` service (`migrate/migrate` image, `profiles: [tools]`) for standalone rollback
- [ ] T010 [P] Create package declaration stub files for all 11 domain modules — one file per module declaring only `package <name>`: `internal/auth/module.go`, `internal/academic/module.go`, `internal/storage/module.go`, `internal/tracing/module.go`, `internal/ai/module.go`, `internal/rag/module.go`, `internal/detection/module.go`, `internal/projects/module.go`, `internal/alerts/module.go`, `internal/analytics/module.go`, `internal/admin/module.go`
- [ ] T011 Create `cmd/worker/main.go`: call `config.Load()` (exit 1 on error), initialise `slog.JSONHandler` logger with `service="worker"`, connect to Redis using `go-redis` (log error + exit 1 if unreachable), log `"worker started"` at INFO level with `correlation_id="system"`, block with `select{}` (job consumption added in M5)

**Checkpoint**: `go build ./cmd/api && go build ./cmd/worker` must both compile successfully before proceeding.

---

## Phase 3: User Story 1 — Developer Starts Full Stack (Priority: P1) 🎯 MVP

**Goal**: `docker compose up --build` starts all services; `GET /health` returns
HTTP 200 with per-dependency JSON status within 500ms.

**Independent Test**: `docker compose up --build && curl -s http://localhost:8080/health | jq .`
must return HTTP 200 with `"status": "ok"` and all three dependencies reporting `"ok"`.

### Implementation for User Story 1

- [ ] T012 [US1] Create `internal/health/handler.go`: define `HealthReport` and `DependencyStatus` structs matching the contract in `contracts/health.md`; implement `NewChecker(db, redis, minio)` constructor; implement `Check(ctx)` method that runs three pings in parallel via goroutines with a shared 400ms deadline context; return HTTP 200 when all ok, HTTP 503 otherwise
- [ ] T013 [US1] Create `cmd/api/main.go`: call `config.Load()` (exit 1 on error) → initialise `slog.JSONHandler` logger (service="api") → open pgx connection pool → run `golang-migrate` `migrate.Up()` using embedded `migrations/` directory (log applied count; exit 1 on failure) → connect Redis client → connect MinIO client → create `*fiber.App` → register `GET /health` route using health checker → `fiber.Listen(cfg.APIPort)`
- [ ] T014 [P] [US1] Add correlation ID middleware in `cmd/api/main.go`: before route registration, add a Fiber middleware that generates a UUID (use `google/uuid` or crypto/rand), stores it in `c.Locals("correlation_id")`, and injects it into a request-scoped `slog` logger stored in locals; all handler log calls use this logger; startup logs use `correlation_id="system"`

**Checkpoint**: Run `docker compose up --build`. Wait for all healthchecks to pass.
`curl -s http://localhost:8080/health | jq .` must return HTTP 200 and `"status": "ok"`.
`docker compose logs api | jq .` must parse without errors (SC-005).

---

## Phase 4: User Story 2 — Module Registration Without Shared Code Changes (Priority: P2)

**Goal**: A developer can add a new domain module directory + Register function and
have its routes served without modifying `cmd/api/main.go` or any sibling module.

**Independent Test**: Create `internal/ping/module.go` (new file only, nothing else changed),
add `GET /ping` route inside it, register it through the module registry — `cmd/api/main.go`
must not be modified. `curl http://localhost:8080/ping` must return a response.

### Implementation for User Story 2

- [ ] T015 [US2] Create `internal/server/router.go`: define `Module` interface with a single `Register(app *fiber.App)` method; implement `Router` struct holding a slice of `Module`; implement `NewRouter(modules ...Module) *Router` constructor and `RegisterAll(app *fiber.App)` method that calls `Register` on each module; move `GET /health` registration into `internal/health/module.go` implementing the `Module` interface
- [ ] T016 [US2] Refactor route registration in `cmd/api/main.go` to use `server.NewRouter(health.New(checker))` and call `router.RegisterAll(app)`; verify that adding a new stub module (`internal/ping/`) with its own `Register` call in `internal/ping/module.go` and adding `ping.New()` to the `NewRouter(...)` call is the **only** change needed to serve new routes — no modifications to any other existing file

**Checkpoint**: `go build ./...` passes. Adding `internal/ping/module.go` (new file) +
one entry in the `NewRouter(...)` list in `cmd/api/main.go` is the complete change required.
SC-004 validated.

---

## Phase 5: User Story 3 — Migration Apply and Rollback (Priority: P3)

**Goal**: Operator can apply pending migrations and roll back the last one using
a standalone command. Both operations are idempotent and leave no orphaned schema objects.

**Independent Test**: `docker compose run --rm migrate down 1` removes the pgvector
extension. `docker compose run --rm migrate up` re-applies it. Running either command
twice produces the same final state (idempotent). No orphaned schema objects remain
after rollback.

### Implementation for User Story 3

- [ ] T017 [US3] Verify `docker-compose.yml` `migrate` service is correctly configured: image `migrate/migrate`, volume mount `./migrations:/migrations`, default command `["-path=/migrations", "-database=${DATABASE_URL}", "up"]`; confirm running `docker compose run --rm migrate down 1` uses the override command to roll back exactly one migration
- [ ] T018 [US3] Validate idempotency in a running stack: run `docker compose run --rm migrate up` a second time and confirm the output reports "no change" or zero migrations applied with exit code 0; document the expected output in `specs/001-m1-foundation/quickstart.md` under the rollback section
- [ ] T019 [P] [US3] Add SC-006 verification commands to `specs/001-m1-foundation/quickstart.md`: exact sequence — apply migrations, connect to psql, run `\dx` to confirm vector extension present; run `docker compose run --rm migrate down 1`; run `\dx` again to confirm extension absent; run `docker compose run --rm migrate up` to restore

**Checkpoint**: Migration apply + rollback cycle verified. `\dx` shows no vector extension
after rollback and the extension present after re-apply. SC-006 satisfied.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation across all user stories and documentation.

- [ ] T020 [P] Create `README.md` at repo root: project name, one-paragraph description, prerequisites (Docker Compose v2+, Git), three-step quickstart (cp .env.example, docker compose up --build, curl /health), link to `specs/001-m1-foundation/quickstart.md` for full validation steps
- [ ] T021 [P] Run `go vet ./...` and fix all reported issues; run `go build ./...` and confirm zero compilation errors across both binaries and all 11 stub modules
- [ ] T022 Run full acceptance criteria validation from `specs/001-m1-foundation/quickstart.md`: verify SC-001 (60s startup), SC-002 (500ms health check), SC-003 (10 min from clone), SC-004 (zero-file module add), SC-005 (all logs valid JSON), SC-006 (migration rollback clean)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 completion — **BLOCKS all user stories**
- **US1 (Phase 3)**: Depends on Phase 2 — MVP deliverable
- **US2 (Phase 4)**: Depends on Phase 2; Phase 3 recommended first (health module is the first Module registrant)
- **US3 (Phase 5)**: Depends on Phase 2 (migrations created in T006/T007); independent of US1 and US2
- **Polish (Phase 6)**: Depends on all user story phases complete

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 2 — no dependencies on US2 or US3
- **US2 (P2)**: Can start after Phase 2 — integrates with US1's health handler
- **US3 (P3)**: Can start after Phase 2 — fully independent of US1 and US2

### Within Each Phase

- Models/structs before services/handlers
- Config before handlers (handlers depend on Config fields)
- Docker files before docker-compose (Dockerfile referenced by compose)
- Core implementation before integration with other phases

### Parallel Opportunities

- T003 + T004 (Phase 1): both are setup files, no dependency
- T006 + T007 (Phase 2): two independent SQL files
- T010 + T005, T008, T009, T011 (Phase 2): T010 stubs are all independent; T005/T008/T009/T011 can proceed in parallel once T001–T004 are done
- T014 (Phase 3): middleware is independent of T012 and T013 implementation details

---

## Parallel Example: Phase 2

```
Launch together (after T001–T004 complete):
  Task T005: internal/config/config.go
  Task T006: migrations/000001_init.up.sql
  Task T007: migrations/000001_init.down.sql
  Task T008: docker/Dockerfile
  Task T009: docker-compose.yml
  Task T010: all 11 stub module files
  Task T011: cmd/worker/main.go
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (**critical — blocks all stories**)
3. Complete Phase 3: US1 — Full stack starts, health check works
4. **STOP and VALIDATE**: `docker compose up --build && curl /health` returns 200
5. Milestone M1 deliverable is met at this point

### Incremental Delivery

1. Setup + Foundational → binaries compile, Docker images build
2. US1 complete → stack starts, health check passes (M1 deliverable ✅)
3. US2 complete → module registration pattern established for M2–M12
4. US3 complete → migration tooling fully verified
5. Polish → all SC validated, README written

### Notes

- [P] tasks = different files, no dependency on each other — safe to implement simultaneously
- Each checkpoint is a testable state — stop and verify before moving to the next phase
- No test tasks generated (not requested in spec)
- `go build ./...` must pass at every checkpoint
