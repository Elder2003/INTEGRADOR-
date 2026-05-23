# Tasks: M12 — Admin Modules

**Input**: Design documents from `specs/012-admin-modules/`

**Prerequisites**: spec.md ✓, research.md ✓, data-model.md ✓, plan.md ✓, contracts/admin-modules.md ✓

---

## Phase 1: Setup

- [ ] T001 Write `migrations/000012_admin_modules.up.sql`: `ALTER TABLE users ADD COLUMN full_name TEXT NOT NULL DEFAULT ''`; `CREATE TABLE format_templates (id UUID PK, name TEXT UNIQUE NOT NULL, description TEXT NOT NULL DEFAULT '', rules JSONB NOT NULL DEFAULT '{}', is_active BOOLEAN NOT NULL DEFAULT TRUE, created_by UUID NOT NULL REFERENCES users(id), created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)`; `CREATE TABLE actividad_format_template (actividad_id UUID PK REFERENCES actividades(id) ON DELETE CASCADE, format_template_id UUID NOT NULL REFERENCES format_templates(id), applied_at TIMESTAMPTZ NOT NULL DEFAULT now(), applied_by UUID NOT NULL REFERENCES users(id))`; `CREATE TABLE actividad_interdisciplinary_scopes (id UUID PK, actividad_id UUID NOT NULL REFERENCES actividades(id) ON DELETE CASCADE, carrera TEXT NOT NULL, semestre SMALLINT NOT NULL CHECK (semestre BETWEEN 1 AND 10), codigo_materia TEXT NOT NULL, UNIQUE (actividad_id, carrera, semestre, codigo_materia)); CREATE INDEX ais_actividad_idx ON actividad_interdisciplinary_scopes(actividad_id)`
- [ ] T002 Write `migrations/000012_admin_modules.down.sql`: drop actividad_interdisciplinary_scopes, actividad_format_template, format_templates; `ALTER TABLE users DROP COLUMN full_name`

---

## Phase 2: Foundational

- [ ] T003 Write `internal/admin/model.go`: define `DocenteAccount` struct (id, email, full_name, is_active, created_at, updated_at), `CreateDocenteRequest` (email, full_name), `UpdateDocenteRequest` (full_name *string, is_active *bool); define `FormatTemplate` struct (id, name, description, rules format.FormatRulesConfig, is_active, created_by, created_at, updated_at), `CreateFormatTemplateRequest`, `UpdateFormatTemplateRequest`, `AppliedTemplate` (format_template_id, name, rules, applied_at); define `CurricularDocument` struct (id, carrera, semestre, codigo_materia, original_filename, is_rubric, minio_path, created_at); define `InterdisciplinaryScope` (id, actividad_id, carrera, semestre, codigo_materia) and `InterdisciplinaryScopeInput` (carrera, semestre, codigo_materia); import `github.com/Elder2003/nextlab-api/internal/format` for FormatRulesConfig
- [ ] T004 Write `internal/admin/repository.go`: `Repository` struct with `*pgxpool.Pool`; implement `NewRepository(db) *Repository`; implement all docente methods: `InsertDocente(ctx, email, fullName, passwordHash string) (DocenteAccount, error)`, `ListDocentes(ctx, activeOnly *bool, limit, offset int) ([]DocenteAccount, int, error)`, `GetDocente(ctx, id uuid.UUID) (*DocenteAccount, error)`, `UpdateDocente(ctx, id uuid.UUID, fullName *string, isActive *bool) (*DocenteAccount, error)`; implement format template methods: `InsertFormatTemplate(ctx, name, description string, rules json.RawMessage, createdBy uuid.UUID) (*FormatTemplate, error)`, `ListFormatTemplates(ctx, activeOnly bool) ([]FormatTemplate, error)`, `GetFormatTemplate(ctx, id uuid.UUID) (*FormatTemplate, error)`, `UpdateFormatTemplate(ctx, id uuid.UUID, req UpdateFormatTemplateRequest) (*FormatTemplate, error)`, `IsTemplateInUse(ctx, id uuid.UUID) (bool, error)`, `DeleteFormatTemplate(ctx, id uuid.UUID) error`; implement apply methods: `ApplyTemplate(ctx, actividadID, templateID, appliedBy uuid.UUID) error`, `GetAppliedTemplate(ctx, actividadID uuid.UUID) (*AppliedTemplate, error)`, `UpdateActividadFormatRules(ctx, actividadID uuid.UUID, rules json.RawMessage) error`, `VerifyDocenteOwnsActividad(ctx, actividadID, docenteID uuid.UUID) (bool, error)`; implement interdisciplinary methods: `SetScopes(ctx, actividadID uuid.UUID, scopes []InterdisciplinaryScopeInput) ([]InterdisciplinaryScope, error)`, `GetScopes(ctx, actividadID uuid.UUID) ([]InterdisciplinaryScope, error)`, `DeleteScopes(ctx, actividadID uuid.UUID) error`
- [ ] T005 Write `internal/admin/module.go`: `Module` struct with `*Repository` and `*minio.Client` and minioBucket string; `NewModule(db *pgxpool.Pool, mc *minio.Client, bucket string) *Module`; stub `RegisterRoutes` returning 501 for all routes; import format package for FormatRulesConfig

