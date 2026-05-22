# Tasks: M8 — Detection Engine

**Input**: Design documents from `specs/008-detection-engine/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/detection-engine.md ✅, quickstart.md ✅

**Tests**: Not requested — no test tasks generated.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)
- Exact file paths are included in every description

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the two new DB tables. No Go code yet.

- [X] T001 Create `migrations/000008_detection_engine.up.sql` with: (a) `CREATE TABLE analysis_results` (id UUID PK DEFAULT gen_random_uuid(), session_id UUID NOT NULL FK→exam_sessions(id), analysis_type TEXT NOT NULL CHECK IN ('heuristic','llm'), created_at TIMESTAMPTZ NOT NULL DEFAULT now(), score INT, flagged_categories TEXT[], event_count INT, advisory_text TEXT, suggested_grade_range TEXT, backend_used TEXT, llm_status TEXT CHECK IN ('success','failed'), UNIQUE(session_id, analysis_type)); INDEX idx_analysis_results_session ON analysis_results(session_id). (b) `CREATE TABLE grade_decisions` (id UUID PK DEFAULT gen_random_uuid(), session_id UUID NOT NULL FK→exam_sessions(id), analysis_result_id UUID NOT NULL FK→analysis_results(id), final_grade NUMERIC(5,2) NOT NULL, suggested_grade_range TEXT, override_reason TEXT, decided_by UUID NOT NULL FK→users(id), decided_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(session_id))
- [X] T002 [P] Create `migrations/000008_detection_engine.down.sql` with: `DROP TABLE IF EXISTS grade_decisions; DROP TABLE IF EXISTS analysis_results;`

**Checkpoint**: Migration files ready; `go build ./...` succeeds with no schema changes yet applied.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Config extension, domain types, and repository — shared by all user stories.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T003 Extend `internal/config/config.go` to add 4 fields to the Config struct: `HeuristicTabWeight int`, `HeuristicClipWeight int`, `HeuristicGapWeight int`, `HeuristicHighThreshold int`; in Load() after the existing cfg struct literal, add: `cfg.HeuristicTabWeight = parseInt(getenv("HEURISTIC_TAB_WEIGHT", "5"))`, `cfg.HeuristicClipWeight = parseInt(getenv("HEURISTIC_CLIP_WEIGHT", "3"))`, `cfg.HeuristicGapWeight = parseInt(getenv("HEURISTIC_GAP_WEIGHT", "2"))`, `cfg.HeuristicHighThreshold = parseInt(getenv("HEURISTIC_HIGH_THRESHOLD", "60"))`
- [X] T004 [P] Create `internal/detection/model.go` with: `HeuristicResult` struct (SessionID uuid.UUID, Score int, FlaggedCategories []string, EventCount int, CreatedAt time.Time); `LLMResult` struct (SessionID uuid.UUID, AdvisoryText string, SuggestedGradeRange string, BackendUsed string, LLMStatus string, CreatedAt time.Time); `DetectionReport` struct (SessionID uuid.UUID, StudentID uuid.UUID, ActividadID uuid.UUID, Heuristic *HeuristicResult, LLM *LLMResult, LLMStatus string json:"llm_status", GradeDecision *GradeDecisionResult); `GradeDecisionResult` struct (FinalGrade float64, SuggestedGradeRange string, OverrideReason string, DecidedBy uuid.UUID, DecidedAt time.Time); `GradeRequest` struct (FinalGrade float64 json:"final_grade", OverrideReason string json:"override_reason,omitempty"); all structs use `json:` tags matching the contract shapes in `contracts/detection-engine.md`
- [X] T005 [P] Create `internal/detection/repository.go` with `DetectionRepository{db *pgxpool.Pool}`, `NewDetectionRepository(db) *DetectionRepository`, and these methods: `InsertHeuristic(ctx, sessionID uuid.UUID, score int, flagged []string, eventCount int) (uuid.UUID, error)` — INSERT INTO analysis_results (session_id, analysis_type, score, flagged_categories, event_count) VALUES ($1,'heuristic',$2,$3,$4) RETURNING id; `InsertLLMResult(ctx, sessionID uuid.UUID, advisoryText, suggestedRange, backendUsed, llmStatus string) (uuid.UUID, error)` — INSERT INTO analysis_results (session_id, analysis_type, advisory_text, suggested_grade_range, backend_used, llm_status) VALUES ($1,'llm',$2,$3,$4,$5) RETURNING id; `HeuristicExists(ctx, sessionID uuid.UUID) (bool, error)` — SELECT COUNT(1) FROM analysis_results WHERE session_id=$1 AND analysis_type='heuristic'; `LLMExists(ctx, sessionID uuid.UUID) (bool, error)` — same for 'llm'; `GetAnalysisResults(ctx, sessionID uuid.UUID) (*HeuristicResult, *LLMResult, error)` — SELECT all columns FROM analysis_results WHERE session_id=$1 ORDER BY analysis_type, return both rows or nil per type; `GetGradeDecision(ctx, sessionID uuid.UUID) (*GradeDecisionResult, error)` — SELECT final_grade, suggested_grade_range, override_reason, decided_by, decided_at FROM grade_decisions WHERE session_id=$1; `GradeDecisionExists(ctx, sessionID uuid.UUID) (bool, error)` — SELECT COUNT(1) FROM grade_decisions WHERE session_id=$1; `InsertGradeDecision(ctx, sessionID uuid.UUID, analysisResultID uuid.UUID, decidedBy uuid.UUID, finalGrade float64, suggestedRange string, overrideReason string) error` — INSERT INTO grade_decisions; `GetSessionForDetection(ctx, sessionID uuid.UUID) (actividadID uuid.UUID, submissionID uuid.UUID, studentID uuid.UUID, err error)` — SELECT actividad_id, submission_id, student_id FROM exam_sessions WHERE id=$1; `GetSubmissionStorageKey(ctx, submissionID uuid.UUID) (storageKey string, originalFilename string, err error)` — SELECT storage_key, original_filename FROM submissions WHERE id=$1; `GetCurricularScope(ctx, actividadID uuid.UUID) (carrera string, codigoMateria string, semestre int, err error)` — SELECT c.carrera, c.codigo_materia, c.semestre FROM actividades a JOIN cursos c ON c.id = a.curso_id WHERE a.id=$1; `GetDocenteScope(ctx, actividadID uuid.UUID) (docenteID uuid.UUID, cursoID uuid.UUID, err error)` — SELECT c.docente_id, c.id FROM actividades a JOIN cursos c ON c.id = a.curso_id WHERE a.id=$1; `GetTraceEvents(ctx, sessionID uuid.UUID) ([]tracing.TraceEvent, error)` — SELECT id, session_id, sequence_num, event_type, received_at, metadata, event_hash FROM trace_events WHERE session_id=$1 ORDER BY sequence_num ASC; import `tracing "github.com/Elder2003/nextlab-api/internal/tracing"`; `VerifyDocenteOwnsSession(ctx, sessionID uuid.UUID, docenteID uuid.UUID) (bool, error)` — SELECT COUNT(1) FROM exam_sessions es JOIN actividades a ON a.id = es.actividad_id JOIN cursos c ON c.id = a.curso_id WHERE es.id=$1 AND c.docente_id=$2

**Checkpoint**: Package compiles (`go build ./internal/detection/...`); no HTTP handlers yet.

---

## Phase 3: User Story 1 — Worker Runs Heuristic Detection (Priority: P1) 🎯 MVP

**Goal**: When the M5 worker picks up a `session.closed` job, it runs the heuristic engine against the session's trace events and stores a `heuristic` row in `analysis_results` — this MUST complete before any LLM attempt.

**Independent Test**: Submit an exam with 5+ tab_blur and 5+ clipboard_copy events. Wait ≤3 minutes. Verify `analysis_results` has one `analysis_type='heuristic'` row with `score > 0` and `flagged_categories` containing `tab_activity` and `clipboard_activity` — independent of any LLM call (quickstart Steps 3–4).

- [X] T006 [US1] Create `internal/detection/heuristic.go` with `HeuristicEngine{TabWeight, ClipWeight, GapWeight, HighThreshold int}`, `NewHeuristicEngine(cfg *config.Config) *HeuristicEngine` (reads the 4 fields added in T003), and `Score(events []tracing.TraceEvent) (score int, flagged []string)` method: count `tab_blur` events → `tabCount`; count `clipboard_copy` + `clipboard_paste` events → `clipCount`; compute `heartbeat_gaps` by iterating heartbeat events in sequence_num order and counting gaps where `received_at` difference > 30 seconds → `gapCount`; compute `raw = tabCount×TabWeight + clipCount×ClipWeight + gapCount×GapWeight`; `score = min(100, raw)`; build `flagged` slice: append `"tab_activity"` if tabCount>0, `"clipboard_activity"` if clipCount>0, `"heartbeat_gap"` if gapCount>0; return score, flagged; import `tracing "github.com/Elder2003/nextlab-api/internal/tracing"`
- [X] T007 [US1] Create `internal/detection/service.go` with `DetectionService{repo *DetectionRepository, heuristic *HeuristicEngine, aiSvc *ai.Service, ragSvc *rag.RetrieveService, minio *minio.Client, minioBucket string, cfg *config.Config}`, `NewDetectionService(repo, heuristic, aiSvc, ragSvc, minioClient, minioBucket, cfg) *DetectionService` constructor, and `RunHeuristic(ctx context.Context, sessionID uuid.UUID) error` method: (1) call repo.HeuristicExists(ctx, sessionID) — if true return nil (idempotent FR-012); (2) call repo.GetSessionForDetection to get actividadID, submissionID, studentID — if session not found return fmt.Errorf("session_not_found"); (3) call repo.GetTraceEvents(ctx, sessionID) to fetch all trace events; (4) call s.heuristic.Score(events) to get score and flagged categories; (5) call repo.InsertHeuristic(ctx, sessionID, score, flagged, len(events)); log at INFO level with session_id, score, flagged; import `"github.com/Elder2003/nextlab-api/internal/ai"`, `"github.com/Elder2003/nextlab-api/internal/rag"`, `"github.com/minio/minio-go/v7"`
- [X] T008 [US1] Rewrite `internal/detection/module.go` stub: define `WorkerHandler{svc *DetectionService}` struct and `NewWorkerHandler(svc *DetectionService) *WorkerHandler`; implement `Run(ctx context.Context, sessionID uuid.UUID, submissionID uuid.UUID) error` that (1) calls svc.RunHeuristic(ctx, sessionID) and logs any error without returning (partial-result guarantee — FR-002); (2) calls `svc.RunLLM(ctx, sessionID)` as a stub that returns nil for now (will be implemented in T010); add `RegisterRoutes` stub function `func RegisterRoutes(...) {}` (will be filled in T014); package must compile after this task
- [X] T009 [US1] Extend `cmd/worker/main.go` to construct and inject detection dependencies: (1) construct `*minio.Client` using `minio.New(cfg.MinioEndpoint, &minio.Options{Creds: credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""), Secure: cfg.MinioUseSSL})` — use the same pattern already in `cmd/api/main.go` line 87–94; (2) construct `*ai.Service` via `ai.NewService(ai.NewRepository(pgPool), cfg)` — use the same imports as `cmd/api/main.go`; (3) construct `*rag.RetrieveService` via `rag.NewRetrieveService(rag.NewCurricularRepository(pgPool), rag.NewDocenteRepository(pgPool), rag.NewEmbedClient(cfg), cfg)`; (4) construct `*detection.DetectionRepository`, `*detection.HeuristicEngine`, `*detection.DetectionService`, `*detection.WorkerHandler`; (5) replace the placeholder `processJob` body with `return workerHandler.Run(ctx, uuid.MustParse(sessionIDStr), uuid.MustParse(submissionIDStr))` where submissionIDStr comes from `values["submission_id"].(string)`; add necessary imports for `detection`, `ai`, `rag`, `minio`, `credentials`

**Checkpoint**: Submit an exam; wait ≤3 minutes; `SELECT * FROM analysis_results WHERE analysis_type='heuristic'` returns a row (quickstart Steps 3–4 pass).

---

## Phase 4: User Story 2 — Worker Runs LLM Analysis with RAG Context (Priority: P2)

**Goal**: After heuristics succeed, the worker builds a RAG-enriched prompt and calls the M6 AI router, storing the `llm` row in `analysis_results`. On failure it stores `llm_status='failed'` — the heuristic row is never touched.

**Independent Test**: Submit an exam with AI router functional. Verify `analysis_results` has a `analysis_type='llm'` row with non-null `advisory_text`, `suggested_grade_range`, `backend_used`, `llm_status='success'` (quickstart Steps 6–8).

- [X] T010 [US2] Add `RunLLM(ctx context.Context, sessionID uuid.UUID) error` method to `internal/detection/service.go`: (1) call repo.LLMExists(ctx, sessionID) — if true return nil (idempotent); (2) call repo.GetSessionForDetection to get actividadID, submissionID; (3) retrieve submission text: call repo.GetSubmissionStorageKey(ctx, submissionID) to get storageKey+originalFilename; call s.minio.GetObject(ctx, s.minioBucket, storageKey, minio.GetObjectOptions{}) → read all bytes → call rag.ExtractText(data, strings.ToLower(filepath.Ext(originalFilename))) — on any MinIO/extraction error set submissionText = "" and log a warning; (4) call repo.GetCurricularScope(ctx, actividadID) to get carrera, codigoMateria, semestre — ignore errors (scope may not exist); (5) call s.ragSvc.QueryCurricular(ctx, rag.CurricularQueryRequest{Carrera: carrera, Semestre: semestre, CodigoMateria: codigoMateria, Query: submissionText, ActividadID: &actividadID}) — on error use empty RetrievalResponse; (6) call repo.GetDocenteScope(ctx, actividadID) to get docenteID, cursoID; (7) call s.ragSvc.QueryDocente(ctx, rag.DocenteQueryRequest{DocenteID: docenteID, CursoID: cursoID, ActividadID: &actividadID, Query: submissionText}) — on error use empty RetrievalResponse; (8) fetch the heuristic result via repo.GetAnalysisResults to get score+flagged+eventCount; (9) build the LLM prompt per the structure in `specs/008-detection-engine/research.md` Decision 8: system message labels output advisory; user message sections: `## Submission Text` (submissionText or "(file content unavailable)"), `## Behavioral Analysis` (score/flagged/eventCount), `## Curricular Context` (join chunk texts from curricular response.Chunks), `## Rubric (full, untruncated)` (join all rubric texts from curricular response.Rubrics — Principle V), `## Task` (JSON format instruction); (10) call s.aiSvc.Dispatch(ctx, ai.DispatchRequest{Messages: []ai.ChatMessage{{Role:"system",Content:systemMsg},{Role:"user",Content:userMsg}}}, cfg.AIPrimaryTimeoutSecs) — if Dispatch fails, call repo.InsertLLMResult with llm_status='failed' and return nil (FR-010); (11) parse response JSON `{"advisory_text":"...","suggested_grade_range":"..."}` using json.Unmarshal — on parse failure use the raw content as advisory_text and "unknown" as suggested_grade_range; (12) call repo.InsertLLMResult(ctx, sessionID, advisoryText, suggestedRange, chatResp.BackendUsed or aiLog.Backend, "success"); add imports for `"encoding/json"`, `"io"`, `"path/filepath"`, `"strings"`, `"github.com/minio/minio-go/v7"`, `"github.com/Elder2003/nextlab-api/internal/rag"`
- [X] T011 [US2] Update `Run` in `internal/detection/module.go` to replace the stub `svc.RunLLM` call with the real call: after `svc.RunHeuristic` completes (even if it errored), call `if err := svc.RunLLM(ctx, sessionID); err != nil { slog.ErrorContext(ctx, "llm analysis failed", "session_id", sessionID, "err", err) }` — LLM failure must never propagate to the job-level error (Principle II)

