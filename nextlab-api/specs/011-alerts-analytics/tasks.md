# Tasks: M11 — Alerts & Analytics

**Input**: Design documents from `specs/011-alerts-analytics/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/alerts-analytics.md ✓

**Organization**: Tasks grouped by user story (US1 → SSE alert stream, US2 → per-activity analytics, US3 → course analytics).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other [P] tasks in the same phase
- **[Story]**: User story this task belongs to

---

## Phase 1: Setup

**Purpose**: Migration and package stubs.

- [ ] T001 Write `migrations/000011_alerts_analytics.up.sql`: create `alerts` table with `BIGSERIAL` primary key `id`, `docente_id UUID NOT NULL REFERENCES users(id)`, `actividad_id UUID REFERENCES actividades(id) ON DELETE SET NULL`, `student_id UUID REFERENCES users(id)`, `type TEXT NOT NULL`, `payload JSONB NOT NULL DEFAULT '{}'`, `advisory BOOLEAN NOT NULL DEFAULT false`, `is_read BOOLEAN NOT NULL DEFAULT false`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`; add indexes `alerts_docente_id_idx ON alerts(docente_id, id)` and `alerts_actividad_id_idx ON alerts(actividad_id)`
- [ ] T002 Write `migrations/000011_alerts_analytics.down.sql`: `DROP TABLE IF EXISTS alerts;`
- [ ] T003 Verify `internal/alerts/module.go` and `internal/analytics/module.go` exist as `package alerts` and `package analytics` stubs — confirm and leave in place

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Models, repositories, and service skeleton that all three stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T004 [P] Write `internal/alerts/model.go`: define `AlertType` string type and constants (`AlertSubmissionReceived`, `AlertLateSubmission`, `AlertFormatViolation`, `AlertIntegrityFlag`, `AlertGitAnalysisDone`); define `Alert` struct (`ID int64`, `DocenteID uuid.UUID`, `ActividadID *uuid.UUID`, `StudentID *uuid.UUID`, `Type AlertType`, `Payload map[string]any`, `Advisory bool`, `IsRead bool`, `CreatedAt time.Time`); define `AlertWriter` interface with `WriteAlert(ctx context.Context, a Alert) error`; define sentinel errors `ErrAlertNotFound`, `ErrForbidden`
- [ ] T005 [P] Write `internal/analytics/model.go`: define `GradeDistribution` struct (`Range0_49`, `Range50_69`, `Range70_89`, `Range90_100 int`); define `ActivityAnalytics` struct (`ActividadID uuid.UUID`, `SubmissionCount`, `PendingCount`, `LateCount`, `FlaggedCount int`, `GradeDistribution GradeDistribution`, `AvgGrade *float64`); define `CourseAnalytics` struct (`CursoID uuid.UUID`, `Activities []ActivityAnalytics`, `TotalSubmissions int`, `AvgGrade *float64`, `FlagRate float64`)
- [ ] T006 Write `internal/alerts/repository.go`: `Repository` struct with `*pgxpool.Pool` and `*redis.Client`; implement `NewRepository(db, rdb) *Repository`; implement `InsertAlert(ctx, Alert) (int64, error)` — INSERT INTO alerts returning id; implement `GetAlertsSince(ctx, docenteID uuid.UUID, sinceID int64, unreadOnly bool, limit int) ([]Alert, error)` — SELECT WHERE docente_id=$1 AND id > $2 ORDER BY id ASC; implement `MarkAlertRead(ctx, alertID int64, docenteID uuid.UUID) error` — UPDATE is_read=true WHERE id=$1 AND docente_id=$2 (returns ErrNotFound if 0 rows); implement `GetDocenteForActividad(ctx, actividadID uuid.UUID) (uuid.UUID, error)` — SELECT docente_id FROM actividades WHERE id=$1
- [ ] T007 Write `internal/alerts/service.go`: `Service` struct holding `*Repository` and `*redis.Client`; implement `NewService(repo, rdb) *Service`; implement `WriteAlert(ctx, Alert) error` — calls repo.InsertAlert to get id, marshals alert to JSON, calls `rdb.Publish(ctx, "alerts:"+docenteID.String(), jsonPayload)`; implement `IssueStreamToken(ctx, docenteID uuid.UUID) (string, error)` — generates UUID v4, calls `rdb.Set(ctx, "sse_token:"+token, docenteID.String(), 60*time.Second)`; implement `ValidateStreamToken(ctx, token string) (uuid.UUID, error)` — calls `rdb.GetDel(ctx, "sse_token:"+token)`, returns ErrForbidden if key missing
- [ ] T008 Write `internal/analytics/repository.go`: `Repository` struct with `*pgxpool.Pool`; implement `NewRepository(db) *Repository`; implement `ActivityAnalyticsQuery(ctx, actividadID uuid.UUID, tipo string) (ActivityAnalytics, error)` — multi-tipo conditional query: for `tarea/informe/examen` aggregate from `submissions`; for `investigacion` aggregate from `stage_submissions` joined to `investigacion_stages`; for `git_analysis` aggregate from `git_submissions`/`git_analyses`; for `defensa_oral` aggregate from `defense_appointments`; use `COUNT(*) FILTER (WHERE ...)` for pending/late/graded counts and grade distribution buckets; `flagged_count` from `alerts` WHERE actividad_id=$1 AND type IN ('integrity_flag','format_violation'); implement `CourseAnalyticsQuery(ctx, cursoID uuid.UUID, docenteID uuid.UUID) (CourseAnalytics, error)` — lists all actividades for the course, calls ActivityAnalyticsQuery per actividad, computes rollup totals
- [ ] T009 Write `internal/alerts/module.go`: `Module` struct holding `*Repository`, `*Service`; implement `NewModule(db *pgxpool.Pool, rdb *redis.Client) *Module`; implement `Service() *Service` accessor (used by cmd/api/main.go to pass AlertWriter to other modules); add `RegisterRoutes(app *fiber.App, jwtMiddleware fiber.Handler, roleGuard func(...string) fiber.Handler)` stub returning 501
- [ ] T010 Write `internal/analytics/module.go`: `Module` struct holding `*Repository`; implement `NewModule(db *pgxpool.Pool) *Module`; add `RegisterRoutes(app *fiber.App, jwtMiddleware fiber.Handler, roleGuard func(...string) fiber.Handler)` stub returning 501

