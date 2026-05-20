# Implementation Plan: M3 — Academic Management

**Branch**: `003-academic-management` | **Date**: 2026-05-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-academic-management/spec.md`

---

## Summary

Implement the academic structure layer for nextlab-api: a degree-program and subject catalog managed by admins, courses and activities managed by docentes, automatic student enrollment via Google OAuth (identity verified by Google; carrera + semester sourced from admin-provisioned student profiles), grupo-based activity organization, per-activity co-teaching grants, and a student-facing view of their enrolled courses. All data access is enforced at the DB query level per Constitution Principle IV.

**Technical approach**: New `internal/academic/` module following the M2 `RegisterRoutes` pattern · `golang.org/x/oauth2` + `google.golang.org/api/idtoken` for Google OAuth student login · admin-provisioned `student_profiles` table as enrollment source of truth · auto-enrollment sync on every student login · `has_submissions` boolean on `actividades` as the M3-side activity-lock guard (M4 will flip it to `true` on first upload) · migration `000003_academic` covering 9 new tables + one ALTER on `users`

---

## Technical Context

**Language/Version**: Go 1.22

**Primary Dependencies**:
- `gofiber/fiber/v2` — HTTP framework (locked by constitution)
- `golang.org/x/oauth2` — Google OAuth 2.0 client *(new in M3)*
- `google.golang.org/api/idtoken` — Google ID token verification *(new in M3)*
- `jackc/pgx/v5/pgxpool` — PostgreSQL pool (from M2)
- `redis/go-redis/v9` — Redis client (from M1; used for OAuth state token TTL)
- `google/uuid` — UUID generation (already in go.sum)
- `golang.org/x/crypto/bcrypt` — unchanged from M2

**Storage**:
- PostgreSQL 16: 9 new tables + `ALTER TABLE users` (see data-model.md)
- Redis 7: OAuth CSRF state tokens (60-second TTL)

**Testing**: `go test` stdlib

**Target Platform**: Linux/AMD64 (Docker Compose, same as M1/M2)

**Performance Goals**:
- Academic CRUD endpoints ≤ 100ms p95 (indexed single-table queries)
- Google OAuth callback (including profile lookup + enrollment sync) ≤ 2s
- Enrollment sync per login: ≤ 500ms for up to 20 matching cursos

**Constraints**:
- All docente-scoped queries MUST include `WHERE docente_id = $N` at the DB layer — never filter in Go after a full-table fetch
- `has_submissions` on `actividades` is initialized to `false` by M3 and respected for edit/delete guards; M4 is responsible for setting it to `true`
- Google OAuth is for **students only**; docentes and admins continue to use email/password (M2)
- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and `GOOGLE_REDIRECT_URL` must be validated at startup with `os.Exit(1)` if absent
- Student carrera + semester come from admin-provisioned `student_profiles` table, **not** from Google Workspace attributes — this avoids a hard dependency on Unifranz IT admin configuration

**Scale/Scope**: ~800–1500 students, ~50–100 docentes, 2 carreras (Unifranz)

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Immutable Evidence | ✅ N/A | No append-only audit tables in M3. `inscripciones` and `codocente_grants` use `revoked_at` timestamps (never deleted) — consistent with spirit of Principle I. |
| II. Async Analysis Isolation | ✅ N/A | No analysis jobs in M3. |
| III. Human Authority Over AI | ✅ N/A | No AI calls in M3. |
| IV. Strict Role-Based Data Isolation | ✅ **PRIMARY** | Every DB query is role-scoped: docente sees only their cursos; estudiante sees only enrolled cursos; co_docente sees only granted actividades; admin has global scope. Co-teaching boundary enforced via `codocente_grants` join. |
| V. Dual RAG Independence | ✅ N/A | No RAG in M3. |
| VI. Non-Invasive Auditing | ✅ Partially exercised | FR-005: `examen` and `defensa_oral` activities get `trace_enabled = true` (computed from type). This is the flag M5 will use to gate trace capture — establishing it correctly here is critical. |
| VII. Submission Always Accepted | ✅ N/A | No submissions in M3. |

**Result: PASS** — No violations. Principle IV is the primary driver; Principle VI is exercised via the `trace_enabled` flag on activities.

---

## Project Structure

### Documentation (this feature)

```text
specs/003-academic-management/
├── plan.md                  ← this file
├── research.md              ← Phase 0 output
├── data-model.md            ← Phase 1 output
├── quickstart.md            ← Phase 1 output
├── contracts/
│   └── academic.md          ← Phase 1 output
└── tasks.md                 ← Phase 2 output (/speckit-tasks)
```

### Source Code Changes

```text
nextlab-api/
├── cmd/
│   └── api/
│       └── main.go                  # Add academic.RegisterRoutes; validate Google config at startup
├── internal/
│   ├── auth/
│   │   ├── google.go                # NEW: GoogleLoginHandler, GoogleCallbackHandler, enrollment sync call
│   │   └── module.go                # Add GET /auth/google + GET /auth/google/callback routes
│   ├── academic/                    # NEW module (zero changes to other internal/ packages)
│   │   ├── module.go                # RegisterRoutes(app, db, rdb, cfg)
│   │   ├── model.go                 # Carrera, Materia, Curso, Grupo, Actividad, Inscripcion,
│   │   │                            #   GrupoAsignacion, CodocenteGrant, StudentProfile structs
│   │   ├── repository.go            # All DB queries, each scoped per role
│   │   ├── service.go               # EnrollmentSync, ActivityLockGuard, GrantValidator
│   │   ├── handler_admin.go         # POST/GET/PUT/DELETE /admin/carreras|materias|student-profiles
│   │   ├── handler_curso.go         # Curso CRUD + GET /cursos/:id/students (docente roster)
│   │   ├── handler_grupo.go         # Grupo CRUD + group assignment endpoints
│   │   ├── handler_actividad.go     # Actividad CRUD (docente) + GET /me/cursos/:id/actividades (student)
│   │   └── handler_grant.go         # Grant create/revoke (docente) + co_docente actividad view
│   └── config/
│       └── config.go                # Add GoogleClientID, GoogleClientSecret, GoogleRedirectURL
├── migrations/
│   ├── 000003_academic.up.sql       # 9 new tables + ALTER TABLE users
│   └── 000003_academic.down.sql     # DROP in reverse order + REVERT users alteration
├── go.mod                           # Add golang.org/x/oauth2, google.golang.org/api
├── go.sum                           # Updated by go mod tidy
└── .env.example                     # Add GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URL
```

**Structure Decision**: All academic logic lives in `internal/academic/`. Google OAuth handlers extend `internal/auth/` with a new `google.go` file, keeping the auth module as the single owner of all authentication flows. No existing files in other packages are modified.

---

## Phase 0: Research Summary

See [research.md](research.md) for the full decision log. Key decisions:

| Topic | Decision |
|---|---|
| Google OAuth library | `golang.org/x/oauth2` + `google.golang.org/api/idtoken` — maintained by Google, standard in Go ecosystem |
| Student identity source | Admin-provisioned `student_profiles` table (email → carrera_id + semester). Google verifies identity only; enrollment data comes from the admin table. |
| Enrollment trigger | On **every** student Google login: query active cursos matching student's carrera + semester; create missing inscripciones; revoke (with `revocation_reason = 'semester_advance'`) inscripciones whose curso no longer matches. |
| Activity submission guard | `has_submissions BOOLEAN DEFAULT false` on `actividades`. M3 initializes it; M4 sets it to `true` on first file upload. M3 service checks it before allowing title/type edits or deletion. |
| Users table extension | `ALTER TABLE users`: make `password_hash` nullable (students have no password); add `google_sub TEXT UNIQUE` (nullable, indexed). Done in migration 000003. |
| Google CSRF state | Short-lived state token (32 random bytes, base64) stored in Redis with 60s TTL. Validated on callback before code exchange to prevent CSRF. |
| New config fields | `GOOGLE_CLIENT_ID` (required), `GOOGLE_CLIENT_SECRET` (required), `GOOGLE_REDIRECT_URL` (required). Startup exits if any are missing or empty. |
| `trace_enabled` | PostgreSQL stored generated column: `GENERATED ALWAYS AS (tipo IN ('examen', 'defensa_oral')) STORED`. Cannot be overridden. |
| Module pattern | Same as M2: `RegisterRoutes(app, db, rdb, cfg)` called from `cmd/api/main.go`. |
| Student group uniqueness | `grupo_asignaciones` includes a denormalized `curso_id` column with `UNIQUE(user_id, curso_id)` — enforces one-group-per-student-per-course at the DB level. |

---

## Phase 1: Design Summary

### Data Model

See [data-model.md](data-model.md). Migration `000003_academic` creates 9 tables and alters 1:

| Table | Purpose |
|---|---|
| `carreras` | Degree program catalog (admin-managed) |
| `materias` | Subject catalog per carrera + semester (admin-managed) |
| `student_profiles` | Admin-provisioned mapping of student email → carrera + semester |
| `cursos` | Course instances (docente-managed; FK → materia + users) |
| `grupos` | Student subgroups within a curso (docente-managed) |
| `actividades` | Assignments and evaluations within a curso (docente-managed) |
| `inscripciones` | System-generated course memberships (student ↔ curso) |
| `grupo_asignaciones` | Docente-assigned group placements (student ↔ grupo, one per curso) |
| `codocente_grants` | Per-activity co-teaching grants (docente → co_docente) |
| `ALTER TABLE users` | Add `google_sub` (nullable, unique); make `password_hash` nullable |

### API Contract

See [contracts/academic.md](contracts/academic.md). Full endpoint list:

| Group | Method + Path | Role |
|---|---|---|
| **Google OAuth** | `GET /auth/google` | public |
| | `GET /auth/google/callback` | public |
| **Student profiles** | `POST /admin/student-profiles` | admin |
| | `GET /admin/student-profiles` | admin |
| | `PUT /admin/student-profiles/:id` | admin |
| **Catalog** | `POST/GET /admin/carreras` | admin |
| | `PUT/DELETE /admin/carreras/:id` | admin |
| | `POST/GET /admin/materias` | admin |
| | `PUT/DELETE /admin/materias/:id` | admin |
| **Cursos** | `POST/GET /cursos` | docente |
| | `GET/PUT/DELETE /cursos/:id` | docente |
| | `GET /cursos/:id/students` | docente (roster) |
| **Grupos** | `POST/GET /cursos/:id/grupos` | docente |
| | `PUT/DELETE /cursos/:id/grupos/:gid` | docente |
| | `POST /cursos/:id/grupos/:gid/students` | docente (assign) |
| | `DELETE /cursos/:id/grupos/:gid/students/:uid` | docente (remove) |
| **Actividades** | `POST/GET /cursos/:id/actividades` | docente |
| | `GET/PUT/DELETE /cursos/:id/actividades/:aid` | docente |
| **Co-docente grants** | `POST /actividades/:aid/grants` | docente |
| | `DELETE /actividades/:aid/grants/:uid` | docente |
| | `GET /grants/actividades/:aid` | co_docente |
| **Student view** | `GET /me/cursos` | estudiante |
| | `GET /me/cursos/:id/actividades` | estudiante |

### Quickstart

See [quickstart.md](quickstart.md). After `docker compose up --build`:
1. Set `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL` in `.env`
2. Admin seeds a carrera + materia via `POST /admin/carreras` and `POST /admin/materias`
3. Admin provisions a student profile: `POST /admin/student-profiles` with the student's `@unifranz.edu.bo` email, carrera, and semester
4. Docente (seeded from M2) creates a curso: `POST /cursos`
5. Test student visits `GET /auth/google` → Google consent → callback → receives JWT
6. Student calls `GET /me/cursos` → sees the auto-enrolled curso
7. Docente creates a grupo and assigns the student; creates an actividad of type `examen`
8. Verify `trace_enabled = true` on the actividad response

---

## Key Implementation Notes

### Google OAuth callback flow

```
GET /auth/google
  → generate state: 32 random bytes, base64url-encoded
  → store state in Redis: SET "oauth_state:{state}" "1" EX 60
  → redirect to Google OAuth URL (scope: openid email profile)