**Checkpoint**: Submit an exam; `SELECT analysis_type, llm_status FROM analysis_results WHERE session_id = $1` returns 2 rows; LLM row has llm_status='success' (quickstart Steps 6–7 pass).

---

## Phase 5: User Story 3 — Docente Views Detection Report (Priority: P3)

**Goal**: Two HTTP endpoints: `GET /sessions/:id/detection-report` returns the full report; `POST /sessions/:id/grade` stores grade with mandatory override reason when outside suggested range.

**Independent Test**: After a session is fully processed, as a docente: GET report → 200 with heuristic+LLM sub-reports. POST grade within range → 201 no reason needed. POST grade outside range without reason → 422. POST grade outside range with reason → 201 (quickstart Steps 9–13).

- [X] T012 [P] [US3] Add `GetReport(ctx context.Context, sessionID uuid.UUID, docenteID uuid.UUID) (*DetectionReport, error)` and `SubmitGrade(ctx context.Context, sessionID uuid.UUID, docenteID uuid.UUID, req GradeRequest) (*GradeDecisionResult, error)` methods to `internal/detection/service.go`: `GetReport` implementation: (1) call repo.VerifyDocenteOwnsSession(ctx, sessionID, docenteID) — return ErrForbidden if false; (2) call repo.GetSessionForDetection to get studentID and actividadID; (3) call repo.GetAnalysisResults(ctx, sessionID) to get heuristic+llm rows — if heuristic is nil, return ErrNotFound ("detection report not yet available"); (4) call repo.GetGradeDecision(ctx, sessionID) — nil is ok (not yet graded); (5) set llmStatus = "success"/"failed"/"unavailable" based on llm row presence; return populated DetectionReport; `SubmitGrade` implementation: (1) call repo.VerifyDocenteOwnsSession — return ErrForbidden if false; (2) call repo.GradeDecisionExists — return ErrConflict if true; (3) call repo.GetAnalysisResults to get LLM row (for analysisResultID and suggestedRange); (4) apply override detection per research.md Decision 9: if suggestedRange contains "–" or "-", parse lo/hi as floats; if req.FinalGrade < lo−5 or > hi+5 AND req.OverrideReason is empty, return ErrOverrideRequired; (5) call repo.InsertGradeDecision(ctx, sessionID, llmRow.ID, docenteID, req.FinalGrade, suggestedRange, req.OverrideReason); return GradeDecisionResult; define sentinel errors: `var ErrForbidden = errors.New("forbidden")`, `ErrNotFound = errors.New("not_found")`, `ErrConflict = errors.New("conflict")`, `ErrOverrideRequired = errors.New("override_reason_required")` in model.go
- [X] T013 [US3] Create `internal/detection/handler.go` with `reportHandler(svc *DetectionService) fiber.Handler` and `gradeHandler(svc *DetectionService) fiber.Handler`: `reportHandler`: (1) parse sessionID from c.Params("sessionID") as uuid.UUID — return 400 on parse error; (2) extract docenteID from c.Locals("user_id") using the same userIDFromLocals pattern (copy from `internal/rag/handler_ingest.go`); (3) call svc.GetReport(c.Context(), sessionID, docenteID); (4) map sentinel errors: ErrForbidden→403, ErrNotFound→404; (5) return 200+JSON; `gradeHandler`: (1) parse sessionID; (2) extract docenteID; (3) parse JSON body into GradeRequest — return 400 if FinalGrade missing or zero; (4) call svc.SubmitGrade(c.Context(), sessionID, docenteID, req); (5) map errors: ErrForbidden→403, ErrConflict→409, ErrNotFound→404, ErrOverrideRequired→422 with message including the grade and range; (6) return 201+JSON of GradeDecisionResult; use response shapes from `contracts/detection-engine.md`
- [X] T014 [US3] Update `internal/detection/module.go` `RegisterRoutes` function to: accept `(app *fiber.App, db *pgxpool.Pool, minioClient *minio.Client, cfg *config.Config, aiSvc *ai.Service, ragSvc *rag.RetrieveService, authMW fiber.Handler, roleGuard func(...string) fiber.Handler)`; construct DetectionRepository, HeuristicEngine, DetectionService; mount `GET /sessions/:sessionID/detection-report` and `POST /sessions/:sessionID/grade` under `authMW + roleGuard("docente")`; add call to `detection.RegisterRoutes(app, pgPool, minioClient, cfg, aiSvc, ragSvc, auth.AuthMiddleware(cfg), auth.RoleGuard)` in `cmd/api/main.go` after the existing module registrations (NOTE: `ai.Service` must be constructed before calling detection.RegisterRoutes — extract it from the existing `ai.RegisterRoutes` call or construct it separately using `ai.NewService(ai.NewRepository(pgPool), cfg)`)

