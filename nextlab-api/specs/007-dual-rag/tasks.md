# Tasks: M7 — Dual RAG Pipeline

**Input**: Design documents from `specs/007-dual-rag/`

**Prerequisites**: plan.md ✅, spec.md ✅, data-model.md ✅, contracts/dual-rag.md ✅, research.md ✅, quickstart.md ✅

**Tests**: Not requested — no test tasks generated.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)
- Exact file paths are included in every description

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the one new dependency and create migration files — no `internal/rag/` logic yet.

- [X] T001 Run `go get github.com/pdfcpu/pdfcpu` from repo root and commit the updated `go.mod` and `go.sum`
- [X] T002 [P] Create `migrations/000007_dual_rag.up.sql` with: (a) `CREATE TABLE rag_curricular` (id UUID PK, carrera TEXT NOT NULL, semestre SMALLINT NOT NULL CHECK 1–10, codigo_materia TEXT NOT NULL, actividad_id UUID NULLABLE FK→actividades(id) ON DELETE SET NULL, is_rubric BOOLEAN NOT NULL DEFAULT FALSE, embed_model TEXT NOT NULL, embed_dimension INT NOT NULL, minio_path TEXT NOT NULL, original_filename TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now()); INDEX rag_curricular_scope_idx ON (carrera, semestre, codigo_materia). (b) `CREATE TABLE rag_curricular_chunks` (id UUID PK, document_id UUID NOT NULL FK→rag_curricular(id) ON DELETE CASCADE, chunk_index INT NOT NULL, chunk_text TEXT NOT NULL, embedding VECTOR(1536), status TEXT NOT NULL CHECK IN ('pending','active') DEFAULT 'pending', created_at TIMESTAMPTZ NOT NULL DEFAULT now()); INDEX rag_curricular_chunks_doc_status_idx ON (document_id, status); INDEX rag_curricular_chunks_vector_idx USING ivfflat (embedding vector_cosine_ops) WITH (lists=10). (c) Mirror tables `rag_docente` (same columns as rag_curricular but with docente_id UUID NOT NULL FK→users(id) and curso_id UUID NOT NULL FK→cursos(id) ON DELETE CASCADE instead of carrera/semestre/codigo_materia) and `rag_docente_chunks` (same as rag_curricular_chunks but FK→rag_docente(id)). INDEX rag_docente_scope_idx ON (docente_id, curso_id). INDEX rag_docente_chunks_doc_status_idx ON (document_id, status). INDEX rag_docente_chunks_vector_idx USING ivfflat (embedding vector_cosine_ops) WITH (lists=10)
- [X] T003 [P] Create `migrations/000007_dual_rag.down.sql` with: `DROP TABLE IF EXISTS rag_docente_chunks; DROP TABLE IF EXISTS rag_docente; DROP TABLE IF EXISTS rag_curricular_chunks; DROP TABLE IF EXISTS rag_curricular;`

