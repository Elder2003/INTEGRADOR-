# Implementation Plan: M7 — Dual RAG Pipeline

**Branch**: `007-dual-rag` | **Date**: 2026-05-20 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/007-dual-rag/spec.md`

---

## Summary

Implement two physically separate RAG knowledge bases (`rag_curricular` and `rag_docente`) backed by pgvector in the existing PostgreSQL instance. An ingest pipeline extracts text from PDF/Word/TXT files (and optional external URLs), chunks the text, calls an OpenAI-compatible embeddings endpoint, and stores vectors with scope metadata. A retrieval API returns similarity-ranked chunks plus any full rubric documents, strictly scoped to prevent cross-docente or cross-programme leakage. Re-ingestion uses a two-phase pending/active swap to guarantee zero stale chunks on failure.

**Technical approach**: Extend `internal/rag/` stub · new `EmbedClient` mirroring M6's `BackendClient` pattern · `pdfcpu` for PDF text extraction · stdlib ZIP/XML for .docx · byte/4 token approximation for chunking · text-encoded pgvector queries (no extra driver) · two-phase ingest with `SELECT FOR UPDATE` concurrency guard · four HTTP endpoints (ingest + query × 2 RAG types) · migration `000007_dual_rag` · six new environment variables

---

## Technical Context

**Language/Version**: Go 1.25.0 (module `github.com/Elder2003/nextlab-api`)

**Primary Dependencies**:
- `gofiber/fiber/v2` — HTTP framework (locked by constitution)
- `jackc/pgx/v5/pgxpool` — PostgreSQL pool (from M1, no change)
- `golang-jwt/jwt/v5` — JWT parsing (from M2, role extraction)
- `google/uuid` — UUID generation (already in go.sum)
- `minio/minio-go/v7` — MinIO file storage (from M4, reused as-is)
- `archive/zip` + `encoding/xml` — .docx parsing (stdlib)
- `net/http` + `encoding/json` — embedding HTTP client + external URL fetch (stdlib)
- `github.com/pdfcpu/pdfcpu` — **NEW** — pure-Go PDF text extraction

**New go.mod dependency**: `github.com/pdfcpu/pdfcpu` only.

**Storage**:
- PostgreSQL 16 + pgvector: 4 new tables (`rag_curricular`, `rag_curricular_chunks`, `rag_docente`, `rag_docente_chunks`)
- MinIO: `rag-docs/` prefix within existing instance (`rag-docs/curricular/{doc_id}/original.{ext}`, `rag-docs/docente/{doc_id}/original.{ext}`)

**New environment variables**:

| Variable | Required | Default | Notes |
|---|---|---|---|
| `AI_EMBED_URL` | No | `AI_CLOUD_URL` | OpenAI-compatible `/v1/embeddings` endpoint |
| `AI_EMBED_MODEL` | No | `text-embedding-3-small` | Embedding model name |
| `AI_EMBED_DIMENSION` | No | `1536` | MUST match `VECTOR(n)` in migration |
| `RAG_CHUNK_TOKENS` | No | `512` | Approx tokens per chunk (byte/4 estimate) |
| `RAG_CHUNK_OVERLAP` | No | `64` | Overlap tokens between consecutive chunks |
| `RAG_TOKEN_BUDGET` | No | `2048` | Max similarity-ranked chunk token budget per retrieval |

**Target Platform**: Linux/AMD64 (Docker Compose, same as M1–M6)

**Performance Goals**:
- Ingest completes within 5 seconds of HTTP response (SC-001); large files may exceed this if embedding batches are slow — document as expected behaviour
- Retrieval latency ≤ 2 seconds for stores up to 10,000 chunks (SC-006)

**Constraints**:
- `rag_curricular` and `rag_docente` MUST NOT be joined in any query (Principle V, FR-002)
- Docente scope filter (`docente_id` + `curso_id`) applied BEFORE similarity search (Principle IV, FR-004)
- Rubric documents returned in full; NEVER retrieved by similarity (FR-005)
- Students MUST NOT access ingest or query endpoints (FR-011)
- Embedding calls bypass M6 AI router — direct to `AI_EMBED_URL` (FR-013)
- Inline synchronous HTTP handler (single-document ingest; no async batch in M7)

**Scale/Scope**: ~100–500 documents/month at peak; up to 10,000 chunks total at launch; pgvector IVFFlat with `lists=10` sufficient.

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Immutable Evidence | ✅ N/A | No trace events or appeal records touched by M7. Document rows in `rag_curricular`/`rag_docente` are append-only. Only chunk rows are deleted (two-phase swap + failure cleanup) — this is not an evidence record. |
| II. Asynchronous Analysis Isolation | ✅ Maintained | M7 ingest runs inline in the HTTP handler (synchronous, single-document). This is safe because ingest is an administrative/docente action, not a student-facing action. The detection engine (M8) calls M7's retrieval endpoints asynchronously via the worker. M7 does not add any inline analysis to student-facing submission handlers. |
| III. Human Authority Over AI | ✅ Maintained | M7 returns raw text chunks. The calling service (M8 detection, M9 format) decides how to use them. No scoring, grading, or sanction is produced by RAG retrieval. Rubric injection is informational only. |
| IV. Strict Role-Based Data Isolation | ✅ **PRIMARY GATE** | Curricular ingest: admin only. Docente ingest/query: docente, own `curso_id` only — verified against `docente_id` from JWT before any DB operation. Students: no access to any RAG endpoint. Retrieval queries apply scope filters before similarity search (FR-004). |
| V. Dual RAG Independence | ✅ **PRIMARY GATE** | `rag_curricular_*` and `rag_docente_*` are separate table pairs. No JOIN across the two pairs. Repository functions for curricular and docente are in separate files. Module routing ensures curricular endpoints cannot be called with docente parameters. |
| VI. Non-Invasive Auditing | ✅ N/A | M7 has no session or WebSocket involvement. |
| VII. Submission Always Accepted | ✅ N/A | M7 is a knowledge base, not a submission handler. |

**Result: PASS** — Principles IV and V are primary enforcement targets. No gate violations.

---

## Project Structure

### Documentation (this feature)

```text
specs/007-dual-rag/
├── plan.md                    ← this file
├── research.md                ← Phase 0 output
├── data-model.md              ← Phase 1 output
├── quickstart.md              ← Phase 1 output
├── contracts/
│   └── dual-rag.md            ← Phase 1 output
├── checklists/
│   └── requirements.md
└── tasks.md                   ← /speckit-tasks output (not created here)
```

### Source Code Changes

```text
nextlab-api/
├── cmd/
│   └── api/
│       └── main.go                       # EXTEND: call rag.RegisterRoutes(...)
├── internal/
│   ├── rag/                              # EXTEND existing stub package
│   │   ├── module.go                     # RegisterRoutes: /rag/curricular/* + /rag/docente/*
│   │   ├── model.go                      # CurricularDocument, DocenteDocument, DocumentChunk,
│   │   │                                 #   IngestRequest, RetrievalRequest, RetrievalResponse
│   │   ├── client_embed.go               # EmbedClient: POST AI_EMBED_URL/v1/embeddings → []float32
│   │   ├── chunker.go                    # ChunkText(text, tokenWindow, overlap) []string
│   │   ├── extractor.go                  # ExtractText(file, ext) (string, error)
│   │   │                                 #   PDF → pdfcpu content streams
│   │   │                                 #   DOCX → archive/zip + encoding/xml
│   │   │                                 #   TXT → pass-through
│   │   ├── repository_curricular.go      # InsertCurricularDoc, InsertChunks, UpdateChunkEmbedding,
│   │   │                                 #   ActivateChunks, DeletePendingChunks, DeleteActiveChunks,
│   │   │                                 #   QuerySimilar, GetRubrics
│   │   ├── repository_docente.go         # Mirror of repository_curricular.go for rag_docente_*
│   │   ├── service_ingest.go             # IngestService.IngestCurricular, IngestDocente
│   │   │                                 #   orchestrates: extract → upload MinIO → chunk → TX-1 →
│   │   │                                 #   embed loop → TX-2 success or TX-2 failure
│   │   ├── service_retrieve.go           # RetrieveService.QueryCurricular, QueryDocente
│   │   │                                 #   runs rubric fetch (full) + similarity search (filtered)
│   │   ├── handler_ingest.go             # POST /rag/curricular/ingest, POST /rag/docente/ingest
│   │   └── handler_retrieve.go          # POST /rag/curricular/query, POST /rag/docente/query
│   └── config/
│       └── config.go                     # EXTEND: add AI_EMBED_URL, AI_EMBED_MODEL,
│                                         #   AI_EMBED_DIMENSION, RAG_CHUNK_TOKENS,
│                                         #   RAG_CHUNK_OVERLAP, RAG_TOKEN_BUDGET
├── migrations/
│   ├── 000007_dual_rag.up.sql            # 4 tables + indexes (VECTOR(1536))
│   └── 000007_dual_rag.down.sql
└── go.mod / go.sum                       # +github.com/pdfcpu/pdfcpu
```

**Structure Decision**: All RAG logic lives in `internal/rag/` (extending the existing stub). Two repository files mirror each other, one per RAG store, ensuring no shared query path between `rag_curricular_*` and `rag_docente_*` tables.

---

## Phase 0: Research Summary

See [research.md](research.md). Key decisions:

| Topic | Decision |
|---|---|
| pgvector Go encoding | Text string `[f1,f2,…]` cast to `VECTOR`; no third-party driver |
| PDF text extraction | `github.com/pdfcpu/pdfcpu` — parse BT/ET content stream operators |
| DOCX text extraction | stdlib `archive/zip` + `encoding/xml` — extract `<w:t>` elements |
| Token counting | byte / 4 approximation; configurable window + overlap |
| Ingest transaction | Two separate transactions — TX-1 inserts pending, TX-2 promotes/cleans |
| Concurrent ingest guard | `SELECT … FOR UPDATE` on document row |
| External URL fetch | `net/http` + HTML tag stripping; YouTube via timedtext API |
| Similarity metric | Cosine distance (`<=>`) with IVFFlat index (`lists=10`) |
| MinIO path | `rag-docs/{curricular\|docente}/{document_id}/original.{ext}` |
| New go.mod deps | `pdfcpu` only |

---

## Phase 1: Design Summary

### Data Model

See [data-model.md](data-model.md). Migration `000007_dual_rag`:

| Table | Description |
|---|---|
| `rag_curricular` | Curricular document metadata; scoped by `(carrera, semestre, codigo_materia)` |
| `rag_curricular_chunks` | Embedded chunks; `status IN ('pending','active')`; `VECTOR(1536)` |
| `rag_docente` | Docente document metadata; scoped by `(docente_id, curso_id)` |
| `rag_docente_chunks` | Embedded chunks; mirror of curricular chunks |

### API Contracts

See [contracts/dual-rag.md](contracts/dual-rag.md):

| Method | Path | Role | Description |
|---|---|---|---|
| `POST` | `/rag/curricular/ingest` | admin | Upload + chunk + embed into curricular store |
| `POST` | `/rag/curricular/query` | any authenticated | Scoped similarity search + rubric injection |
| `POST` | `/rag/docente/ingest` | docente (own course) | Upload + chunk + embed into docente store |
| `POST` | `/rag/docente/query` | docente (own course) | Scoped similarity search + rubric injection |

### Quickstart

See [quickstart.md](quickstart.md). 15-step verification covering: migration, curricular ingest, chunk activation, similarity query, rubric injection, two-phase re-ingest, docente ingest, cross-scope isolation (403), empty-scope empty-response, unsupported file rejection, .docx text extraction, MinIO path structure, student-access denial, latency check.

---

## Key Implementation Notes

### config.go: new RAG fields

```go
type Config struct {
    // ... existing fields ...
    AIEmbedURL       string
    AIEmbedModel     string
    AIEmbedDimension int
    RAGChunkTokens   int
    RAGChunkOverlap  int
    RAGTokenBudget   int
}
```

In `Load()`:
```go
cfg.AIEmbedURL       = getenv("AI_EMBED_URL", cfg.AICloudURL)
cfg.AIEmbedModel     = getenv("AI_EMBED_MODEL", "text-embedding-3-small")
cfg.AIEmbedDimension = parseInt(getenv("AI_EMBED_DIMENSION", "1536"))
cfg.RAGChunkTokens   = parseInt(getenv("RAG_CHUNK_TOKENS", "512"))
cfg.RAGChunkOverlap  = parseInt(getenv("RAG_CHUNK_OVERLAP", "64"))
cfg.RAGTokenBudget   = parseInt(getenv("RAG_TOKEN_BUDGET", "2048"))
```

---

### client_embed.go: embedding client

```go
type EmbedClient struct {
    BaseURL    string
    APIKey     string
    Model      string
    HTTPClient *http.Client
}

