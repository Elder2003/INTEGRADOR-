# Tasks: M4 — File Storage

**Input**: Design documents from `specs/004-file-storage/`

**Prerequisites**: plan.md ✅ | spec.md ✅ | research.md ✅ | data-model.md ✅ | contracts/submissions.md ✅ | quickstart.md ✅

**Organization**: Tasks are grouped by user story. Each phase is independently deliverable and testable. No test tasks generated (not requested in spec).

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the MinIO SDK dependency, extend config, and add env-var documentation before any feature work begins.

- [X] T001 Add `github.com/minio/minio-go/v7` to `nextlab-api/go.mod` via `go get github.com/minio/minio-go/v7` and run `go mod tidy`
- [X] T002 Add `MinioEndpoint string`, `MinioAccessKey string`, `MinioSecretKey string`, `MinioBucket string`, `MinioUseSSL bool` fields to `nextlab-api/internal/config/config.go`; add startup validation — if `MinioEndpoint`, `MinioAccessKey`, or `MinioSecretKey` are empty → `slog.Error` + `os.Exit(1)` matching the existing JWT_SECRET pattern; default `MinioBucket` to `"nextlab"` if empty
- [X] T003 [P] Add `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET` (default: `nextlab`), and `MINIO_USE_SSL` (default: `false`) entries with comments to `nextlab-api/.env.example`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Database migration, Go model structs, MinIO client initialization, and the new module skeleton — everything all user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T004 Write `nextlab-api/migrations/000004_file_storage.up.sql` per `data-model.md`: `ALTER TABLE actividades ADD COLUMN late_submission_allowed BOOLEAN NOT NULL DEFAULT true`; `CREATE TABLE actividad_config_changes` (id, actividad_id FK, changed_by FK, field_name TEXT, previous_value TEXT, new_value TEXT, changed_at TIMESTAMPTZ) with index on `(actividad_id, changed_at DESC)`; `CREATE TABLE submissions` (id, student_id FK, actividad_id FK, storage_key TEXT, original_filename TEXT, file_size_bytes BIGINT, mime_type TEXT, submitted_at TIMESTAMPTZ, is_late BOOLEAN, edit_window_expires_at TIMESTAMPTZ nullable, last_replaced_at TIMESTAMPTZ nullable, created_at TIMESTAMPTZ, `UNIQUE(student_id, actividad_id)`) with indexes on `actividad_id` and `student_id`
- [X] T005 Write `nextlab-api/migrations/000004_file_storage.down.sql`: `DROP TABLE IF EXISTS submissions`; `DROP TABLE IF EXISTS actividad_config_changes`; `ALTER TABLE actividades DROP COLUMN IF EXISTS late_submission_allowed`
- [X] T006 [P] Create `nextlab-api/internal/submission/model.go` with Go structs: `Submission` (all DB columns), `SubmissionResponse` (JSON output: id, actividad_id, original_filename, file_size_bytes, mime_type, submitted_at, is_late, edit_window_expires_at, is_locked bool derived field, last_replaced_at), `SubmissionListItem` (adds student name + email for docente view), `SubmissionListResponse` (actividad_id + submissions slice + total), `ConfigChangeResponse` (id, changed_by name, field_name, previous_value, new_value, changed_at)
- [X] T007 [P] Extend `nextlab-api/internal/academic/model.go`: add `LateSubmissionAllowed bool` to `Actividad` struct and to the actividad response/update request types; add `ActivityConfigChange` struct (id, actividad_id, changed_by UUID, field_name, previous_value, new_value, changed_at)
- [X] T008 [P] Create `nextlab-api/internal/submission/module.go` skeleton: `RegisterRoutes(app *fiber.App, db *pgxpool.Pool, mc *minio.Client, cfg *config.Config, authMW fiber.Handler, roleGuard func(...) fiber.Handler)` — route wiring will be completed as each user story phase is implemented; include the handler + repo + service struct initializations
- [X] T009 Extend `nextlab-api/cmd/api/main.go`: initialize `*minio.Client` using `minio.New(cfg.MinioEndpoint, &minio.Options{Creds: credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""), Secure: cfg.MinioUseSSL})`; call `mc.BucketExists` + `mc.MakeBucket` (idempotent) at startup with `os.Exit(1)` on error; log `[minio] bucket "nextlab" ready`; add `submission.RegisterRoutes(app, pgPool, mc, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard)` call after existing route registrations

