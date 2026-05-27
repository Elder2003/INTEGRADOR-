# Quickstart: MW4 — Student Portal

**Feature**: MW4 — Student Portal
**Date**: 2026-05-26

## Prerequisites

1. `nextlab-api` running: `docker compose up` in `nextlab-api/`
2. `nextlab-web` dev server running: `pnpm dev` in `nextlab-web/`
3. Admin account exists; student profile pre-provisioned with `@unifranz.edu.bo` email
4. A docente account exists with at least one curso and one actividad
5. The student has been assigned to a grupo in that curso

To set up test data:
```powershell
# 1. Create admin (if needed)
Invoke-RestMethod -Uri "http://localhost:8080/auth/register" -Method Post -ContentType "application/json" -Body '{"email":"admin@unifranz.edu.bo","password":"Admin123!","full_name":"Admin","role":"admin"}'

# 2. As admin: create carrera, materia, student profile via /admin/... endpoints

# 3. Create docente, curso, actividad, assign student to grupo via /cursos/... endpoints
```

## Smoke Test Sequence

### 1. Student sees their course list (US1)
- Open `http://localhost:5173/login`
- Sign in with a student account that has active enrollments
- Should redirect to `/student/dashboard`
- Verify: one card per enrolled curso, showing materia name, docente email, semester label, section, and grupo name

### 2. Empty state (US1)
- Sign in with a student account with NO enrollments
- Navigate to `/student/dashboard`
- Verify: "Aún no estás inscrito en ningún curso." message shown

### 3. Student sees activity list (US2)
- As a student with enrollments, navigate to `/student/dashboard`
- Click on a course card
- Should navigate to `/student/cursos/:id/actividades`
- Verify: activity list shows title, tipo badge, formatted deadline, "Requiere monitoreo" badge if trace_enabled

### 4. Past deadline styling (US2)
- Verify: activities with deadlines in the past show a "Vencido" label or muted color

### 5. Access denied — not enrolled (US2)
- Type a curso UUID directly into `/student/cursos/:id/actividades` for a course not enrolled in
- Verify: "No tienes acceso a este curso" message shown with link back to dashboard

### 6. Activity detail — from list (US3)
- On the activity list page, click an activity
- Should navigate to `/student/cursos/:id/actividades/:aid`
- Verify: title, tipo, full instructions, formatted deadline, monitoring badge visible
- Verify: "Volver al curso" link navigates back to the activity list

### 7. Activity detail — direct URL (US3)
- Copy the detail URL from step 6
- Open in a new tab (fresh navigation, no route state)
- Verify: page loads correctly without redirect — detail is re-fetched from the list endpoint

### 8. Role guard
- Sign in as a docente account
- Navigate directly to `http://localhost:5173/student/dashboard`
- Verify: redirected to `/docente/dashboard`

## Notes

- `grupo` may be null — card shows "Sin grupo" in that case
- The detail page calls `GET /me/cursos/:id/actividades` (NOT a single-activity endpoint — that doesn't exist)
- `tipo` values: `practica`, `tarea`, `informe`, `examen`, `defensa_oral`
- All deadline comparisons are display-only — the backend enforces submission windows