**Checkpoint**: go.mod has pdfcpu; migration files ready to apply.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Config extension, all shared types, embedding client, chunker, extractor, and both repositories. No HTTP handlers yet.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T004 Extend `internal/config/config.go` to add struct fields AIEmbedURL (string), AIEmbedModel (string), AIEmbedDimension (int), RAGChunkTokens (int), RAGChunkOverlap (int), RAGTokenBudget (int); in Load() set: AIEmbedURL=getenv("AI_EMBED_URL", cfg.AICloudURL), AIEmbedModel=getenv("AI_EMBED_MODEL","text-embedding-3-small"), AIEmbedDimension=parseInt(getenv("AI_EMBED_DIMENSION","1536")), RAGChunkTokens=parseInt(getenv("RAG_CHUNK_TOKENS","512")), RAGChunkOverlap=parseInt(getenv("RAG_CHUNK_OVERLAP","64")), RAGTokenBudget=parseInt(getenv("RAG_TOKEN_BUDGET","2048"))
- [X] T005 [P] Create `internal/rag/model.go` replacing the empty stub with: CurricularDocument struct (ID, Carrera, Semestre, CodigoMateria, ActividadID *uuid.UUID, IsRubric, EmbedModel, EmbedDimension, MinioPath, OriginalFilename, CreatedAt), DocenteDocument struct (ID, DocenteID, CursoID, ActividadID *uuid.UUID, IsRubric, EmbedModel, EmbedDimension, MinioPath, OriginalFilename, CreatedAt), DocumentChunk struct (ID, DocumentID uuid.UUID, ChunkIndex int, ChunkText string, Status string), ChunkResult struct (DocumentID uuid.UUID, ChunkIndex int, ChunkText string, SimilarityScore float32, Scope map[string]any), IngestResult struct (DocumentID uuid.UUID, ChunkCount int, Status string), CurricularQueryRequest struct (Carrera string, Semestre int, CodigoMateria string, Query string, MaxChunks int, ActividadID *uuid.UUID), DocenteQueryRequest struct (DocenteID uuid.UUID, CursoID uuid.UUID, ActividadID *uuid.UUID, Query string, MaxChunks int), RetrievalResponse struct (Rubrics []ChunkResult, Chunks []ChunkResult)
- [X] T006 [P] Create `internal/rag/client_embed.go` with EmbedClient struct (BaseURL, APIKey, Model string, HTTPClient *http.Client), NewEmbedClient(cfg *config.Config) *EmbedClient constructor (use cfg.AIEmbedURL as BaseURL, cfg.AIEmbedModel, 30s timeout), Embed(ctx context.Context, text string) ([]float32, error) method that POSTs to BaseURL+"/v1/embeddings" with body {"model":"<model>","input":"<text>"}, sets Authorization header only when APIKey non-empty, decodes response {"data":[{"embedding":[...]}]}, returns data[0].embedding as []float32; also add formatVector(v []float32) string helper that formats as "[f1,f2,...,fn]" for pgvector text cast
- [X] T007 [P] Create `internal/rag/chunker.go` with ChunkText(text string, tokenWindow, overlap int) []string: byteWindow=tokenWindow*4, byteOverlap=overlap*4; loop from start=0; each iteration: end=min(start+byteWindow, len(text)), snap end backward to last space/newline boundary within 100 bytes to avoid splitting mid-word, append strings.TrimSpace(text[start:end]), advance start by (byteWindow-byteOverlap); stop when start>=len(text); return chunks; handle edge case where text is shorter than byteWindow (return single chunk)
- [X] T008 [P] Create `internal/rag/extractor.go` with ExtractText(file io.ReaderAt, size int64, ext string) (string, error): ext must be one of ".pdf", ".docx", ".txt" — return error "unsupported file type" for others; for ".txt" read all bytes as string; for ".docx" open zip archive from file/size, locate "word/document.xml" entry, XML-decode collecting text content of `<w:t>` elements (namespace "http://schemas.openxmlformats.org/wordprocessingml/2006/main"), join with space; for ".pdf" use pdfcpu api to extract content, parse BT/ET blocks for Tj/TJ operators to collect text strings, join lines; all extraction errors for non-text elements (images, charts) are silently skipped — return whatever text was extracted plus nil error; also add FetchURL(ctx context.Context, rawURL string) (string, error) that does GET with 30s timeout, for YouTube URLs (contains "youtube.com/watch" or "youtu.be/") extracts video ID and fetches "https://www.youtube.com/api/timedtext?v={id}&lang=en" then strips VTT timestamps, for other URLs strips all HTML tags with a regexp and normalises whitespace
- [X] T009 Create `internal/rag/repository_curricular.go` with CurricularRepository struct (db *pgxpool.Pool), NewCurricularRepository(db) constructor, and these methods: InsertDocument(ctx, doc *CurricularDocument) (uuid.UUID, error) — INSERT INTO rag_curricular RETURNING id; LockDocument(ctx, docID uuid.UUID) (*CurricularDocument, error) — SELECT … FROM rag_curricular WHERE id=$1 FOR UPDATE; InsertChunks(ctx, docID uuid.UUID, texts []string) ([]uuid.UUID, error) — batch INSERT into rag_curricular_chunks (document_id, chunk_index, chunk_text, status='pending') RETURNING id; UpdateChunkEmbedding(ctx, chunkID uuid.UUID, embedding []float32) error — UPDATE rag_curricular_chunks SET embedding=$1::vector WHERE id=$2 (pass formatVector(embedding) as $1); ActivateChunks(ctx, docID uuid.UUID) error — in single transaction: DELETE FROM rag_curricular_chunks WHERE document_id=$1 AND status='active'; UPDATE rag_curricular_chunks SET status='active' WHERE document_id=$1 AND status='pending'; DeletePendingChunks(ctx, docID uuid.UUID) error — DELETE FROM rag_curricular_chunks WHERE document_id=$1 AND status='pending'; GetRubrics(ctx, carrera string, semestre int, codigoMateria string, actividadID *uuid.UUID) ([]ChunkResult, error) — JOIN query selecting active chunks for documents with is_rubric=TRUE, filtered by carrera/semestre/codigo_materia and actividad_id if non-nil; QuerySimilar(ctx, carrera string, semestre int, codigoMateria string, queryVec string, limit int) ([]ChunkResult, error) — JOIN query with cosine similarity ORDER BY embedding <=> $1::vector LIMIT $5, filtered to is_rubric=FALSE and status='active'
- [X] T010 [P] Create `internal/rag/repository_docente.go` as a mirror of `repository_curricular.go` but operating on `rag_docente` and `rag_docente_chunks` tables: DocenteRepository struct with InsertDocument, LockDocument, InsertChunks, UpdateChunkEmbedding, ActivateChunks, DeletePendingChunks, GetRubrics(ctx, docenteID, cursoID uuid.UUID, actividadID *uuid.UUID), QuerySimilar(ctx, docenteID, cursoID uuid.UUID, actividadID *uuid.UUID, queryVec string, limit int) — all queries scope by docente_id+curso_id first; QuerySimilar adds actividad_id filter when non-nil

