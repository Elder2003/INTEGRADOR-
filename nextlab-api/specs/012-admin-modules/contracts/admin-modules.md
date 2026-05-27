# API Contracts: Admin Modules (M12)

All admin endpoints require `Authorization: Bearer <token>` with role=admin. Non-admin → 403.

---

## US1 — Docente Account Management

### POST /admin/docentes
Create a new docente account. Returns the created account plus a one-time plain-text temporary password.

**Request**:
```json
{ "email": "jdoe@unifranz.edu.bo", "full_name": "Juan Doe" }
```

**Response 201**:
```json
{
  "id": "<uuid>",
  "email": "jdoe@unifranz.edu.bo",
  "full_name": "Juan Doe",
  "is_active": true,
  "created_at": "<timestamp>",
  "temporary_password": "Xk9mP2qRvZ3n"
}
```

**Errors**: 400 (invalid body), 409 (email exists)

---

### GET /admin/docentes
List docente accounts with optional filtering.

**Query params**: `?page=1&limit=20&active=true`

**Response 200**:
```json
{
  "docentes": [{ "id": "...", "email": "...", "full_name": "...", "is_active": true, "created_at": "..." }],
  "total": 5,
  "page": 1,
  "limit": 20
}
```

---

### GET /admin/docentes/:id
Get a single docente by ID.

**Response 200**: `DocenteAccount` object. **404** if not found.

---

### PUT /admin/docentes/:id
Update docente full_name or is_active status.

**Request** (all fields optional):
```json
{ "full_name": "Juan Carlos Doe", "is_active": false }
```

**Response 200**: Updated `DocenteAccount`. **404** if not found.

---

### DELETE /admin/docentes/:id
Soft-deactivate a docente account (sets is_active=false).

**Response 204**. **404** if not found.

---

## US2 — Format Template Library

### POST /admin/format-templates
Create a format rule template.

**Request**:
```json
{
  "name": "TFG Standard",
  "description": "Standard format for final graduation work",
  "rules": { "min_pages": 10, "max_pages": 80, "required_types": ["pdf"] }
}
```

**Response 201**: `FormatTemplate` object (without `temporary_password`). **409** if name exists.

---

### GET /admin/format-templates
List all format templates.

**Query params**: `?active_only=true`

**Response 200**: `{ "templates": [...], "total": N }`

---

### GET /admin/format-templates/:id
Get a single format template. **404** if not found.

---

### PUT /admin/format-templates/:id
Update template fields (partial update).

**Request** (all optional): `{ "name": "...", "description": "...", "rules": {...}, "is_active": false }`

**Response 200**: Updated `FormatTemplate`. **404** if not found.

---

### DELETE /admin/format-templates/:id
Delete a format template.

**Response 204** if unused. **409** if template is currently applied to any actividad.

---

### POST /actividades/:aid/format-template *(docente only)*
Apply a format template to an actividad owned by the calling docente. Copies template rules into `actividades.format_rules` and records the association.

**Request**: `{ "template_id": "<uuid>" }`

**Response 200**: `AppliedTemplate` object. **403** if docente doesn't own the actividad. **404** if template or actividad not found.

---

### GET /actividades/:aid/format-template *(docente only)*
Get the format template currently applied to an actividad.

**Response 200**: `AppliedTemplate` or `null` if no template applied. **403** if docente doesn't own the actividad.

---

## US3 — Curriculum Document Management

### GET /admin/curriculum/documents
List curriculum documents with optional filters.

**Query params**: `?carrera=ISI&semestre=3&codigo_materia=TFG001&page=1&limit=20`

**Response 200**:
```json
{
  "documents": [{ "id": "...", "carrera": "ISI", "semestre": 3, "codigo_materia": "TFG001", "original_filename": "...", "is_rubric": false, "created_at": "..." }],
  "total": 12
}
```

---

### DELETE /admin/curriculum/documents/:id
Delete a curriculum document and all its chunks; remove MinIO file.

**Response 204**. **404** if not found.

---

## US4 — Interdisciplinary Activity Configuration

### POST /actividades/:aid/interdisciplinary *(admin only)*
Set interdisciplinary scopes for an actividad. Replaces any existing scopes.

**Request**:
```json
{
  "scopes": [
    { "carrera": "Medicina", "semestre": 4, "codigo_materia": "BIO401" },
    { "carrera": "Farmacia",  "semestre": 3, "codigo_materia": "QUI301" }
  ]
}
```

**Response 200**: `{ "actividad_id": "...", "scopes": [...] }`. Empty array clears all scopes.

---

### GET /actividades/:aid/interdisciplinary
Get current interdisciplinary scopes for an actividad (admin or docente owner).

**Response 200**: `{ "actividad_id": "...", "scopes": [...] }`

---

### DELETE /actividades/:aid/interdisciplinary *(admin only)*
Remove all interdisciplinary scopes.

**Response 204**.
