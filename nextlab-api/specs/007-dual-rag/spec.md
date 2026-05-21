# Feature Specification: M7 — Dual RAG Pipeline

**Feature Branch**: `007-dual-rag`

**Created**: 2026-05-20

**Status**: Draft

**Input**: User description: "Lets do the milestone 7"

## Clarifications

### Session 2026-05-20

- Q: Should M7 route embedding requests through the M6 AI Router or use a dedicated embedding endpoint? → A: M7 calls `{AI_EMBED_URL}/v1/embeddings` directly — both Vercel AI Gateway and Ollama support OpenAI-compatible `/v1/embeddings`. M7 has its own embedding config (`AI_EMBED_URL`, `AI_EMBED_MODEL`, `AI_EMBED_DIMENSION`) independent of M6. The M6 router handles only chat completions and is not involved in any M7 operation.
- Q: On re-ingest failure, should old chunks survive, accept partial write, or use a full transaction? → A: Full two-phase transaction in PostgreSQL — insert new chunks with `status='pending'`; only if ALL embeddings succeed, delete old `active` chunks and promote new ones to `active`; if any step fails, delete pending chunks so old active chunks remain intact, and return an error to the caller.
- Q: What uniquely identifies a document for deduplication (re-ingest targeting)? → A: System-assigned UUID (`document_id`) generated on first ingest and returned in the response. The caller stores and reuses this UUID to trigger a re-ingest replace. The UUID also forms the MinIO storage path: `rag-docs/{scope}/{document_id}/original.{ext}`.
- Q: Does M7 handle PDF and Word text extraction, or defer to M9? → A: M7 implements basic text extraction — PDF via pure-Go library (`pdfcpu`), Word (.docx) via ZIP/XML parsing — sufficient for embedding. M9 implements format compliance validation (structure, page limits, required sections), which is a different concern. Both coexist without duplication.
- Q: Is the RAG retrieval independent of which chat model is used? → A: Yes. RAG retrieval returns plain text chunks. Those chunks are injected into a prompt that can be processed by any chat model via M6. The embedding model is the only AI dependency baked into M7 — it is stable per deployment and changing it requires full re-ingest. The chat model used downstream is irrelevant to M7.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Ingest Knowledge Documents into the RAG Stores (Priority: P1)

A docente uploads course materials — syllabi, rubrics, past exam questions, reference documents — which are stored, chunked, embedded, and indexed in the appropriate RAG knowledge base. An administrator uploads or updates curricular documents — programme descriptors, competency frameworks — which populate the curricular knowledge base. Once ingested, these documents are immediately available for retrieval.

**Why this priority**: Without ingestion there is nothing to retrieve. All downstream AI analysis (detection engine, practice feedback, interdisciplinary proposals) depends on the knowledge bases being populated. Ingestion is the prerequisite for every other M7 story.

**Independent Test**: Can be fully tested by uploading a document via the ingest endpoint and then performing a similarity query that returns the ingested content in its top-N results.

**Acceptance Scenarios**:

1. **Given** a docente uploads a course rubric, **When** the ingest completes, **Then** the document appears in the docente RAG store filtered to that docente's course, and can be retrieved by a similarity query scoped to that course.
2. **Given** an administrator uploads a curriculum document, **When** the ingest completes, **Then** the document appears in the curricular RAG store filtered to the correct programme and subject code.
3. **Given** a rubric document is uploaded with the rubric flag set, **When** it is ingested, **Then** it is stored with `is_rubric = TRUE` and will always be included in full in any retrieval response for that activity — never fragmented.
4. **Given** an external URL or YouTube link is referenced in a docente upload, **When** the ingest runs, **Then** the referenced resource is fetched and stored in MinIO so retrieval does not depend on external availability.
5. **Given** a duplicate document is ingested (same source file, same course), **When** the ingest completes, **Then** the existing chunks are replaced with fresh embeddings, and no stale chunks remain retrievable.

---

### User Story 2 — Retrieve Relevant Context via Curricular RAG (Priority: P2)

