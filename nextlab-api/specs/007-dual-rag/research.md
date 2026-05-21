# Research: M7 — Dual RAG Pipeline

**Date**: 2026-05-20  
**Feature**: [spec.md](spec.md) | [plan.md](plan.md)

---

## Decision Log

### 1. pgvector Go Integration

**Decision**: Encode embedding vectors as PostgreSQL text literals `[f1,f2,…]` and pass as `TEXT`, letting PostgreSQL cast to `VECTOR`. No third-party pgvector Go driver required.

**Rationale**: `pgx/v5` does not natively understand pgvector column types. The pgvector wire format accepts the string literal `[0.1,-0.2,0.3]` when cast via `$1::vector`. This works without adding `github.com/pgvector/pgvector-go` to go.mod:

```go
// format []float32 as "[f1,f2,...,fn]"
func formatVector(v []float32) string {
    parts := make([]string, len(v))
    for i, f := range v {
        parts[i] = strconv.FormatFloat(float64(f), 'f', -1, 32)
    }
    return "[" + strings.Join(parts, ",") + "]"
}

// usage in SQL
`embedding <=> $1::vector`
// pass formatVector(embedding) as the $1 argument (TEXT)
```

**Alternatives considered**:
- `github.com/pgvector/pgvector-go` — adds explicit pgvector encoding; cleaner but extra dependency
- Raw binary encoding — avoids string conversion overhead; not worth the complexity at this scale

---

### 2. PDF Text Extraction

**Decision**: Use `github.com/pdfcpu/pdfcpu` for PDF text extraction. Invoke `api.ExtractContent()` to obtain raw content streams per page, then parse BT/ET (Begin Text / End Text) blocks to extract text operands (`Tj`, `TJ`, `'`, `"` operators).

**Rationale**: Specified in spec clarification. Pure-Go with no system dependencies (no poppler, ghostscript). pdfcpu v0.7+ exposes content stream access via its API. For M7's purpose (embedding text), raw content stream parsing is sufficient — the goal is machine-readable text, not pixel-perfect layout reconstruction.

**Implementation note**: Content stream parsing must handle:
- `Tj` — show string (single argument)
- `TJ` — show string array (array of strings and kerning values)
- `'` — move to next line and show string
- All other operators can be skipped

Non-text elements (images, form XObjects) produce no text operators and are silently skipped.

**Alternatives considered**:
- `ledongthuc/pdf` — simpler API specifically for text extraction; ruled out because spec names pdfcpu
- `unidoc/unipdf` — commercial license, rejected
- poppler / pdftotext — C dependency, violates constitution constraint (pure-Go)

**New go.mod dependency**: `github.com/pdfcpu/pdfcpu v0.7.0` (or latest stable at implementation time)

---

### 3. Word (.docx) Text Extraction

**Decision**: Parse .docx files using stdlib `archive/zip` + `encoding/xml`. A .docx is a ZIP archive; text lives in `word/document.xml`. Extract text nodes within `<w:t>` elements.

**Rationale**: Zero new dependencies. The XML structure is stable across Word versions. Non-text elements (images, charts, embedded objects) produce no `<w:t>` content and are silently skipped.

```go
func extractDocx(r io.ReaderAt, size int64) (string, error) {
    zr, _ := zip.NewReader(r, size)
    for _, f := range zr.File {
        if f.Name == "word/document.xml" {
            rc, _ := f.Open()
            // parse XML, collect <w:t> text nodes
            return parseWordXML(rc)
        }
    }
    return "", fmt.Errorf("word/document.xml not found in archive")
}
```

**Alternatives considered**: `ugurcandede/go-docx` — extra dependency; stdlib approach is sufficient for text-only extraction.

---

### 4. Token Counting (Chunk Window)

**Decision**: Use byte-based approximation: `len(text) / 4` ≈ token count. Default chunk window = 512 tokens ≈ 2048 bytes. Overlap = 64 tokens ≈ 256 bytes. Both are configurable via `RAG_CHUNK_TOKENS` and `RAG_CHUNK_OVERLAP` env vars.

**Rationale**: Avoids adding a tokenizer dependency (`tiktoken-go` or similar). For embedding purposes, the exact token boundary is not critical — the goal is approximate chunk sizing. The 1 token ≈ 4 bytes rule-of-thumb is well-established for GPT-family tokenizers and sufficient for text-embedding-3-small inputs.

**Alternatives considered**:
- `tiktoken-go` — accurate BPE tokenization; adds ~10MB dependency and CGo; overkill for chunking
- Word count × 1.33 — similar accuracy; bytes approach is simpler to implement

---

### 5. Two-Phase Ingest Transaction

**Decision**: Split ingest into two separate transactions to avoid holding DB locks during external embedding API calls:

