# Feature Specification: Admin Modules

**Feature Branch**: `012-admin-modules`

**Created**: 2026-05-23

**Status**: Draft

**Input**: M12 — Admin Modules: Curriculum, templates, interdisciplinary (Principle IV)

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Docente Account Management (Priority: P1)

An administrator needs to provision, update, and deactivate docente (teacher) accounts. Currently the admin can manage student profiles, carreras, and materias, but has no way to manage docente accounts through the API.

**Why this priority**: Docente accounts gate all course/activity creation. Without admin control over docente accounts, the platform cannot be managed institutionally.

**Independent Test**: `POST /admin/docentes` → 201; `GET /admin/docentes` → list contains new docente; `PUT /admin/docentes/:id` → 200; `DELETE /admin/docentes/:id` → 204 (soft deactivation); duplicate email → 409.

**Acceptance Scenarios**:

1. **Given** admin is authenticated, **When** POST `/admin/docentes` with valid email and name, **Then** a new docente user is created with role=docente and a bcrypt-hashed temporary password, returning 201.
2. **Given** docente accounts exist, **When** admin GET `/admin/docentes`, **Then** all docentes (active and inactive) are returned with pagination.
3. **Given** a docente exists, **When** admin PUT `/admin/docentes/:id` with updated fields, **Then** the account is updated and 200 returned.
4. **Given** an active docente, **When** admin DELETE `/admin/docentes/:id`, **Then** is_active=false; 204 returned; the docente cannot log in.
5. **Given** a non-existent ID, **When** admin GET/PUT/DELETE `/admin/docentes/:id`, **Then** 404.
6. **Given** a duplicate email, **When** admin POST `/admin/docentes`, **Then** 409 Conflict.

---

### User Story 2 — Format Template Library (Priority: P2)

An administrator can create reusable format rule templates (e.g., "TFG Standard: PDF, 10–50 pages") that docentes apply to activities instead of configuring rules from scratch.

**Why this priority**: Without templates, every docente manually configures format rules per activity. Admin-managed templates enforce institutional standards.

**Independent Test**: Admin POST `/admin/format-templates` → 201; GET list → template present; docente POST `/actividades/:aid/format-template` with template_id → 200; GET `/actividades/:aid/format-template` → returns applied template; non-admin → 403 on admin endpoints.

**Acceptance Scenarios**:

1. **Given** admin is authenticated, **When** POST `/admin/format-templates` with name and rules, **Then** template is created and 201 returned.
2. **Given** templates exist, **When** GET `/admin/format-templates`, **Then** all templates are listed (paginated).
3. **Given** a template and an actividad owned by the docente, **When** docente POST `/actividades/:aid/format-template` with `template_id`, **Then** `actividades.format_rules` is updated to the template's rules.
4. **Given** a template in use, **When** admin DELETE `/admin/format-templates/:id`, **Then** 409 Conflict.
5. **Given** an unused template, **When** admin DELETE `/admin/format-templates/:id`, **Then** 204 (hard delete).
6. **Given** non-admin user, **When** admin POST/PUT/DELETE format-template endpoints, **Then** 403.

---

### User Story 3 — Curriculum Document Management (Priority: P3)

An administrator can list and delete curriculum documents stored in the curricular RAG. Currently only upload exists; admin needs to curate and remove stale content.

**Why this priority**: Without list/delete, the curriculum RAG accumulates stale documents. This is operational maintenance that does not block day-to-day usage.

**Independent Test**: `GET /admin/curriculum/documents?carrera=X` → list; `DELETE /admin/curriculum/documents/:id` → 204; list again → doc removed.

**Acceptance Scenarios**:

1. **Given** curriculum documents exist, **When** admin GET `/admin/curriculum/documents` (optional filters: carrera, semestre, codigo_materia), **Then** matching documents returned with metadata.
2. **Given** a document ID, **When** admin DELETE `/admin/curriculum/documents/:id`, **Then** document + all its chunks deleted from DB; file removed from MinIO; 204 returned.
3. **Given** a non-existent ID, **When** admin DELETE, **Then** 404.
4. **Given** non-admin user, **When** calling these endpoints, **Then** 403.

---

### User Story 4 — Interdisciplinary Activity Configuration (Priority: P4)

An administrator can mark an actividad as interdisciplinary by attaching additional carrera+semestre+materia scopes. The detection RAG then queries all scopes, providing richer cross-disciplinary context.