**Checkpoint**: Migration runs cleanly, both packages compile, routes registered (501 at this stage).

---

## Phase 3: User Story 1 — Real-Time SSE Alert Stream (Priority: P1) 🎯 MVP

**Goal**: Docente connects to SSE stream via short-lived token; receives real-time alerts for their activities; replays missed alerts on reconnect via Last-Event-ID; can list and mark alerts as read.

**Independent Test**: `POST /alerts/stream-token` → open `GET /alerts/stream?token=<value>` → submit file as student → stream delivers `submission_received` within 5 s → kill connection → reconnect with `Last-Event-ID: 0` → stream replays the alert → `PATCH /alerts/1/read` returns `{"ok":true}`. Also verify second docente's stream stays silent. (Quickstart Story 1, steps 1–7.)

### Implementation for User Story 1

- [ ] T011 [P] [US1] Write `StreamTokenHandler` in `internal/alerts/handler.go`: extracts docente JWT claims, calls `svc.IssueStreamToken`, returns 200 `{"token":"<uuid>"}` — roleGuard: docente/co-docente only
- [ ] T012 [US1] Write `SSEStreamHandler` in `internal/alerts/handler.go`: reads `?token` query param, calls `svc.ValidateStreamToken` (401 on fail); reads `Last-Event-ID` request header (default 0); sets SSE response headers (`Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`); calls `c.Context().SetBodyStreamWriter` with a closure that: (1) queries `repo.GetAlertsSince(ctx, docenteID, sinceID, false, 500)` and writes each as SSE `id:`/`event:`/`data:` lines, (2) subscribes to Redis `"alerts:"+docenteID` via `rdb.Subscribe`, (3) forwards messages from `sub.Channel()` until `ctx.Done()` — depends on T006, T007
- [ ] T013 [P] [US1] Write `GetAlertsHandler` in `internal/alerts/handler.go`: GET /alerts; reads `?since_id` (default 0), `?unread_only` (default false), `?limit` (default 50, max 200) query params; calls `repo.GetAlertsSince`; returns `{"alerts":[...],"total":N}`
- [ ] T014 [P] [US1] Write `MarkReadHandler` in `internal/alerts/handler.go`: PATCH /alerts/:id; parses alert id; calls `repo.MarkAlertRead(ctx, id, docenteID)`; maps `ErrAlertNotFound`→404, `ErrForbidden`→403; returns `{"ok":true}`
- [ ] T015 [US1] Wire US1 routes in `internal/alerts/module.go` `RegisterRoutes`: `POST /alerts/stream-token` (jwtMiddleware + roleGuard("docente","co-docente")), `GET /alerts/stream` (no standard JWT — token in query param), `GET /alerts` (jwtMiddleware + roleGuard("docente","co-docente")), `PATCH /alerts/:id/read` (jwtMiddleware + roleGuard("docente","co-docente")) — depends on T011–T014
- [ ] T016 [US1] Inject `AlertWriter` into existing submission pipeline: (a) add `alertWriter alerts.AlertWriter` field to `internal/submission` module/service and call `alertWriter.WriteAlert(ctx, Alert{Type: AlertSubmissionReceived, ...})` after file stored; (b) add `alertWriter alerts.AlertWriter` to `internal/projects/InvestigacionService` and call `WriteAlert` for `submission_received` and additionally `late_submission` when `isLate=true` in `SubmitToStage` — depends on T007
- [ ] T017 [US1] Inject `AlertWriter` into async workers: (a) add `alertWriter alerts.AlertWriter` param to `internal/format` service and call `WriteAlert(ctx, Alert{Type: AlertFormatViolation, Advisory: true, Payload: {violations, severity}})` after non-compliant result; (b) add `alertWriter` to `internal/detection` service and call `WriteAlert(ctx, Alert{Type: AlertIntegrityFlag, Advisory: true, Payload: {flags, score}})` after detection completes with flags; (c) add `alertWriter` param to `ProcessGitAnalysisJob` in `internal/projects/git_worker.go` and call `WriteAlert(ctx, Alert{Type: AlertGitAnalysisDone, Advisory: true, Payload: {flag_count, flags}})` after analysis stored — depends on T007