**Checkpoint**: GET /sessions/:id/detection-report returns 200 with full report; POST /sessions/:id/grade 201/409/422 all work (quickstart Steps 9–13 pass).

---

## Phase 6: User Story 4 — Student Cannot Access Detection Report (Priority: P4)

**Goal**: Student role is blocked at route middleware before any DB query — 0% data leakage.

**Independent Test**: Authenticate as student; call GET + POST detection endpoints for own session → 403 both times (quickstart Steps 14–15).

- [X] T015 [US4] Verify in `internal/detection/module.go` that the route group uses `roleGuard("docente")` which already blocks the `estudiante` role (check `internal/auth/middleware.go` for the RoleGuard implementation to confirm role name constants); verify the student role name used in the DB/JWT matches what RoleGuard checks — co_docente should also pass if it maps to the "docente" guard (cross-check against how `internal/rag/module.go` mounts docente routes); no code change needed if already correct — document finding in a code comment; if role name mismatch found, fix the roleGuard call in module.go

**Checkpoint**: Student JWT receives 403 on both detection endpoints before any DB query (quickstart Steps 14–15 pass).

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Environment wiring, build validation, and end-to-end quickstart verification.

- [X] T016 [P] Update `docker-compose.yml` api and worker service environment blocks to add: `HEURISTIC_TAB_WEIGHT: "5"`, `HEURISTIC_CLIP_WEIGHT: "3"`, `HEURISTIC_GAP_WEIGHT: "2"`, `HEURISTIC_HIGH_THRESHOLD: "60"` — add under a `# Detection Engine (M8)` comment in both service blocks
- [X] T017 [P] Run `go build ./...` from repo root and fix any compilation errors in `internal/detection/`, `internal/config/`, `cmd/worker/main.go`, or `cmd/api/main.go`
- [ ] T018 Run quickstart.md verification steps 1–18 against a live Docker Compose stack and confirm all expected responses

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on T001 (migration must exist before testing compiles); T003 (config fields) required by T006 (heuristic engine) — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 (T003–T005); T006 → T007 → T008 → T009 (sequential)
- **US2 (Phase 4)**: Depends on T007 (service.go must exist); T010 extends service.go; T011 extends module.go
- **US3 (Phase 5)**: Depends on Phase 2 (repository must have GetReport-related methods); T012 [P] can start after T005; T013 depends on T012; T014 depends on T013
- **US4 (Phase 6)**: Depends on T014 (routes must be mounted); T015 is a verification task
- **Polish (Phase 7)**: Depends on all phases; T016 and T017 can run in parallel

