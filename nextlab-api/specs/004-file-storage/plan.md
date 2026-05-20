# Implementation Plan: M4 — File Storage

**Branch**: `004-file-storage` | **Date**: 2026-05-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/004-file-storage/spec.md`

---

## Summary

Implement the file submission layer for nextlab-api: students upload academic files for activities (stored in MinIO), a 30-minute editing window allows on-time submissions to be replaced or deleted, late submissions are controlled per-activity by a docente-configurable flag (audited), and docentes/co-docentes can list and download submissions through the API. All file access is API-proxied — no storage URLs are ever exposed.

**Technical approach**: New `internal/submission/` module · `minio/minio-go/v7` SDK · multipart upload streamed directly to MinIO · API-proxied download via `fiber.SendStream` · edit window lockedness derived at read time from `edit_window_expires_at` (no background job) · UNIQUE constraint on `(student_id, actividad_id)` prevents duplicate submissions · `internal/academic/` extended minimally to accept `late_submission_allowed` in the actividad PUT and write to the new `actividad_config_changes` audit table · migration `000004_file_storage` adds 2 tables and 1 ALTER

---

## Technical Context

**Language/Version**: Go 1.22

**Primary Dependencies**:
- `gofiber/fiber/v2` — HTTP framework (locked by constitution)
- `minio/minio-go/v7` — MinIO/S3 SDK *(new in M4)*
- `jackc/pgx/v5/pgxpool` — PostgreSQL pool (from M1)
- `redis/go-redis/v9` — Redis client (from M1; no new usage in M4)
- `google/uuid` — UUID generation (already in go.sum)
- `golang.org/x/oauth2` — unchanged from M3

**Storage**:
- PostgreSQL 16: 2 new tables (`submissions`, `actividad_config_changes`) + `ALTER TABLE actividades`
- MinIO (S3-compatible): single bucket `nextlab`, objects under `submissions/` prefix

**Testing**: `go test` stdlib

**Target Platform**: Linux/AMD64 (Docker Compose, same as M1–M3)

**Performance Goals**:
- Upload confirmation (10 MB file) ≤ 10 seconds end-to-end (SC-001)
- Submission list endpoint ≤ 150ms p95 (indexed query on `actividad_id`)
- Download stream initiation ≤ 500ms before first bytes delivered

**Constraints**:
- Max file size: 50 MB (checked before streaming to MinIO; return 400 if exceeded)
- All submission queries scoped by role: student sees only their own row; docente sees only activities in their courses; co_docente sees only activities with an active grant
- `submitted_at` is set server-side at `PutObject` completion — never from the client clock
- No submission record created if MinIO upload fails (atomicity per FR-014)
- `is_locked` is derived at runtime: `edit_window_expires_at IS NULL OR NOW() > edit_window_expires_at`
- `late_submission_allowed` changes MUST be written in the same DB transaction as the `actividad_config_changes` INSERT

**Scale/Scope**: ~800–1500 students, ~50–100 docentes; 50 MB max file; expected peak: handful of concurrent uploads per activity deadline

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Immutable Evidence | ✅ Exercised | `actividad_config_changes` is append-only (no UPDATE/DELETE). `submitted_at` is set at storage-confirmation time and never altered. Within the edit window, `submissions` rows MAY be updated (replacement) or deleted (student removes); after the window closes they become immutable by application policy. This is consistent with Principle I — the edit window is the student's right to finalize their submission before it becomes an academic record. |
| II. Async Analysis Isolation | ✅ N/A | No async analysis in M4. No Redis Streams jobs enqueued. M8 (detection engine) will consume `submissions` rows; M4 only produces them. |
| III. Human Authority Over AI | ✅ N/A | No AI calls in M4. |
| IV. Strict Role-Based Data Isolation | ✅ **PRIMARY** | Student access scoped to `WHERE student_id = $N`. Docente access requires `actividades.curso_id → cursos.docente_id = $N`. Co-docente access requires `codocente_grants.codocente_id = $N AND is_active = true`. All scoping applied at the DB query level, not in Go after a full-table fetch. |
| V. Dual RAG Independence | ✅ N/A | No RAG in M4. |
| VI. Non-Invasive Auditing | ✅ Noted | File submissions are not trace events. Trace capture (M5) is gated by `trace_enabled` on actividades, which is set by M3. M4 does not modify `trace_enabled`. |
| VII. Submission Always Accepted | ✅ **Directly exercised** | Within-deadline submissions always accepted (FR-002). Post-deadline acceptance is controlled by `late_submission_allowed` (docente-configurable, default `true`). This is an explicit, audited docente decision — consistent with the constitution's principle that docentes have authority over grading and acceptance decisions. Edit window does not violate Principle VII: within the window the student is finalizing their submission, not being blocked by the system. |

**Result: PASS** — No violations. Principle IV is the primary enforcement driver; Principle VII is the primary design driver; Principle I is exercised via the `actividad_config_changes` audit log.

---

## Project Structure

### Documentation (this feature)

```text
specs/004-file-storage/
├── plan.md                  ← this file
├── research.md              ← Phase 0 output
├── data-model.md            ← Phase 1 output
├── quickstart.md            ← Phase 1 output
├── contracts/
│   └── submissions.md       ← Phase 1 output
├── checklists/
│   └── requirements.md
└── tasks.md                 ← Phase 2 output (/speckit-tasks)
```

### Source Code Changes

```text
nextlab-api/
├── cmd/
│   └── api/
│       └── main.go                      # Add submission.RegisterRoutes; add MinIO client init; validate MinIO config at startup
├── internal/
│   ├── submission/                      # NEW module
│   │   ├── module.go                    # RegisterRoutes(app, db, minioClient, cfg)
│   │   ├── model.go                     # Submission struct + SubmissionResponse + ConfigChangeResponse
│   │   ├── repository.go                # All DB queries, role-scoped
│   │   ├── service.go                   # UploadFile, ReplaceFile, DeleteSubmission, DownloadFile, IsLocked()
│   │   ├── handler_student.go           # POST/PUT/DELETE/GET /actividades/:aid/submission[/download]
│   │   └── handler_docente.go           # GET /cursos/:cid/actividades/:aid/submissions
│   │                                    # GET /submissions/:sid/download
│   │                                    # GET /cursos/:cid/actividades/:aid/config-changes
│   ├── academic/
│   │   ├── handler_actividad.go         # EXTEND: PUT actividad now reads + validates late_submission_allowed
│   │   ├── service.go                   # EXTEND: WriteConfigChange() called when late_submission_allowed changes
│   │   ├── model.go                     # EXTEND: ActividadResponse gains late_submission_allowed field
│   │   └── repository.go               # EXTEND: UpdateActividad now sets late_submission_allowed; GetConfigChanges query
│   └── config/
│       └── config.go                    # Add MinioEndpoint, MinioAccessKey, MinioSecretKey, MinioBucket, MinioUseSSL
├── migrations/
│   ├── 000004_file_storage.up.sql       # ALTER actividades + 2 new tables
│   └── 000004_file_storage.down.sql     # DROP new tables + REVERT ALTER
├── go.mod                               # Add minio/minio-go/v7
├── go.sum                               # Updated by go mod tidy
└── .env.example                         # Add MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, MINIO_BUCKET, MINIO_USE_SSL
```

**Structure Decision**: All submission logic lives in `internal/submission/`. The academic module (`internal/academic/`) is extended with minimal targeted changes to support `late_submission_allowed` — no new files added to that module. No other existing packages are modified.

---

## Phase 0: Research Summary

See [research.md](research.md) for the full decision log. Key decisions:

| Topic | Decision |
|---|---|
| MinIO Go SDK | `minio/minio-go/v7` — official, idiomatic, actively maintained |
| Upload flow | Multipart form → API → MinIO `PutObject` (API-proxied; client never addresses MinIO) |
| Download flow | `minio.GetObject` streamed to client via `fiber.SendStream` (API-proxied; no presigned URLs) |
| Atomicity (FR-014) | MinIO `PutObject` first → DB `INSERT`; if INSERT fails, `RemoveObject` cleans up the orphan |
| Concurrent upload prevention | DB `UNIQUE(student_id, actividad_id)` catches duplicate INSERTs; handler catches violation → 409 + MinIO cleanup |
| `is_locked` | Derived at API read time from `edit_window_expires_at`; no stored column, no background job |
| Edit window cap | `min(submitted_at + 30min, deadline)` when `late_submission_allowed = false`; uncapped when `true` |
| File replacement | MinIO `PutObject` new key → DB `UPDATE` in transaction → `RemoveObject` old key after commit |
| Student delete within window | DB `DELETE` + MinIO `RemoveObject`; fresh 30-min window on next upload |
| Storage key | `submissions/{actividad_id}/{student_id}/{uuidv4}{ext}` — server-generated, never exposed |
| MinIO bucket | Single bucket `nextlab`, `submissions/` prefix; bucket created idempotently at startup |
| Migration | `000004_file_storage`: ALTER `actividades` + CREATE `actividad_config_changes` + CREATE `submissions` |

---

## Phase 1: Design Summary

### Data Model

See [data-model.md](data-model.md). Migration `000004_file_storage`:

| Change | Description |
|---|---|
| `ALTER TABLE actividades` | Add `late_submission_allowed BOOLEAN NOT NULL DEFAULT true` |
| `CREATE TABLE actividad_config_changes` | Immutable audit log for activity config flag changes |
| `CREATE TABLE submissions` | Core submission record with `UNIQUE(student_id, actividad_id)` |

### API Contract

See [contracts/submissions.md](contracts/submissions.md). Full endpoint list:

| Method | Path | Role | Description |
|---|---|---|---|
| `POST` | `/actividades/:aid/submission` | estudiante | Upload first file |
| `PUT` | `/actividades/:aid/submission` | estudiante | Replace within edit window |
| `DELETE` | `/actividades/:aid/submission` | estudiante | Remove within edit window |
| `GET` | `/actividades/:aid/submission` | estudiante | Get own submission metadata |
| `GET` | `/actividades/:aid/submission/download` | estudiante | Download own file |
| `GET` | `/cursos/:cid/actividades/:aid/submissions` | docente, co_docente | List all submissions for activity |
| `GET` | `/submissions/:sid/download` | docente, co_docente | Download a specific submission |
| `PUT` | `/cursos/:cid/actividades/:aid` | docente | Extended from M3: now accepts `late_submission_allowed` |
| `GET` | `/cursos/:cid/actividades/:aid/config-changes` | docente, admin | Activity config audit log |

### Quickstart

See [quickstart.md](quickstart.md). After `docker compose up --build`:
1. Add `MINIO_*` env vars to `.env`; verify bucket created at startup
2. Run migration `000004_file_storage`
3. Upload a file as a student → verify `201` with `edit_window_expires_at`
4. List submissions as docente → verify student's record appears
5. Download via docente endpoint → verify file content (md5 check)
6. Replace within edit window → verify `last_replaced_at` updated
7. Attempt replace after window → verify `403`
8. Toggle `late_submission_allowed = false` → submit late → verify `422`
9. Check config-changes audit log → verify entry recorded
10. Attempt cross-student download → verify `403`

---

## Key Implementation Notes

### Upload handler (student)

```
POST /actividades/:aid/submission
  1. Validate student is enrolled in the curso that owns actividad (JOIN inscripciones)
  2. Check actividad.tipo ∈ {practica, tarea, informe, examen, defensa_oral}
  3. Check no existing submission: SELECT FROM submissions WHERE student_id=$1 AND actividad_id=$2
     → if found: return 409 (use PUT to replace)
  4. Parse multipart file; check size ≤ 50 MB → 400 if exceeded
  5. Compute:
       submitted_at = now()     (server clock, not client)
       is_late      = submitted_at > actividad.deadline
       if is_late AND NOT actividad.late_submission_allowed → return 422
  6. Compute edit_window_expires_at:
       if is_late: NULL
       elif actividad.late_submission_allowed: submitted_at + 30min
       else: min(submitted_at + 30min, actividad.deadline)
  7. Generate storage_key = "submissions/{aid}/{student_id}/{uuid}{ext}"
  8. minio.PutObject(ctx, bucket, storage_key, reader, size, opts)
     → on error: return 503 (no DB write)
  9. BEGIN TRANSACTION
       INSERT INTO submissions (student_id, actividad_id, storage_key, ...)
       if first submission: UPDATE actividades SET has_submissions = true WHERE id = $aid
     COMMIT
     → on unique violation: minio.RemoveObject(key); return 409
     → on other DB error: minio.RemoveObject(key); return 500
 10. Return 201 SubmissionResponse (with is_locked derived = false)
