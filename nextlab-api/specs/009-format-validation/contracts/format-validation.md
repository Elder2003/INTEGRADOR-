# API Contracts: M9 — Format Validation

## Modified Endpoints

### `POST /cursos/:cursoID/actividades` — Create activity with optional format rules

**Extension**: Adds optional `format_rules` field to existing request body. No breaking change.

**Request body (addition)**:
```json
{
  "title": "Examen Parcial 1",
  "tipo": "examen",
  "deadline": "2026-06-01T23:59:00Z",
  "instructions": "...",
  "format_rules": {
    "min_pages": 2,
    "max_pages": 10,
    "min_word_count": 500,
    "required_types": ["pdf"]
  }
}
```

`format_rules` is fully optional. When absent or null, no rules are configured for the activity.

**Responses**: unchanged from M3 (`201`, `400`, `403`).

---

### `PUT /cursos/:cursoID/actividades/:actividadID` — Update activity format rules

**Extension**: Adds optional `format_rules` field. Omitting `format_rules` leaves existing rules unchanged. Setting `format_rules: null` clears all rules.

**Responses**: unchanged from M3 (`200`, `400`, `403`, `404`).

---

### `GET /sessions/:sessionID/detection-report` — Detection report (M8 extended)

**Extension**: Adds `format_report` sub-object when format validation has completed.

**Response `200`**:
```json
{
  "session_id": "uuid",
  "student_id": "uuid",
  "actividad_id": "uuid",
  "format_report": {
    "id": "uuid",
    "submission_id": "uuid",
    "session_id": "uuid",
    "parse_status": "success",
    "file_type": "pdf",
    "page_count": 5,
    "word_count": 1234,
    "char_count": 7890,
    "doc_metadata": {
      "author": "Juan Perez",
      "created_at": "2026-05-22T10:00:00Z"
    },
    "compliant": true,
    "rule_results": [
      { "rule": "min_pages", "status": "pass", "actual": 5, "required": 2 },
      { "rule": "file_type",  "status": "pass", "actual": "pdf", "required": ["pdf"] }
    ],
    "created_at": "2026-05-22T10:05:00Z"
  },
  "heuristic": { ... },
  "llm": { ... },
  "llm_status": "success",
  "grade_decision": null
}
```

**When format validation failed (parse error)**:
```json
{
  "format_report": {
    "parse_status": "failed",
    "fail_reason": "password_protected",
    "compliant": false,
    "rule_results": [],
    "created_at": "..."
  }
}
```

**When format validation is still pending (not yet run)**:
```json
{
  "format_report": null
}
```

**When no file was submitted**:
```json
{
  "format_report": {
    "parse_status": "failed",
    "fail_reason": "no_submission",
    "compliant": false,
    "rule_results": []
  }
}
```

**Access**: Docente-only. Student role → `403 Forbidden` (Principle IV, same as M8).

---

## Error Responses (format_rules validation)

| Code | Error | Trigger |
|------|-------|---------|
| `422` | `invalid_format_rules` | `min_pages > max_pages`, negative values, empty `required_types` array |
| `403` | `forbidden` | Attempting to set rules on another docente's activity |
