# Data Model: M4 — File Storage

## Migration: `000004_file_storage`

### ALTER TABLE actividades

```sql
-- Adds the docente-configurable late submission flag.
-- Default true: existing activities accept late submissions unless the docente disables it.
ALTER TABLE actividades
  ADD COLUMN IF NOT EXISTS late_submission_allowed BOOLEAN NOT NULL DEFAULT true;
```

**Notes**:
- `late_submission_allowed = true` (default): submissions are accepted after the deadline and marked `is_late = true`.
- `late_submission_allowed = false`: submissions received after the deadline are rejected with HTTP 422.
- The edit window duration for on-time submissions is capped at `min(submitted_at + 30min, deadline)` when this flag is `false`, and always `submitted_at + 30min` when `true`.
- Every change to this column is recorded in `actividad_config_changes` (see below). Never update this column without writing a config change log entry in the same transaction.

---

### actividad_config_changes

```sql
CREATE TABLE actividad_config_changes (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  actividad_id   UUID        NOT NULL REFERENCES actividades(id),
  changed_by     UUID        NOT NULL REFERENCES users(id),
  field_name     TEXT        NOT NULL,           -- e.g. 'late_submission_allowed'
  previous_value TEXT        NOT NULL,
  new_value      TEXT        NOT NULL,
  changed_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_actividad_config_changes_actividad
  ON actividad_config_changes(actividad_id, changed_at DESC);
```

**Notes**:
- Append-only — no UPDATE or DELETE ever permitted on this table (Constitution Principle I spirit).
- `field_name` is generic (`TEXT`) to support future activity config fields without a schema change.
- `previous_value` and `new_value` are stored as text representations (`'true'` / `'false'` for boolean).
- Written in the same DB transaction as the `UPDATE actividades SET late_submission_allowed = ...`.
- Readable by: the owning docente (scoped to their actividades) and admins (global scope).

---

### submissions

```sql
CREATE TABLE submissions (
  id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  student_id             UUID        NOT NULL REFERENCES users(id),
  actividad_id           UUID        NOT NULL REFERENCES actividades(id),
  storage_key            TEXT        NOT NULL,   -- internal MinIO object key, never exposed to client
  original_filename      TEXT        NOT NULL,
  file_size_bytes        BIGINT      NOT NULL CHECK (file_size_bytes > 0),
  mime_type              TEXT        NOT NULL,
  submitted_at           TIMESTAMPTZ NOT NULL,   -- server-side timestamp at storage confirmation
  is_late                BOOLEAN     NOT NULL,   -- true iff submitted_at > actividades.deadline
  edit_window_expires_at TIMESTAMPTZ,            -- NULL for late submissions (locked immediately)
  last_replaced_at       TIMESTAMPTZ,            -- NULL if file was never replaced within the window
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (student_id, actividad_id)
);

CREATE INDEX idx_submissions_actividad
  ON submissions(actividad_id);

CREATE INDEX idx_submissions_student
  ON submissions(student_id);
```

**Notes**:
- `UNIQUE (student_id, actividad_id)` enforces the single-active-submission invariant at the DB level. Concurrent first uploads from the same student cause a unique violation on the second INSERT; the handler cleans up the orphan MinIO object and returns HTTP 409.
- `storage_key` format: `submissions/{actividad_id}/{student_id}/{uuid}{ext}`. Generated server-side; never sent to clients.
- `submitted_at` records the moment `minio.PutObject` returned successfully — not the moment the HTTP request arrived. Set explicitly in the INSERT.
- `is_late` is set at insert time from `submitted_at > actividades.deadline` and is immutable after creation. For replacements within the edit window, `is_late` MAY change (a within-window replacement after the deadline flips it to `true`).
- `edit_window_expires_at`:
  - For on-time submissions with `late_submission_allowed = false`: `submitted_at + 30 minutes`, capped at `actividades.deadline`.
  - For on-time submissions with `late_submission_allowed = true`: `submitted_at + 30 minutes` (uncapped).
  - For late submissions: `NULL` (locked immediately).
- `is_locked` is NOT stored. It is derived at query time: `(edit_window_expires_at IS NULL) OR (NOW() > edit_window_expires_at)`.
- `last_replaced_at` is set to `NOW()` each time the student successfully replaces the file within the edit window.
- Hard DELETE is permitted within the edit window (student exercises their right to remove and re-upload). After the window expires or if `is_late = true`, the record becomes immutable and cannot be deleted.
- On first submission for an activity, `has_submissions` on `actividades` is flipped to `true` in the same transaction (enabling M3's title/type edit guard).

---

## Storage Object Lifecycle

| Event | MinIO action | DB action |
|---|---|---|
| First upload (on-time or late) | `PutObject(key)` | `INSERT INTO submissions` + `UPDATE actividades SET has_submissions = true` (if first) |
| Replace within edit window | `PutObject(new_key)` then `RemoveObject(old_key)` after DB commit | `UPDATE submissions SET storage_key, filename, size, mime_type, submitted_at, is_late, last_replaced_at` |
| Delete within edit window | `RemoveObject(key)` after DB delete | `DELETE FROM submissions` |
| Download (any authorized role) | `GetObject(key)` streamed to client | No write; auth check only |
| Upload fails mid-transfer | MinIO object may or may not exist | No submission record created |
| DB insert fails after MinIO success | `RemoveObject(key)` (best-effort cleanup) | No submission record |

---

## Entity Relationship Summary

```
actividades (M3) ──< submissions >── users(student)
     │
     └──< actividad_config_changes >── users(docente)
```

Extended from M3:

```
carreras ──< materias ──< cursos >── users(docente)
                              │
                  ┌───────────┼──────────────────────────────────┐
                  │           │                                  │
               grupos      actividades                      inscripciones >── users(student)
                  │           │         └──< actividad_config_changes
         grupo_asignaciones   ├──< codocente_grants >── users(co_docente)
         >── users(student)   └──< submissions >── users(student)
```

---

## Down Migration

```sql
-- 000004_file_storage.down.sql
DROP TABLE IF EXISTS submissions;
DROP TABLE IF EXISTS actividad_config_changes;

ALTER TABLE actividades
  DROP COLUMN IF EXISTS late_submission_allowed;
```
