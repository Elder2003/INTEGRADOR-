# Tasks: M12 — Admin Modules

**Input**: Design documents from `specs/012-admin-modules/`

**Prerequisites**: spec.md ✓, research.md ✓, data-model.md ✓, plan.md ✓, contracts/admin-modules.md ✓

---

## Phase 1: Setup

- [X] T001 Write `migrations/000012_admin_modules.up.sql`: `ALTER TABLE users ADD COLUMN IF NOT EXISTS full_name TEXT NOT NULL DEFAULT ''`; `CREATE TABLE format_templates (id UUID PK DEFAULT gen_random_uuid(), name TEXT UNIQUE NOT NULL, description TEXT NOT NULL DEFAULT '', rules JSONB NOT NULL DEFAULT '{}', is_active BOOLEAN NOT NULL DEFAULT TRUE, created_by UUID NOT NULL REFERENCES users(id), created_at/updated_at TIMESTAMPTZ)`; `CREATE TABLE actividad_format_template (actividad_id UUID PK REFERENCES actividades(id) ON DELETE CASCADE, format_template_id UUID NOT NULL REFERENCES format_templates(id), applied_at TIMESTAMPTZ NOT NULL DEFAULT now(), applied_by UUID NOT NULL REFERENCES users(id))`; `CREATE TABLE actividad_interdisciplinary_scopes (id UUID PK DEFAULT gen_random_uuid(), actividad_id UUID NOT NULL REFERENCES actividades(id) ON DELETE CASCADE, carrera TEXT NOT NULL, semestre SMALLINT NOT NULL CHECK (semestre BETWEEN 1 AND 10), codigo_materia TEXT NOT NULL, UNIQUE (actividad_id, carrera, semestre, codigo_materia))`; `CREATE INDEX ais_actividad_idx ON actividad_interdisciplinary_scopes(actividad_id)`
- [X] T002 Write `migrations/000012_admin_modules.down.sql`: `DROP TABLE IF EXISTS actividad_interdisciplinary_scopes`; `DROP TABLE IF EXISTS actividad_format_template`; `DROP TABLE IF EXISTS format_templates`; `ALTER TABLE users DROP COLUMN IF EXISTS full_name`

---

## Phase 2: Foundational

- [X] T003 [P] Write `internal/admin/model.go`: `DocenteAccount` {id, email, full_name, is_active, created_at, updated_at}; `CreateDocenteRequest` {email, full_name, test_password *string}; `UpdateDocenteRequest` {full_name *string, is_active *bool}; `FormatTemplate` {id, name, description, rules format.FormatRulesConfig, is_active, created_by, created_at, updated_at}; `CreateFormatTemplateRequest` {name, description, rules format.FormatRulesConfig}; `UpdateFormatTemplateRequest` {name *string, description *string, rules *format.FormatRulesConfig, is_active *bool}; `AppliedTemplate` {format_template_id, name, rules format.FormatRulesConfig, applied_at}; `CurricularDocument` {id, carrera, semestre, codigo_materia, original_filename, is_rubric, minio_path, created_at}; `InterdisciplinaryScope` {id, actividad_id, carrera, semestre, codigo_materia}; `InterdisciplinaryScopeInput` {carrera, semestre, codigo_materia}; `ErrorResponse` {error, message}; import `github.com/Elder2003/nextlab-api/internal/format`
- [X] T004 Write `internal/admin/repository.go`: `Repository` struct with `db *pgxpool.Pool`; `NewRepository(db *pgxpool.Pool) *Repository`; docente methods: `InsertDocente(ctx, email, fullName, passwordHash string) (*DocenteAccount, error)` — INSERT INTO users (email, full_name, password_hash, role) VALUES ($1,$2,$3,'docente') RETURNING id,email,full_name,is_active,created_at,updated_at; `ListDocentes(ctx, activeOnly *bool, limit, offset int) ([]DocenteAccount, int, error)` — SELECT with optional is_active filter + COUNT; `GetDocente(ctx, id uuid.UUID) (*DocenteAccount, error)` — nil on no rows; `UpdateDocente(ctx, id uuid.UUID, fullName *string, isActive *bool) (*DocenteAccount, error)` — COALESCE patch, nil on no rows; format template methods: `InsertFormatTemplate(ctx, name, description string, rules json.RawMessage, createdBy uuid.UUID) (*FormatTemplate, error)`, `ListFormatTemplates(ctx, activeOnly bool) ([]FormatTemplate, error)`, `GetFormatTemplate(ctx, id uuid.UUID) (*FormatTemplate, error)`, `UpdateFormatTemplate(ctx, id uuid.UUID, req UpdateFormatTemplateRequest) (*FormatTemplate, error)`, `IsTemplateInUse(ctx, id uuid.UUID) (bool, error)` — SELECT EXISTS(SELECT 1 FROM actividad_format_template WHERE format_template_id=$1), `DeleteFormatTemplate(ctx, id uuid.UUID) error`; apply methods: `ApplyTemplate(ctx, actividadID, templateID, appliedBy uuid.UUID) error` — INSERT INTO actividad_format_template ON CONFLICT (actividad_id) DO UPDATE; `GetAppliedTemplate(ctx, actividadID uuid.UUID) (*AppliedTemplate, error)` — JOIN format_templates; `UpdateActividadFormatRules(ctx, actividadID uuid.UUID, rules json.RawMessage) error` — UPDATE actividades SET format_rules=$2 WHERE id=$1; `VerifyDocenteOwnsActividad(ctx, actividadID, docenteID uuid.UUID) (bool, error)` — SELECT EXISTS(SELECT 1 FROM actividades a JOIN cursos c ON c.id=a.curso_id WHERE a.id=$1 AND c.docente_id=$2); interdisciplinary methods: `SetScopes(ctx, actividadID uuid.UUID, scopes []InterdisciplinaryScopeInput) ([]InterdisciplinaryScope, error)` — DELETE then INSERT in tx; `GetScopes(ctx, actividadID uuid.UUID) ([]InterdisciplinaryScope, error)`; `DeleteScopes(ctx, actividadID uuid.UUID) error`
- [X] T005 Write `internal/admin/module.go`: `Module` struct {repo *Repository, curRepo *rag.CurricularRepository, mc *minio.Client, bucket string, appEnv string}; `NewModule(db *pgxpool.Pool, mc *minio.Client, bucket, appEnv string) *Module` — instantiates NewRepository(db) and rag.NewCurricularRepository(db) internally; stub `RegisterRoutes(app *fiber.App, jwtMW fiber.Handler, roleGuard func(...string) fiber.Handler)` returning 501 for all routes; import `github.com/Elder2003/nextlab-api/internal/rag`

