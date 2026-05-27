# Quickstart: MW6 — Tracing

**Feature**: MW6 — Tracing
**Date**: 2026-05-26

## Prerequisites

1. `nextlab-api` running: `docker compose up` in `nextlab-api/`
2. `nextlab-web` dev server running: `pnpm dev` in `nextlab-web/`
3. A docente account with at least one active curso and one actividad with `trace_enabled = true`
4. A student account enrolled in that curso and assigned to a grupo
5. A test file ready (PDF < 50 MB)

## Smoke Test Sequence

### 1. No prompt on non-traced activity (US1)
- Sign in as student → navigate to an activity where `trace_enabled = false`
- Verify: no session prompt shown; upload form appears directly (if submission section exists)

### 2. Session start prompt on traced activity (US1 — P1)
- Navigate to an activity where `trace_enabled = true`
- Verify: a session prompt appears explaining monitoring before other content is accessible
- Verify: the upload form is NOT visible until the session is started

### 3. Start a session (US1 — P1)
- On the session prompt, click "Iniciar sesión de monitoreo"
- Verify: prompt disappears; "Sesión activa" panel appears with a green indicator and start time
- Verify: the upload form is now visible below the session panel

### 4. Events stream during session (US1 — P1)
- With a session active, type in any text field, copy and paste text, tab away and back
- Verify: no visible disruption to the student's workflow
- Verify (via docente live feed — see test 10): events appear in the backend

### 5. Session ends on navigation (US1)
- With a session active, click "← Volver al curso" to navigate away
- Verify: navigation completes without error
- Verify (via docente viewer): session is marked as ended in the trace log

### 6. Explicit session termination (US3 — P3)
- Start a new session on the activity detail page
- Click "Terminar sesión"
- Verify: confirmation modal appears
- Confirm → Verify: panel updates to "Sesión terminada" with end time; WebSocket disconnects
- Verify: an option to start a new session appears (if edit window still open)

### 7. Cancel session termination (US3)
- With a session active, click "Terminar sesión" → click "Cancelar"
- Verify: session remains active; events continue streaming

### 8. Upload with linked session (FR-014)
- Start a session → check integrity declaration → upload a file
- Verify: submission card appears (same as MW5)
- Sign in as docente → navigate to the submissions list → click the submission row
- Verify: "Ver trazas" button is visible on the submission detail page

### 9. Upload without active session (FR-013)
- Navigate to the activity without starting a session (or with session ended)
- Upload a file (if the session panel allows it)
- Sign in as docente → navigate to the submission detail for that file
- Verify: "Ver trazas" button is NOT visible

### 10. Docente views historical trace log (US2 — P2)
- From the submission detail page, click "Ver trazas"
- Verify: trace viewer page opens with a chronological table
- Verify: columns shown: timestamp, event type, content/detail
- Verify: no edit controls — all read-only
- Verify: back link returns to submission detail

### 11. Empty trace log (US2)
- Navigate to a submission whose session has no recorded events
- Verify: "No hay eventos de trazas registrados." empty state shown

### 12. Docente views live sessions list (US4 — P3)
- While a student has an active session on an activity:
- Sign in as docente → click "Ver Sesiones en Vivo" on the activity detail page
- Verify: the student appears in the list with name, start time, and event count

### 13. Docente watches live event feed (US4)
- From the live sessions list, click the student's row
- Verify: live feed page opens showing recent events
- Interact as the student (type, paste) → Verify: new events appear in the feed within ~2 seconds
- Student ends session → Verify: feed shows "Sesión terminada"

### 14. No active sessions state (US4)
- Navigate to live sessions monitor when no students have active sessions
- Verify: "No hay sesiones activas en este momento." empty state shown

### 15. Co-docente access (FR-020)
- Sign in as co-docente with grant for the activity
- Navigate to `/codocente/cursos/:cid/actividades/:aid/sessions` → verify same live list visible
- Navigate to trace viewer from a submission → verify same historical log visible

### 16. Role guard
- Sign in as admin → navigate to `/docente/cursos/:cid/actividades/:aid/sessions`
- Verify: redirected to `/admin/dashboard`

## Notes

- The session prompt gates the upload form: students must start a session before they can upload
- Session state machine: idle → starting → connecting → active → reconnecting → ended/error
- Docente live feed polls every 2 seconds using `since` timestamp; stops on page unmount
- "Ver trazas" on the submission detail page is only shown when `submission.session_id !== null`
- `TracingPanel.tsx` (Bloque 4 scaffold) has been replaced by `SessionPanel.tsx` with Tailwind + named export
