# API Contract: M7 — Dual RAG Pipeline

**Base path**: internal HTTP service — all endpoints require `Authorization: Bearer <jwt>`.  
**Content-Type**: `application/json` (query endpoints); `multipart/form-data` (ingest endpoints).

---

## Endpoints

### POST /rag/curricular/ingest

Ingest a document into the curricular knowledge base. Admin only.

**Auth**: `role=admin`

**Request** — `multipart/form-data`:

| Field | Type | Required | Description |
|---|---|---|---|
| `file` | binary | Yes | PDF, .docx, or .txt file to ingest |
| `carrera` | string | Yes | Programme code, e.g. `"ISC"` |
| `semestre` | string | Yes | Academic semester 1–10 |
| `codigo_materia` | string | Yes | Subject code, e.g. `"INT-6001"` |
| `is_rubric` | string | No | `"true"` or `"false"` (default `"false"`) |
| `actividad_id` | string | No | UUID of the associated activity (required when `is_rubric=true`) |
| `document_id` | string | No | Existing UUID to trigger a two-phase re-ingest replace |

**Response 201** — first ingest:
```json
{
  "document_id": "550e8400-e29b-41d4-a716-446655440000",
  "chunk_count": 42,
  "status": "ingested"
}
```

**Response 200** — re-ingest (existing `document_id` supplied):
```json
{
  "document_id": "550e8400-e29b-41d4-a716-446655440000",
  "chunk_count": 45,
  "status": "re-ingested"
}
```

**Response 422** — unsupported file type:
```json
{
  "error": "unsupported_file_type",
  "message": "only PDF, DOCX, and TXT files are accepted"
}
```

**Response 400** — missing required field:
```json
{
  "error": "invalid_request",
  "message": "carrera is required"
}
```

**Response 500** — embedding service unavailable during ingest:
```json
{
  "error": "embedding_failed",
  "message": "embedding service unavailable; document saved but not indexed"
}
```

---

### POST /rag/curricular/query

Retrieve context from the curricular RAG. Scoped by programme, semester, and subject code. Used by the detection engine (M8) and interdisciplinary analysis (M12).

**Auth**: any authenticated user (internal service call — students cannot reach this endpoint)

**Request body**:
```json
{
  "carrera": "ISC",
  "semestre": 6,
  "codigo_materia": "INT-6001",
  "query": "machine learning classification algorithm",
  "max_chunks": 10,
  "actividad_id": "uuid"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `carrera` | string | Yes | Programme code filter |
| `semestre` | integer | Yes | Semester filter |
| `codigo_materia` | string | Yes | Subject code filter |
| `query` | string | Yes | Free-text similarity query |
| `max_chunks` | integer | No | Max non-rubric chunks to return (default: `RAG_TOKEN_BUDGET / avg_chunk_size`) |
| `actividad_id` | string | No | UUID to include the activity's rubric (if any) at the top of the response |

**Response 200**:
```json
{
  "rubrics": [
    {
      "document_id": "uuid",
      "chunk_index": 0,
      "chunk_text": "Rubric: Criterion 1 — Complexity (30%)...",
      "scope": {
        "carrera": "ISC",
        "semestre": 6,
        "codigo_materia": "INT-6001"
      }
    }
  ],
  "chunks": [
    {
      "document_id": "uuid",
      "chunk_index": 3,
      "chunk_text": "Neural networks are a class of machine learning models...",
      "similarity_score": 0.871,
      "scope": {
        "carrera": "ISC",
        "semestre": 6,
        "codigo_materia": "INT-6001"
      }
    }
  ]
}
```

`rubrics` is an empty array when no rubric exists or `actividad_id` is not provided.  
`chunks` is an empty array when no documents are ingested for the scope (no error — SC-003 compliance).

**Response 400** — missing required scope field:
```json
{
  "error": "invalid_request",
  "message": "carrera, semestre, and codigo_materia are required"
}
```

---

### POST /rag/docente/ingest

Ingest a document into the docente knowledge base. Docentes can only ingest for their own courses.

**Auth**: `role=docente`; `docente_id` derived from JWT claims.

**Request** — `multipart/form-data`:

| Field | Type | Required | Description |
|---|---|---|---|
| `file` | binary | Yes | PDF, .docx, or .txt file |
| `curso_id` | string | Yes | UUID of the course to scope this document |
| `actividad_id` | string | No | UUID of the associated activity |
| `is_rubric` | string | No | `"true"` or `"false"` (default `"false"`) |
| `document_id` | string | No | Existing UUID to trigger re-ingest |

**Response 201**:
```json
{
  "document_id": "uuid",
  "chunk_count": 18,
  "status": "ingested"
}
```

**Response 200** — re-ingest:
```json
{
  "document_id": "uuid",
  "chunk_count": 20,
  "status": "re-ingested"
}
```

**Response 403** — docente does not own `curso_id`:
```json
{
  "error": "forbidden",
  "message": "you do not have access to this course"
}
```

**Response 422** — unsupported file type (same as curricular ingest)

---

### POST /rag/docente/query

Retrieve context from the docente RAG. Scoped to the authenticated docente's course.

**Auth**: `role=docente`; `docente_id` derived from JWT claims.

**Request body**:
```json
{
  "curso_id": "uuid",
  "actividad_id": "uuid",
  "query": "student struggled with recursion in the last assignment",
  "max_chunks": 8
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `curso_id` | string | Yes | Course scope filter |
| `actividad_id` | string | No | Optional activity sub-filter; also triggers rubric injection |
| `query` | string | Yes | Free-text similarity query |
| `max_chunks` | integer | No | Max non-rubric chunks (default from `RAG_TOKEN_BUDGET`) |

**Response 200** — same structure as `/rag/curricular/query`:
```json
{
  "rubrics": [...],
  "chunks": [
    {
      "document_id": "uuid",
      "chunk_index": 1,
      "chunk_text": "...",
      "similarity_score": 0.812,
      "scope": {
        "docente_id": "uuid",
        "curso_id": "uuid"
      }
    }
  ]
}
```

**Response 403** — querying another docente's course:
```json
{
  "error": "forbidden",
  "message": "you do not have access to this course"
}
```

---

## Role Access Matrix

| Endpoint | admin | docente | estudiante |
|---|---|---|---|
| POST /rag/curricular/ingest | ✅ | ❌ | ❌ |
| POST /rag/curricular/query | ✅ | ✅ (M8 service) | ❌ |
| POST /rag/docente/ingest | ❌ | ✅ (own courses only) | ❌ |
| POST /rag/docente/query | ❌ | ✅ (own courses only) | ❌ |

---

## Common Error Codes

| Code | HTTP Status | Meaning |
|---|---|---|
| `invalid_request` | 400 | Missing required field or bad value |
| `unsupported_file_type` | 422 | File extension not PDF/DOCX/TXT |
| `forbidden` | 403 | JWT role mismatch or cross-scope access |
| `not_found` | 404 | `document_id` supplied for re-ingest does not exist |
| `embedding_failed` | 500 | Embedding service unavailable; ingest aborted |