---

## Phase 3: US1 — Docente Account Management (P1) 🎯 MVP

**Goal**: Admin creates, lists, updates, and deactivates docente accounts.

**Independent Test**: POST → 201; GET list → contains new docente; PUT → name updated; DELETE → 204; non-admin → 403.

- [X] T006 [P] [US1] Write `internal/admin/handler_docente.go`: `CreateDocenteHandler(repo *Repository, appEnv string) fiber.Handler` — parse CreateDocenteRequest; validate email matches `^doc\..+@unifranz\.edu\.bo$` → 400 if not; if appEnv=="development" and test_password non-empty, bcrypt hash it else use "" as passwordHash; InsertDocente; return 201 DocenteAccount (no password in response); map unique violation → 409; `ListDocentesHandler(repo *Repository) fiber.Handler` — parse page/limit/?active query params; return 200 {docentes, total, page, limit}; `GetDocenteHandler(repo *Repository) fiber.Handler` — parse id, 404 if nil; `UpdateDocenteHandler(repo *Repository) fiber.Handler` — parse id, partial update, 404 if nil; `DeleteDocenteHandler(repo *Repository) fiber.Handler` — parse id, call UpdateDocente with isActive=false, 204 (404 if nil)
- [X] T007 [US1] Wire US1 routes in `internal/admin/module.go` `RegisterRoutes`: `admin := app.Group("/admin", jwtMW, roleGuard("admin"))`; admin.Post("/docentes", CreateDocenteHandler(m.repo, m.appEnv)); admin.Get("/docentes", ListDocentesHandler(m.repo)); admin.Get("/docentes/:id", GetDocenteHandler(m.repo)); admin.Put("/docentes/:id", UpdateDocenteHandler(m.repo)); admin.Delete("/docentes/:id", DeleteDocenteHandler(m.repo))

**Checkpoint**: Admin can provision, list, update, and deactivate docente accounts. 403 for non-admin.

---

## Phase 4: US2 — Format Template Library (P2)

**Goal**: Admin manages reusable format templates; docentes apply them to their actividades.

**Independent Test**: Admin POST template → 201; GET list → present; docente applies → 200 with AppliedTemplate; GET applied → returns template; delete in-use → 409; non-admin on admin endpoints → 403.

