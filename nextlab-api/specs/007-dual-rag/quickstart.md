# Quickstart & Verification: M7 — Dual RAG Pipeline

**Prerequisites**: Docker Compose stack running (`postgres`, `minio`, `api`), migration `000007` applied, admin JWT and docente JWT available.

---

## Setup

```bash
# 1. Apply migration
docker compose run --rm migrate up

# Verify 4 new tables exist
docker compose exec postgres psql -U postgres -d nextlab -c \
  "\dt rag_*"
# Expected: rag_curricular, rag_curricular_chunks, rag_docente, rag_docente_chunks

# 2. Obtain tokens (from /auth/login)
ADMIN_TOKEN="<admin-jwt>"
DOCENTE_A_TOKEN="<docente-a-jwt>"
DOCENTE_B_TOKEN="<docente-b-jwt>"
```

---

## Step 1 — Ingest Curricular Document (admin)

```bash
curl -s -X POST http://localhost:8080/rag/curricular/ingest \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "file=@testdata/syllabus.pdf" \
  -F "carrera=ISC" \
  -F "semestre=6" \
  -F "codigo_materia=INT-6001"
```

**Expected**:
```json
{ "document_id": "<uuid>", "chunk_count": 42, "status": "ingested" }
```

Save `document_id` as `CURR_DOC_ID`.

---

## Step 2 — Verify Chunks Are Active

```bash
docker compose exec postgres psql -U postgres -d nextlab -c \
  "SELECT COUNT(*), status FROM rag_curricular_chunks
   WHERE document_id = '$CURR_DOC_ID'
   GROUP BY status;"
```

**Expected**: rows with `status = 'active'`, count matches `chunk_count` from step 1.

---

## Step 3 — Query Curricular RAG

```bash
curl -s -X POST http://localhost:8080/rag/curricular/query \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "carrera": "ISC",
    "semestre": 6,
    "codigo_materia": "INT-6001",
    "query": "evaluation criteria for the final project",
    "max_chunks": 5
  }'
```

**Expected**: `chunks` array non-empty with `similarity_score` values; `rubrics` array empty (no rubric yet).

---

## Step 4 — Ingest Rubric Document (admin)

```bash
curl -s -X POST http://localhost:8080/rag/curricular/ingest \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "file=@testdata/rubric.pdf" \
  -F "carrera=ISC" \
  -F "semestre=6" \
  -F "codigo_materia=INT-6001" \
  -F "is_rubric=true" \
  -F "actividad_id=<actividad-uuid>"
```

**Expected**:
```json
{ "document_id": "<uuid>", "chunk_count": 1, "status": "ingested" }
```

Rubrics produce exactly 1 chunk (no splitting). Save `document_id` as `RUBRIC_DOC_ID`.

---

## Step 5 — Query with Rubric Injection

```bash
curl -s -X POST http://localhost:8080/rag/curricular/query \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "carrera": "ISC",
    "semestre": 6,
    "codigo_materia": "INT-6001",
    "query": "grading criteria",
    "actividad_id": "<actividad-uuid>",
    "max_chunks": 3
  }'
```

**Expected**: `rubrics` array contains exactly 1 entry with the full rubric text; `chunks` contains up to 3 similarity-ranked entries. Rubric is NOT included in `chunks`.

---

## Step 6 — Re-Ingest Curricular Document (two-phase replace)

```bash
curl -s -X POST http://localhost:8080/rag/curricular/ingest \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "file=@testdata/syllabus_v2.pdf" \
  -F "carrera=ISC" \
  -F "semestre=6" \
  -F "codigo_materia=INT-6001" \
  -F "document_id=$CURR_DOC_ID"
```

**Expected**:
```json
{ "document_id": "<same-uuid>", "chunk_count": 45, "status": "re-ingested" }
```

Verify no stale chunks:
```bash
docker compose exec postgres psql -U postgres -d nextlab -c \
  "SELECT COUNT(*), status FROM rag_curricular_chunks
   WHERE document_id = '$CURR_DOC_ID'
   GROUP BY status;"
```

