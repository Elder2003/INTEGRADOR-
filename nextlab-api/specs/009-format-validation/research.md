# Research: M9 — Format Validation

## Decision 1: Migration strategy for `format_rules`

**Decision**: Add a nullable JSONB column `format_rules` to the existing `actividades` table via `ALTER TABLE` in the M9 migration — no separate `activity_format_rules` table.

**Rationale**: The four mechanical rules (`min_pages`, `max_pages`, `min_word_count`, `required_types`) are a small, cohesive configuration object that logically belongs to the activity. A JSONB column avoids a JOIN on every activity read, keeps the academic module's existing repository functions largely unchanged, and fits the existing M4 pattern of extending `actividades` via `ALTER TABLE` (`late_submission_allowed` was added the same way).

**Alternatives considered**: Separate `activity_format_rules` table with FK to `actividades(id)`. Rejected because it adds a JOIN to a frequently-read table and requires a new repository layer for a simple config object.

---

## Decision 2: PDF page count extraction

**Decision**: Use `pdfapi.PageCountFile` from `github.com/pdfcpu/pdfcpu/pkg/api` (same package already imported in `internal/rag/extractor.go`). Write the PDF bytes to a temp file, call `PageCountFile(path, nil)`, clean up temp file.

**Rationale**: pdfcpu v0.12.1 is already a dependency (go.mod line 13). `PageCountFile` returns `(int, error)` — the exact primitive needed. No new dependency required.

**Alternatives considered**: Counting `%Page` markers manually in raw PDF bytes — fragile and inaccurate for PDFs with cross-reference streams. Rejected.

---

## Decision 3: DOCX page count extraction

**Decision**: Parse `docProps/app.xml` inside the DOCX ZIP archive. This file contains an `<Pages>` element with the page count as reported by the authoring application. Fall back to `0` if the element is absent (e.g., LibreOffice sometimes omits it).

**Rationale**: DOCX is a ZIP. `docProps/app.xml` is a standard OOXML component that records document statistics including page count, word count, and character count — exactly the metadata fields needed. The existing `extractDocx` function in `internal/rag/extractor.go` already opens the ZIP with `archive/zip`, so the same pattern applies. No new dependency.

**Alternatives considered**: Counting `<w:lastRenderedPageBreak>` elements in `word/document.xml` — unreliable across editors; not always present. Rejected.

---

## Decision 4: DOCX word/char count

**Decision**: Parse `docProps/app.xml` for `<Words>` and `<Characters>` elements. These are set by the authoring application and are the most reliable source for word/char counts without re-running the document layout engine.

**Rationale**: Same ZIP parsing logic as Decision 3. Already present as part of the OOXML standard. Avoids re-implementing a tokeniser.

**Alternatives considered**: Count words by tokenising extracted text (split on whitespace). Kept as fallback when `docProps/app.xml` is absent.

---

## Decision 5: File type detection

**Decision**: Detect file type by magic bytes, not by filename extension. PDF magic: first 4 bytes = `%PDF`. DOCX/ZIP magic: first 4 bytes = `PK\x03\x04`. Then confirm DOCX vs. generic ZIP by checking for `[Content_Types].xml` in the ZIP central directory.

**Rationale**: Students may submit files with incorrect extensions. Relying on extension would produce misleading `file_type` values. Magic-byte detection is O(1) and requires no additional library.

**Alternatives considered**: `net/http.DetectContentType` — only returns `application/zip` for DOCX, not a specific MIME type. Not specific enough to distinguish DOCX from arbitrary ZIP. Rejected as primary method (kept as secondary hint).

---

## Decision 6: `format_reports` table placement

**Decision**: Create a new `format_reports` table in migration `000009_format_validation.up.sql` with `UNIQUE(submission_id)` — one report per submission file (idempotent, append-only per Principle I).

**Rationale**: Sessions have exactly one associated `submission_id` (from `exam_sessions.submission_id`). A UNIQUE constraint on `submission_id` enforces idempotency without application-layer duplication checks. The append-only constraint comes from the migration design (no UPDATE/DELETE grants needed).