- [X] T008 [P] [US2] Write `internal/admin/handler_template.go`: `CreateFormatTemplateHandler(repo *Repository) fiber.Handler` — parse CreateFormatTemplateRequest, marshal rules to JSON, InsertFormatTemplate, return 201 FormatTemplate; map unique violation → 409; `ListFormatTemplatesHandler(repo *Repository) fiber.Handler` — parse active_only bool param, return {templates, total}; `GetFormatTemplateHandler(repo *Repository) fiber.Handler` — 404 if nil; `UpdateFormatTemplateHandler(repo *Repository) fiber.Handler` — partial update, 404 if nil; `DeleteFormatTemplateHandler(repo *Repository) fiber.Handler` — IsTemplateInUse → 409 {error:"conflict"} if true; else DeleteFormatTemplate, 204; `ApplyTemplateHandler(repo *Repository) fiber.Handler` — parse aid from path params; parse {template_id} from body; VerifyDocenteOwnsActividad → 403 if false; GetFormatTemplate → 404 if nil; marshal rules to json.RawMessage; call ApplyTemplate then UpdateActividadFormatRules in sequence; return 200 AppliedTemplate; `GetAppliedTemplateHandler(repo *Repository) fiber.Handler` — VerifyDocenteOwnsActividad → 403; GetAppliedTemplate; 200 with result or null
- [X] T009 [US2] Wire US2 routes in `RegisterRoutes`: admin group: admin.Post("/format-templates", ...); admin.Get("/format-templates", ...); admin.Get("/format-templates/:id", ...); admin.Put("/format-templates/:id", ...); admin.Delete("/format-templates/:id", ...); docente group: `docente := app.Group("/actividades", jwtMW, roleGuard("docente"))`; docente.Post("/:aid/format-template", ApplyTemplateHandler(m.repo)); docente.Get("/:aid/format-template", GetAppliedTemplateHandler(m.repo))

**Checkpoint**: Template CRUD works. Docente apply updates actividad.format_rules. Delete of in-use template blocked with 409.

---

## Phase 5: US3 — Curriculum Document Management (P3)

**Goal**: Admin lists and deletes curriculum documents from rag_curricular.

**Independent Test**: GET list → correct docs returned; DELETE → 204; list again → doc gone; MinIO file removed.

- [X] T010 [P] [US3] Add `CurricularDocumentSummary` struct to `internal/rag/model.go` {ID uuid.UUID, Carrera string, Semestre int, CodigoMateria string, OriginalFilename string, IsRubric bool, MinioPath string, CreatedAt time.Time}; add `ListDocuments(ctx context.Context, carrera string, semestre int, codigoMateria string, page, limit int) ([]CurricularDocumentSummary, int, error)` to `internal/rag/repository_curricular.go` — SELECT id,carrera,semestre,codigo_materia,original_filename,is_rubric,minio_path,created_at FROM rag_curricular WHERE ($1='' OR carrera=$1) AND ($2=0 OR semestre=$2) AND ($3='' OR codigo_materia=$3) ORDER BY created_at DESC LIMIT $4 OFFSET $5; separate COUNT query with same WHERE
- [X] T011 [US3] Add `DeleteDocument(ctx context.Context, id uuid.UUID) (string, error)` to `internal/rag/repository_curricular.go` — use tx: SELECT minio_path FROM rag_curricular WHERE id=$1 (ErrNoRows → return "","not_found"); DELETE FROM rag_curricular WHERE id=$1 (chunks cascade via FK); commit; return minio_path
- [X] T012 [US3] Write `internal/admin/handler_curriculum.go`: `ListCurriculumDocsHandler(curRepo *rag.CurricularRepository) fiber.Handler` — parse carrera/semestre/codigo_materia/page/limit query params (semestre defaults to 0); call ListDocuments; return {documents, total}; `DeleteCurriculumDocHandler(curRepo *rag.CurricularRepository, mc *minio.Client, bucket string) fiber.Handler` — parse id; call DeleteDocument; if minio_path=="" → 404; mc.RemoveObject(ctx, bucket, minioPath, minio.RemoveObjectOptions{}); return 204
- [X] T013 [US3] Wire US3 routes in `RegisterRoutes` admin group: admin.Get("/curriculum/documents", ListCurriculumDocsHandler(m.curRepo)); admin.Delete("/curriculum/documents/:id", DeleteCurriculumDocHandler(m.curRepo, m.mc, m.bucket))

**Checkpoint**: Admin can list and delete curriculum docs. MinIO file removed on delete. 404 for unknown id.

---

## Phase 6: US4 — Interdisciplinary Activity Configuration (P4)