type EmbedRequest struct {
    Model string `json:"model"`
    Input string `json:"input"`
}

type EmbedResponse struct {
    Data []struct {
        Embedding []float32 `json:"embedding"`
    } `json:"data"`
}

func (c *EmbedClient) Embed(ctx context.Context, text string) ([]float32, error) {
    body, _ := json.Marshal(EmbedRequest{Model: c.Model, Input: text})
    req, _ := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v1/embeddings",
        bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    if c.APIKey != "" {
        req.Header.Set("Authorization", "Bearer "+c.APIKey)
    }
    // ... POST, decode EmbedResponse, return Data[0].Embedding ...
}
```

---

### chunker.go: fixed-window chunking

```go
// ChunkText splits text into overlapping chunks.
// tokenWindow and overlap are token estimates (byte/4 approximation).
func ChunkText(text string, tokenWindow, overlap int) []string {
    byteWindow  := tokenWindow * 4
    byteOverlap := overlap * 4
    var chunks []string
    start := 0
    for start < len(text) {
        end := start + byteWindow
        if end > len(text) {
            end = len(text)
        }
        // snap to word boundary
        chunks = append(chunks, strings.TrimSpace(text[start:end]))
        step := byteWindow - byteOverlap
        if step <= 0 {
            step = byteWindow
        }
        start += step
    }
    return chunks
}
```

---

### service_ingest.go: two-phase ingest

```go
func (s *IngestService) ingestChunks(ctx context.Context, repo interface{ /* curricular or docente repo */ },
    docID uuid.UUID, chunks []string) error {

    // TX-1: insert pending chunks
    if err := repo.InsertChunks(ctx, docID, chunks); err != nil {
        return err
    }

    // Embed each chunk (outside DB transaction)
    var failed bool
    for i, text := range chunks {
        vec, err := s.embedClient.Embed(ctx, text)
        if err != nil {
            failed = true
            break
        }
        if err := repo.UpdateChunkEmbedding(ctx, docID, i, vec); err != nil {
            failed = true
            break
        }
    }

    if failed {
        // TX-2 failure: delete pending, old active survives
        return repo.DeletePendingChunks(ctx, docID)
    }

    // TX-2 success: delete old active, promote pending to active
    return repo.ActivateChunks(ctx, docID)
}
```

---

### service_retrieve.go: rubric + similarity

```go
func (s *RetrieveService) QueryCurricular(ctx context.Context, req CurricularQueryRequest) (*RetrievalResponse, error) {
    // 1. Embed the query text
    queryVec, err := s.embedClient.Embed(ctx, req.Query)
    if err != nil {
        return nil, err
    }

    // 2. Fetch rubrics first (full, unsplit, no similarity ranking)
    rubrics, _ := s.curricularRepo.GetRubrics(ctx, req.Carrera, req.Semestre, req.CodigoMateria, req.ActividadID)

    // 3. Similarity search for non-rubric active chunks within scope
    chunks, _ := s.curricularRepo.QuerySimilar(ctx,
        req.Carrera, req.Semestre, req.CodigoMateria,
        formatVector(queryVec), req.MaxChunks)

    return &RetrievalResponse{Rubrics: rubrics, Chunks: chunks}, nil
}
```

---

### repository_curricular.go: key SQL patterns

```sql
-- Rubric fetch (full text, no similarity)
SELECT c.id, c.chunk_text, c.chunk_index, d.id AS document_id,
       d.carrera, d.semestre, d.codigo_materia
