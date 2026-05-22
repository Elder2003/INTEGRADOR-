# Quickstart: M9 — Format Validation

End-to-end verification of the format validation pipeline. Prerequisites: Docker Compose stack running with migrations applied through M9.

---

## Step 1 — Create activity with format rules

```http
POST /cursos/{cursoID}/actividades
Authorization: Bearer {docente_jwt}
Content-Type: application/json

{
  "title": "Examen Parcial 1",
  "tipo": "examen",
  "deadline": "2026-12-31T23:59:00Z",
  "instructions": "Entregar en PDF, mínimo 2 páginas.",
  "format_rules": {
    "min_pages": 2,
    "max_pages": 10,
    "min_word_count": 300,
    "required_types": ["pdf"]
  }
}
```

**Expected**: `201 Created`. Response body includes `format_rules` object in the actividad.

---

## Step 2 — Verify rules are persisted

```http
GET /cursos/{cursoID}/actividades/{actividadID}
Authorization: Bearer {docente_jwt}
```

**Expected**: `200 OK`. `format_rules` in response matches what was set in Step 1.

---

## Step 3 — Open exam session as student

```http
POST /actividades/{actividadID}/sessions
Authorization: Bearer {student_jwt}
```

**Expected**: `201 Created`. Note `session_id`.

---

## Step 4 — Upload a compliant PDF (≥2 pages, ≥300 words, PDF type)

```http
POST /submissions
Authorization: Bearer {student_jwt}
Content-Type: multipart/form-data

file: [2-page PDF with 350+ words]
actividad_id: {actividadID}
```

**Expected**: `201 Created`. Note `submission_id`.

---

## Step 5 — Close exam session

```http
POST /sessions/{sessionID}/close
Authorization: Bearer {student_jwt}
```

**Expected**: `200 OK`. Worker job enqueued.

---

## Step 6 — Wait for worker (≤3 minutes)

Wait up to 3 minutes for the async worker to process the job.

```sql
SELECT parse_status, file_type, page_count, word_count, compliant, rule_results
FROM format_reports
WHERE session_id = '{sessionID}';
```

**Expected**: One row with `parse_status = 'success'`, `file_type = 'pdf'`, `page_count >= 2`, `word_count >= 300`, `compliant = true`, `rule_results` contains pass entries for all four rules.

---

## Step 7 — Verify format report in detection report endpoint

```http
GET /sessions/{sessionID}/detection-report
Authorization: Bearer {docente_jwt}
```

**Expected**: `200 OK`. Response includes `format_report` sub-object with `compliant: true` and `rule_results` array with all rules passing.

---

## Step 8 — Upload a non-compliant DOCX (wrong type)

Repeat Steps 3–5 for a new session with the same activity, but upload a DOCX file instead of PDF.

**Expected after worker runs**: `format_reports` row has `compliant = false`, `rule_results` contains `{"rule": "file_type", "status": "fail", ...}`. Session is still accepted (Principle VII).

---

## Step 9 — Upload a single-page PDF (below min_pages)

Repeat Steps 3–5 for a new session, submitting a 1-page PDF.

**Expected after worker runs**: `format_reports` row has `compliant = false`, `rule_results` contains `{"rule": "min_pages", "status": "fail", "actual": 1, "required": 2}`. Submission accepted regardless.

---

## Step 10 — Verify student access is blocked

```http
GET /sessions/{sessionID}/detection-report
Authorization: Bearer {student_jwt}
```

**Expected**: `403 Forbidden`. Format report not visible to students (Principle IV).

---

## Step 11 — Verify password-protected PDF handling

Upload a password-protected PDF as the submission and close the session.

**Expected after worker runs**: `format_reports` row has `parse_status = 'failed'`, `fail_reason = 'password_protected'`, `compliant = false`. Detection report endpoint returns `format_report.parse_status = "failed"` with the reason.

---

## Step 12 — Verify activity with no format rules

Create an activity without `format_rules` and submit any valid PDF.

**Expected after worker runs**: `format_reports` row has `parse_status = 'success'`, `compliant = true`, `rule_results = []`. No rules configured means no violations.

---

## Step 13 — Verify idempotency

Trigger the worker job twice for the same session (simulate by re-enqueueing).

**Expected**: Only one `format_reports` row exists for the submission. No duplicate key error; second run is a no-op.

---

## Step 14 — Update activity rules and verify new submissions use updated rules

Update the activity to change `min_pages` to 5.

```http
PUT /cursos/{cursoID}/actividades/{actividadID}
Authorization: Bearer {docente_jwt}
Content-Type: application/json

{
  "format_rules": { "min_pages": 5, "required_types": ["pdf"] }
}
```

Then submit a 3-page PDF in a new session.

**Expected**: New format report shows `min_pages: fail, actual: 3, required: 5`. The old format report from earlier sessions is unchanged.
