# Tasks: M10 — Project Management

**Input**: Design documents from `specs/010-project-management/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/project-management.md ✓

**Organization**: Tasks grouped by user story (US1 → investigacion, US2 → git_analysis, US3 → defensa_oral).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other [P] tasks in the same phase
- **[Story]**: User story this task belongs to

---

## Phase 1: Setup

**Purpose**: Add the `go-git` dependency and project skeleton for `internal/projects/`.

- [X] T001 Add `github.com/go-git/go-git/v5` dependency by running `go get github.com/go-git/go-git/v5` and committing the updated `go.mod` and `go.sum`
- [X] T002 Verify `internal/projects/` package exists as `package projects` stub in `internal/projects/module.go` (already present — confirm and leave in place)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Database schema and shared package skeleton that all three stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T003 Write `migrations/000010_project_management.up.sql`: `ALTER TYPE actividad_tipo ADD VALUE 'investigacion'; ALTER TYPE actividad_tipo ADD VALUE 'git_analysis';` then create tables `investigacion_stages`, `stage_submissions`, `git_submissions`, `git_analyses`, `git_analysis_jobs`, `defense_appointments` per `specs/010-project-management/data-model.md`
- [X] T004 Write `migrations/000010_project_management.down.sql`: drop all 6 tables in reverse dependency order; note that removing enum values requires a full enum recreate — document this limitation in a comment
- [X] T005 Write `internal/projects/model.go`: define Go structs `InvestigacionStage`, `StageSubmission`, `GitSubmission`, `GitAnalysis`, `GitAnalysisJob`, `DefenseAppointment`, plus sentinel errors `ErrForbidden`, `ErrNotFound`, `ErrConflict`, `ErrStageHasSubmissions`, `ErrNoStages`
- [X] T006 Write `internal/projects/repository.go`: define `Repository` struct with `*pgxpool.Pool`; implement a stub `NewRepository(db) *Repository`; leave method stubs (no-op bodies returning `nil`) for all DB operations referenced in the service files — one method per contract endpoint; methods added in later tasks will fill in the SQL
- [X] T007 Write `internal/projects/module.go`: define `Module` struct holding `*Repository`, `*minio.Client`, `minioBucket string`, `*ai.Service`, `*config.Config`; implement `NewModule(db, minio, bucket, ai, cfg) *Module`; add `RegisterRoutes(app *fiber.App, jwtMiddleware fiber.Handler)` stub that wires all routes from `specs/010-project-management/contracts/project-management.md`

**Checkpoint**: Migration runs cleanly (`migrate up`), package compiles, routes registered (handlers return 501 at this stage).

---

## Phase 3: User Story 1 — Multi-Stage Research Report (Priority: P1) 🎯 MVP

**Goal**: Docente creates an `investigacion` actividad, defines stages, students submit per-stage, docente views progress and grades each stage independently.

**Independent Test**: Create `investigacion` actividad → add 3 stages → submit stage 1 on time, stage 3 after deadline → GET `/stage-progress` → see stage1=submitted, stage2=pending, stage3=submitted_late → grade stage 1 → attempt DELETE stage 1 → get 409. (Quickstart Story 1, steps 1–7.)

### Implementation for User Story 1

- [X] T008 [P] [US1] Implement `Repository` methods for stage CRUD in `internal/projects/repository.go`: `InsertStage`, `ListStages`, `UpdateStage`, `GetStage`, `StageHasSubmissions`, `DeleteStage` — SQL queries against `investigacion_stages`
- [X] T009 [P] [US1] Implement `Repository` methods for stage submissions in `internal/projects/repository.go`: `UpsertStageSubmission`, `GetStageSubmission`, `ListStageSubmissionsForActividad` (returns all submissions across all stages for the progress matrix)
- [X] T010 [US1] Write `internal/projects/service_investigacion.go`: implement `CreateStage`, `ListStages`, `UpdateStage`, `DeleteStage` (checks `StageHasSubmissions` → `ErrStageHasSubmissions` on conflict), `SubmitToStage` (uploads file to MinIO, computes `is_late` by comparing `now()` to stage deadline, calls `UpsertStageSubmission`), `GetStageProgress` (builds the per-student matrix from `ListStageSubmissionsForActividad`), `GradeStageSubmission` — depends on T008, T009
- [X] T011 [US1] Write Fiber handlers in `internal/projects/handler.go` for US1 endpoints: `CreateStageHandler`, `ListStagesHandler`, `UpdateStageHandler`, `DeleteStageHandler`, `SubmitToStageHandler` (multipart), `GetStageProgressHandler`, `GradeStageSubmissionHandler` — each handler extracts JWT claims, validates actividad tipo is `investigacion`, delegates to service methods
- [X] T012 [US1] Wire US1 routes in `internal/projects/module.go` `RegisterRoutes`: `POST /actividades/:actividadId/stages`, `GET /actividades/:actividadId/stages`, `PUT /actividades/:actividadId/stages/:stageId`, `DELETE /actividades/:actividadId/stages/:stageId`, `POST /actividades/:actividadId/stages/:stageId/submit`, `GET /actividades/:actividadId/stage-progress`, `PUT /actividades/:actividadId/stages/:stageId/submissions/:studentId/grade`
- [X] T013 [US1] Add publication guard in `CreateActividadHandler` (or a new `PublishActividadHandler`) in `internal/academic/handler_actividad.go`: for `tipo = investigacion`, reject publish/create-with-publish if zero stages exist by calling `Repository.CountStagesForActividad` (FR-007); add `CountStagesForActividad` to `internal/projects/repository.go`
- [X] T014 [US1] Register `internal/projects` Module in `cmd/api/main.go` (or wherever routes are wired): call `projectsModule.RegisterRoutes(app, jwtMiddleware)`

**Checkpoint**: Full US1 flow works via `quickstart.md` Story 1 steps 1–7. Stage-progress returns correct statuses. Grade saved. Delete blocked with 409.

---

## Phase 4: User Story 2 — Git Repository Analysis Activity (Priority: P2)

**Goal**: Docente creates a `git_analysis` actividad; student submits a Git URL; async worker clones, runs heuristics, dispatches to LLM; docente reviews report and grades.

**Independent Test**: Create `git_analysis` actividad → student submits `https://github.com/octocat/Hello-World` → wait up to 3 min → GET report as docente → verify `commit_count > 0`, `authors` array, `flags` array, `llm_advisory` present → GET same report as student → 403 → POST grade. (Quickstart Story 2, steps 1–5.)