**Checkpoint**: Full US1 flow works via quickstart Story 1. SSE delivers events within 5 s. Last-Event-ID replay works. Mark-read persists. Role isolation verified.

---

## Phase 4: User Story 2 — Per-Activity Analytics (Priority: P2)

**Goal**: Docente requests analytics for a single activity and receives submission count, pending count, late count, flagged count, grade distribution, and avg grade — all computed live.

**Independent Test**: Activity with 3 enrolled students, 2 submitted (1 late), 1 flagged, 1 graded 85 → `GET /actividades/:id/analytics` → `submission_count=2`, `pending_count=1`, `late_count=1`, `flagged_count=1`, `avg_grade=85.0`. (Quickstart Story 2, steps 1–3.)

### Implementation for User Story 2

- [ ] T018 [US2] Write `ActivityAnalyticsHandler` in `internal/analytics/handler.go`: GET /actividades/:actividadId/analytics; extracts docenteID from JWT; fetches actividad to verify ownership (403 if wrong docente); fetches actividad.tipo; calls `repo.ActivityAnalyticsQuery(ctx, actividadID, tipo)`; returns `ActivityAnalytics` as JSON; maps errors to 403/404 — depends on T008
- [ ] T019 [US2] Wire US2 route in `internal/analytics/module.go` `RegisterRoutes`: `GET /actividades/:actividadId/analytics` (jwtMiddleware + roleGuard("docente","co-docente")) — depends on T018

**Checkpoint**: Full US2 flow works via quickstart Story 2. Counts correct for all tipos. 403 for wrong docente. Null avg_grade when no grades.

---

## Phase 5: User Story 3 — Course-Level Analytics (Priority: P3)

