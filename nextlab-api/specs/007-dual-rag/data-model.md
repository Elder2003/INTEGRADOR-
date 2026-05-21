# Data Model: M7 — Dual RAG Pipeline

**Migration**: `000007_dual_rag`  
**Date**: 2026-05-20

---

## Tables

### `rag_curricular` — Curricular Knowledge Documents

Stores metadata for documents in the curricular RAG. Scoped by `carrera` code, `semestre`, and `codigo_materia` code (denormalized text fields, not FK UUIDs — allows ingestion without requiring matching rows in `carreras`/`materias`).

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | Returned as `document_id` in responses |
| `carrera` | TEXT | NOT NULL | Programme code, e.g. `"ISC"` |
| `semestre` | SMALLINT | NOT NULL, CHECK 1–10 | Academic semester |
| `codigo_materia` | TEXT | NOT NULL | Subject code, e.g. `"INT-6001"` |
| `actividad_id` | UUID | NULLABLE, FK → actividades(id) ON DELETE SET NULL | Set when document is a rubric for a specific activity |
| `is_rubric` | BOOLEAN | NOT NULL DEFAULT FALSE | Rubric documents skip chunking; returned in full |
| `embed_model` | TEXT | NOT NULL | Model used for embeddings, e.g. `"text-embedding-3-small"` |
| `embed_dimension` | INT | NOT NULL | Vector dimension, must match migration VECTOR(n) |
| `minio_path` | TEXT | NOT NULL | `rag-docs/curricular/{id}/original.{ext}` |
| `original_filename` | TEXT | NOT NULL | Original upload filename |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT now() | |

**Indexes**:
- `rag_curricular_scope_idx ON (carrera, semestre, codigo_materia)` — used by all retrieval filters

---

### `rag_curricular_chunks` — Curricular Document Chunks

Each row is one embedded chunk from a `rag_curricular` document. Both `pending` and `active` rows coexist during two-phase re-ingest.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | |
| `document_id` | UUID | NOT NULL, FK → rag_curricular(id) ON DELETE CASCADE | |
| `chunk_index` | INT | NOT NULL | 0-based sequence index within the document |
| `chunk_text` | TEXT | NOT NULL | Raw extracted text for this chunk |
| `embedding` | VECTOR(1536) | NULLABLE | NULL until embedding API responds; dimension = AI_EMBED_DIMENSION |
| `status` | TEXT | NOT NULL CHECK IN ('pending','active') DEFAULT 'pending' | Only 'active' chunks are returned by retrieval |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT now() | |

**Indexes**:
- `rag_curricular_chunks_doc_status_idx ON (document_id, status)` — used by two-phase ingest queries
- `rag_curricular_chunks_vector_idx USING ivfflat (embedding vector_cosine_ops) WITH (lists = 10)` — used by similarity search

**Lifecycle**:
```
INSERT (status='pending')
  → embedding API called
  → embedding stored in row
  → if ALL succeed: old 'active' chunks deleted, pending promoted to 'active'
  → if ANY fail:   pending chunks deleted, old 'active' chunks survive
```

---

### `rag_docente` — Docente Knowledge Documents

Stores metadata for documents in the docente RAG. Scoped by `docente_id` + `curso_id`, with optional `actividad_id` sub-filter.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | Returned as `document_id` in responses |
| `docente_id` | UUID | NOT NULL, FK → users(id) | Owner; enforced for all reads and writes |
| `curso_id` | UUID | NOT NULL, FK → cursos(id) ON DELETE CASCADE | |
| `actividad_id` | UUID | NULLABLE, FK → actividades(id) ON DELETE SET NULL | Optional sub-filter; required when `is_rubric=TRUE` |
| `is_rubric` | BOOLEAN | NOT NULL DEFAULT FALSE | |
| `embed_model` | TEXT | NOT NULL | |
| `embed_dimension` | INT | NOT NULL | |
| `minio_path` | TEXT | NOT NULL | `rag-docs/docente/{id}/original.{ext}` |
| `original_filename` | TEXT | NOT NULL | |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT now() | |

**Indexes**:
- `rag_docente_scope_idx ON (docente_id, curso_id)` — used by all retrieval filters

---

### `rag_docente_chunks` — Docente Document Chunks

Mirror of `rag_curricular_chunks`, backed by `rag_docente` documents. Physically separate table (FR-002, Principle V).

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | |
| `document_id` | UUID | NOT NULL, FK → rag_docente(id) ON DELETE CASCADE | |
| `chunk_index` | INT | NOT NULL | |
| `chunk_text` | TEXT | NOT NULL | |
| `embedding` | VECTOR(1536) | NULLABLE | Same lifecycle as curricular chunks |
| `status` | TEXT | NOT NULL CHECK IN ('pending','active') DEFAULT 'pending' | |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT now() | |

**Indexes**:
- `rag_docente_chunks_doc_status_idx ON (document_id, status)`
- `rag_docente_chunks_vector_idx USING ivfflat (embedding vector_cosine_ops) WITH (lists = 10)`

---

## Relationships

```
users (M2)
  └── rag_docente.docente_id

cursos (M3)
  └── rag_docente.curso_id

actividades (M3)
  ├── rag_curricular.actividad_id   (nullable — rubric association)
  └── rag_docente.actividad_id      (nullable — rubric association)

rag_curricular
  └── rag_curricular_chunks.document_id  (CASCADE DELETE)

rag_docente
  └── rag_docente_chunks.document_id     (CASCADE DELETE)
```

---

## Migration Notes

- **VECTOR(1536)**: Hardcoded at migration time. Matches the default `AI_EMBED_DIMENSION=1536` for `text-embedding-3-small`. If the embedding model is changed, a new migration with a different VECTOR(n) dimension is required.
- **IVFFlat index**: Requires at least one row to be created before the index can be built. In practice, the index is populated as chunks are ingested.
- **pgvector extension**: Already enabled in migration `000001_init.up.sql` (`CREATE EXTENSION IF NOT EXISTS vector`).
- **No UPDATE/DELETE on document rows**: Only `rag_*_chunks` rows are deleted (pending cleanup + two-phase swap). Document rows in `rag_curricular` and `rag_docente` are append-only.

---

## State Transition Diagram (Chunk Status)

```
                  ingest begins
                       │
                       ▼
               status = 'pending'
               embedding = NULL
                       │
           ┌───────────┴───────────┐
           │ embedding call        │ any embedding
           │ succeeds              │ call fails
           ▼                       ▼
   embedding stored           DELETE pending chunks
   in chunk row               old 'active' survive
           │
   all chunks embedded
           │
           ▼
    TX-2: DELETE old 'active'
          UPDATE pending → 'active'
          COMMIT
           │
           ▼
               status = 'active'
          (retrievable by queries)
```
