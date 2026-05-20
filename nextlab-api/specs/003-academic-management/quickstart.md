# Quickstart: M3 — Academic Management

## Prerequisites

- M1 and M2 are running (`docker compose up --build` passes health check and login works).
- A Google Cloud project with OAuth 2.0 credentials (Client ID + Secret) configured for your redirect URL.

## 1. Environment Setup

Add to `.env`:

```env
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback
```

Rebuild and start:

```bash
docker compose up --build
```

Verify migration applied:

```bash
docker compose exec db psql -U postgres -d nextlab -c "\dt"
# Should list: carreras, materias, student_profiles, cursos, grupos,
#              actividades, inscripciones, grupo_asignaciones, codocente_grants
```

## 2. Seed Admin and Docente (M2 flow)

If not already seeded from M2:

```bash
# Seed admin user
docker compose exec db psql -U postgres -d nextlab -c \
  "INSERT INTO users (email, password_hash, role) VALUES
   ('admin@unifranz.edu.bo', '\$2a\$12\$...bcrypt...', 'admin'),
   ('docente@unifranz.edu.bo', '\$2a\$12\$...bcrypt...', 'docente');"
```

Login as admin:

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@unifranz.edu.bo","password":"admin123"}'
# → {"access_token": "...", "refresh_token": "..."}
# Save as ADMIN_TOKEN
```

## 3. Admin Creates the Catalog

```bash
# Create a carrera
curl -X POST http://localhost:8080/admin/carreras \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code":"IS","name":"Ingeniería de Sistemas"}'
# → {"id": "CARRERA_ID", ...}

# Create a materia for semester 4
curl -X POST http://localhost:8080/admin/materias \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"carrera_id":"CARRERA_ID","code":"PROG3","name":"Programación III","semester":4}'
# → {"id": "MATERIA_ID", ...}
```

## 4. Admin Provisions a Student Profile

```bash
curl -X POST http://localhost:8080/admin/student-profiles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "juan.perez@unifranz.edu.bo",
    "carrera_id": "CARRERA_ID",
    "semester": 4,
    "student_code": "20221234"
  }'
# → {"id": "PROFILE_ID", ...}
```

## 5. Docente Creates a Curso

Login as docente:

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"docente@unifranz.edu.bo","password":"docente123"}'
# Save as DOCENTE_TOKEN
```

Create a curso:

```bash
curl -X POST http://localhost:8080/cursos \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"materia_id":"MATERIA_ID","semester_label":"2026-1","section":"A"}'
# → {"id": "CURSO_ID", ...}
```

## 6. Student Google OAuth Login

Open in a browser (or simulate with a Google OAuth test tool):

```
GET http://localhost:8080/auth/google
```

After Google consent, the callback returns:

```json
{
  "access_token": "...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "..."
}
```

Save as `STUDENT_TOKEN`.

Verify enrollment was auto-created:

```bash
curl http://localhost:8080/me/cursos \
  -H "Authorization: Bearer $STUDENT_TOKEN"
# → [{"id":"CURSO_ID","materia_name":"Programación III", ...}]
```

## 7. Docente Creates a Grupo and Assigns the Student

```bash
# Create a grupo
curl -X POST http://localhost:8080/cursos/CURSO_ID/grupos \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Grupo A","capacity":10}'
# → {"id": "GRUPO_ID", ...}

# Look up student's user_id
curl http://localhost:8080/cursos/CURSO_ID/students \
  -H "Authorization: Bearer $DOCENTE_TOKEN"
# → [{user_id: "STUDENT_USER_ID", ...}]

# Assign student to grupo
curl -X POST http://localhost:8080/cursos/CURSO_ID/grupos/GRUPO_ID/students \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"STUDENT_USER_ID"}'
# → 201 Created
```

## 8. Docente Creates an Actividad

```bash
curl -X POST http://localhost:8080/cursos/CURSO_ID/actividades \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Examen Parcial 1",
    "tipo": "examen",
    "deadline": "2026-06-15T18:00:00Z",
    "instructions": "Resolver todos los ejercicios del módulo 3."
  }'
# → {"id":"ACTIVIDAD_ID","trace_enabled":true,"has_submissions":false,...}
```

Verify `trace_enabled: true` (because tipo = "examen").

## 9. Student Sees the Actividad

```bash
curl http://localhost:8080/me/cursos/CURSO_ID/actividades \
  -H "Authorization: Bearer $STUDENT_TOKEN"
# → [{"title":"Examen Parcial 1","trace_enabled":true,"posted_at":"..."}]
```

## 10. Verify Role Isolation

```bash
# Student cannot access another docente's curso
curl http://localhost:8080/me/cursos/OTHER_CURSO_ID/actividades \
  -H "Authorization: Bearer $STUDENT_TOKEN"
# → 403 Forbidden

# Docente cannot see another docente's cursos
curl http://localhost:8080/cursos \
  -H "Authorization: Bearer $OTHER_DOCENTE_TOKEN"
# → [] (empty list, not OTHER_DOCENTE's cursos)

# Non-admin cannot create a carrera
curl -X POST http://localhost:8080/admin/carreras \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -d '{"code":"CC","name":"Ciencias de la Computación"}'
# → 403 Forbidden
```

## M3 Deliverable Verified ✓

All checks pass when:
- `GET /cursos` returns only the authenticated docente's cursos
- `GET /me/cursos` returns only enrolled cursos for the student
- `trace_enabled = true` for `examen` and `defensa_oral` actividades
- Non-admin users receive `403` on catalog write endpoints
- Co_docente without a grant receives `403` on any actividad