The detection engine and interdisciplinary analysis components query the curricular RAG to obtain relevant academic context for a student's submission. The query is scoped by programme, semester, and subject code before any similarity ranking occurs. The returned context is used to enrich an AI prompt — never to make autonomous decisions.

**Why this priority**: The curricular RAG feeds the anti-plagiarism and interdisciplinary detection flows. Without a working retrieval path, the detection engine cannot produce grounded analysis. This is the first retrieval flow to be operational.

**Independent Test**: Can be fully tested by issuing a retrieval query with a programme and subject code scope and verifying that only documents from that programme and subject code appear in results, ranked by relevance.

**Acceptance Scenarios**:

1. **Given** the curricular RAG contains documents for two different programmes, **When** a retrieval query is issued for programme A, **Then** only programme A documents appear in the results — no programme B documents leak through.
2. **Given** a retrieval query is issued for a subject code with ingested rubrics and reference materials, **When** results are returned, **Then** rubrics are always included in full at the top of the context, followed by similarity-ranked non-rubric documents up to the configured token budget.
3. **Given** a retrieval query is issued for a subject code with no ingested documents, **When** results are returned, **Then** the response is an empty context set — no fallback to other programmes.
4. **Given** a retrieval query specifies a maximum number of chunks, **When** non-rubric results exceed that limit, **Then** only the top-N most similar chunks are returned (rubrics excluded from this limit).

---

### User Story 3 — Retrieve Relevant Context via Docente RAG (Priority: P3)

A docente's practice feedback and post-activity analysis components query the docente RAG to obtain course-specific context for a student's work. The query is scoped by docente identity and course (and optionally activity) before similarity ranking. The returned context enriches the feedback prompt.

**Why this priority**: The docente RAG enables personalised, course-specific feedback. It is distinct from the curricular RAG and must never share data across docentes. This story is independently testable once ingestion works and delivers the remaining retrieval capability.

**Independent Test**: Can be fully tested by querying the docente RAG with docente A's credentials and verifying that only docente A's documents are returned, with no documents from docente B or from a different course.

**Acceptance Scenarios**:

1. **Given** two docentes have ingested materials for their respective courses, **When** docente A queries the RAG for their course, **Then** only docente A's documents for that course appear — docente B's documents never leak.
2. **Given** the query includes an optional activity filter, **When** results are returned, **Then** only documents tagged to that specific activity are returned; if no activity filter is supplied, all of the docente's course documents are eligible.
3. **Given** a docente queries their RAG for a course with no ingested materials, **When** results are returned, **Then** the response is an empty context set with no error — the calling service decides how to handle an empty context.
4. **Given** a retrieval query returns results, **When** the response is inspected, **Then** every chunk includes the source document identifier so the calling service can reference the original document if needed.

---

### User Story 4 — Retrieve Rubric in Full for Any Activity (Priority: P4)

When any analysis service requests context for an activity that has a rubric document, the rubric is always included entirely in the retrieval response — never chunked, never truncated, never replaced by a similarity-ranked fragment. Non-rubric documents fill the remaining context budget.

**Why this priority**: The constitution mandates this behaviour explicitly (Principle V). Rubric completeness is an integrity constraint, not a quality-of-life feature. Partial rubrics produce incorrect scoring guidance.

**Independent Test**: Can be fully tested by uploading a rubric of known size, performing a retrieval for the activity, and confirming the full rubric text is present in the response with no truncation.

**Acceptance Scenarios**:

1. **Given** a rubric document is stored for activity X with a token count that exceeds the similarity-chunk budget, **When** a retrieval is requested for activity X, **Then** the full rubric is still included in the response regardless of size.
2. **Given** a rubric and three similarity-ranked documents are available for an activity, **When** the token budget is tight, **Then** the rubric is kept whole and the similarity-ranked documents are reduced or omitted — never the reverse.
3. **Given** no rubric exists for an activity, **When** a retrieval is requested, **Then** the response contains only similarity-ranked chunks with no error.

---

### Edge Cases