### User Story Dependencies

- **US1 (P1)**: After Phase 2 — the MVP; blocks US2 (service.go created in T007)
- **US2 (P2)**: After T007 (extends service.go RunLLM); otherwise independent of US3/US4
- **US3 (P3)**: After Phase 2 — T012 can start once T005 exists; T013/T014 depend on T012; independent of US1/US2 logic (reads already-stored rows)
- **US4 (P4)**: After T014 (routes must exist); verification only

### Within Each User Story

- T003 → T004 [P], T005 [P] → T006 [US1] → T007 [US1] → T008 [US1] → T009 [US1]
- T007 → T010 [US2] → T011 [US2]
- T005 → T012 [P] [US3] → T013 [US3] → T014 [US3] → T015 [US4]

### Parallel Opportunities

| Phase | Parallel tasks |
|---|---|
| Phase 1 | T001 sequential (T002 can parallel after DB file is created) |
| Phase 2 | T004, T005 (different files, both depend on T003 config) |
| Phase 5 | T012 can start after T005 while US1/US2 worker tasks continue |
| Phase 7 | T016, T017 (independent) |

---

## Parallel Example: US1 + US3 in parallel

```
After T005 (repository) completes:
  Worker path:   T006 → T007 → T008 → T009  (US1 heuristic + worker)
  HTTP path:     T012 [P] can begin         (US3 service methods for report/grade)

After T007 completes:
  T010 → T011  (US2 LLM extension of service.go)

After T013 completes:
  T014  (RegisterRoutes + main.go wiring)
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Setup (T001–T002)
2. Complete Phase 2: Foundational (T003–T005)
3. Complete Phase 3: US1 (T006–T009)
4. **STOP and VALIDATE**: Heuristic row appears in DB within 3 minutes of submission (quickstart Steps 3–4 pass)
5. Proceed to US2, US3, US4

### Incremental Delivery

1. T001–T005 → Package compiles, migration ready
2. T006–T009 → Heuristic detection running (MVP — analysis always stored)
3. T010–T011 → LLM analysis running (full detection pipeline)
4. T012–T014 → Detection report endpoints live
5. T015 → Student access denied confirmed
6. T016–T018 → Full quickstart validation

---

## Notes

- `internal/detection/module.go` is currently a stub (`package detection`) — T008 rewrites it and T014 extends it
- The `ai.Service` instance needed by detection can be constructed with `ai.NewService(ai.NewRepository(pgPool), cfg)` — the AI package exposes these constructors; verify exported names before implementing T009 and T014
- `GetTraceEvents` in the detection repository queries the `trace_events` table directly — no cross-package coupling beyond the `tracing.TraceEvent` type imported for the return type
- Constitution Principle I compliance: `repo.InsertHeuristic` and `repo.InsertLLMResult` are INSERT only; no UPDATE path; `InsertGradeDecision` is INSERT only with UNIQUE(session_id) enforcing one decision per session
- Constitution Principle V compliance: `RunLLM` builds the prompt with rubrics from `response.Rubrics` (full, untruncated) and chunks from `response.Chunks` separately — rubrics are never mixed into the chunk list
- T015 is intentionally lightweight — US4 is enforced at the route-mounting level by `roleGuard("docente")`, not by detection-specific code; the task verifies correctness, not implementation