**Why this priority**: Applies to a small subset of activities. Depends on all prior features being stable.

**Independent Test**: Admin POST `/actividades/:aid/interdisciplinary` with extra scopes → 200; verify detection prompt includes content from all scopes; DELETE `/actividades/:aid/interdisciplinary` → reverts to single-carrera.

**Acceptance Scenarios**:

1. **Given** an actividad, **When** admin POST `/actividades/:aid/interdisciplinary` with extra scope array, **Then** scopes are stored and 200 returned.
2. **Given** an interdisciplinary actividad, **When** detection runs RAG, **Then** curricular context is retrieved from ALL scopes (union, deduplicated by chunk_id).
3. **Given** an interdisciplinary actividad, **When** admin DELETE `/actividades/:aid/interdisciplinary`, **Then** all extra scopes removed; 204 returned.
4. **Given** non-admin user, **When** POST/DELETE interdisciplinary endpoints, **Then** 403.

---

### Edge Cases

- Deactivating a docente with active courses: courses and data are preserved; docente cannot log in.
- Updating a format template after it was applied: existing actividad rules are NOT updated retroactively (snapshot on application).
- Deleting a curriculum document while RAG is in use: in-flight requests may use stale embeddings; subsequent requests will not include the deleted doc.
- Interdisciplinary scope duplicating the actividad's own carrera: silently deduplicated.
- Empty interdisciplinary scopes array in POST: clears all extra scopes (equivalent to DELETE).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Admin MUST be able to create docente accounts (email, full name, temporary password) with role=docente.
- **FR-002**: Admin MUST be able to list docente accounts with pagination and optional active/inactive filter.
- **FR-003**: Admin MUST be able to update docente name and is_active status.
- **FR-004**: Admin MUST be able to soft-deactivate docente accounts (is_active=false); no hard delete.
- **FR-005**: Admin MUST be able to create named format rule templates (JSONB rules payload matching FormatRulesConfig schema).
- **FR-006**: Admin MUST be able to list, update, and delete format templates (delete blocked if template is applied to any actividad).
- **FR-007**: Docentes MUST be able to apply an existing format template to one of their actividades via a single call that copies rules into actividades.format_rules.
- **FR-008**: Admin MUST be able to list curriculum documents with optional carrera/semestre/codigo_materia filters.
- **FR-009**: Admin MUST be able to delete a curriculum document, cascading to chunks in DB and the file in MinIO.
- **FR-010**: Admin MUST be able to set interdisciplinary scopes for an actividad (replaces existing scopes).
- **FR-011**: Admin MUST be able to remove all interdisciplinary scopes from an actividad.
- **FR-012**: Detection RAG MUST union-query all interdisciplinary scopes when present (deduplicated by chunk_id).
- **FR-013**: All /admin/* endpoints MUST return 403 for non-admin roles.
- **FR-014**: Format template apply endpoint MUST verify docente ownership of the actividad (403 otherwise).

### Key Entities

- **FormatTemplate**: name, description, rules (JSONB — FormatRulesConfig), is_active, created_by, created_at, updated_at.
- **ActividadFormatTemplate**: actividad_id (PK), format_template_id, applied_at, applied_by.
- **InterdisciplinaryScope**: id, actividad_id, carrera, semestre, codigo_materia. Unique on (actividad_id, carrera, semestre, codigo_materia).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Admin provisions a new docente account in a single API call under 2 seconds.
- **SC-002**: Admin lists and deletes a stale curriculum document in under 3 API calls.
- **SC-003**: Applying a format template to an actividad requires exactly 1 API call.
- **SC-004**: All admin-only endpoints return 403 for docente and estudiante roles.
- **SC-005**: When interdisciplinary scopes are set, the RAG prompt includes content from all scopes (verifiable by inspection).

## Assumptions

- Existing `users` table with `role` and `is_active` columns is reused for docente accounts.
- FormatRulesConfig Go type from `internal/format/model.go` defines the rules JSONB schema.
- Curriculum documents are stored in `rag_curricular_documents` table (created in M7).
- Interdisciplinary scopes use a new table `actividad_interdisciplinary_scopes`.
- Detection RAG union uses a separate SELECT per scope and deduplicates by chunk_id in Go before building the prompt.
- Admin JWT claims are already handled by the auth package; no new auth plumbing needed.
- Temporary docente passwords are bcrypt-hashed; plain-text is never stored.