### Implementation for User Story 2

- [X] T015 [P] [US2] Implement `Repository` methods for git submissions in `internal/projects/repository.go`: `UpsertGitSubmission`, `GetGitSubmission`, `GetGitSubmissionByActividad`
- [X] T016 [P] [US2] Implement `Repository` methods for git analyses in `internal/projects/repository.go`: `InsertGitAnalysis`, `UpdateGitAnalysis`, `GetLatestGitAnalysis`, `GetGitAnalysisForReport`; and for git analysis jobs: `InsertGitAnalysisJob`, `ClaimNextGitAnalysisJob`, `CompleteGitAnalysisJob`, `FailGitAnalysisJob`
- [X] T017 [US2] Write `internal/projects/service_git_analysis.go`: implement `SubmitGitURL` (upsert submission, insert git_analysis row with status=pending, insert git_analysis_job row), `GetGitReport` (checks docente ownership via actividad; returns 403 for student role), `GradeGitAnalysis` (validates grade range, stores grade + override_reason on `git_analyses`) — depends on T015, T016
- [X] T018 [US2] Write Fiber handlers in `internal/projects/handler.go` for US2: `GitSubmitHandler`, `GetGitReportHandler` (role check: student → 403), `GitGradeHandler` — delegates to service
- [X] T019 [US2] Wire US2 routes in `internal/projects/module.go` `RegisterRoutes`: `POST /actividades/:actividadId/git-submit`, `GET /actividades/:actividadId/git-report/:studentId`, `POST /actividades/:actividadId/git-grade/:studentId`
- [X] T020 [US2] Implement `internal/projects/git_worker.go`: function `ProcessGitAnalysisJob(ctx, job, db, aiSvc, cfg) error` — (1) shallow clone repo via `go-git` (`CloneOptions{Depth: 200, SingleBranch: true}`); on clone failure set `fail_reason=clone_error`, update analysis status=failed, return nil; (2) iterate commits via `CommitIterator`: count total commits, collect distinct author names, record first/last commit timestamps, count commits in last 24h before `actividad.deadline`; (3) compute `consistency_score` (100 − deadline_rush_pct, clamped 0–100); (4) set flags `deadline_rush` when `deadline_rush_pct > 70`, `multiple_authors` when `len(authors) > 1`; (5) call `ai.Service.Dispatch` with git-specific prompt (commit count, authors, dates, flags); store `llm_advisory` and `llm_status`; (6) update `git_analyses` row to status=success
- [X] T021 [US2] Extend `cmd/worker/main.go` poll loop: add a second ticker (or extend existing loop) that calls `Repository.ClaimNextGitAnalysisJob` and dispatches to `ProcessGitAnalysisJob`; keep existing exam-session job processing intact; ensure graceful shutdown handles both job types