GET /auth/google/callback?code=...&state=...
  → validate state: GET "oauth_state:{state}" from Redis; must be non-nil; DEL key
  → exchange code for tokens via Google token endpoint
  → verify ID token with google.golang.org/api/idtoken (validates aud + sig)
  → extract email from ID token claims
  → look up student_profiles by email → get carrera_id + semester
    (if not found: return 403 "account not provisioned — contact admin")
  → upsert users row: INSERT ... ON CONFLICT (email) DO UPDATE SET google_sub = ...
  → call EnrollmentSync(ctx, userID, carreraID, semester)
  → call issueTokenPair (same function as M2 email/password login)
  → return LoginResponse{access_token, refresh_token, ...}
```

### EnrollmentSync

```
EnrollmentSync(ctx, userID, carreraID, semester):
  activeCursos = SELECT cursos WHERE materia.carrera_id = carreraID
                   AND materia.semester = semester AND cursos.is_active = true
  existingInscripciones = SELECT * FROM inscripciones WHERE user_id = userID

  for each activeCurso not in existingInscripciones:
    INSERT INTO inscripciones (user_id, curso_id, carrera_id, semester) VALUES (...)

  for each existingInscripcion where curso not in activeCursos AND is_active = true:
    UPDATE inscripciones SET is_active = false, revoked_at = now(),
      revocation_reason = 'semester_advance' WHERE id = ...