**Checkpoint**: Run `docker compose up --build` — server starts, migration `000004` applies cleanly (`\dt` shows `submissions` and `actividad_config_changes`), MinIO bucket log line appears, no compilation errors.

---

## Phase 3: User Story 1 — Student Submits a File (Priority: P1) 🎯 MVP

**Goal**: A student can upload a file for an activity. The system validates enrollment and activity type, records `submitted_at` server-side, computes `is_late` and `edit_window_expires_at`, stores the file in MinIO atomically with the DB record, and returns a confirmation. Late submissions are rejected when `late_submission_allowed = false`.

**Independent Test**: Student uploads a PDF for an on-time activity → 201 with `is_late=false` and a valid `edit_window_expires_at`. Student uploads for a past-deadline activity with `late_submission_allowed=true` → 201 with `is_late=true` and `edit_window_expires_at=null`. Same activity with `late_submission_allowed=false` → 422. Non-enrolled student → 403.

### Implementation

- [X] T010 [P] Implement validation queries in `nextlab-api/internal/submission/repository.go`: `StudentEnrolledInCurso(ctx, studentID, actividadID)` (JOIN inscripciones → cursos → actividades; returns bool); `GetActividadForSubmission(ctx, actividadID)` (returns actividad id, tipo, deadline, late_submission_allowed, has_submissions, curso is_active); `GetExistingSubmission(ctx, studentID, actividadID)` (returns *Submission or nil)
- [X] T011 [P] Implement write queries in `nextlab-api/internal/submission/repository.go`: `InsertSubmission(ctx, tx, sub *Submission)` (INSERT INTO submissions); `MarkActivityHasSubmissions(ctx, tx, actividadID)` (UPDATE actividades SET has_submissions=true WHERE id=$1 AND has_submissions=false)
- [X] T012 [US1] Implement `UploadFile(ctx, studentID, actividadID, header *multipart.FileHeader)` in `nextlab-api/internal/submission/service.go`
- [X] T013 [US1] Implement `POST /actividades/:aid/submission` in `nextlab-api/internal/submission/handler_student.go`
- [X] T014 [US1] Implement `GET /actividades/:aid/submission` in `nextlab-api/internal/submission/handler_student.go`
- [X] T015 [US1] Wire `POST /actividades/:aid/submission` and `GET /actividades/:aid/submission` in `nextlab-api/internal/submission/module.go` with `authMW` + `roleGuard("estudiante")`

**Checkpoint**: US1 fully testable — follow steps 4–5 from `quickstart.md`. Verify `has_submissions` flips to `true` on `actividades` after first upload. Verify MinIO bucket contains the uploaded object.

---

## Phase 4: User Story 2 — Edit Window: Replace or Delete (Priority: P2)

**Goal**: A student with an on-time submission in the edit window can replace the file (new file supersedes old, `edit_window_expires_at` unchanged, `is_late` updated to reflect new timestamp) or delete the submission entirely and re-upload. Late submissions and expired-window submissions are locked and cannot be changed.

**Independent Test**: Student submits on time → replaces within window → GET shows new filename, `last_replaced_at` set. Student waits for window to expire → PUT returns 403. Late submission → DELETE immediately returns 403.

### Implementation

