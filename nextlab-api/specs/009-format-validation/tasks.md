# Tasks: M9 — Format Validation

**Input**: Design documents from `specs/009-format-validation/`

**Prerequisites**: plan.md ✓ spec.md ✓ research.md ✓ data-model.md ✓ contracts/ ✓ quickstart.md ✓

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks in the same phase)
- **[Story]**: Which user story this task belongs to (US1 = P1, US2 = P2, US3 = P3, US4 = P4)
- Exact file paths are given in every description

---

## Phase 1: Setup — Migrations

**Purpose**: Create the database schema for M9 before any Go code touches it.

- [X] T001 Create `migrations/000009_format_validation.up.sql` — `CREATE TABLE IF NOT EXISTS format_reports` with columns `id`, `submission_id` (FK), `session_id` (FK), `parse_status`, `fail_reason`, `file_type`, `page_count`, `word_count`, `char_count`, `doc_metadata JSONB`, `compliant`, `rule_results JSONB`, `created_at`; `UNIQUE(submission_id)`; index on `session_id`; `ALTER TABLE actividades ADD COLUMN IF NOT EXISTS format_rules JSONB`
- [X] T002 Create `migrations/000009_format_validation.down.sql` — `DROP TABLE IF EXISTS format_reports`; `ALTER TABLE actividades DROP COLUMN IF EXISTS format_rules`

---

## Phase 2: Foundational — Core Types

**Purpose**: Define Go structs that every user-story phase imports. MUST be complete before Phases 3–6.

**⚠️ CRITICAL**: T003 must finish before T004 or T005 can start (they import the format package).

- [X] T003 Create `internal/format/model.go` — define `FormatReport` struct (id, submission_id, session_id, parse_status, fail_reason, file_type, page_count, word_count, char_count, doc_metadata `map[string]any`, compliant, rule_results `[]RuleResult`, created_at), `RuleResult` struct (rule, status, actual `any`, required `any`), `FormatRulesConfig` struct (min_pages `*int`, max_pages `*int`, min_word_count `*int`, required_types `[]string`), and `FileInfo` struct (file_type string, page_count int, word_count int, char_count int, doc_metadata `map[string]any`) — all with JSON tags matching data-model.md
- [X] T004 [P] Extend `internal/detection/model.go` — add `FormatReport *format.FormatReport \`json:"format_report"\`` field to `DetectionReport` struct; add import `"nextlab-api/internal/format"` (depends on T003)
- [X] T005 [P] Extend `internal/academic/model.go` — add `FormatRules *format.FormatRulesConfig \`json:"format_rules,omitempty"\`` field to both `CreateActividadRequest` and `UpdateActividadRequest` structs; add import `"nextlab-api/internal/format"` (depends on T003)

**Checkpoint**: All struct types defined — user story phases can now proceed.

---

## Phase 3: User Story 1 — Worker Validates Format (Priority: P1) 🎯 MVP

**Goal**: Async worker fetches the submitted file from MinIO, detects its type, extracts structural metadata (page count, word count, character count, doc metadata), and stores a `format_reports` row — regardless of whether any format rules are configured. Submissions are never blocked.

**Independent Test**: Close an exam session that has a PDF submission, wait up to 3 minutes, then `SELECT parse_status, file_type, page_count, word_count FROM format_reports WHERE session_id = '<id>'` — expect one row with `parse_status = 'success'`. No format rules needed for this test.

