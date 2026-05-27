# Quickstart: MW7 — Grading

**Feature**: MW7 — Grading
**Date**: 2026-05-26

## Prerequisites

1. `nextlab-api` running: `docker compose up` in `nextlab-api/`
2. `nextlab-web` dev server running: `pnpm dev` in `nextlab-web/`
3. A docente account with at least one active curso
4. A student account enrolled in that curso with an existing submission (from MW5)
5. Optional: accounts for co-docente, admin to verify role guards

---

## Smoke Test Sequence

### US1 — Docente Grades a Submission (P1)

#### 1. Grade a submission (no prior grade)
- Sign in as docente → navigate to a curso → click on an actividad → click "Ver Entregas"
- Click on a student's submission row → submission detail page opens
- Verify: **GradePanel** is visible at the bottom with editable score (0–100) and feedback fields
- Enter score `85` and feedback `"Buen trabajo"` → click "Calificar"
- Verify: button shows "Calificando..." (disabled) while in-flight
- Verify: after save → GradePanel switches to read-only mode showing score `85` and feedback text
- Verify: inline "Calificación guardada" success message visible for ~3 seconds

#### 2. Grade is reflected in the submissions list
- Navigate back to "Ver Entregas" list
- Verify: the graded submission shows a "Calificado" badge or the score in the row

#### 3. Overwrite an existing grade
- On the same submission detail page (grade already saved), click "Editar calificación"
- Change score to `90`, update feedback → click "Calificar"
- Verify: updated values shown in read-only mode after save

#### 4. Score out of range
- Enter score `150` → click "Calificar"
- Verify: error message shown inline; grade not saved; form remains editable

#### 5. Co-docente sees grade read-only
- Sign in as co-docente with grant for that actividad → navigate to the submission detail
- Verify: grade (score + feedback) is displayed in **read-only mode** with no edit button or input fields

---

### US2 — Student Views Their Grade (P2)

#### 6. Student sees grade after docente saves
- Sign in as student → navigate to the actividad where their submission was graded
- Verify: "Mi Calificación" section shows score `85 / 100` and the feedback text
- Verify: no editing controls are present — all read-only

#### 7. Student sees "pending" before grading
- Navigate to an actividad where submission exists but no grade has been saved yet
- Verify: "Pendiente de calificación" message shown in the grade area; no score or feedback

#### 8. No grade section without submission
- Navigate to an actividad where student has NOT submitted
- Verify: grade section is NOT shown at all

---

### US3 — Exam Session Form (P2)

#### 9. Exam form visible on examen-type activity (no trace)
- Navigate to an actividad with `tipo = 'examen'` and `trace_enabled = false`
- Verify: a multi-line text area is shown instead of the file upload panel
- Verify: a "Enviar respuesta" submit button is present

#### 10. Submit exam answer
- Type answers into the text area → click "Enviar respuesta"
- Verify: loading state during submission
- Verify: after success — text area and button disappear; read-only view of the answers + submission timestamp shown

#### 11. Cannot re-submit
- After a successful exam submit, reload the page
- Verify: answers are displayed read-only; submit button is absent

#### 12. Exam with trace gate (trace_enabled = true)
- Navigate to an `examen`-type activity with `trace_enabled = true` without an active session
- Verify: exam form is blocked by session prompt (same gate as file upload in MW6)
- Start session → verify exam form becomes accessible

#### 13. Exam closed after deadline
- Navigate to an `examen`-type activity where the deadline has passed
- Verify: "Examen cerrado" label shown; answers displayed read-only if submitted, empty state if not

---

### US4 — Investigación Stage Tracking (P3)

#### 14. Stage selector on informe submission
- Sign in as docente → navigate to an `informe`-type actividad → open a submission detail
- Verify: **stage selector** (dropdown) visible showing current stage (default: `propuesta`)

#### 15. Advance stage
- Select `desarrollo` from the dropdown → click "Actualizar etapa"
- Verify: stage label updates immediately to "Desarrollo" without page reload

#### 16. Student sees current stage
- Sign in as student → navigate to the same `informe` actividad
- Verify: "Etapa: Desarrollo" label shown in read-only mode below the submission card

#### 17. Stage selector absent for non-informe activities
- Navigate to a `practica` or `tarea` submission detail as docente
- Verify: stage selector is NOT shown

---

### US5 — Defensa Oral Scheduling (P3)

#### 18. Scheduling page accessible
- Sign in as docente → navigate to a `defensa_oral` actividad detail page
- Verify: "Programar Defensas" button is visible
- Verify: for non-`defensa_oral` activities, this button is NOT shown

#### 19. Assign a slot
- Click "Programar Defensas" → `DefensaSchedulingPage` opens
- Verify: list of students (those with submissions) shown with date/time input fields
- Enter a valid datetime for one student → click "Guardar"
- Verify: slot saved; datetime shown for that student; no page reload needed

#### 20. Student views assigned slot
- Sign in as student → navigate to the `defensa_oral` actividad
- Verify: "Tu defensa: [datetime] — [location]" section is displayed

#### 21. Student without a slot
- Navigate to a `defensa_oral` actividad where no slot has been assigned yet
- Verify: "Defensa aún no programada" shown in the scheduling area

#### 22. Slot conflict
- Try to assign two students the same datetime slot
- Verify: backend returns conflict error; frontend shows inline error for the conflicting assignment

---

## Role Guard Checks

#### 23. Student cannot access GradePanel
- Sign in as student → navigate to any activity detail page
- Verify: GradePanel component (score input + feedback textarea + submit button) is NOT rendered

#### 24. Admin cannot access docente pages
- Sign in as admin → navigate to `/docente/cursos/:cid/actividades/:aid/submissions/:sid`
- Verify: redirected to `/admin/dashboard`

## Notes

- `GradePanel` decides `createGrade` vs `updateGrade` by inspecting `item.grade !== null` — no try/catch retry needed
- Exam form (`ExamForm`) renders only when `actividad.tipo === 'examen'`; file upload renders for all other types
- `DefensaSchedulingPage` uses `listSubmissions` (MW5) as the student roster — only students with submissions appear
- Stage selector (`StageSelector`) renders only when `actividad.tipo === 'informe'`
- "Programar Defensas" button renders only when `actividad.tipo === 'defensa_oral'`