- [X] T016 [P] Implement mutation queries in `nextlab-api/internal/submission/repository.go`: `UpdateSubmission` and `DeleteSubmission`
- [X] T017 [US2] Implement `ReplaceFile(ctx, studentID, actividadID, header)` in `nextlab-api/internal/submission/service.go`
- [X] T018 [US2] Implement `DeleteOwnSubmission(ctx, studentID, actividadID)` in `nextlab-api/internal/submission/service.go`
- [X] T019 [US2] Implement `PUT /actividades/:aid/submission` in `nextlab-api/internal/submission/handler_student.go`
- [X] T020 [US2] Implement `DELETE /actividades/:aid/submission` in `nextlab-api/internal/submission/handler_student.go`
- [X] T021 [US2] Wire `PUT /actividades/:aid/submission` and `DELETE /actividades/:aid/submission` in `nextlab-api/internal/submission/module.go`

**Checkpoint**: US2 fully testable — follow steps 7 from `quickstart.md`. Verify old MinIO object is removed after replacement. Verify window-expiry enforcement (create activity with deadline 1 minute out, submit, wait, try to replace → 403).

---

## Phase 5: User Story 3 — Docente Reviews, Downloads & Configures (Priority: P3)

**Goal**: A docente (or co-docente with an active grant) can list all submissions for an activity and download individual files via API-proxied streaming. Docente can toggle `late_submission_allowed` on any of their activities; every change is recorded in an immutable audit log.

**Independent Test**: Three students submit for one activity (one late, one on time, one replaced). Docente `GET .../submissions` returns 3 records with correct is_late, filename, file_size_bytes. Docente downloads each file — content matches upload. Co-docente with grant accesses same endpoint → 200. Co-docente without grant → 403. Docente toggles `late_submission_allowed=false` → audit log shows 1 entry. Late student submission rejected with 422.

### Implementation

- [X] T022 [P] Implement role-scoped list queries in `nextlab-api/internal/submission/repository.go`
- [X] T023 [P] Implement download resolution queries in `nextlab-api/internal/submission/repository.go`
- [X] T024 [US3] Implement `StreamFile` in `nextlab-api/internal/submission/service.go`
- [X] T025 [US3] Implement `GET /cursos/:cid/actividades/:aid/submissions` in `nextlab-api/internal/submission/handler_docente.go`
- [X] T026 [US3] Implement `GET /submissions/:sid/download` in `nextlab-api/internal/submission/handler_docente.go`
- [X] T027 [P] [US3] Implement config-change queries: `UpdateLateSubmissionFlag` in `academic/repository.go`; `GetConfigChanges` in `submission/repository.go`
- [X] T028 [US3] Extend `UpdateActividadHandler` in `nextlab-api/internal/academic/handler_actividad.go` to handle `LateSubmissionAllowed` toggle with atomic audit log
- [X] T029 [US3] Extend `PUT /cursos/:cid/actividades/:aid` — `late_submission_allowed` now included in response (repo scans updated)
- [X] T030 [US3] `GET /cursos/:cid/actividades/:aid/config-changes` wired in `submission/module.go`; docente/co_docente submission routes wired

**Checkpoint**: US3 fully testable — follow steps 5–6 and 8–9 from `quickstart.md`. Verify file content integrity via md5sum comparison. Verify config-changes audit log entry on toggle.

---

## Phase 6: User Story 4 — Student Downloads Own File (Priority: P4)

**Goal**: A student can download the file they submitted for any activity they are enrolled in. Cross-student access is denied.

**Independent Test**: Student A downloads their own file → 200 with correct content. Student A attempts to download with Student B's submission URL structure → 403.

### Implementation

- [X] T031 [US4] Implement `GET /actividades/:aid/submission/download` in `nextlab-api/internal/submission/handler_student.go`
- [X] T032 [US4] Wire `GET /actividades/:aid/submission/download` in `nextlab-api/internal/submission/module.go`

**Checkpoint**: US4 fully testable — steps 4 and 9 from `quickstart.md` (download own file; cross-student 403 attempt).

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: End-to-end validation, error handling audit, and log completeness.