**Expected**: only rows with `status = 'active'`; no `pending` rows; count = 45.

---

## Step 7 — Ingest Docente Document (docente A)

```bash
CURSO_A_ID="<uuid of docente-a's course>"

curl -s -X POST http://localhost:8080/rag/docente/ingest \
  -H "Authorization: Bearer $DOCENTE_A_TOKEN" \
  -F "file=@testdata/lecture_notes.docx" \
  -F "curso_id=$CURSO_A_ID"
```

**Expected**:
```json
{ "document_id": "<uuid>", "chunk_count": 18, "status": "ingested" }
```

---

## Step 8 — Query Docente RAG as Docente A

```bash
curl -s -X POST http://localhost:8080/rag/docente/query \
  -H "Authorization: Bearer $DOCENTE_A_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"curso_id\": \"$CURSO_A_ID\", \"query\": \"recursion implementation\"}"
```

**Expected**: `chunks` non-empty with documents belonging to docente A.

---

## Step 9 — Cross-Scope Isolation (Docente B cannot access Docente A's data)

```bash
curl -s -X POST http://localhost:8080/rag/docente/query \
  -H "Authorization: Bearer $DOCENTE_B_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"curso_id\": \"$CURSO_A_ID\", \"query\": \"recursion\"}"
```

**Expected**: `403 Forbidden` — docente B does not own `$CURSO_A_ID`.

---

## Step 10 — Empty Scope Returns Empty Results (No Error)

```bash
curl -s -X POST http://localhost:8080/rag/curricular/query \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"carrera":"ISC","semestre":9,"codigo_materia":"NONEXISTENT","query":"test"}'
```

**Expected** `200` with:
```json
{ "rubrics": [], "chunks": [] }
```

---

## Step 11 — Unsupported File Type Rejected

```bash
curl -s -X POST http://localhost:8080/rag/curricular/ingest \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "file=@testdata/spreadsheet.xlsx" \
  -F "carrera=ISC" \
  -F "semestre=6" \
  -F "codigo_materia=INT-6001"
```

**Expected** `422`:
```json
{ "error": "unsupported_file_type", "message": "only PDF, DOCX, and TXT files are accepted" }
```

---

## Step 12 — Word (.docx) Ingest

```bash
curl -s -X POST http://localhost:8080/rag/curricular/ingest \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "file=@testdata/competency_framework.docx" \
  -F "carrera=ISC" \
  -F "semestre=5" \
  -F "codigo_materia=INT-5002"
```

**Expected**: `201` with `chunk_count > 0`. Verify text in DB:
```bash
docker compose exec postgres psql -U postgres -d nextlab -c \
  "SELECT LEFT(chunk_text, 100) FROM rag_curricular_chunks
   WHERE document_id = '<returned-uuid>' AND status = 'active' LIMIT 3;"
```

**Expected**: readable text extracted from the .docx (not XML tags or binary).

---

## Step 13 — MinIO Path Verification

```bash
docker compose exec minio mc ls --recursive local/rag-docs/
```

**Expected path format**: `rag-docs/curricular/<document-id>/original.pdf` and `rag-docs/docente/<document-id>/original.docx`.

---

## Step 14 — Student Cannot Access Ingest or Query Endpoints

```bash
STUDENT_TOKEN="<student-jwt>"

curl -s -o /dev/null -w "%{http_code}" \
  -X POST http://localhost:8080/rag/curricular/ingest \
  -H "Authorization: Bearer $STUDENT_TOKEN" \
  -F "file=@testdata/syllabus.pdf" \
  -F "carrera=ISC" -F "semestre=6" -F "codigo_materia=INT-6001"
# Expected: 403
```

---

## Step 15 — Retrieval Latency Check

```bash
time curl -s -X POST http://localhost:8080/rag/curricular/query \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"carrera":"ISC","semestre":6,"codigo_materia":"INT-6001","query":"algorithms","max_chunks":10}'
```

**Expected**: completes in under 2 seconds (SC-006 — up to 10,000 chunks).