---

## Phase 3: US1 — Docente Account Management (P1) 🎯 MVP

**Goal**: Admin creates, lists, updates, and deactivates docente accounts.

**Independent Test**: POST → 201 with temp_password; GET list → contains new docente; PUT → name updated; DELETE → 204; non-admin → 403.

- [ ] T006 [P] [US1] Write `internal/admin/handler_docente.go`: implement `CreateDocenteHandler(repo *Repository)` — parse CreateDocenteRequest, validate email ends in @unifranz.edu.bo or is non-empty, generate 12-char random alphanumeric temp password with `crypto/rand`, bcrypt-hash it, call `repo.InsertDocente`, return 201 with account + temporary_password; implement `ListDocentesHandler(repo)` — parse page/limit/active query params, call `repo.ListDocentes`, return paginated response; implement `GetDocenteHandler(repo)` — parse id, call `repo.GetDocente`, 404 if nil; implement `UpdateDocenteHandler(repo)` — parse id + UpdateDocenteRequest, call `repo.UpdateDocente`, 404 if nil; implement `DeleteDocenteHandler(repo)` — parse id, call `repo.UpdateDocente` with is_active=false, 204 on success; extract docenteID from `c.Locals("user_id")` for admin identity if needed; map unique constraint violation → 409
- [ ] T007 [US1] Wire US1 routes in `internal/admin/module.go` `RegisterRoutes`: under `app.Group("/admin", jwtMW, roleGuard("admin"))`: POST /docentes, GET /docentes, GET /docentes/:id, PUT /docentes/:id, DELETE /docentes/:id

**Checkpoint**: Full US1 flow works. Admin can provision, list, and deactivate docente accounts. 403 for non-admin.

---

## Phase 4: US2 — Format Template Library (P2)

**Goal**: Admin manages reusable format templates; docentes apply them to their actividades.

**Independent Test**: Admin POST template → 201; GET list → present; docente applies → actividad.format_rules updated; GET applied → returns template; delete in-use → 409; non-admin admin endpoints → 403.

- [ ] T008 [P] [US2] Write `internal/admin/handler_template.go`: implement `CreateFormatTemplateHandler(repo)` — parse CreateFormatTemplateRequest, marshal rules to JSON, call `repo.InsertFormatTemplate`, return 201; implement `ListFormatTemplatesHandler(repo)` — parse active_only param; implement `GetFormatTemplateHandler(repo)`; implement `UpdateFormatTemplateHandler(repo)` — partial update; implement `DeleteFormatTemplateHandler(repo)` — call `repo.IsTemplateInUse`, return 409 if true, else call `repo.DeleteFormatTemplate`, 204; implement `ApplyTemplateHandler(repo)` — parse actividad_id and template_id, verify docente ownership via `repo.VerifyDocenteOwnsActividad` (403 if false), fetch template, marshal rules, call `repo.ApplyTemplate` + `repo.UpdateActividadFormatRules` in same handler, return AppliedTemplate 200; implement `GetAppliedTemplateHandler(repo)` — verify docente ownership, call `repo.GetAppliedTemplate`
- [ ] T009 [US2] Wire US2 routes in `RegisterRoutes`: admin group: POST/GET /format-templates, GET/PUT/DELETE /format-templates/:id; docente group (jwtMW + roleGuard("docente")): POST/GET /actividades/:aid/format-template

**Checkpoint**: Full US2 flow. Template CRUD works. Docente apply flow updates actividad.format_rules. Delete of in-use template blocked.

---

## Phase 5: US3 — Curriculum Document Management (P3)

**Goal**: Admin lists and deletes curriculum documents from rag_curricular.

**Independent Test**: GET list with filter → correct docs; DELETE → 204; doc no longer in list; MinIO file gone; non-admin → 403.

