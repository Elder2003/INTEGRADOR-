# Research: MW2 — Docente Course Management

**Date**: 2026-05-25  
**Spec**: specs/002-mw2-docente-courses/spec.md

---

## Decision 1: Activity tipo enum — 5 values, not 7

**Decision**: The `actividades` table uses a PostgreSQL ENUM (`actividad_tipo`) with exactly 5 values: `practica`, `tarea`, `informe`, `examen`, `defensa_oral`. The types `investigacion` and `git_analysis` are handled by the separate `projects` module (MW7/MW8) with their own tables — they are NOT in `actividades`.

**Rationale**: The DB migration `000003_academic.up.sql` defines:
```sql
CREATE TYPE actividad_tipo AS ENUM (
  'practica', 'tarea', 'informe', 'examen', 'defensa_oral'
);
```
The API error message confirms: "allowed: practica, tarea, informe, examen, defensa_oral".

**Impact on spec**: US5 AC1 and FR-009 listed 7 types. Both are corrected: the activity creation form shows only these 5 valid tipos.

**Alternatives considered**: Including investigacion/git_analysis in the dropdown — rejected because the API would return a 400 validation error.

---

## Decision 2: Materia listing is admin-only — API gap

**Decision**: `GET /admin/materias` requires the `admin` role (registered on an admin-gated route group). Docentes receive a 403 if they call this endpoint. For MW2, the course creation form will call `GET /admin/materias` and gracefully degrade: if the response is 403 or the list is empty, the form shows an instructional message ("Contact your admin to get the materia ID") and falls back to a plain UUID text input for `materia_id`.

**Rationale**: The spec assumed a docente-accessible materias dropdown. The backend does not expose one. Adding a new backend endpoint (`GET /cursos/materias` accessible to docentes) is the long-term fix but is a backend change outside this milestone's scope.

**Alternatives considered**:
- Block course creation entirely until a proper endpoint exists — rejected (too disruptive; docentes can still work with a UUID input)
- Store materias on the client after an admin login — rejected (violates session isolation)

**Future work**: Track as a backend improvement: add `GET /materias?is_active=true` accessible to authenticated docentes.

---

## Decision 3: Activity edit field locking is submission-based, not creation-based

**Decision**: The `title` and `tipo` fields of an actividad are editable as long as `has_submissions = false`. Once any submission exists (`has_submissions = true`), the API returns `field_locked` (422) if title or tipo are changed. The `deadline`, `instructions`, and `late_submission_allowed` fields remain editable regardless.

**Rationale**: `UpdateActividadHandler` explicitly checks `existing.HasSubmissions` and branches:
- `has_submissions = false`: full update allowed
- `has_submissions = true`: only deadline, instructions, format_rules can change

**Impact on spec**: US5 AC6 said "Activity type cannot be changed after creation." This was wrong — it can change before any submissions exist. The edit form should:
- Always render title and tipo as editable inputs
- Disable them and show "Locked: submissions exist" when `actividad.has_submissions = true`
- Rely on the API's `field_locked` error as the authoritative enforcement

---

## Decision 4: Curso deletion blocks on submissions, not enrolled students

**Decision**: `DELETE /cursos/:id` returns 409 (`has_submissions`) if the course has any submission records — it does NOT check enrolled student count. The spec said "delete a course that has no enrolled students" but the actual constraint is broader.

**Rationale**: `DeleteCursoHandler` calls `CursoHasSubmissions` and blocks on that. Enrolled students do not block deletion. The frontend ConfirmModal should say "Delete this course and all its activities? This cannot be undone." — the API provides the actual constraint enforcement.

---

## Decision 5: No GET endpoint for listing co-docente grants

**Decision**: The API exposes `POST /actividades/:aid/grants` (create) and `DELETE /actividades/:aid/grants/:uid` (revoke) but has no `GET /actividades/:aid/grants` (list). US7 AC2 ("Granted co-docentes appear in a list") cannot be fully implemented from the API alone.

**Rationale**: Confirmed from `academic/module.go` — only POST and DELETE are registered on the grants path.

**Resolution for MW2**: The activity detail page tracks grants in React component state: after a successful `POST /actividades/:aid/grants`, the returned `CodocenteGrant` object is added to local state; on revoke (`DELETE`), it is removed. The list is lost on page refresh. A note is shown: "Grants refresh on reload."

**Future work**: Backend should add `GET /actividades/:aid/grants` returning `[]CodocenteGrant`.

---

## Decision 6: Deadline field requires datetime input

**Decision**: The API stores `deadline` as `TIMESTAMPTZ` and accepts ISO 8601 datetime strings. The activity creation and edit forms use `<input type="datetime-local">`, which produces a `YYYY-MM-DDTHH:mm` string. Before POSTing, append `:00` and convert to full ISO 8601 with timezone offset (or send as UTC `Z`).

**Rationale**: A date-only input would default to midnight UTC, which may not be the intended deadline. Giving the docente time control is necessary.

---

## Decision 7: Student enrollment happens automatically — no docente action needed

**Decision**: `inscripciones` rows are created automatically when a student logs in for the first time via Google OAuth (the student's `student_profile` is matched to active courses by carrera/semester). Docentes do NOT enroll students manually. The roster endpoint (`GET /cursos/:id/students`) returns whoever is already enrolled.

**Rationale**: Migration `000003_academic.up.sql` shows `inscripciones` has `carrera_id` and `semester` matching the student profile. Auto-enrollment is implied. There is no `POST /cursos/:id/students` endpoint.

**Impact on spec**: Assumption 2 (enrollment is database-level) is confirmed correct. The Estudiantes tab is read-only — docentes can only assign enrolled students to groups.

---

## Summary of spec corrections

| Issue | Original spec | Corrected |
|-------|--------------|-----------|
| Activity tipos | 7 (incl. investigacion, git_analysis) | 5 (practica, tarea, informe, examen, defensa_oral) |
| Title/tipo mutability | Immutable after creation | Mutable until first submission |
| Materia dropdown | Populated from API | Falls back to manual UUID input (admin-only endpoint) |
| Curso delete constraint | No enrolled students | No submissions |
| Grant list | Always visible | Session-state only (no GET endpoint) |
