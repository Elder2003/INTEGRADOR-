# Quickstart: MW5 — File Submissions

**Feature**: MW5 — File Submissions
**Date**: 2026-05-26

## Prerequisites

1. `nextlab-api` running: `docker compose up` in `nextlab-api/`
2. `nextlab-web` dev server running: `pnpm dev` in `nextlab-web/`
3. A docente account with at least one active curso and one actividad (tipo: `tarea` or `practica`)
4. A student account enrolled in that curso and assigned to a grupo
5. A test file ready (PDF < 50 MB works best)

## Smoke Test Sequence

### 1. Student uploads a file (US1 — P1)
- Sign in as student → navigate to `/student/dashboard`
- Click a course card → navigate to activity list
- Click an activity with tipo `tarea` → navigate to activity detail
- Check the integrity declaration checkbox
- Drop or select a file ≤ 50 MB
- Upload button becomes active after checkbox → click it
- Verify: submission metadata shown (filename, size, submitted-at, edit window expiry)
- Verify: Replace and Delete buttons visible

### 2. Student replaces submission (US1)
- On the same activity detail page, click "Replace"
- Select a different file → upload
- Verify: new filename and size shown; `last_replaced_at` timestamp visible

### 3. Student deletes submission (US1)
- On the activity detail page with a submission, click "Delete"
- Confirm the modal
- Verify: upload form reappears (no submission state)

### 4. Locked submission (US1)
- Wait for the edit window to expire (or use a test activity with `edit_window_expires_at` in the past)
- Navigate to the activity detail page
- Verify: submission shown read-only; "Replace" and "Delete" buttons not rendered; "Entrega bloqueada" badge shown

### 5. Late submission disabled (US1)
- Set an activity deadline to the past and `late_submission_allowed = false`
- Sign in as student with no submission → navigate to that activity
- Attempt to upload
- Verify: "late submissions are not allowed" error shown

### 6. File too large (US1)
- Select a file > 50 MB
- Verify: "file too large" error shown immediately (before any network request)

### 7. Integrity declaration gate (US1)
- Navigate to an activity with no submission
- Verify: Upload button is disabled before checking the integrity declaration

### 8. Docente sees submissions list (US2)
- Sign in as docente → navigate to an activity that has student submissions
- Click "Ver Entregas" on the activity detail page → navigate to `/docente/cursos/:cid/actividades/:aid/submissions`
- Verify: table shows student name, email, filename, size, submitted-at, late badge, locked status

### 9. Empty submissions list (US2)
- Navigate to an activity with no submissions yet
- Verify: empty state message shown

### 10. Co-docente sees submissions list (US2)
- Sign in as co-docente with a grant for the activity
- Navigate to `/codocente/cursos/:cid/actividades/:aid/submissions`
- Verify: same list visible as docente

### 11. Docente downloads a submission (US3)
- From the submissions list, click a row → navigate to submission detail page
- Verify: student name, filename, size, submitted-at, late/locked shown
- Click "Download" → file downloads with correct filename and content type

### 12. Co-docente downloads a submission (US3)
- Same as above but signed in as co-docente
- Verify: download works

### 13. Config changes audit log (FR-011)
- On the submissions page (`/docente/cursos/:cid/actividades/:aid/submissions`), look for the config changes section
- Edit an activity field (e.g., deadline) as docente
- Reload the submissions page
- Verify: the change appears in the audit log with field name, old/new value, and timestamp

### 14. Role guard
- Sign in as admin → navigate to `/student/cursos/:id/actividades/:aid`
- Verify: redirected to `/admin/dashboard` (not student page)

## Notes

- Edit window = 30 minutes from upload, capped at the deadline if late submissions disabled
- All 5 activity tipos (`practica`, `tarea`, `informe`, `examen`, `defensa_oral`) accept file uploads
- The student download button (`GET /actividades/:aid/submission/download`) is shown on the student activity detail page only when a submission exists
- Config changes are visible on the docente submissions page alongside the student list