- [ ] T010 [P] [US3] Add `ListDocuments(ctx context.Context, carrera, semestre, codigoMateria string, page, limit int) ([]rag.CurricularDocumentSummary, int, error)` to `internal/rag/repository_curricular.go` — SELECT id, carrera, semestre, codigo_materia, original_filename, is_rubric, minio_path, created_at FROM rag_curricular WHERE (carrera=$1 OR $1='') AND (semestre=$2 OR $2=0) AND (codigo_materia=$3 OR $3='') ORDER BY created_at DESC LIMIT $4 OFFSET $5; add count query; define `CurricularDocumentSummary` struct in `internal/rag/model.go`
- [ ] T011 [P] [US3] Add `DeleteDocument(ctx context.Context, id uuid.UUID) (minioPath string, err error)` to `internal/rag/repository_curricular.go` — SELECT minio_path then DELETE FROM rag_curricular WHERE id=$1 (chunks cascade); return minio_path for caller to delete from MinIO
- [ ] T012 [US3] Write `internal/admin/handler_curriculum.go`: implement `ListCurriculumDocsHandler(ragCurRepo *rag.CurricularRepository)` — parse carrera/semestre/codigo_materia/page/limit query params, call `ragCurRepo.ListDocuments`; implement `DeleteCurriculumDocHandler(ragCurRepo *rag.CurricularRepository, mc *minio.Client, bucket string)` — parse id, call `ragCurRepo.DeleteDocument` (get minioPath), call `mc.RemoveObject(ctx, bucket, minioPath)`, return 204; 404 if minio_path empty (doc not found)
- [ ] T013 [US3] Wire US3 routes in `RegisterRoutes` admin group: GET /curriculum/documents, DELETE /curriculum/documents/:id; pass ragCurRepo from module

**Checkpoint**: Full US3 flow. List and delete curriculum docs. MinIO file removed on delete.

---

## Phase 6: US4 — Interdisciplinary Activity Configuration (P4)

**Goal**: Admin sets extra RAG scopes on an actividad; detection queries all scopes.

**Independent Test**: POST scopes → 200; GET → scopes listed; run RAG query for that actividad (unit test or inspect prompt) → includes content from all scopes; DELETE → 204; scopes removed.

- [ ] T014 [P] [US4] Write `internal/admin/handler_interdisciplinary.go`: implement `SetScopesHandler(repo)` — parse actividad_id, parse scopes array from body, call `repo.SetScopes` (replaces existing), return 200 with result; implement `GetScopesHandler(repo)` — call `repo.GetScopes`; implement `DeleteScopesHandler(repo)` — call `repo.DeleteScopes`, 204
- [ ] T015 [US4] Update `internal/rag/service_retrieve.go` `QueryCurricular` method: after building primary `CurricularQueryRequest`, add DB query for `actividad_interdisciplinary_scopes WHERE actividad_id=$1`; for each extra scope, call `r.curRepo.QuerySimilar(ctx, scope.Carrera, scope.Semestre, scope.CodigoMateria, nil, query, limit)` (or equivalent); union all chunks from primary + extra scopes; deduplicate by chunk_id keeping first occurrence; return combined `RetrievalResponse`. If actividad_id is nil in request, skip interdisciplinary lookup. Add a direct db query helper to `rag/repository_curricular.go`: `GetInterdisciplinaryScopes(ctx, actividadID uuid.UUID) ([]InterdisciplinaryScopeRow, error)` — this avoids a circular import by keeping the query in the rag package
- [ ] T016 [US4] Wire US4 routes in `RegisterRoutes`: admin group: POST /actividades/:aid/interdisciplinary, DELETE /actividades/:aid/interdisciplinary; authenticated (admin or docente): GET /actividades/:aid/interdisciplinary

**Checkpoint**: Full US4 flow. Scopes stored. GET returns scopes. Detection RAG union includes extra scope content.

---

## Phase 7: Polish

- [ ] T017 Register `internal/admin` module in `cmd/api/main.go`: `adminModule := admin.NewModule(pgPool, minioClient, cfg.MinioBucket); adminModule.RegisterRoutes(app, auth.AuthMiddleware(cfg), auth.RoleGuard)`; add `import "github.com/Elder2003/nextlab-api/internal/admin"`
- [ ] T018 Add `ragCurRepo *rag.CurricularRepository` to `admin.Module` struct and accept it in `NewModule` — OR instantiate it directly inside `NewModule(db)` via `rag.NewCurricularRepository(db)` to avoid requiring callers to pass it; update `handler_curriculum.go` accordingly
- [ ] T019 Run `go build ./...` and `go vet ./...` from repo root; fix any compile errors or vet warnings
- [ ] T020 Run quickstart scenarios from `specs/012-admin-modules/quickstart.md` against live stack; document any gaps

---

## Dependencies & Execution Order

- Phase 1 (setup) → blocks all others
- Phase 2 (foundational model/repo/module) → blocks all user story phases
- US1 (P1) → independent of US2/US3/US4
- US2 (P2) → independent of US1/US3/US4
- US3 (P3) → depends on rag repository additions (T010/T011) before handler (T012)
- US4 (P4) → depends on rag service update (T015); T014 can run in parallel with T015
- Phase 7 (polish) → depends on all user stories

## Notes

- T010/T011 modify `internal/rag/repository_curricular.go` — run sequentially to avoid conflicts
- T015 modifies `internal/rag/service_retrieve.go` — no conflict with T010/T011 (different file)
- To avoid circular imports: `rag` package MUST NOT import `admin` package. The interdisciplinary scope lookup in T015 MUST be a direct DB query inside the rag package, not a call to `admin.Repository`.
- Docente apply endpoint (T008) must update BOTH `actividad_format_template` JOIN table AND `actividades.format_rules` in a single transaction.