```

### Role-scoped repository pattern

Every query that returns docente-owned resources MUST include the `docente_id` filter in the SQL itself:

```sql
-- GetCursosByDocente (repository.go)
SELECT id, materia_id, semester_label, section, is_active, created_at
FROM cursos
WHERE docente_id = $1 AND is_active = true
ORDER BY created_at DESC

-- GetActividadForCodocente (repository.go)
SELECT a.* FROM actividades a
JOIN codocente_grants g ON g.actividad_id = a.id
WHERE a.id = $1
  AND g.codocente_id = $2
  AND g.is_active = true
  AND g.revoked_at IS NULL
```

### Activity edit/delete guard (service.go)

```go
func (s *Service) GuardActivityMutation(a *Actividad, req *UpdateActividadRequest, isDelete bool) error {
    if !a.HasSubmissions {
        return nil // no submissions yet — all edits allowed
    }
    if isDelete {
        return ErrActivityHasSubmissions // 409
    }
    if req.Title != "" || req.Tipo != "" {
        return ErrActivityFieldLocked // 422 — title and type locked after submissions
    }
    return nil
}
```

### Startup config validation (config.go)

```go
if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" || cfg.GoogleRedirectURL == "" {
    slog.Error("Google OAuth config missing", "required", "GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URL")
    os.Exit(1)
}
```