**Goal**: Admin sets extra RAG scopes on an actividad; detection RAG queries all scopes (union, deduplicated).

**Independent Test**: POST scopes → 200; GET → scopes listed; RAG query for that actividad includes chunks from all scopes; DELETE → 204; non-admin POST/DELETE → 403.

- [X] T014 [P] [US4] Write `internal/admin/handler_interdisciplinary.go`: `SetScopesHandler(repo *Repository) fiber.Handler` — parse aid from path; parse {scopes []InterdisciplinaryScopeInput} from body; call SetScopes (empty array clears all); return 200 {actividad_id, scopes}; `GetScopesHandler(repo *Repository) fiber.Handler` — parse aid; call GetScopes; return {actividad_id, scopes}; check role from c.Locals("user_role"): if "docente", call VerifyDocenteOwnsActividad → 403 if false; `DeleteScopesHandler(repo *Repository) fiber.Handler` — parse aid; DeleteScopes; return 204
- [X] T015 [P] [US4] Add `GetInterdisciplinaryScopes(ctx context.Context, actividadID uuid.UUID) ([]struct{ Carrera string; Semestre int; CodigoMateria string }, error)` to `internal/rag/repository_curricular.go` — SELECT carrera, semestre, codigo_materia FROM actividad_interdisciplinary_scopes WHERE actividad_id=$1; return slice (avoids circular import: rag does NOT import admin)
- [X] T016 [US4] Update `QueryCurricular` in `internal/rag/service_retrieve.go`: after primary QuerySimilar call, if req.ActividadID != nil call r.curRepo.GetInterdisciplinaryScopes; for each extra scope call r.curRepo.QuerySimilar with same queryVec and limit; union all chunks (primary + extra); deduplicate by DocumentID keeping first occurrence; return combined RetrievalResponse{Rubrics: primary rubrics, Chunks: deduped union}
- [X] T017 [US4] Wire US4 routes in `RegisterRoutes`: admin group: admin.Post("/actividades/:aid/interdisciplinary", SetScopesHandler(m.repo)); admin.Delete("/actividades/:aid/interdisciplinary", DeleteScopesHandler(m.repo)); shared group: `shared := app.Group("/actividades", jwtMW, roleGuard("admin", "docente"))`; shared.Get("/:aid/interdisciplinary", GetScopesHandler(m.repo))

**Checkpoint**: Scopes stored and retrieved. Detection RAG union includes content from all scopes. Non-admin POST/DELETE → 403.

---

## Phase 7: Polish

- [X] T018 Register admin module in `cmd/api/main.go`: add import `"github.com/Elder2003/nextlab-api/internal/admin"`; after projects module: `adminModule := admin.NewModule(pgPool, minioClient, cfg.MinioBucket, cfg.AppEnv)`; `adminModule.RegisterRoutes(app, auth.AuthMiddleware(cfg), auth.RoleGuard)`
- [X] T019 Run `go build ./...` and `go vet ./...` from repo root; fix any compile errors or vet warnings
- [ ] T020 Run quickstart scenarios from `specs/012-admin-modules/quickstart.md` against live stack; document any gaps

---

## Dependencies & Execution Order

- Phase 1 (setup) → blocks all others
- Phase 2 (foundational) → blocks all user story phases; T003 [P] can run with T004, T005 depends on T004
- US1 (P1) → T006 [P] and T004 can run together; T007 depends on T006
- US2 (P2) → T008 [P] independent of US1; T009 depends on T008
- US3 (P3) → T010 and T011 modify same file (run sequentially); T012 depends on T010+T011; T013 depends on T012
- US4 (P4) → T014 [P] and T015 [P] can run in parallel; T015 modifies same file as T010/T011 (run after T011); T016 depends on T015; T017 depends on T014+T016
- Phase 7 (polish) → depends on all user stories

## Notes

- T010, T011, T015 all modify `internal/rag/repository_curricular.go` — run sequentially: T010 → T011 → T015
- T016 modifies `internal/rag/service_retrieve.go` — no conflict with above; can start after T015
- **No circular imports**: `rag` package MUST NOT import `admin`. T015 adds `GetInterdisciplinaryScopes` directly to the rag package.
- T008 `ApplyTemplateHandler` must call `ApplyTemplate` + `UpdateActividadFormatRules` sequentially (not in a DB transaction — handler-level sequencing is sufficient since both ops succeed or the request fails).
- `InsertDocente` uses the existing `users` table; role must be set to 'docente' and password_hash to "" for Google-only accounts.