```

### Replace handler (student)

```
PUT /actividades/:aid/submission
  1. Validate enrollment
  2. Fetch existing submission; if not found → 404
  3. Check is_locked = (edit_window_expires_at IS NULL) OR (now() > edit_window_expires_at)
     → if locked: return 403
  4. Parse multipart file; check size ≤ 50 MB
  5. Compute new submitted_at = now(); new is_late = now() > actividad.deadline
  6. Generate new_key = "submissions/{aid}/{student_id}/{new_uuid}{ext}"
  7. minio.PutObject(ctx, bucket, new_key, reader, size, opts)
     → on error: return 503 (no DB write)
  8. BEGIN TRANSACTION
       old_key = submission.storage_key
       UPDATE submissions SET storage_key=new_key, original_filename=..., file_size_bytes=...,
         mime_type=..., submitted_at=now(), is_late=..., last_replaced_at=now()
         WHERE student_id=$1 AND actividad_id=$2
     COMMIT
     → on error: minio.RemoveObject(new_key); return 500
  9. minio.RemoveObject(old_key)  -- best-effort after commit; log failure if it occurs
 10. Return 200 SubmissionResponse
```

### Download handler (proxy streaming)

```go
func (h *Handler) DownloadSubmission(c *fiber.Ctx) error {
    sub := // fetch submission + verify role access
    obj, err := h.minio.GetObject(ctx, h.bucket, sub.StorageKey, minio.GetObjectOptions{})
    if err != nil { return fiber.ErrNotFound }
    defer obj.Close()

    stat, _ := obj.Stat()
    c.Set("Content-Disposition", `attachment; filename="`+sub.OriginalFilename+`"`)
    c.Set("Content-Type", sub.MimeType)
    c.Set("Content-Length", strconv.FormatInt(stat.Size, 10))
    return c.SendStream(obj, int(stat.Size))
}
```

### Activity config change (academic module extension)

```go
// academic/service.go
func (s *Service) UpdateActividad(ctx context.Context, docenteID, actividadID uuid.UUID, req *UpdateActividadRequest) (*Actividad, error) {
    existing := s.repo.GetActividad(ctx, actividadID, docenteID) // role-scoped

    // ... existing M3 mutation guards (title/tipo locked after submissions) ...

    if req.LateSubmissionAllowed != nil && *req.LateSubmissionAllowed != existing.LateSubmissionAllowed {
        // Write config change log in same transaction as UPDATE
        s.repo.UpdateActividadWithConfigLog(ctx, actividadID, docenteID, req, existing.LateSubmissionAllowed)
    } else {
        s.repo.UpdateActividad(ctx, actividadID, docenteID, req)
    }
    return s.repo.GetActividad(ctx, actividadID, docenteID)
}
```

### MinIO startup initialization (main.go)

```go
mc, err := minio.New(cfg.MinioEndpoint, &minio.Options{
    Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
    Secure: cfg.MinioUseSSL,
})
if err != nil { slog.Error("minio init failed", "err", err); os.Exit(1) }