**Checkpoint**: Package compiles (`go build ./internal/rag/...`); no HTTP handlers yet.

---

## Phase 3: User Story 1 — Ingest Knowledge Documents (Priority: P1) 🎯 MVP

**Goal**: POST /rag/curricular/ingest and POST /rag/docente/ingest accept files (PDF/DOCX/TXT), extract text, chunk, embed via AI_EMBED_URL, store vectors in rag_curricular_chunks / rag_docente_chunks with two-phase pending→active swap, upload original to MinIO, and return document_id + chunk_count.

**Independent Test**: Upload a PDF via POST /rag/curricular/ingest → verify 201 with document_id; query DB to confirm active chunks exist with non-null embeddings; query MinIO to confirm file stored at `rag-docs/curricular/{document_id}/original.pdf`.

- [X] T011 [US1] Create `internal/rag/service_ingest.go` with IngestService struct (curRepo *CurricularRepository, docRepo *DocenteRepository, embedClient *EmbedClient, minio *minio.Client, cfg *config.Config), NewIngestService constructor, and two methods: IngestCurricular(ctx, req IngestCurricularRequest) (*IngestResult, error) and IngestDocente(ctx, req IngestDocenteRequest) (*IngestResult, error). Both methods share the same flow: (1) validate file extension via strings.ToLower(filepath.Ext(req.OriginalFilename)); (2) read file bytes into memory; (3) call ExtractText(bytes.NewReader(data), size, ext) — return 422 on "unsupported file type" error; (4) if is_rubric=TRUE skip ChunkText and use []string{fullText} as single chunk; else call ChunkText(text, cfg.RAGChunkTokens, cfg.RAGChunkOverlap); (5) determine document_id: if req.DocumentID!=nil use existing (re-ingest), else generate uuid.New(); (6) upload original file to MinIO at path "rag-docs/curricular/{docID}/original{ext}" (curricular) or "rag-docs/docente/{docID}/original{ext}" (docente); (7) upsert document row — if first ingest call InsertDocument, if re-ingest call LockDocument (SELECT FOR UPDATE); (8) call InsertChunks to insert pending chunks, collect returned chunk IDs; (9) embed loop: for each chunk call embedClient.Embed(ctx, text), call UpdateChunkEmbedding(ctx, chunkID, vec) — if ANY call fails, call DeletePendingChunks and return embedding_failed error; (10) call ActivateChunks (TX-2: deletes old active, promotes pending to active); (11) return IngestResult{DocumentID: docID, ChunkCount: len(chunks), Status: "ingested"/"re-ingested"}; also define IngestCurricularRequest struct (File io.ReaderAt, FileSize int64, OriginalFilename, Carrera string, Semestre int, CodigoMateria string, IsRubric bool, DocumentID *uuid.UUID, ActividadID *uuid.UUID) and IngestDocenteRequest struct (same but with DocenteID, CursoID uuid.UUID instead of carrera fields); include optional external URL fetching: if req.ExternalURL non-empty call FetchURL and append fetched text to extracted text before chunking
- [X] T012 [US1] Create `internal/rag/handler_ingest.go` with two handlers: ingestCurricularHandler(svc *IngestService) fiber.Handler — parse multipart form, validate carrera/semestre/codigo_materia all present (return 400 if missing), parse is_rubric/document_id/actividad_id optional fields, call c.FormFile("file") to get the upload, call svc.IngestCurricular, return 422 on "unsupported file type", 500 on embedding failure, 201 (first ingest) or 200 (re-ingest) with IngestResult JSON; and ingestDocenteHandler(svc *IngestService) fiber.Handler — same pattern but extract docente_id from JWT claims (`c.Locals("userID")`), validate curso_id form field, verify docente owns the curso by querying DB (return 403 if not), call svc.IngestDocente; both handlers log a warning (not error) for each skipped external URL per SC-007
- [X] T013 [US1] Rewrite `internal/rag/module.go` stub (currently `package rag` only) with RegisterRoutes(app *fiber.App, db *pgxpool.Pool, minioClient *minio.Client, cfg *config.Config, authMW fiber.Handler, roleGuard func(...string) fiber.Handler): construct CurricularRepository, DocenteRepository, EmbedClient, IngestService; mount POST /rag/curricular/ingest under authMW+roleGuard("admin"); mount POST /rag/docente/ingest under authMW+roleGuard("docente"); leave retrieve endpoints commented as TODO placeholders (will be added in T017, T020)
- [X] T014 [US1] Extend `cmd/api/main.go` to import `github.com/Elder2003/nextlab-api/internal/rag`, retrieve the existing *minio.Client from the storage package (or reconstruct with cfg+minio.New — check how M4 exposes minioClient), and call `rag.RegisterRoutes(app, pgPool, minioClient, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard)` after the existing module registrations