- What happens when a very large document (> token limit for a single chunk) is ingested — how are chunks sized and overlapped?
- What happens when an external URL referenced in a docente upload is unreachable during ingest?
- How does the system handle two concurrent ingest requests for the same document from the same course?
- What happens if the embedding model is unavailable during ingest?
- What happens when a retrieval query returns zero results in the curricular RAG — does the detection engine still run?
- What happens if a rubric document is updated (re-ingested) while an analysis is in progress?
- How does the system behave when the token budget would be entirely consumed by the rubric alone, leaving no room for similarity-ranked context?

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide an ingest endpoint that accepts a document (file or URL reference), identifies the target knowledge base (curricular or docente), chunks and embeds the content, and stores the vectors with appropriate scope metadata.
- **FR-002**: The system MUST maintain two physically separate knowledge bases: `rag_curricular` and `rag_docente`. These MUST NOT be joined or mixed in any retrieval query.
- **FR-003**: Curricular RAG documents MUST be scoped by `carrera`, `semestre`, and `codigo_materia` (programme, semester, subject code). Retrieval MUST apply these filters before similarity search.
- **FR-004**: Docente RAG documents MUST be scoped by `docente_id` and `curso_id`, with an optional `actividad_id` sub-filter. Retrieval MUST apply the `docente_id` and `curso_id` filters before similarity search.
- **FR-005**: Rubric documents (`is_rubric = TRUE`) MUST be stored as a single unsplit unit and MUST be injected in full into any retrieval response for the associated activity. They MUST NOT be retrieved by similarity ranking.
- **FR-006**: The system MUST expose a retrieval endpoint for the curricular RAG that accepts a scope (`carrera`, `semestre`, `codigo_materia`) and a query text, and returns the top-N most relevant chunks plus any full rubrics for the matched activity.
- **FR-007**: The system MUST expose a retrieval endpoint for the docente RAG that accepts a scope (`docente_id`, `curso_id`, optional `actividad_id`) and a query text, and returns the top-N most relevant chunks plus any full rubrics.
- **FR-008**: External URLs and video links referenced in a docente ingest payload MUST be fetched and stored in MinIO at ingest time. Retrieved content (page text or transcript) is embedded alongside the local file content.
- **FR-009**: Re-ingesting an existing document (identified by `document_id`) MUST use a two-phase replace: insert new chunks as `status='pending'`; only if all embeddings succeed, delete old `active` chunks and promote new ones to `active`; if any step fails, delete pending chunks and leave the old active chunks intact. No stale or partial chunks may remain in `active` state after a failed re-ingest.
- **FR-010**: The system MUST enforce role-based access: docentes can only ingest and query their own course knowledge base; administrators can ingest curricular documents; neither role can cross-query the other's scope.
- **FR-011**: All ingest and retrieval operations MUST be accessible only to authenticated internal callers (JWT required). Students MUST NOT have access to ingest or retrieval endpoints.
- **FR-012**: The retrieval response MUST include, for each returned chunk: the source `document_id`, the chunk sequence index, the similarity score (for non-rubric chunks), and the scope metadata under which it was stored.
- **FR-013**: Embedding requests MUST be sent to `{AI_EMBED_URL}/v1/embeddings` using the OpenAI-compatible format. The M6 router MUST NOT be involved in any M7 embedding, ingest, or retrieval operation. The embedding model and vector dimension are fixed per deployment via `AI_EMBED_MODEL` and `AI_EMBED_DIMENSION`.
- **FR-014**: The ingest endpoint MUST return a system-assigned `document_id` (UUID) on first ingest. Subsequent ingest requests for the same document MUST supply the same `document_id` to trigger the two-phase replace defined in FR-009.
- **FR-015**: The system MUST extract readable text from PDF files and Word (.docx) files at ingest time. Non-text elements (images, charts) MUST be silently skipped. Plain text files MUST be accepted without transformation. Unsupported file types MUST be rejected with a clear error before any embedding is attempted.

### Key Entities