**Goal**: Docente requests analytics across all activities in a course and receives a per-activity breakdown plus a course-level rollup.

**Independent Test**: Course with 2 activities → `GET /cursos/:id/analytics` → response has `activities` array with 2 entries and `total_submissions`, `avg_grade`, `flag_rate` rollup. (Quickstart Story 3, steps 1–3.)

### Implementation for User Story 3

- [ ] T020 [US3] Write `CourseAnalyticsHandler` in `internal/analytics/handler.go`: GET /cursos/:cursoId/analytics; validates docente owns the course (403 if not); calls `repo.CourseAnalyticsQuery(ctx, cursoID, docenteID)`; returns `CourseAnalytics` as JSON — depends on T008
- [ ] T021 [US3] Wire US3 route in `internal/analytics/module.go` `RegisterRoutes`: `GET /cursos/:cursoId/analytics` (jwtMiddleware + roleGuard("docente","co-docente")) — depends on T020

**Checkpoint**: Full US3 flow works via quickstart Story 3. Per-activity breakdown correct. Rollup totals correct. Null avg_grade and zero flag_rate when no data.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Wire modules into main.go, pass AlertWriter to existing modules, build clean.

- [ ] T022 Register `internal/alerts` and `internal/analytics` modules in `cmd/api/main.go`: `alertsModule := alerts.NewModule(pgPool, redisClient)` and `analyticsModule := analytics.NewModule(pgPool)`; call `alertsModule.RegisterRoutes(app, auth.AuthMiddleware(cfg), auth.RoleGuard)` and `analyticsModule.RegisterRoutes(app, auth.AuthMiddleware(cfg), auth.RoleGuard)`; pass `alertsModule.Service()` as the `AlertWriter` to submission, format, detection, and projects modules when constructing them
- [ ] T023 Run `go build ./...` and `go vet ./...` from repo root; fix any compile errors or vet warnings in `internal/alerts/`, `internal/analytics/`, `cmd/api/`, and any modified existing packages
- [ ] T024 Run the quickstart scenarios from `specs/011-alerts-analytics/quickstart.md` against a live stack and confirm all steps pass (or document known gaps)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — **blocks all user stories**
- **US1 (Phase 3)**: Depends on Phase 2
- **US2 (Phase 4)**: Depends on Phase 2 — independent of US1
- **US3 (Phase 5)**: Depends on Phase 2 — independent of US1 and US2
- **Polish (Phase 6)**: Depends on US1, US2, US3

### Within Each User Story

- T004/T005 (models) → T006/T008 (repositories) → T007 (service) → handlers → route wiring → integration hooks
- T004 and T005 can run in parallel (different packages)
- T011, T013, T014 can run in parallel (different handlers, same file — non-overlapping functions)

---

## Parallel Example: User Story 1

```
# Parallel (independent handlers):
T011 StreamTokenHandler
T013 GetAlertsHandler
T014 MarkReadHandler

# Then sequential:
T012 SSEStreamHandler (depends on repo + service)
T015 Route wiring (depends on all handlers)
T016 Submission pipeline injection (depends on service)
T017 Worker injection (depends on service)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 + Phase 2 (migration + package skeletons)
2. Complete Phase 3 (US1 — SSE stream)
3. **STOP and VALIDATE**: run quickstart Story 1 steps 1–7
4. Deliver/demo US1 independently

### Incremental Delivery

1. Phase 1 + 2 → foundation ready
2. Phase 3 (US1) → real-time alert stream working
3. Phase 4 (US2) → per-activity analytics working
4. Phase 5 (US3) → course analytics working
5. Phase 6 → wire, build clean, validate all

---

## Notes

- [P] tasks operate on different files or non-overlapping function groups within a file
- Each story has an independent quickstart test path — validate before proceeding to next
- T016/T017 inject `AlertWriter` into existing packages — never import `internal/alerts` from those packages directly; use the interface type only to avoid circular imports
- T024 requires `docker compose up` — document gaps if live stack is unavailable
- The `alerts` table is append-only (Principle I) — no UPDATE except `is_read`, no DELETE