FROM rag_curricular_chunks c
JOIN rag_curricular d ON d.id = c.document_id
WHERE d.carrera = $1 AND d.semestre = $2 AND d.codigo_materia = $3
  AND d.actividad_id = $4 AND d.is_rubric = TRUE
  AND c.status = 'active';

-- Similarity search (non-rubric, scoped, top-N)
SELECT c.id, c.chunk_text, c.chunk_index, c.document_id,
       1 - (c.embedding <=> $1::vector) AS similarity_score,
       d.carrera, d.semestre, d.codigo_materia
FROM rag_curricular_chunks c
JOIN rag_curricular d ON d.id = c.document_id
WHERE d.carrera = $2 AND d.semestre = $3 AND d.codigo_materia = $4
  AND d.is_rubric = FALSE
  AND c.status = 'active'
ORDER BY c.embedding <=> $1::vector
LIMIT $5;

-- Two-phase: activate pending chunks (delete old active first)
DELETE FROM rag_curricular_chunks
WHERE document_id = $1 AND status = 'active';

UPDATE rag_curricular_chunks
SET status = 'active'
WHERE document_id = $1 AND status = 'pending';
```

---

### handler_ingest.go: role enforcement

```go
func ingestCurricularHandler(svc *IngestService) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Role guard applied in RegisterRoutes — admin only

        form, err := c.MultipartForm()
        if err != nil { return c.Status(400).JSON(...) }

        carrera     := form.Value["carrera"][0]       // validate non-empty
        semestre    := parseInt(form.Value["semestre"][0])
        codigoMat   := form.Value["codigo_materia"][0]
        isRubric    := form.Value["is_rubric"][0] == "true"
        documentID  := parseOptionalUUID(form.Value["document_id"])
        actividadID := parseOptionalUUID(form.Value["actividad_id"])

        file, err := getUploadedFile(form) // validates extension
        if err != nil { return c.Status(422).JSON(...) }

        result, err := svc.IngestCurricular(c.Context(), IngestCurricularRequest{
            File: file, Carrera: carrera, Semestre: semestre,
            CodigoMateria: codigoMat, IsRubric: isRubric,
            DocumentID: documentID, ActividadID: actividadID,
        })
        if err != nil { return c.Status(500).JSON(...) }
        status := 201
        if documentID != nil { status = 200 }
        return c.Status(status).JSON(result)
    }
}
```

---

### module.go: RegisterRoutes

```go
func RegisterRoutes(
    app *fiber.App,
    db *pgxpool.Pool,
    minioClient *minio.Client,
    cfg *config.Config,
    authMW fiber.Handler,
    roleGuard func(roles ...string) fiber.Handler,
) {
    embedClient := NewEmbedClient(cfg)
    curRepo     := NewCurricularRepository(db)
    docRepo     := NewDocenteRepository(db)
    ingestSvc   := NewIngestService(curRepo, docRepo, embedClient, minioClient, cfg)
    retrieveSvc := NewRetrieveService(curRepo, docRepo, embedClient)

    // Curricular RAG: admin ingest, any-auth query
    curr := app.Group("/rag/curricular", authMW)
    curr.Post("/ingest", roleGuard("admin"), ingestCurricularHandler(ingestSvc))
    curr.Post("/query",  retrieveCurricularHandler(retrieveSvc))

    // Docente RAG: docente ingest+query (own course enforced in handler)
    doc := app.Group("/rag/docente", authMW, roleGuard("docente"))
    doc.Post("/ingest", ingestDocenteHandler(ingestSvc))
    doc.Post("/query",  retrieveDocenteHandler(retrieveSvc))
}
```

---

## Key Risks & Mitigations

| Risk | Mitigation |
|---|---|
| pdfcpu content stream parsing misses encoded text | Log byte count of extracted text; if 0 bytes extracted, warn and continue (no empty embed) |
| AI_EMBED_DIMENSION mismatch with VECTOR(n) in migration | Validate at startup: if `cfg.AIEmbedDimension != 1536` log a startup warning; actual failure is caught at embed time by pgvector type check |
| Two-phase TX-2 failure leaves document in no-active-chunks state | On TX-2 success failure, TX-2 is retried once; if still fails, document row exists but has zero active chunks — retrieval returns empty for that document, operator must re-ingest |
| Large rubric exceeds response buffer | No size limit on rubric text in retrieval response — returned in full per FR-005/Principle V |
| Concurrent re-ingest races | `SELECT FOR UPDATE` on document row serialises concurrent ingests per document |
| External URL fetch hangs | 30-second timeout per URL; failure skips URL with warning (SC-007: 95% target) |
| IVFFlat index requires rows at creation | Create index in migration without `WITH (lists=10)` initially; index will be empty but functional. Re-run `REINDEX` or create as `CONCURRENTLY` after first ingest batch. |
