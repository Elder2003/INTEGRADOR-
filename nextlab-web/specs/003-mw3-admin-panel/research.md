# Research: MW3 — Admin Panel

**Feature**: MW3 — Admin Panel  
**Date**: 2026-05-26  
**Source**: `nextlab-api` source code inspection (`internal/academic/`, migrations)

---

## Decision 1 — Student profiles have no DELETE endpoint

**Decision**: Student profile "deactivation" calls `PUT /admin/student-profiles/:id` with `{ is_active: false }`. There is no `DELETE /admin/student-profiles/:id` route.

**Rationale**: The server only registers `Get`, `Post`, and `Put` for `/admin/student-profiles`. Hard-deleting a student profile would break referential integrity with `inscripciones` and Google OAuth login history. Soft-deactivation is the correct approach.

**Alternatives considered**: Hard DELETE — rejected because no endpoint exists and would lose enrollment history.

**Impact on implementation**: The `deactivateStudentProfile(id)` function in `src/api/admin.ts` calls `PUT /admin/student-profiles/:id` with `{ is_active: false }`, not `DELETE`. The confirm modal label should say "Desactivar" not "Eliminar".

---

## Decision 2 — Carreras and materias use soft-delete via DELETE endpoint

**Decision**: `DELETE /admin/carreras/:id` and `DELETE /admin/materias/:id` exist and trigger server-side soft-deactivation (`is_active = false`). The response is `204 No Content`. The frontend calls `api.delete(...)` normally.

**Rationale**: The handler internally checks for active dependents (materias for carreras, cursos for materias) and returns `409 Conflict` if blocked. If not blocked, it sets `is_active = false`. From the frontend's perspective this looks exactly like a hard delete — 204 on success, 409 on rejection.

**Alternatives considered**: Using PUT with `{ is_active: false }` — not needed since a DELETE route exists.

---

## Decision 3 — `getMaterias()` in `academic.ts` is reused by docentes; admin needs its own module

**Decision**: Create a new `src/api/admin.ts` file with all admin-specific CRUD functions. Do not add admin mutations to `academic.ts`.

**Rationale**: `academic.ts` already exports `getMaterias()` which catches 403 and returns `[]` for non-admin roles. Admin functionality requires full CRUD (create/update/delete) that should not be visible or accidentally called from docente contexts. Separation also keeps the file sizes manageable.

**Alternatives considered**: Adding admin functions to `academic.ts` — rejected to preserve role-specific module boundaries.

---

## Decision 4 — Carrera requires both `code` and `name`

**Decision**: The creation form collects both `code` (short unique identifier, e.g. `"IS"`) and `name` (full name, e.g. `"Ingeniería de Sistemas"`). Both are required. Code must be unique across all carreras.

**Rationale**: Confirmed from `CreateCarreraHandler` which validates `req.Code == "" || req.Name == ""`. Uniqueness is enforced server-side (409 Conflict on duplicate code).

**Impact**: The CarrerasPage creation form has two inputs: Code and Name. The list table shows both columns.

---

## Decision 5 — Materia requires code, name, semester, and carrera_id

**Decision**: Materia creation form collects: `code` (unique short code), `name`, `semester` (integer 1–10, `<select>` with 10 options), and `carrera_id` (dropdown from carrera list). All four fields are required.

**Rationale**: Confirmed from `CreateMateriaHandler` validation: `req.Code == "" || req.Name == "" || req.Semester < 1 || req.Semester > 10 || req.CarreraID == uuid.Nil`.

**Impact**: MateriasPage creation form has 4 fields. Materia list table has columns: Code, Name, Semester, Carrera.

---

## Decision 6 — Student profile email must be `@unifranz.edu.bo`

**Decision**: The server enforces `strings.HasSuffix(req.Email, "@unifranz.edu.bo")`. The frontend shows whatever validation error the server returns (FR-012 — thin client principle).

**Rationale**: The check is server-side; there is no need to duplicate it in the frontend. The UI just displays the server error message via `<ErrorMessage>`.

---

## Decision 7 — `Materia` type already exists in `src/types/academic.ts`

**Decision**: Reuse the existing `Materia` interface. Add `Carrera` and `StudentProfile` to `src/types/academic.ts` (alongside the existing type). Do not create a separate `src/types/admin.ts`.

**Rationale**: All three entities are part of the academic domain. Adding them to the existing file keeps type definitions centralized and avoids import path proliferation.

**Impact**: T001 updates `src/types/academic.ts`, not a new file.