- [X] T006 [US1] Implement `internal/format/extractor.go` — define `ParseFile(data []byte) (FileInfo, error)` that: (1) detects file type via magic bytes (`%PDF` prefix → pdf; `PK\x03\x04` + contains `[Content_Types].xml` in ZIP directory → docx; empty → fail `empty_file`; else → fail `unsupported_file_type`); (2) for PDF: write bytes to `os.CreateTemp`, call `pdfapi.PageCountFile(path, nil)` from `github.com/pdfcpu/pdfcpu/pkg/api`, parse word/char counts by tokenising extracted text (split on whitespace); (3) for DOCX: open `archive/zip.NewReader`, read `docProps/app.xml`, unmarshal `<Pages>`, `<Words>`, `<Characters>`, `<Author>`, `<Created>` fields using `encoding/xml`; (4) return populated `FileInfo`; password-protected PDFs produce a pdfcpu error — detect and return fail reason `password_protected`
- [X] T007 [US1] Implement `internal/format/repository.go` — define `Repository` struct with pgx pool; implement `InsertFormatReport(ctx, *FormatReport) error` using `INSERT INTO format_reports ... ON CONFLICT (submission_id) DO NOTHING` for idempotency; implement `GetFormatReportBySubmissionID(ctx, submissionID uuid.UUID) (*FormatReport, error)` that unmarshals `doc_metadata` and `rule_results` JSONB columns; implement `GetFormatReportBySessionID(ctx, sessionID uuid.UUID) (*FormatReport, error)`
- [X] T008 [US1] Implement `internal/format/service.go` — define `FormatService` struct with fields `repo *Repository`, `minioClient *minio.Client`, `bucket string`; implement `Run(ctx context.Context, sessionID uuid.UUID) error` that: (1) queries `exam_sessions` to get `submission_id` and `actividad_id`; (2) if `submission_id` IS NULL, calls `InsertFormatReport` with `parse_status='failed'`, `fail_reason='no_submission'`, `compliant=false`, `rule_results=[]`, and returns; (3) queries `submissions` to get `storage_key`; (4) fetches file bytes from MinIO using `GetObject`; (5) calls `ParseFile`; (6) on parse error, inserts failed report; (7) on success, inserts report with `parse_status='success'`, `compliant=true` (rule evaluation added in T014), `rule_results=[]`
- [X] T009 [US1] Create `internal/format/module.go` — implement `NewRepository(db *pgxpool.Pool) *Repository` and `NewFormatService(repo *Repository, minioClient *minio.Client, bucket string) *FormatService` constructors; no HTTP routes registered
- [X] T010 [US1] Modify `internal/detection/module.go` — add `fmtSvc *format.FormatService` field to `WorkerHandler` struct; update `NewWorkerHandler` signature to accept `fmtSvc *format.FormatService`; in `WorkerHandler.Run`, call `w.fmtSvc.Run(ctx, sessionID)` as the first step (phase 0) before `w.svc.RunHeuristic` — log warning on error but do not abort remaining phases
- [X] T011 [US1] Modify `cmd/worker/main.go` — construct `format.Repository` and `format.FormatService` using the existing `minioClient` and `cfg.MinioBucket`; update `detection.NewWorkerHandler` call to pass the `FormatService`

**Checkpoint**: Worker now produces a format_reports row for every processed session. US1 independently testable.

---

## Phase 4: User Story 2 — Format Report Checked Against Rules (Priority: P2)

**Goal**: The worker evaluates each configured format rule independently and stores per-rule pass/fail results. Activities without rules still produce a report with `compliant=true` and `rule_results=[]`.

**Independent Test**: Create an activity with `format_rules = {"min_pages":3,"required_types":["pdf"]}` directly in the DB. Submit a 1-page DOCX. After worker runs: `SELECT compliant, rule_results FROM format_reports WHERE session_id='<id>'` — expect `compliant=false`, rule_results contains entries for both `min_pages` (fail, actual=1) and `file_type` (fail, actual="docx").

- [X] T012 [US2] Implement `internal/format/validator.go` — define `EvaluateRules(info FileInfo, rules *FormatRulesConfig) ([]RuleResult, bool)` that iterates over each non-nil rule field and produces a `RuleResult` with `rule` name, `status` ("pass"/"fail"), `actual` value, and `required` value; overall `compliant` is `true` only if all rules pass; when `rules` is nil, returns `([]RuleResult{}, true)`; rules evaluated: `min_pages` (info.PageCount >= *rules.MinPages), `max_pages` (info.PageCount <= *rules.MaxPages), `min_word_count` (info.WordCount >= *rules.MinWordCount), `file_type` (info.FileType contained in rules.RequiredTypes)
- [X] T013 [US2] Add `GetFormatRulesForSession(ctx context.Context, sessionID uuid.UUID) (*format.FormatRulesConfig, error)` to `internal/format/repository.go` — `SELECT a.format_rules FROM actividades a JOIN exam_sessions es ON es.actividad_id = a.id WHERE es.id = $1`; unmarshal JSONB into `*FormatRulesConfig`; return `nil, nil` when column is NULL (no rules configured)
- [X] T014 [US2] Extend `internal/format/service.go` `Run` method — after `ParseFile` succeeds: call `repo.GetFormatRulesForSession(ctx, sessionID)`, call `validator.EvaluateRules(fileInfo, rules)`, and include the resulting `rule_results` and `compliant` flag in the `InsertFormatReport` call (replace the hardcoded `compliant=true, rule_results=[]` from T008)

