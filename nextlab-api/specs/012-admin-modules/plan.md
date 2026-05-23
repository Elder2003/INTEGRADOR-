# Implementation Plan: Admin Modules (M12)

## Technical Context

**Stack**: Go 1.25, Fiber v2, pgx/v5, MinIO, Redis (not used for M12 features)

**Existing packages affected**:
- `internal/academic` — already has `/admin/*` routes; M12 admin routes live in new `internal/admin` package
- `internal/rag` — `service_retrieve.go` QueryCurricular must union-query interdisciplinary scopes; `repository_curricular.go` needs ListDocuments + DeleteDocument
- `cmd/api/main.go` — register new admin module

**New packages**:
- `internal/admin` — DocenteHandler, FormatTemplateHandler, CurriculumHandler, InterdisciplinaryHandler, Repository, module.go

## Constitution Check

| Principle | Gate | Status |
|-----------|------|--------|
| I. Immutable Evidence | No new mutable-after-insert evidence tables | ✅ PASS — format_templates and interdisciplinary_scopes are mutable by design (admin config, not evidence) |
| II. Async Isolation | No analysis work in HTTP handlers | ✅ PASS — M12 is CRUD only |
| III. Human Authority | No AI decisions | ✅ PASS — no AI calls in M12 |
| IV. Role Isolation | All /admin/* require role=admin; apply-template requires docente ownership | ✅ PASS |
| V. Dual RAG | Interdisciplinary union keeps rag_curricular physically separate from rag_docente | ✅ PASS |
| VI. Non-Invasive | No trace capture changes | ✅ PASS |
| VII. Submission Always Accepted | No submission gates added | ✅ PASS |

## Project Structure

```
internal/admin/
    model.go          — DocenteAccount, FormatTemplate, AppliedTemplate, CurricularDocument, InterdisciplinaryScope + request types
    repository.go     — all DB queries for admin module
    handler_docente.go    — CRUD handlers for /admin/docentes
    handler_template.go   — CRUD handlers for /admin/format-templates + docente apply/get
    handler_curriculum.go — list/delete /admin/curriculum/documents
    handler_interdisciplinary.go — set/get/delete /actividades/:aid/interdisciplinary
    module.go         — NewModule, RegisterRoutes

migrations/
    000012_admin_modules.up.sql
    000012_admin_modules.down.sql

internal/rag/
    repository_curricular.go  — add ListDocuments(ctx, filters) and DeleteDocument(ctx, id)
    service_retrieve.go       — update QueryCurricular to union interdisciplinary scopes
```

## Phase 0: Research
All NEEDS CLARIFICATION resolved. See research.md.

## Phase 1: Migration + Package Skeleton

1. Write `migrations/000012_admin_modules.up.sql` — ALTER TABLE users ADD COLUMN full_name; CREATE TABLE format_templates; CREATE TABLE actividad_format_template; CREATE TABLE actividad_interdisciplinary_scopes.
2. Write `migrations/000012_admin_modules.down.sql`.
3. Create `internal/admin/model.go` with all types.
4. Create `internal/admin/repository.go` with stub methods.
5. Create `internal/admin/module.go` with `NewModule` + stub `RegisterRoutes`.
6. Wire module into `cmd/api/main.go` (routes registered, returns 501 until implemented).

## Phase 2: US1 — Docente Account Management

1. Implement `repository.go` methods: `InsertDocente`, `ListDocentes`, `GetDocente`, `UpdateDocente`.
2. Write `handler_docente.go`: CreateDocenteHandler, ListDocentesHandler, GetDocenteHandler, UpdateDocenteHandler, DeleteDocenteHandler.
3. Wire routes in `RegisterRoutes`.

## Phase 3: US2 — Format Template Library

1. Implement `repository.go` methods: `InsertFormatTemplate`, `ListFormatTemplates`, `GetFormatTemplate`, `UpdateFormatTemplate`, `DeleteFormatTemplate`, `IsTemplateInUse`, `ApplyTemplateToActividad`, `GetAppliedTemplate`, `UpdateActividadFormatRules`, `VerifyDocenteOwnsActividad`.
2. Write `handler_template.go`.
3. Wire routes.

## Phase 4: US3 — Curriculum Document Management

1. Add `ListDocuments(ctx, carrera, semestre, codigoMateria string, page, limit int) ([]CurricularDocument, int, error)` to `rag/repository_curricular.go`.
2. Add `DeleteDocument(ctx, id uuid.UUID) (minioPath string, err error)` to `rag/repository_curricular.go`.
3. Write `handler_curriculum.go` using `rag.CurricularRepository` + MinIO client for deletion.
4. Wire routes.

## Phase 5: US4 — Interdisciplinary Activity Configuration

1. Implement `repository.go` methods: `SetInterdisciplinaryScopes`, `GetInterdisciplinaryScopes`, `DeleteInterdisciplinaryScopes`.
2. Write `handler_interdisciplinary.go`.
3. Update `rag/service_retrieve.go` `QueryCurricular`: after primary scope query, load extra scopes via `admin.Repository.GetInterdisciplinaryScopes` (or a lightweight direct DB query), run additional `QuerySimilar` per scope, union and deduplicate by chunk_id.
4. Wire routes.

## Phase 6: Polish

1. Register `internal/admin` module in `cmd/api/main.go`.
2. Run `go build ./...` and `go vet ./...`.
3. Validate quickstart scenarios.