- **CurricularDocument**: A knowledge item in the curricular RAG. Scoped by `carrera`, `semestre`, `codigo_materia`. Tracks `document_id` (caller-reusable UUID), `is_rubric`, `embed_model`, `embed_dimension`, and MinIO source path. Current active chunks are derived from its associated `DocumentChunk` rows with `status='active'`.
- **DocenteDocument**: A knowledge item in the docente RAG. Scoped by `docente_id`, `curso_id`, and optionally `actividad_id`. Same metadata fields as CurricularDocument. Rubric flag applies here too.
- **DocumentChunk**: A single embedding unit from a parent document. Holds the chunk text (no prompt content stored here), embedding vector, chunk sequence index, parent `document_id`, and `status` (`pending` or `active`). Only `active` chunks are returned by retrieval queries.
- **RetrievalRequest**: An internal service call to a RAG endpoint specifying scope filters, query text, max chunk count, and optional `actividad_id` for rubric injection.
- **RetrievalResponse**: Ordered list of chunks — full rubric text first (if any), followed by similarity-ranked `active` non-rubric chunks — each with `document_id`, chunk index, similarity score, and scope metadata.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A document ingested into either RAG store is retrievable within 5 seconds of the ingest request completing.
- **SC-002**: A retrieval query scoped to a specific programme and subject code returns zero results from any other programme or subject code — no cross-scope leakage in 100% of tested queries.
- **SC-003**: A retrieval query scoped to a specific docente returns zero results from any other docente — no cross-docente leakage in 100% of tested queries.
- **SC-004**: Rubric documents are returned in full (untruncated) in 100% of retrieval responses for their associated activity, regardless of the configured token budget.
- **SC-005**: Re-ingesting a document results in zero stale chunks from the previous version remaining in the store, verified by a subsequent retrieval query.
- **SC-006**: Retrieval latency (from query to response) is under 2 seconds for stores containing up to 10,000 chunks.
- **SC-007**: External URLs referenced during ingest are fetched and stored in MinIO successfully for 95% of reachable URLs; unreachable URLs produce a warning log and the ingest continues with the remaining content.

---

## Assumptions

- PostgreSQL 16 with the `pgvector` extension is the vector store — already provisioned from M1. No additional vector database is needed.
- Embedding generation uses an OpenAI-compatible `/v1/embeddings` endpoint. Both Vercel AI Gateway (primary) and Ollama (fallback) support this endpoint format. M7 introduces three new env vars: `AI_EMBED_URL` (defaults to `AI_CLOUD_URL`), `AI_EMBED_MODEL` (e.g. `text-embedding-3-small`), and `AI_EMBED_DIMENSION` (e.g. `1536`). The M6 AI router is not involved in any embedding operation.
- The embedding vector dimension is fixed at migration time (`VECTOR(n)` where n = `AI_EMBED_DIMENSION`). If the embedding model changes, all documents in both RAG stores must be re-ingested — this is an operator responsibility, not a runtime feature.
- Each document row stores `embed_model` and `embed_dimension` so the system can detect a model mismatch and warn the operator before retrieval produces incorrect similarity scores.
- PDF text is extracted using a pure-Go PDF library (no new Docker dependencies). Word (.docx) text is extracted by parsing the underlying ZIP/XML format. Plain text files pass through unchanged. Images, charts, and non-text elements are skipped silently.
- Documents are chunked with a fixed token window (default 512 tokens) and configurable overlap (default 64 tokens), tunable via environment variables.
- The ingestion pipeline runs inline (synchronous HTTP handler) for single-document uploads. Batch ingestion is out of scope for M7.
- The detection engine (M8) is the primary consumer of retrieval responses; it calls the RAG endpoints over HTTP as internal service calls and passes retrieved chunks to whichever chat model M6 is currently routing to.
- Both RAG knowledge bases share the same PostgreSQL instance but reside in separate tables — not separate schemas.
- The chunk-level token budget for similarity results (excluding rubrics) defaults to 2,048 tokens and is configurable.
- The system does not evaluate the quality or accuracy of embeddings — it trusts the embedding model's output.
- Student-facing features that display AI feedback do not call RAG directly — the detection engine and format validator are the intermediaries.