**Checkpoint**: Full US2 flow works via `quickstart.md` Story 2 steps 1–5. Report visible to docente, 403 for student. Grade stored.

---

## Phase 5: User Story 3 — Oral Defense Scheduling and Grading (Priority: P3)

**Goal**: Docente schedules a per-student defense appointment; student sees their date; docente records grade directly with no file or tracing required.

**Independent Test**: Create `defensa_oral` actividad → PUT schedule for student → GET appointment as student → see `scheduled_at` set → GET for unscheduled student → see `scheduled_at: null` → PUT grade → see grade stored. (Quickstart Story 3, steps 1–5.)

### Implementation for User Story 3

- [X] T022 [P] [US3] Implement `Repository` methods for defense appointments in `internal/projects/repository.go`: `UpsertDefenseAppointment`, `GetDefenseAppointment` (scoped: student sees own only, docente sees any)
- [X] T023 [US3] Write `internal/projects/service_defensa.go`: implement `ScheduleDefense` (upserts appointment, replaces prior), `GetDefenseAppointment` (student scoping: if caller role=student, enforce `caller_id == student_id` else 403), `GradeDefense` (validates grade 0–100, stores grade + graded_at + graded_by) — depends on T022
- [X] T024 [US3] Write Fiber handlers in `internal/projects/handler.go` for US3: `ScheduleDefenseHandler`, `GetDefenseAppointmentHandler`, `GradeDefenseHandler`
- [X] T025 [US3] Wire US3 routes in `internal/projects/module.go` `RegisterRoutes`: `PUT /actividades/:actividadId/defensa/:studentId/schedule`, `GET /actividades/:actividadId/defensa/:studentId`, `PUT /actividades/:actividadId/defensa/:studentId/grade`

**Checkpoint**: Full US3 flow works via `quickstart.md` Story 3 steps 1–5. Schedule visible to student. Grade stored. No tracing triggered.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validation guard on actividad tipo for all new endpoints; error consistency; `go vet` and compile clean.

- [X] T026 Add tipo guard helper in `internal/projects/handler.go`: shared function `requireTipo(c *fiber.Ctx, db, actividadID, expectedTipo) error` that returns 400 with `"invalid tipo"` if the actividad exists but has the wrong tipo — call from CreateStageHandler, GitSubmitHandler, ScheduleDefenseHandler
- [X] T027 Ensure all handlers return the shared error shape `{"error":"code","message":"...","correlation_id":"..."}` consistent with `internal/academic/` handlers — verify against `contracts/project-management.md` error shapes
- [X] T028 Run `go build ./...` and `go vet ./...` from repo root; fix any compile errors or vet warnings in `internal/projects/` and `cmd/worker/`
- [ ] T029 Run the quickstart scenarios from `specs/010-project-management/quickstart.md` against a live stack and confirm all steps pass (or document known gaps)

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

- Repository methods (T008/T009, T015/T016, T022) → service → handlers → route wiring
- T008 and T009 can run in parallel (different repository method groups)
- T015 and T016 can run in parallel

---

## Parallel Example: User Story 2

```
# Parallel (different method groups, same file — can be split by batch):
T015 git_submissions repository methods
T016 git_analyses + git_analysis_jobs repository methods

# Then sequential:
T017 service_git_analysis.go (depends on T015, T016)
T018 handler.go US2 handlers (depends on T017)
T019 route wiring (depends on T018)
T020 git_worker.go (can overlap with T018/T019 — separate file)
T021 cmd/worker/main.go extension (depends on T020)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 + Phase 2 (migration + package skeleton)
2. Complete Phase 3 (US1 — investigacion stages)
3. **STOP and VALIDATE**: run quickstart Story 1 steps 1–7
4. Deliver/demo US1 independently

### Incremental Delivery

1. Phase 1 + 2 → foundation ready
2. Phase 3 (US1) → stage-based research reports working
3. Phase 4 (US2) → git analysis working
4. Phase 5 (US3) → oral defense scheduling working
5. Phase 6 → polish and validate all

### Parallel Team Strategy

After Phase 2 completes:
- Developer A: Phase 3 (US1)
- Developer B: Phase 4 (US2) — independent
- Developer C: Phase 5 (US3) — independent

---

## Notes

- [P] tasks operate on different files or non-overlapping method groups within a file
- Each story has an independent quickstart test path — validate before proceeding to next story
- T021 (worker extension) must not break existing M4–M9 exam-session job processing
- The `trace_enabled` generated column evaluates to `false` for `investigacion` and `git_analysis` — no WebSocket sessions are created for these tipos
- Shallow clone depth 200 (`go-git CloneOptions.Depth = 200`) bounds worker memory