**Checkpoint**: Format reports now include per-rule results and accurate compliant flag.

---

## Phase 5: User Story 3 — Docente Views Format Compliance Report (Priority: P3)

**Goal**: `GET /sessions/:sessionID/detection-report` response includes a `format_report` sub-object. When validation is pending (not yet run), `format_report` is `null`. When run, it contains all metadata and rule results. Access restricted to docentes (already enforced by existing RoleGuard).

**Independent Test**: After worker processes a session, call `GET /sessions/<id>/detection-report` with a docente JWT — expect `200` with `format_report` object containing `compliant`, `file_type`, `page_count`, `word_count`, `rule_results`. Call with a student JWT — expect `403`.

- [X] T015 [US3] Extend `internal/detection/repository.go` `GetReport` method — after populating the base `DetectionReport`, add a query: `SELECT id, submission_id, session_id, parse_status, fail_reason, file_type, page_count, word_count, char_count, doc_metadata, compliant, rule_results, created_at FROM format_reports WHERE session_id = $1`; if row found, unmarshal JSONB fields (doc_metadata, rule_results) and assign to `report.FormatReport`; if no row found (`pgx.ErrNoRows`), leave `report.FormatReport` as `nil` (serializes as `null`)
- [X] T016 [US3] Verify `internal/detection/handler.go` `report` handler serializes `format_report` field correctly — confirm `DetectionReport.FormatReport *format.FormatReport` with `json:"format_report"` tag appears as either a populated object or `null` in the JSON response (no code change expected if struct tag is correct from T004; add `omitempty` removal if needed to ensure `null` appears explicitly)

**Checkpoint**: Detection report endpoint returns format_report sub-object. Docente-only enforcement already in place from M8.

---

## Phase 6: User Story 4 — Docente Configures Format Rules (Priority: P4)

**Goal**: Docentes can set `format_rules` via `POST /cursos/:id/actividades` and `PUT /cursos/:id/actividades/:id`. Omitting the field leaves existing rules unchanged. Setting `format_rules: null` clears all rules. Invalid rule combinations return `422`.

**Independent Test**: `POST /cursos/<id>/actividades` with `format_rules: {"min_pages":2,"max_pages":10,"required_types":["pdf"]}` — expect `201` and `format_rules` in response. `GET` the activity — expect same rules. `PUT` with `format_rules: null` — expect rules cleared.

- [X] T017 [US4] Extend `internal/academic/repository.go` `CreateActividad` function — add `formatRules *format.FormatRulesConfig` parameter; marshal to JSONB (`json.Marshal`) when non-nil; include `format_rules` in INSERT statement (`$N` parameter, NULL when nil)
- [X] T018 [US4] Extend `internal/academic/repository.go` `UpdateActividad` function — add `formatRules **format.FormatRulesConfig` parameter (pointer-to-pointer to distinguish "omitted" from "explicit null"); when the outer pointer is nil, do not update `format_rules`; when outer pointer is non-nil and inner is nil, set `format_rules = NULL`; when both non-nil, marshal and update
- [X] T019 [US4] Extend `internal/academic/handler.go` `createActividad` handler — read `FormatRules` from the parsed `CreateActividadRequest`, validate it (return `422 invalid_format_rules` if `min_pages > max_pages`, any negative values, or `required_types` is present but empty), then pass to `repo.CreateActividad`
- [X] T020 [US4] Extend `internal/academic/handler.go` `updateActividad` handler — read `FormatRules` from the parsed `UpdateActividadRequest` using pointer detection to distinguish "field absent" from "field null", validate the same rules as T019, then pass to `repo.UpdateActividad`
- [X] T021 [US4] Extend GET actividad response in `internal/academic/repository.go` and handler — add `format_rules JSONB` to SELECT query for `GetActividad`/`GetActividades`; unmarshal into `*format.FormatRulesConfig`; include field in the response struct so docentes see the current rules
- [X] T022 [US4] Add `format_rules` field to actividad response struct in `internal/academic/model.go` — add `FormatRules *format.FormatRulesConfig \`json:"format_rules,omitempty"\`` to `ActividadResponse` (or whichever struct is serialized in GET responses)

