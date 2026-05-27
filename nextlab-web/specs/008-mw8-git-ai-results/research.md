# Research: MW8 — Git & AI Results

## Decision 1: API module placement

**Decision**: New file `src/api/gitAnalysis.ts` for all git submission and analysis report functions.

**Rationale**: Matches the pattern of `src/api/grading.ts` (MW7) and `src/api/submissions.ts` (MW5). Each domain gets its own API module. Grouping git submission + all three analysis reports in one file keeps the boundary clean — they all relate to the analysis pipeline.

**Alternatives considered**: Extending `src/api/submissions.ts` with git functions — rejected because that file handles file-based submissions; mixing git and file logic would blur the domain boundary.

---

## Decision 2: Type extension strategy

**Decision**: Extend existing interfaces in `src/types/academic.ts` (adding fields to `StudentActividadEntry`, `Actividad`, `SubmissionResponse`, `SubmissionListItem`) and append new interfaces at the end of the same file.

**Rationale**: All MW1–MW7 types live in `academic.ts`. A new file would require re-exporting and creates no organizational benefit for a small set of additions.

**New fields on existing interfaces**:
- `StudentActividadEntry`: `git_enabled: boolean`, `ai_enabled: boolean`, `detection_enabled: boolean`
- `Actividad`: `git_enabled: boolean`, `ai_enabled: boolean`, `detection_enabled: boolean`
- `SubmissionResponse`: `repo_url: string | null`
- `SubmissionListItem`: `repo_url: string | null`

---

## Decision 3: Backend API endpoint shapes

**Decision**: Inferred from MW8 spec and backend milestone M6/M7/M8 conventions (matching REST patterns used throughout the project).

| Operation | Method | Path |
|-----------|--------|------|
| Submit git repo | POST | `/actividades/:aid/git-submission` |
| Update git repo URL | PATCH | `/actividades/:aid/git-submission` |
| Get git analysis | GET | `/submissions/:sid/git-analysis` |
| Get AI report | GET | `/submissions/:sid/ai-report` |
| Get detection report | GET | `/submissions/:sid/detection-report` |

**Rationale**: POST/PATCH on the submission resource scoped to actividad (same as existing `/actividades/:aid/submission/exam`). GET on the analysis resources scoped to submission. Consistent with existing routes.

---

## Decision 4: Report pages — docente / codocente sharing

**Decision**: Three report page components (`GitAnalysisPage`, `AIReportPage`, `DetectionReportPage`) — each component handles both docente and codocente access. Two separate route entries per page (docente prefix + codocente prefix), both pointing to the same component. No code duplication.

**Pattern**: Same as `SubmissionDetailPage` which is already shared between docente and codocente routes. The components don't need to know their role prefix — navigation links in the parent page (`SubmissionDetailPage`) are constructed with the correct prefix.

---

## Decision 5: System git account env var

**Decision**: `import.meta.env.VITE_GIT_SYSTEM_ACCOUNT ?? 'nextlab-bot'` — accessed inline in `GitSubmissionForm`. Default fallback `'nextlab-bot'` ensures the form works in dev without a `.env` file.

**Rationale**: Spec FR-001a requires the username to be read from a frontend environment variable. Vite exposes `VITE_*` vars via `import.meta.env`. Fallback prevents the note from showing an empty username.

---

## Decision 6: GitSubmissionForm state model

**Decision**: Local state `mode: 'form' | 'submitting' | 'confirmed' | 'editing' | 'updating' | 'closed'`.

- `'form'`: no submission yet, edit window open or no deadline passed → show URL input
- `'submitting'`: POST in flight → loading state
- `'confirmed'`: submission exists → show read-only confirmation with URL + timestamp
- `'editing'`: "Editar" clicked (submission exists, edit window open) → show URL input pre-filled
- `'updating'`: PATCH in flight
- `'closed'`: deadline passed, no submission → show closed message

`repoUrl` tracked in local state; after successful submit/update, parent `onSuccess(submission)` callback updates the parent page's `submission` state.

---

## Decision 7: Analysis report empty/pending/error states

**Decision**: Each report page renders inline state messages without redirects.
- `status === 'pending'` → info banner "Análisis en curso..."
- `status === 'error'` → error banner with `error_message` from API
- `status === 'done'` → full report content

**Rationale**: Spec FR-011 requires inline messages. No redirect or polling — the user refreshes manually. Consistent with constitution Principle VII (thin client).

---

## Decision 8: URL structure for report pages

**Docente**:
- `/docente/cursos/:cid/actividades/:aid/submissions/:subid/git-analysis`
- `/docente/cursos/:cid/actividades/:aid/submissions/:subid/ai-report`
- `/docente/cursos/:cid/actividades/:aid/submissions/:subid/detection-report`

**Codocente** (same pages, different prefix):
- `/codocente/cursos/:cid/actividades/:aid/submissions/:subid/git-analysis`
- `/codocente/cursos/:cid/actividades/:aid/submissions/:subid/ai-report`
- `/codocente/cursos/:cid/actividades/:aid/submissions/:subid/detection-report`

**Rationale**: Scoped under the submission URL, consistent with `/submissions/:subid/trace` pattern already in the router.