exists, err := mc.BucketExists(ctx, cfg.MinioBucket)
if err != nil { slog.Error("minio bucket check failed", "err", err); os.Exit(1) }
if !exists {
    if err = mc.MakeBucket(ctx, cfg.MinioBucket, minio.MakeBucketOptions{}); err != nil {
        slog.Error("minio bucket create failed", "err", err); os.Exit(1)
    }
}
slog.Info("minio ready", "bucket", cfg.MinioBucket)
```

### Role-scoped submission list query (repository.go)

```sql
-- GetSubmissionsForDocente
SELECT s.id, s.student_id, u.name, u.email,
       s.original_filename, s.file_size_bytes, s.mime_type,
       s.submitted_at, s.is_late, s.edit_window_expires_at,
       s.last_replaced_at
FROM submissions s
JOIN users u ON u.id = s.student_id
JOIN actividades a ON a.id = s.actividad_id
JOIN cursos c ON c.id = a.curso_id
WHERE s.actividad_id = $1
  AND c.docente_id = $2          -- role scope: docente owns the course
ORDER BY u.name ASC

-- GetSubmissionsForCodocente
SELECT s.id, s.student_id, u.name, u.email,
       s.original_filename, s.file_size_bytes, s.mime_type,
       s.submitted_at, s.is_late, s.edit_window_expires_at,
       s.last_replaced_at
FROM submissions s
JOIN users u ON u.id = s.student_id
JOIN codocente_grants g ON g.actividad_id = s.actividad_id
WHERE s.actividad_id = $1
  AND g.codocente_id = $2        -- role scope: active grant only
  AND g.is_active = true
  AND g.revoked_at IS NULL
ORDER BY u.name ASC
```