**Checkpoint**: POST /rag/curricular/ingest functional; quickstart steps 1–6 pass; `go build ./...` succeeds.

---

## Phase 4: User Story 2 — Retrieve Relevant Context via Curricular RAG (Priority: P2)

**Goal**: POST /rag/curricular/query returns similarity-ranked chunks scoped to carrera+semestre+codigo_materia, with rubric injection when actividad_id is supplied. Zero results from other scopes (SC-002).

**Independent Test**: Ingest a document for scope A and scope B; query scope A → verify zero scope B documents in results; query with actividad_id for a rubric → verify rubric in `rubrics` array (not in `chunks`), untruncated.

- [X] T015 [US2] Create `internal/rag/service_retrieve.go` with RetrieveService struct (curRepo *CurricularRepository, docRepo *DocenteRepository, embedClient *EmbedClient, cfg *config.Config), NewRetrieveService constructor, and QueryCurricular(ctx context.Context, req CurricularQueryRequest) (*RetrievalResponse, error): (1) call embedClient.Embed(ctx, req.Query) to embed the query text — return 500 on failure; (2) if req.ActividadID != nil call curRepo.GetRubrics(ctx, carrera, semestre, codigoMateria, actividadID), convert to []ChunkResult with no similarity_score; (3) determine limit: if req.MaxChunks>0 use it, else derive from cfg.RAGTokenBudget / avg_chunk_tokens (default 512); (4) call curRepo.QuerySimilar(ctx, carrera, semestre, codigoMateria, formatVector(queryVec), limit), convert to []ChunkResult with similarity_score; (5) return &RetrievalResponse{Rubrics: rubrics, Chunks: chunks} — rubrics and chunks are always separate arrays, rubrics never included in chunks count or limit
- [X] T016 [US2] Create `internal/rag/handler_retrieve.go` with retrieveCurricularHandler(svc *RetrieveService) fiber.Handler: parse JSON body into CurricularQueryRequest, validate carrera/semestre/codigo_materia all present (400 if missing), validate query non-empty (400 if missing), call svc.QueryCurricular, return 200 with RetrievalResponse JSON; empty rubrics and chunks arrays (not null) when no results — never return 404 for empty scope
- [X] T017 [US2] Extend `internal/rag/module.go` to add RetrieveService construction (NewRetrieveService) and mount POST /rag/curricular/query under authMW (any authenticated user — used by M8 detection engine and M12 interdisciplinary analysis); remove the TODO comment placeholder for this route