- [ ] T033 [P] Run full `quickstart.md` validation checklist top-to-bottom: upload → list → download → replace → expire → late-block → config-change log → cross-student 403 → MinIO orphan check (inject DB error after PutObject and verify RemoveObject is called and no submission row exists)
- [ ] T034 [P] Audit error responses across all M4 handlers: verify MinIO unavailability returns 503 (not 500); verify all role violations return 403 with `"error"` key in JSON body; verify window-expiry violations return 403 with `"submission_locked"` error; verify late-block returns 422 with `"late_submissions_disabled"`; fix any missing or incorrect mappings in `nextlab-api/internal/submission/handler_student.go` and `handler_docente.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (go.mod + config) — **BLOCKS all user stories**
- **US1 (Phase 3)**: Depends on Phase 2 — no dependency on US2/US3/US4
- **US2 (Phase 4)**: Depends on US1 (needs existing submission to replace/delete)
- **US3 (Phase 5)**: Depends on Phase 2; T027–T030 depend on M3 academic module (already exists); independent of US2/US4
- **US4 (Phase 6)**: Depends on US1 (needs an existing submission to download); independent of US2/US3
- **Polish (Phase 7)**: Depends on all stories complete

### User Story Dependencies

- **US1 (P1)**: Only depends on Foundational phase — MVP deliverable
- **US2 (P2)**: Depends on US1 (edit window applies to an existing submission)
- **US3 (P3)**: Depends on Foundational only; independent of US2 and US4
- **US4 (P4)**: Depends on US1 (needs a submission to download); independent of US2 and US3

### Within Each Phase

- `[P]`-marked tasks within a phase touch different files and can run in parallel
- Repository tasks precede service tasks; service tasks precede handler tasks
- Module wiring (route registration) is always last within a phase

### Parallel Opportunities

- T003 (env example), T006 (submission model), T007 (academic model extend), T008 (module skeleton) — all Phase 2 parallel
- T010 + T011 (two independent repo blocks for US1)
- T016 (mutation queries) is independent of T017/T018 service implementation
- T022 + T023 (two independent repo blocks for US3)
- T024 (stream service) parallel with T022/T023 (different file)
- T027 (config-change queries) parallel with T022–T024 (different file and module)
- T033 + T034 (polish tasks, different focus areas)

---

## Parallel Example: User Story 1

```bash
# Can run in parallel (different files):
Task T010: "Implement validation queries in internal/submission/repository.go"
Task T011: "Implement write queries in internal/submission/repository.go"

# Then sequentially (service depends on both repo functions):
Task T012: "Implement UploadFile in internal/submission/service.go"
Task T013: "Implement POST handler in internal/submission/handler_student.go"
Task T014: "Implement GET handler in internal/submission/handler_student.go"

# Then wire (depends on both handlers):
Task T015: "Wire routes in internal/submission/module.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001–T003)
2. Complete Phase 2: Foundational — **CRITICAL** (T004–T009)
3. Complete Phase 3: US1 (T010–T015)
4. **STOP and VALIDATE**: Student can upload, system correctly marks late flag, MinIO object exists, docente can see `has_submissions=true` on the actividad
5. Demo / integrate with M3 before proceeding

### Incremental Delivery

1. Phases 1–2 → Foundation ready
2. Phase 3 → Students can submit → **MVP**
3. Phase 4 → Students can edit within the window
4. Phase 5 → Docentes/co-docentes can review and download; late-submission control active
5. Phase 6 → Students can verify their own submission by downloading
6. Phase 7 → End-to-end validation complete → M4 milestone deliverable

---

## Notes

- `[P]` = can run in parallel with other `[P]` tasks in the same phase (different files)
- `[USn]` = maps task to specific user story for traceability
- `is_locked` is never stored in the DB — always derived as `(edit_window_expires_at IS NULL) OR (NOW() > edit_window_expires_at)` in service/handler
- MinIO `RemoveObject` after failed DB operations is best-effort; log failures but do not block the error response to the client
- All DB queries that access submission data MUST include the role-scoping join (`docente_id` or `codocente_grants` join) — never filter in Go after a full-table fetch (Constitution Principle IV)
- The `has_submissions` flag on `actividades` is flipped in the same transaction as the first `InsertSubmission` call — M3's title/type edit guard already depends on this flag