**Checkpoint**: Docentes can configure rules via API; rules persist and appear in GET responses.

---

## Phase 7: Polish & Validation

**Purpose**: Verify clean compile and end-to-end correctness.

- [X] T023 Run `go build ./...` from repo root — fix any compile errors (import cycles, missing methods, type mismatches) before proceeding
- [X] T024 Apply M9 migration against local Postgres (`migrate -path migrations -database $DATABASE_URL up`) and run quickstart.md steps 1–14 against the live Docker Compose stack to verify the full pipeline end-to-end

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Migrations)**: No dependencies — start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 (migration files should exist before Go code references new tables) — BLOCKS all user story phases
  - T003 must complete before T004 and T005
  - T004 and T005 can run in parallel with each other after T003
- **Phase 3 (US1)**: Depends on Phase 2 completion (format/model.go and updated detection/model.go must exist)
  - T006, T007 can run in parallel (different files)
  - T008 depends on T006 + T007
  - T009 depends on T008
  - T010 depends on T009 + Phase 2 T004
  - T011 depends on T010
- **Phase 4 (US2)**: Depends on Phase 3 completion (service.go must exist to extend)
  - T012 and T013 can run in parallel
  - T014 depends on T012 + T013
- **Phase 5 (US3)**: Depends on Phase 2 T004 (detection/model.go extended) and Phase 3 T007 (format/repository.go exists)
  - T015 and T016 can run in parallel
- **Phase 6 (US4)**: Depends on Phase 2 T005 (academic/model.go extended)
  - T017, T018, T021, T022 can run in parallel
  - T019 depends on T017
  - T020 depends on T018
- **Phase 7 (Polish)**: Depends on all user story phases

### Parallel Opportunities

Within Phase 3 (US1):
```
# T006 and T007 can run in parallel:
Task: "Implement internal/format/extractor.go"
Task: "Implement internal/format/repository.go"
```

Within Phase 4 (US2):
```
# T012 and T013 can run in parallel:
Task: "Implement internal/format/validator.go"
Task: "Add GetFormatRulesForSession to internal/format/repository.go"
```

Within Phase 6 (US4):
```
# T017, T018, T021, T022 can run in parallel:
Task: "Extend CreateActividad in academic/repository.go"
Task: "Extend UpdateActividad in academic/repository.go"
Task: "Extend GET actividad query in academic/repository.go"
Task: "Add format_rules to ActividadResponse struct"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Migrations
2. Complete Phase 2: Core types (T003 → T004+T005)
3. Complete Phase 3: Worker format validation (T006–T011)
4. **STOP and VALIDATE**: Verify format_reports rows appear after session close
5. Ship US1 — docentes now always get a format report, even without rules

### Incremental Delivery

1. Setup + Foundational → schema and types ready
2. US1 → worker stores format metadata (no rules yet) → independently testable
3. US2 → add rule evaluation → reports now show pass/fail per rule
4. US3 → expose via detection-report endpoint → docentes can see it
5. US4 → docentes configure rules via API → full self-service

---

## Notes

- `format_reports.UNIQUE(submission_id)` enforces idempotency — `ON CONFLICT DO NOTHING` in InsertFormatReport is the correct pattern
- `pdfcpu` `PageCountFile` requires a real file path, not an io.Reader — write bytes to `os.CreateTemp` and clean up with `defer os.Remove`
- The `no_submission` case (session.submission_id IS NULL) must store a format_report row immediately — do not skip it — so the detection-report endpoint always returns a non-null format_report after worker runs
- Detection repository queries format_reports directly (same DB) — no inter-package service call needed; import `internal/format` only for the struct types
- For `UpdateActividad` format_rules: JSON body with `"format_rules": null` is distinct from omitting the field entirely — use `**FormatRulesConfig` or a wrapper struct to distinguish
- Do not add error handling for impossible states (e.g., session not found in format service — the worker job would not have been enqueued without a valid session)
