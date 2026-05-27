# Quickstart: MW3 — Admin Panel

**Feature**: MW3 — Admin Panel  
**Date**: 2026-05-26

## Prerequisites

1. `nextlab-api` running: `docker compose up` in `nextlab-api/`
2. `nextlab-web` dev server running: `pnpm dev` in `nextlab-web/`
3. An admin account exists. Create one via the API if needed:
   ```powershell
   Invoke-RestMethod -Uri "http://localhost:8080/auth/register" -Method Post -ContentType "application/json" -Body '{"email":"admin@unifranz.edu.bo","password":"Admin123!","full_name":"Admin","role":"admin"}'
   ```

## Smoke Test Sequence

### 1. Sign in as admin
- Open `http://localhost:5173/login`
- Enter admin credentials → should redirect to `/admin/dashboard`

### 2. Carreras
- Navigate to `/admin/carreras`
- Create a carrera: Code = `"IS"`, Name = `"Ingeniería de Sistemas"` → appears in list
- Edit the name → reflected immediately
- Try creating another with code `"IS"` → conflict error shown
- Deactivate the carrera (no active materias) → removed from list

### 3. Materias
- Re-create the carrera first (or use one that exists)
- Navigate to `/admin/materias`
- Create a materia: Code = `"MAT101"`, Name = `"Matemáticas I"`, Semester = `1`, Carrera = IS
- Verify it appears in the flat table with Carrera column showing `"Ingeniería de Sistemas"`
- Edit name → reflected
- Try deactivating with a carrera that has active cursos → 409 error shown inline

### 4. Student Profiles
- Navigate to `/admin/student-profiles`
- Create a profile: Email = `"test@unifranz.edu.bo"`, Carrera = IS, Semester = `5`
- Verify it appears in the list
- Try creating with email `"test@gmail.com"` → validation error shown
- Update the semester → reflected
- Deactivate → removed from active list (confirm modal shown)

### 5. Role guard
- Sign in as a docente account
- Try navigating to `http://localhost:5173/admin/carreras` directly → redirected to `/docente/dashboard`

### 6. Empty states
- On a fresh database, navigate to each page → verify empty state with creation call-to-action is shown

## Notes

- The `getCarreras()` function in `src/api/admin.ts` is separate from `getMaterias()` in `academic.ts`
- Student profile deactivation calls `PUT /admin/student-profiles/:id` with `{ is_active: false }` — not DELETE
- All form submissions display API error messages via `<ErrorMessage>` — no custom frontend validation beyond empty-field checks