**Checkpoint**: POST /rag/curricular/query functional; quickstart steps 3, 5, 10 pass (similarity results, rubric injection, empty-scope empty-response).

---

## Phase 5: User Story 3 — Retrieve Relevant Context via Docente RAG (Priority: P3)

**Goal**: POST /rag/docente/query returns similarity-ranked chunks scoped to the authenticated docente's course (docente_id+curso_id) with optional actividad_id filter. Zero cross-docente leakage (SC-003).

**Independent Test**: Ingest documents as docente A; query as docente A → verify A's documents; query as docente B with docente A's curso_id → verify 403 forbidden (cross-scope blocked at handler).

- [X] T018 [US3] Extend `internal/rag/service_retrieve.go` by adding QueryDocente(ctx context.Context, req DocenteQueryRequest) (*RetrievalResponse, error) method: same flow as QueryCurricular but (1) scoped by docenteID+cursoID; (2) if req.ActividadID != nil call docRepo.GetRubrics(ctx, docenteID, cursoID, actividadID); (3) call docRepo.QuerySimilar(ctx, docenteID, cursoID, req.ActividadID, formatVector(queryVec), limit); (4) return RetrievalResponse with docente scope metadata {docente_id, curso_id} in each chunk's Scope field
- [X] T019 [US3] Extend `internal/rag/handler_retrieve.go` by adding retrieveDocenteHandler(svc *RetrieveService) fiber.Handler: parse JSON body into DocenteQueryRequest, extract docente_id from JWT claims (c.Locals("userID")), validate curso_id non-empty, verify docente owns curso_id via DB check (return 403 if not — same ownership check as handler_ingest.go docente path), call svc.QueryDocente, return 200 with RetrievalResponse; empty arrays on no results (never 404)
- [X] T020 [US3] Extend `internal/rag/module.go` to mount POST /rag/docente/query under authMW+roleGuard("docente"); remove the remaining TODO comment placeholder

**Checkpoint**: POST /rag/docente/query functional; quickstart steps 7–9 pass (docente ingest, own-course query, cross-scope 403).

---

## Phase 6: User Story 4 — Retrieve Rubric in Full for Any Activity (Priority: P4)

**Goal**: Confirm that is_rubric=TRUE documents are stored as a single unsplit chunk, are always injected into retrieval responses in full, and are excluded from the max_chunks token budget.

**Independent Test**: Ingest a large rubric (known token count > RAG_TOKEN_BUDGET); retrieve with that actividad_id; verify rubric appears in full in `rubrics` array; verify chunk_count=1 in DB.

- [X] T021 [US4] Verify in `internal/rag/service_ingest.go` that the rubric path (is_rubric=TRUE) skips ChunkText entirely and uses `[]string{extractedText}` as the sole chunk, resulting in chunk_index=0 as the only row; add an explicit comment: "rubric documents are stored as a single unsplit unit per FR-005 and Constitution Principle V"
- [X] T022 [US4] Verify in `internal/rag/service_retrieve.go` QueryCurricular and QueryDocente that (a) rubric results come from GetRubrics (separate DB call, no similarity ranking), (b) rubric ChunkResult entries are placed in RetrievalResponse.Rubrics, NEVER in .Chunks, (c) the `limit` passed to QuerySimilar is unaffected by rubric count — rubrics do not consume from max_chunks budget; add an explicit comment: "rubrics excluded from token budget per FR-005/SC-004"

**Checkpoint**: Quickstart step 5 passes with a large rubric; rubric always present in full regardless of RAGTokenBudget setting.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Environment wiring, build validation, and end-to-end quickstart verification.