**Alternatives considered**: UNIQUE on `(session_id, submission_id)` — redundant because a session already has at most one submission_id. UNIQUE on `session_id` alone — incorrect since versioned re-uploads (same session, new submission row) should each get their own report.

---

## Decision 7: Worker integration point

**Decision**: Add `RunFormatValidation(ctx, sessionID)` as the first phase in `WorkerHandler.Run`, executed before `RunHeuristic`. Format validation and heuristic analysis are independent; format validation runs first because it is faster and its result is the most immediately useful to the docente.

**Rationale**: The worker already runs phases sequentially (RunHeuristic then RunLLM per M8). Adding format validation as phase 0 extends the same sequential pattern. Running it first means a docente checking the report while heuristics are still computing already sees the format result.

**Alternatives considered**: Run format validation and heuristic in parallel (goroutines). Rejected for M9 — the sequential pattern is simpler, and the 3-minute SLA allows sequential execution with margin.

---

## Decision 8: Detection report extension

**Decision**: Extend the existing `GET /sessions/:sessionID/detection-report` response (M8) with a nullable `format_report` sub-object. The detection report repository's `GetReport` method is extended to fetch the format report via `GetFormatReport(ctx, submissionID)`. The M8 `DetectionReport` struct gets a new `FormatReport *FormatReport` field.

**Rationale**: The spec explicitly states no new endpoint is needed (FR-009). Extending the existing endpoint keeps the docente's workflow in one call. The `submissionID` is already available in `GetSessionForDetection`.

**Alternatives considered**: A separate `GET /sessions/:id/format-report` endpoint — adds URL surface and requires the docente client to make two calls. Rejected per FR-009.

---

## Decision 9: Academic handler extension

**Decision**: Extend `CreateActividadRequest` and `UpdateActividadRequest` in `internal/academic/model.go` with an optional `FormatRules *FormatRulesConfig` field. Add `FormatRulesConfig` struct: `MinPages *int`, `MaxPages *int`, `MinWordCount *int`, `RequiredTypes []string`. Persist as JSONB via `json.Marshal`/`json.Unmarshal`.

**Rationale**: The `CreateActividad` and `UpdateActividad` repository functions need one additional nullable parameter. JSONB serialization is the simplest path: no schema migration beyond the new column, no new repository methods beyond a nullable column read/write.

**Alternatives considered**: Flat columns (`min_pages INT`, `max_pages INT`, etc.) — would require more migration columns and more nullable parameters in the function signatures. JSONB is simpler for an optional, rarely-queried config object.

---

## Decision 10: `parse_status` and `fail_reason` design

**Decision**: `parse_status` is an enum-like TEXT column: `'success'` or `'failed'`. `fail_reason` is a nullable TEXT column holding a machine-readable slug for the failure cause: `'password_protected'`, `'unsupported_file_type'`, `'empty_file'`, `'corrupt_file'`, `'no_submission'`. When `parse_status = 'success'`, `fail_reason` is NULL.

**Rationale**: A machine-readable slug allows clients to display localised messages and programmatically distinguish failure modes. A free-text description alone would make programmatic handling fragile.

**Alternatives considered**: A boolean `success` column + a free-text `error` column — less structured; harder to query. Rejected.

---

## Decision 11: No submission scenario

**Decision**: If `session.submission_id` is NULL (student never uploaded a file), the format validation worker step immediately stores a `format_reports` row with `parse_status = 'failed'`, `fail_reason = 'no_submission'`, and `compliant = false`.

**Rationale**: The detection report endpoint must always return a consistent shape. A `NULL` format report field would force the client to distinguish "not yet computed" from "no file was submitted". Storing an explicit `no_submission` record makes the state deterministic.

**Alternatives considered**: Skip format validation entirely when submission_id is NULL and return `null` in the API response — acceptable but less informative. Rejected to keep the report shape consistent.
