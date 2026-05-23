# Implementation Plan: Alerts & Analytics (M11)

**Branch**: `011-alerts-analytics` | **Date**: 2026-05-23 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/011-alerts-analytics/spec.md`

## Summary

M11 adds two capabilities on top of the existing analysis pipeline: (1) a real-time SSE alert stream that pushes events to docentes as submissions arrive and workers complete analysis, using Redis pub/sub for fan-out and `Last-Event-ID` for lossless reconnect; (2) on-demand analytics endpoints that aggregate submission, grade, and flag data per activity and per course, enforcing docente ownership throughout.

## Technical Context

**Language/Version**: Go 1.25.0

**Primary Dependencies**:
- `github.com/gofiber/fiber/v2 v2.52.0` — SSE via `SetBodyStreamWriter`
- `github.com/redis/go-redis/v9 v9.5.1` — pub/sub fan-out (`alerts:<docente_id>` channels) + short-lived token storage
- `github.com/jackc/pgx/v5 v5.6.0` — alerts persistence and analytics queries
- `github.com/golang-jwt/jwt/v5 v5.2.1` — standard auth middleware (SSE uses a separate short-lived token)
- `github.com/google/uuid v1.6.0` — stream token generation

**Storage**:
- PostgreSQL — `alerts` table (append-only, sequential BIGSERIAL id for Last-Event-ID)
- Redis — `alerts:<docente_id>` pub/sub channels; `sse_token:<uuid>` keys with 60 s TTL

**Target Platform**: Linux server (Docker Compose, same VPS topology as M1)

**Project Type**: Web service — extending existing `cmd/api` with two new internal packages

**Performance Goals**:
- SSE alert delivery ≤ 5 s from trigger to connected client (SC-001)
- Analytics endpoints ≤ 2 s for up to 500 students and 20 activities (SC-003)

**Constraints**:
- No new infrastructure — Redis and PostgreSQL already provisioned
- SSE uses Fiber's `SetBodyStreamWriter`; no additional WebSocket or long-poll mechanism
- Short-lived token stored in Redis (not a new table) — expires after 60 s or first use

**Scale/Scope**: Up to ~50 concurrent SSE connections per API instance (one per active docente session)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Evaluation | Status |
|---|---|---|
| I — Immutable Evidence | `alerts` rows are append-only; no UPDATE/DELETE operations permitted after insertion | ✅ PASS |
| II — Async Analysis Isolation | Alert writing by workers (detection, format, git) occurs after analysis completes and never blocks the student-facing HTTP path; SSE is a separate read-only stream | ✅ PASS |
| III — Human Authority Over AI | All AI-sourced alert payloads (format_violation, integrity_flag, git_analysis_done) include `"advisory": true` — no autonomous grade or sanction action is taken | ✅ PASS |
| IV — Strict Role-Based Data Isolation | SSE stream scoped to authenticated docente's own activities via token; analytics endpoints return 403 for wrong docente; students cannot access any alert or analytics endpoint | ✅ PASS |
| V — Dual RAG Independence | Not touched by M11 | ✅ PASS |
| VI — Non-Invasive Auditing | M11 introduces no new student monitoring; alerts are server-side events triggered by student actions already captured in prior milestones | ✅ PASS |
| VII — Submission Always Accepted | Alert creation is a side effect of submission; it never gates, delays, or blocks the submission response | ✅ PASS |

**All 7 gates pass. Proceeding to Phase 0.**

## Project Structure

### Documentation (this feature)

```text
specs/011-alerts-analytics/
├── plan.md              ← this file
├── research.md          ← Phase 0 output
├── data-model.md        ← Phase 1 output
├── quickstart.md        ← Phase 1 output
├── contracts/
│   └── alerts-analytics.md   ← Phase 1 output
└── tasks.md             ← /speckit-tasks output
```

### Source Code

```text
internal/alerts/
├── module.go         # Module struct, NewModule, RegisterRoutes
├── model.go          # Alert struct, sentinel errors, AlertType enum
├── repository.go     # InsertAlert, GetAlertsSince, MarkRead, GetDocenteForActividad
├── service.go        # WriteAlert (DB insert + Redis publish), IssueStreamToken, ValidateStreamToken
└── handler.go        # StreamTokenHandler, SSEStreamHandler, MarkReadHandler

internal/analytics/
├── module.go         # Module struct, NewModule, RegisterRoutes
├── repository.go     # ActivityAnalyticsQuery, CourseAnalyticsQuery
└── handler.go        # ActivityAnalyticsHandler, CourseAnalyticsHandler

migrations/
└── 000011_alerts_analytics.up.sql   # alerts table
└── 000011_alerts_analytics.down.sql
```

**Existing files modified**:
- `cmd/api/main.go` — register alerts and analytics modules
- `internal/submission/` — call `alertsSvc.WriteAlert` after accepting a file submission
- `internal/format/service.go` — call `alertsSvc.WriteAlert` after format analysis completes
- `internal/detection/service.go` — call `alertsSvc.WriteAlert` after detection analysis completes
- `internal/projects/git_worker.go` — call `alertsSvc.WriteAlert` after git analysis completes
- `internal/projects/service_investigacion.go` — call `alertsSvc.WriteAlert` on late stage submission

## Complexity Tracking

No constitution violations. Standard append-only alert pattern.