- [X] T023 [P] Update `docker-compose.yml` api service environment block to add: `AI_EMBED_URL: ""` (comment: defaults to AI_CLOUD_URL), `AI_EMBED_MODEL: "text-embedding-3-small"`, `AI_EMBED_DIMENSION: "1536"`, `RAG_CHUNK_TOKENS: "512"`, `RAG_CHUNK_OVERLAP: "64"`, `RAG_TOKEN_BUDGET: "2048"`
- [X] T024 [P] Run `go build ./...` from repo root and fix any compilation errors in `internal/rag/`, `internal/config/`, or `cmd/api/main.go`
- [ ] T025 Run quickstart.md verification steps 1–15 against a live Docker Compose stack and confirm all expected responses

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on T001 (pdfcpu in go.mod before it can be imported in extractor.go); T004 (config fields) required by T006 (EmbedClient) and T009/T010 (repo constructors) — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 completion (T004–T010)
- **US2 (Phase 4)**: Depends on Phase 2 completion; can start after T009 (curricular repo) without waiting for US1 service layer
- **US3 (Phase 5)**: Depends on Phase 2 completion + T015 (service_retrieve.go must exist before extension in T018) + T016 (handler_retrieve.go before T019)
- **US4 (Phase 6)**: Depends on T011 (service_ingest.go) and T015 (service_retrieve.go) — verification tasks
- **Polish (Phase 7)**: Depends on all phases complete; T023 and T024 can run in parallel

### User Story Dependencies

- **US1 (P1)**: After Phase 2 — blocks US2/US3/US4 only through shared service files
- **US2 (P2)**: After Phase 2 — independent of US1 service layer; only needs repos (T009) and model (T005)
- **US3 (P3)**: After US2 (T015, T016 must exist before T018, T019 extend them)
- **US4 (P4)**: After US1 (T011) and US2 (T015) — verification/annotation only

### Within Each User Story

- T004 → T005 [P], T006 [P], T007 [P], T008 [P] → T009 → T010 [P] → T011 → T012 → T013 → T014
- T009 → T015 → T016 → T017 (US2 can begin independently of US1's T011–T014)
- T015 → T018; T016 → T019; T017 → T020 (US3 extends US2 files)
- T011 → T021; T015 → T022 (US4 verifies US1+US2 implementation)

### Parallel Opportunities

| Phase | Parallel tasks |
|---|---|
| Phase 1 | T002, T003 (different files) |
| Phase 2 | T005, T006, T007, T008 (all different files, all depend on T004 only) |
| Phase 2 | T009, T010 (different files, both depend on T004+T005) |
| Phase 7 | T023, T024 (independent) |

---

## Parallel Example: US1 Implementation

```
After T010 completes (both repos ready):
  └─ T011 (service_ingest.go) → T012 (handler_ingest.go) → T013 (module.go) → T014 (main.go)

After T009 completes (curricular repo ready):
  └─ T015 (service_retrieve.go — US2) → T016 → T017  [can begin in parallel with T011]
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Setup (T001–T003)
2. Complete Phase 2: Foundational (T004–T010)
3. Complete Phase 3: US1 (T011–T014)
4. **STOP and VALIDATE**: POST /rag/curricular/ingest works; quickstart steps 1–6 pass
5. Proceed to US2, US3, US4 in priority order

### Incremental Delivery

1. T001–T010 → Package compiles, migrations ready
2. T011–T014 → Ingest working (MVP — knowledge bases can be populated)
3. T015–T017 → Curricular retrieval working
4. T018–T020 → Docente retrieval working
5. T021–T022 → Rubric integrity confirmed
6. T023–T025 → Full quickstart validation

---

## Notes

- `internal/rag/` already exists as an empty `package rag` stub — T005–T010 ADD files; T013 REPLACES the stub module.go
- pdfcpu content stream parsing may produce empty text for scanned/image-only PDFs — log the byte count on extraction; if 0 bytes returned, skip embedding (no empty-vector chunks)
- `formatVector` helper defined in `client_embed.go` (T006) is used by both `repository_curricular.go` (T009) and `repository_docente.go` (T010) — they import from the same package, no duplication
- MinIO client: check how `internal/storage` or `cmd/api/main.go` currently exposes the *minio.Client before implementing T014 — may need a small adjustment to make it accessible to the rag package
- Constitution Principle V compliance: `repository_curricular.go` and `repository_docente.go` MUST contain no SQL that references the other's tables; no JOIN across rag_curricular_* and rag_docente_* in any query
- T025 (live quickstart) is the only non-automated task — requires Docker Compose stack with a real embedding endpoint configured