```
TX-1: INSERT document row (ON CONFLICT DO NOTHING for re-ingest)
      INSERT new chunks with status='pending'
      COMMIT

[External: call AI_EMBED_URL/v1/embeddings for each pending chunk]
[UPDATE chunk SET embedding=... WHERE id=... for each successful call]

TX-2 (success path): BEGIN
      DELETE FROM rag_*_chunks WHERE document_id=$1 AND status='active'
      UPDATE rag_*_chunks SET status='active' WHERE document_id=$1 AND status='pending'
      COMMIT

TX-2 (failure path): BEGIN
      DELETE FROM rag_*_chunks WHERE document_id=$1 AND status='pending'
      COMMIT
      → old 'active' chunks survive intact
```

**Rationale**: Embedding calls are network I/O (potentially 1-5 seconds per batch). Holding a DB transaction open across network calls would consume connection pool resources and risk timeouts. The two-transaction model isolates DB locking from network I/O. The spec (FR-009) mandates this pattern explicitly.

**Alternatives considered**:
- Single transaction wrapping all embedding calls — holds DB connection pool slot; rejected
- Optimistic concurrency with CAS — more complex without meaningful benefit at this scale

---

### 6. Concurrent Ingest Guard

**Decision**: Use `SELECT … FOR UPDATE` on the document row (`rag_curricular` or `rag_docente`) at the start of TX-1. Only one ingest per document can proceed at a time; the second caller blocks until the first completes.

**Rationale**: Concurrent re-ingests of the same document would produce duplicate pending chunks. Row-level locking prevents this with no application-layer coordination. Works correctly across multiple API pods.

---

### 7. External URL Fetching

**Decision**: Use `net/http` GET with a 30-second timeout. For HTML pages: strip all `<…>` tags, normalize whitespace. For YouTube URLs: extract video ID, call `https://www.youtube.com/api/timedtext?v={id}&lang=en` to fetch auto-generated captions (VTT format), strip timestamps. Store fetched content in MinIO alongside the original file.

**SC-007 compliance**: Unreachable URLs (connection error, non-2xx, timeout) produce a warning log entry and do NOT fail the ingest — the document is ingested without the external content. The response body notes which URLs were skipped.

**Alternatives considered**:
- YouTube Data API v3 — requires API key, unnecessary overhead
- colly / go-rod — headless scraping; adds C dependencies, rejected

---

### 8. Similarity Metric

**Decision**: Use cosine distance (`<=>` operator) for vector similarity search.

**Rationale**: Cosine similarity is standard for text embeddings. text-embedding-3-small produces L2-normalized vectors, making cosine and dot-product equivalent — but cosine is the pgvector convention.

**IVFFlat index**: `USING ivfflat (embedding vector_cosine_ops) WITH (lists = 10)`. With up to 10,000 chunks (SC-006), 10 lists gives adequate recall with fast lookup. Can be increased without data migration.

---

### 9. MinIO Path Convention

**Decision**: `rag-docs/curricular/{document_id}/original.{ext}` and `rag-docs/docente/{document_id}/original.{ext}`. Use the existing MinIO instance; create `rag-docs` bucket on startup if absent (same pattern as M4's document upload).

**Rationale**: The `document_id` UUID in the path provides natural uniqueness and matches the retrieval response's `document_id` field. Scope prefix distinguishes the two RAGs at the storage layer.

---

### 10. New Environment Variables

| Variable | Default | Notes |
|---|---|---|
| `AI_EMBED_URL` | `AI_CLOUD_URL` | OpenAI-compatible `/v1/embeddings` endpoint |
| `AI_EMBED_MODEL` | `text-embedding-3-small` | Embedding model name |
| `AI_EMBED_DIMENSION` | `1536` | Must match `VECTOR(n)` in migration |
| `RAG_CHUNK_TOKENS` | `512` | Approx tokens per chunk (byte/4 estimate) |
| `RAG_CHUNK_OVERLAP` | `64` | Overlap tokens between consecutive chunks |
| `RAG_TOKEN_BUDGET` | `2048` | Max tokens for similarity-ranked chunks per retrieval |

`AI_EMBED_DIMENSION` MUST match the `VECTOR(n)` dimension in migration `000007_dual_rag.up.sql`. Mismatch causes runtime errors. If the embedding model is changed, a new migration is required.

---

### 11. New go.mod Dependencies

| Package | Purpose |
|---|---|
| `github.com/pdfcpu/pdfcpu` | PDF text extraction (pure-Go) |

All other functionality uses stdlib: `archive/zip`, `encoding/xml`, `net/http`, `strings`, `strconv`.

No new database driver — existing `jackc/pgx/v5` supports pgvector via text encoding.
