# Quickstart & Integration Scenarios: MW8 — Git & AI Results

## Scenario 1: Student submits a git repo (first submission)

1. Student opens activity page → `actividad.git_enabled === true`
2. `GitSubmissionForm` renders in `'form'` mode
3. Student enters `https://github.com/alice/proyecto`
4. Clicks "Enviar repositorio" → mode transitions to `'submitting'`
5. `submitGitRepo(aid, repoUrl)` called → `POST /actividades/:aid/git-submission`
6. On success: parent updates `submission` state; form shows `'confirmed'` view with URL and timestamp
7. On reload: `getSubmission(aid)` returns submission with `repo_url` set → `GitSubmissionForm` receives `existingSubmission` prop, renders `'confirmed'` immediately

## Scenario 2: Student edits their git URL

1. Student on confirmed view → clicks "Editar"
2. Mode transitions to `'editing'`; URL input pre-filled with current `repo_url`
3. Student changes URL, clicks "Actualizar repositorio" → mode `'updating'`
4. `updateGitRepo(aid, newUrl)` called → `PATCH /actividades/:aid/git-submission`
5. On success: parent updates `submission`; form shows `'confirmed'` with new URL
6. All three analysis sections in `SubmissionDetailPage` (docente) reset to pending on next load

## Scenario 3: Docente opens git analysis for a completed report

1. Docente on `SubmissionDetailPage` → `submission.repo_url !== null` → "Ver análisis Git" link shown
2. Navigate to `/docente/cursos/:cid/actividades/:aid/submissions/:subid/git-analysis`
3. `GitAnalysisPage` mounts → calls `getGitAnalysis(subid)`
4. Response `status === 'done'` → renders commit count, authors, date range, file change table
5. Back link returns to `SubmissionDetailPage`

## Scenario 4: Docente opens AI report while analysis is pending

1. Navigate to `/docente/cursos/:cid/actividades/:aid/submissions/:subid/ai-report`
2. `AIReportPage` calls `getAIReport(subid)`
3. Response `{ status: 'pending' }` → renders "Análisis IA en curso..." banner
4. User refreshes manually later to check again

## Scenario 5: Detection report link absent (detection disabled)

1. Activity was created with `detection_enabled: false`
2. `SubmissionDetailPage` checks `actividadFlags.detection_enabled` → link not rendered
3. No route navigation possible; page never shown

## Scenario 6: Codocente reads AI report

1. Codocente navigates to `/codocente/cursos/:cid/actividades/:aid/submissions/:subid/ai-report`
2. Same `AIReportPage` component (no edit controls exist anyway — all reports are read-only)
3. Report displays identically to docente view

## Component integration summary

```
StudentActividadDetailPage
  → actividad.git_enabled? → <GitSubmissionForm> : file upload panel

DocenteSubmissionDetailPage
  → submission.repo_url?   → "Ver análisis Git" link
  → actividad.ai_enabled?  → "Ver informe IA" link
  → actividad.detection_enabled? → "Ver informe de detección" link

Routes:
  /docente/.../submissions/:subid/git-analysis   → <GitAnalysisPage>
  /docente/.../submissions/:subid/ai-report       → <AIReportPage>
  /docente/.../submissions/:subid/detection-report→ <DetectionReportPage>
  (same three under /codocente/...)
```
