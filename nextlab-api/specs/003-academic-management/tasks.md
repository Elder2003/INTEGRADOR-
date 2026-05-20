# Tasks: M3 — Academic Management

**Input**: Design documents from `specs/003-academic-management/`

**Prerequisites**: plan.md ✅ | spec.md ✅ | research.md ✅ | data-model.md ✅ | contracts/academic.md ✅ | google-oauth-integration.md ✅

**Organization**: Tasks are grouped by user story. Each phase is independently deliverable and testable.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add dependencies, config fields, Google OAuth future-proofing doc, and directory structure before any feature work begins.

- [X] T001 Add `golang.org/x/oauth2`, `golang.org/x/oauth2/google`, and `google.golang.org/api/idtoken` to `nextlab-api/go.mod` via `go get` and run `go mod tidy`
- [X] T002 Add `GoogleClientID string`, `GoogleClientSecret string`, `GoogleRedirectURL string` fields to `internal/config/config.go`; add startup validation (if any field is empty → `slog.Error` + `os.Exit(1)`) matching the JWT_SECRET pattern from M2
- [X] T003 [P] Add `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and `GOOGLE_REDIRECT_URL` entries with comments to `nextlab-api/.env.example`
- [X] T004 [P] Verify `specs/003-academic-management/google-oauth-integration.md` exists and is complete — this is the canonical reference if the auth provider changes in the future (see that file for replacement instructions)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Database schema, Go model structs, and module skeleton — everything all user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T005 Write `nextlab-api/migrations/000003_academic.up.sql` with all tables and alterations per `data-model.md`: `ALTER TABLE users` (nullable `password_hash`, add `google_sub`); `CREATE TABLE carreras`, `materias`, `student_profiles`, `cursos`, `grupos`, `actividades` (including `actividad_tipo` enum and `trace_enabled` stored generated column), `inscripciones`, `grupo_asignaciones` (with denormalized `curso_id` and `UNIQUE(user_id, curso_id)`), `codocente_grants`; all indexes
- [X] T006 Write `nextlab-api/migrations/000003_academic.down.sql` dropping all M3 tables in reverse dependency order and reverting the `users` alteration per `data-model.md`
- [X] T007 Create `nextlab-api/internal/academic/model.go` with Go structs for all entities: `Carrera`, `Materia`, `StudentProfile`, `Curso`, `Grupo`, `Actividad`, `Inscripcion`, `GrupoAsignacion`, `CodocenteGrant`; plus request/response types (`CreateCursoRequest`, `UpdateActividadRequest`, `LoginResponse` reuse from auth, etc.)
- [X] T008 [P] Create `nextlab-api/internal/academic/module.go` with `RegisterRoutes(app *fiber.App, db *pgxpool.Pool, rdb *redis.Client, cfg *config.Config, authMW fiber.Handler, roleGuard func(...) fiber.Handler)` — full route wiring for all phases
- [X] T009 [P] Add `academic.RegisterRoutes(app, pgPool, rdb, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard)` call to `nextlab-api/cmd/api/main.go` after `auth.RegisterRoutes`

**Checkpoint**: Run `docker compose up --build` — server must start without error and migration must apply (`\dt` shows all M3 tables).

---

## Phase 3: User Story 1 — Docente Manages a Course (Priority: P1) 🎯 MVP

**Goal**: Admin can manage the subject catalog and student profiles. Docente can create cursos, grupos, and actividades scoped to their own account only.

**Independent Test**: Docente logs in, creates a curso linked to a seeded materia, adds a grupo, creates an `examen` actividad. Response includes `trace_enabled: true`. A second docente's `GET /cursos` returns an empty list — not the first docente's cursos.

### Implementation

- [X] T010 [P] [US1] Implement catalog DB queries in `nextlab-api/internal/academic/repository.go`: `GetCarreras`, `CreateCarrera`, `UpdateCarrera`, `DeactivateCarrera` (set `is_active=false`, reject if active materias exist), `GetMaterias` (filterable by `carrera_id` + `semester`), `CreateMateria`, `UpdateMateria`, `DeactivateMateria` (reject if active cursos exist), `GetActiveMateriasByCarreraSemester` (used by EnrollmentSync)
- [X] T011 [P] [US1] Implement student profile DB queries in `nextlab-api/internal/academic/repository.go`: `CreateStudentProfile`, `GetStudentProfileByEmail`, `UpdateStudentProfile`, `ListStudentProfiles` (filterable by `carrera_id` + `semester`)
- [X] T012 [US1] Implement admin handlers in `nextlab-api/internal/academic/handler_admin.go`: `POST/GET/PUT/DELETE /admin/carreras`, `POST/GET/PUT/DELETE /admin/materias`, `POST/GET/PUT /admin/student-profiles` — all guarded by `RoleGuard("admin")` per `contracts/academic.md`
- [X] T013 [P] [US1] Implement curso DB queries in `nextlab-api/internal/academic/repository.go`: `GetCursosByDocente` (WHERE `docente_id=$1 AND is_active=true`), `GetCursoByIDAndDocente` (ownership check), `CreateCurso` (duplicate check via UNIQUE constraint error), `UpdateCurso`, `DeactivateCurso`, `GetCursoRoster` (students with active inscripciones + grupo assignment), `CursoHasSubmissions` (checks `actividades.has_submissions=true` for any actividad in this curso)
- [X] T014 [US1] Implement `ActivityLockGuard` and curso-deletion guard in `nextlab-api/internal/academic/service.go`: `GuardActivityMutation(actividad, req, isDelete) error` — returns `ErrActivityHasSubmissions` (409) on delete if `has_submissions=true`, returns `ErrActivityFieldLocked` (422) if `title` or `tipo` change is attempted after submissions exist
- [X] T015 [US1] Implement curso handlers in `nextlab-api/internal/academic/handler_curso.go`: `POST /cursos`, `GET /cursos`, `GET /cursos/:id`, `PUT /cursos/:id`, `DELETE /cursos/:id` (soft, blocked if has_submissions), `GET /cursos/:id/students` — all guarded by `AuthMiddleware` + `RoleGuard("docente")` per `contracts/academic.md`
- [X] T016 [P] [US1] Implement grupo DB queries in `nextlab-api/internal/academic/repository.go`: `CreateGrupo` (unique name per curso), `GetGruposByCurso`, `UpdateGrupo`, `DeleteGrupo`, `AssignStudentToGrupo` (INSERT with denormalized `curso_id`; handles `UNIQUE(user_id, curso_id)` conflict as 409), `RemoveStudentFromGrupo`, `StudentEnrolledInCurso` (guard for assignment)
- [X] T017 [US1] Implement grupo handlers in `nextlab-api/internal/academic/handler_grupo.go`: `POST/GET /cursos/:id/grupos`, `PUT/DELETE /cursos/:id/grupos/:gid`, `POST /cursos/:id/grupos/:gid/students`, `DELETE /cursos/:id/grupos/:gid/students/:uid` — all guarded by `RoleGuard("docente")` per `contracts/academic.md`
- [X] T018 [P] [US1] Implement actividad DB queries in `nextlab-api/internal/academic/repository.go`: `CreateActividad`, `GetActividadesByCurso`, `GetActividadByIDAndCurso`, `UpdateActividad` (respects has_submissions lock — partial fields), `DeleteActividad` (blocked if has_submissions)
- [X] T019 [US1] Implement docente actividad handlers in `nextlab-api/internal/academic/handler_actividad.go`: `POST/GET /cursos/:id/actividades`, `GET/PUT/DELETE /cursos/:id/actividades/:aid` — calls `GuardActivityMutation` before mutating; returns `created_at` and `updated_at` per `contracts/academic.md`
- [X] T020 [US1] Wire all US1 routes in `nextlab-api/internal/academic/module.go`: `/admin/carreras`, `/admin/materias`, `/admin/student-profiles` (admin), `/cursos` and sub-routes (docente), `/cursos/:id/grupos` and sub-routes (docente), `/cursos/:id/actividades` and sub-routes (docente)

**Checkpoint**: US1 fully testable — run steps 2–5 and 8 from `quickstart.md`. Verify docente isolation and `trace_enabled=true` on examen actividades.

---

## Phase 4: User Story 2 — Student Course Access and Group Assignment (Priority: P2)

**Goal**: Students authenticate via Google OAuth, are auto-enrolled in matching cursos, and can view their courses and activities. Docente can see the pre-populated roster and assign students to grupos.

**Independent Test**: Admin provisions a student profile. Student visits `GET /auth/google`, completes consent, receives a JWT. `GET /me/cursos` returns their auto-enrolled curso. Student cannot access a curso they are not enrolled in.

### Implementation

- [X] T021 [P] [US2] Implement inscripcion DB queries in `nextlab-api/internal/academic/repository.go`: `GetActiveInscripcionesByUser`, `GetActiveInscripcionesByCurso`, `CreateInscripcion`, `RevokeInscripcion` (sets `revoked_at`, `revocation_reason`, `is_active=false` — never hard-deletes)
- [X] T022 [US2] Implement `EnrollmentSync(ctx, db, userID, carreraID, semester)` in `nextlab-api/internal/academic/service.go`: query active cursos for `carrera+semester`; insert inscripciones for cursos not yet enrolled; revoke inscripciones for cursos no longer matching (reason=`"semester_advance"`)
- [X] T023 [US2] Create `nextlab-api/internal/auth/google.go`: `GoogleLoginHandler` (generate CSRF state → store in Redis 60s TTL → redirect to Google OAuth URL with `scope: openid email profile`); `GoogleCallbackHandler` (validate state via Redis GET+DEL → exchange code → verify ID token with `idtoken.Validate` → extract email → `GetStudentProfileByEmail` → 403 if not found → `UpsertUser` with `google_sub` → call `academic.EnrollmentSync` → call `issueTokenPair` → return `LoginResponse`)
- [X] T024 [US2] Register Google OAuth routes in `nextlab-api/internal/auth/module.go`: `GET /auth/google` → `GoogleLoginHandler`; `GET /auth/google/callback` → `GoogleCallbackHandler` (no auth middleware — these are public endpoints)
- [X] T025 [P] [US2] Implement student-facing DB queries in `nextlab-api/internal/academic/repository.go`: `GetCursosByStudent` (JOIN inscripciones WHERE `user_id=$1 AND is_active=true`; includes grupo assignment if any), `GetActividadesByCursoForStudent` (excludes `has_submissions` field; returns `created_at` as `posted_at` and `updated_at` as `last_edited_at`)
- [X] T026 [US2] Implement student-facing handlers in `nextlab-api/internal/academic/handler_actividad.go`: `GET /me/cursos` → `GetCursosByStudent`; `GET /me/cursos/:id/actividades` → verifies active inscripcion for requesting student before returning actividades (403 if not enrolled)
- [X] T027 [US2] Wire student routes in `nextlab-api/internal/academic/module.go`: `GET /me/cursos`, `GET /me/cursos/:id/actividades` — guarded by `AuthMiddleware` + `RoleGuard("estudiante")`

**Checkpoint**: US2 fully testable — run steps 4–7 from `quickstart.md`. Verify auto-enrollment on Google login and student isolation (403 on foreign curso).

---

## Phase 5: User Story 3 — Co-Teaching Grant for a Specific Activity (Priority: P3)

**Goal**: Docente grants a co_docente user access to exactly one actividad. Co_docente can view that actividad only — all other resources in the same curso return 403.

**Independent Test**: Docente grants co_docente access to actividad X. Co_docente calls `GET /grants/actividades/X` → 200. Co_docente calls `GET /grants/actividades/Y` (different actividad, same curso) → 403. Docente revokes grant → co_docente call → 403.

### Implementation

- [X] T028 [P] [US3] Implement co-docente grant DB queries in `nextlab-api/internal/academic/repository.go`: `CreateGrant` (validates target user has `role="co_docente"`; handles `UNIQUE(actividad_id, codocente_id)` conflict as 409 if already active), `GetActiveGrantByActividadAndCodocente`, `RevokeGrant` (sets `revoked_at`, `revoked_by`, `is_active=false` — never hard-deletes), `ListGrantsByActividad`
- [X] T029 [US3] Implement `ValidateCodocenteAccess(ctx, db, actividadID, codocenteID) error` in `nextlab-api/internal/academic/service.go`: returns `ErrGrantNotFound` (403) if no active grant exists for the requesting co_docente on the given actividad
- [X] T030 [US3] Implement grant handlers in `nextlab-api/internal/academic/handler_grant.go`: `POST /actividades/:aid/grants` (docente — validates ownership of actividad's curso before granting); `DELETE /actividades/:aid/grants/:uid` (docente — validates ownership); `GET /grants/actividades/:aid` (co_docente — calls `ValidateCodocenteAccess` then returns actividad details) per `contracts/academic.md`
- [X] T031 [US3] Wire grant routes in `nextlab-api/internal/academic/module.go`: `POST/DELETE /actividades/:aid/grants` with `RoleGuard("docente")`; `GET /grants/actividades/:aid` with `RoleGuard("co_docente")`

**Checkpoint**: US3 fully testable — grant, verify access, verify isolation across other actividades, revoke, verify denial.

---

## Phase 6: User Story 4 — Admin Manages the Subject Catalog (Priority: P4)

**Note**: US4 catalog CRUD (carreras, materias, student_profiles) was implemented in Phase 3 (T010–T012, T020). This phase verifies non-admin rejection and catalog deactivation behavior.

**Goal**: Confirm admin-only enforcement on all catalog endpoints; confirm deactivated materias are unavailable for new cursos.

**Independent Test**: Non-admin (docente, estudiante) attempts `POST /admin/carreras` → 403. Admin deactivates a materia → docente cannot create a nuevo curso referencing it → 404.

### Implementation

- [X] T032 [US4] Verify `RoleGuard("admin")` is applied to all `/admin/*` routes in `nextlab-api/internal/academic/module.go`; confirm `DeactivateMateria` service returns an error (409) if the materia has active cursos referencing it; confirm `GetActiveMateriasByCarreraSemester` excludes `is_active=false` materias

**Checkpoint**: Run steps 2–3 and 10 (non-admin check) from `quickstart.md`.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Consistency, observability, and final validation across all stories.

- [X] T033 [P] Audit all DB queries in `nextlab-api/internal/academic/repository.go` — confirm every docente-scoped SELECT includes `WHERE docente_id = $N` or equivalent ownership JOIN in the SQL itself (not in Go post-fetch)
- [X] T034 [P] Add structured slog event logging to all handlers: `"event"` key with values like `"curso_created"`, `"actividad_updated"`, `"grant_issued"`, `"grant_revoked"`, `"enrollment_synced"` — consistent with the M2 auth event naming pattern in `internal/auth/service.go`
- [ ] T035 Run the full `quickstart.md` end-to-end validation (steps 1–10) in a clean `docker compose down -v && docker compose up --build` environment; confirm all role-isolation checks pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 — **BLOCKS all user stories**
- **Phase 3 (US1)**: Depends on Phase 2
- **Phase 4 (US2)**: Depends on Phase 2; also depends on Phase 3 (T020 must wire routes before T024 can extend the auth module's route list, and `EnrollmentSync` uses `GetActiveMateriasByCarreraSemester` from T010)
- **Phase 5 (US3)**: Depends on Phase 2; depends on Phase 3 (actividades must exist before grants can reference them)
- **Phase 6 (US4)**: Depends on Phase 3 (catalog code exists)
- **Phase 7 (Polish)**: Depends on Phases 3–6

### Within Each Story

- DB queries (repository) → service logic → handlers → route wiring (module)
- Parallel tasks (`[P]`) within the same phase can be implemented simultaneously across different files

### Parallel Opportunities

```bash
# Phase 1: All 4 tasks can run in parallel
T001  T002  T003  T004

# Phase 2: T007, T008, T009 can run in parallel (after T005+T006 for the migration)
T005 → T006 → T007 | T008 | T009

# Phase 3: Parallel within the phase
T010 | T011  →  T012
T013         →  T014  →  T015
T016         →  T017
T018         →  T019
All above    →  T020

# Phase 4: T021, T025 can run in parallel; T022 depends on T021
T021  →  T022  →  T023  →  T024
T025                     →  T026  →  T027

# Phase 5: T028 first, then T029, then T030, then T031
T028  →  T029  →  T030  →  T031
```

---

## Implementation Strategy

### MVP First (Phase 1 + 2 + 3 only)

1. Complete Phase 1: Setup (dependencies + config)
2. Complete Phase 2: Foundational (migration + models + module skeleton)
3. Complete Phase 3: US1 (admin catalog + docente cursos/grupos/actividades)
4. **STOP and VALIDATE**: Admin seeds catalog; docente creates curso + actividad; verify isolation; verify `trace_enabled`
5. Deploy / demo if ready

### Incremental Delivery

| After phase | What works |
|---|---|
| Phase 3 | Admin manages catalog; docente manages full academic structure |
| Phase 4 | Students log in via Google OAuth; auto-enrollment; student course view |
| Phase 5 | Co-docentes can be granted access to specific activities |
| Phase 6 | Confirmed admin-only enforcement on catalog |
| Phase 7 | Fully validated, observable, production-ready M3 |

---

## Notes

- `[P]` tasks touch different files with no cross-task dependencies — safe to parallelize
- `[USN]` label maps each task to a specific user story for traceability
- The Google OAuth integration is documented in `specs/003-academic-management/google-oauth-integration.md` — read that file before modifying `internal/auth/google.go` or if switching to a different identity provider
- `trace_enabled` is a stored generated column — it cannot be set in Go code; verify after INSERT that the DB computes it correctly
- `has_submissions` starts as `false` in M3; M4 (file storage) will flip it — the service guard (`GuardActivityMutation`) must be in place before M4 to avoid a later refactor
- Commits should happen after each checkpoint (T020, T027, T031, T035)
