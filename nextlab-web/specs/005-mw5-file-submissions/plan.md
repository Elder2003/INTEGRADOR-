# Implementation Plan: MW5 — File Submissions

**Branch**: `main` | **Date**: 2026-05-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/005-mw5-file-submissions/spec.md`
**Research**: `specs/005-mw5-file-submissions/research.md`
**Data Model**: `specs/005-mw5-file-submissions/data-model.md`
**Contracts**: `specs/005-mw5-file-submissions/contracts/submissions.md`

---

## Summary

Build file submission support across three roles. Student sees an upload section on the activity detail page — file picker with integrity declaration, upload progress, and a Replace/Delete interface for unlocked submissions. Docente (and co-docente) gets a submissions list page and a detail/download page. One new API module (`submissions.ts`), six new types, two rewritten student components (Tailwind migration), one new docente component, three new pages, two updated pages, and four new routes.

Key decisions from research:
- File upload via `FormData` through the centralized axios instance with `onUploadProgress` (no `fetch()`)
- File download via `responseType: 'blob'` + `URL.createObjectURL` (keeps auth interceptor in place)
- All 5 activity tipos accept file submissions — spec assumption about `examen`/`defensa_oral` was incorrect; corrected here
- `FileUpload.tsx` and `IntegrityDeclaration.tsx` rewritten to Tailwind + named exports for consistency
- Edit window displayed as a static timestamp; no countdown (Principle VII — no client-side enforcement)
- Submission detail route: `/docente/cursos/:cid/actividades/:aid/submissions/:sid` and `/codocente/cursos/:cid/actividades/:aid/submissions/:sid`; single `SubmissionDetailPage` shared via `backPath` prop

---

## Technical Context

**Language/Version**: TypeScript 5.9 (strict mode)

**Primary Dependencies**: React 19, React Router v7, axios, Tailwind CSS v4, Lucide React

**Storage**: N/A — frontend only; files stored in MinIO via nextlab-api

**Testing**: Manual smoke tests via `quickstart.md`. No automated tests for MW5.

**Target Platform**: Browser (React SPA at `http://localhost:5173`)

**Project Type**: Web application — frontend display layer

**Performance Goals**: Upload confirmation visible within 5 seconds for files < 10 MB (SC-001)

**Constraints**: No external UI component library. No business logic on the frontend (Principle VII). File size check client-side is a "basic format check for immediate UI feedback only" — API always re-validates.

**Scale/Scope**: ~3 new pages, 1 updated student page, 1 updated docente page, 1 new API module, 6 new types, 2 rewritten components, 1 new component, 4 new routes.

---

## Constitution Check

*GATE: Must pass before implementation.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Role-Gated Rendering | ✅ Pass | Upload UI under `ProtectedRoute allowedRoles={['estudiante']}`. Submissions list/detail under `docente` and `co_docente` ProtectedRoutes. Replace/Delete buttons hidden (not disabled) when `is_locked = true`. |
| II. API Client Isolation | ✅ Pass | `src/api/submissions.ts` uses `api` from `../lib/api`. FormData sent via `api.post()`. Downloads via `api.get({ responseType: 'blob' })`. No raw `fetch()`. |
| III. React Quality Gate | ✅ Pass | React Doctor scan required after all MW5 pages complete. |
| IV. Token Security | ✅ Pass | No token handling changes. Axios interceptor handles auth automatically. |
| V. Read-Only Evidence | ✅ Pass | No evidence data in MW5. Submission timestamps are displayed read-only. |
| VI. Brand Baseline | ✅ Pass | Tailwind CSS v4 throughout. No external component library. FileUpload and IntegrityDeclaration migrated to Tailwind. |
| VII. Thin Client | ✅ Pass | No business logic. Edit window displayed as static timestamp (no countdown). Lock state derived from `is_locked` returned by API. 50 MB check is UI feedback only — API enforces independently. No deadline enforcement. |

No violations. All gates pass.

---

## Project Structure

### Documentation (this feature)

```text
specs/005-mw5-file-submissions/
├── plan.md            ← this file
├── research.md        ← Phase 0 output (8 decisions)
├── data-model.md      ← Phase 1 output
├── quickstart.md      ← Phase 1 output
├── contracts/
│   └── submissions.md ← Phase 1 output
└── tasks.md           ← Phase 2 output (speckit-tasks — NOT created by speckit-plan)
```

### Source Code Changes

Files to **create** (new):

```text
src/
├── api/
│   └── submissions.ts              # 8 functions: student upload lifecycle + docente review
│
├── components/
│   └── docente/
│       └── SubmissionTable.tsx     # Paginated table of SubmissionListItem rows
│
└── pages/
    ├── docente/
    │   ├── SubmissionsPage.tsx     # GET /cursos/:cid/actividades/:aid/submissions
    │   └── SubmissionDetailPage.tsx # Metadata + download; shared by docente + co-docente
    └── codocente/
        └── SubmissionsPage.tsx     # Thin wrapper: mounts SubmissionsPage under co-docente route
```

Files to **update** (existing):

```text
src/
├── types/
│   └── academic.ts       # Add 6 new types: SubmissionResponse, StudentInfo,
│                         # SubmissionListItem, SubmissionListResponse,
│                         # ConfigChangeResponse, ConfigChangesResponse
│
├── components/
│   └── student/
│       ├── FileUpload.tsx          # Rewrite: inline styles → Tailwind, default → named export
│       └── IntegrityDeclaration.tsx # Rewrite: inline styles → Tailwind, default → named export
│
├── pages/
│   ├── student/
│   │   └── ActividadDetailPage.tsx # Extend: add submission section (upload + current submission display)
│   └── docente/
│       └── ActividadDetailPage.tsx # Add: "Ver Entregas" link to submissions list
│
└── router/
    └── index.tsx         # Add 4 new routes (2 docente, 2 co-docente)
```

---

## Key Implementation Details

### `src/api/submissions.ts`

```ts
// Student
export async function getSubmission(aid: string): Promise<SubmissionResponse | null>
  // GET /actividades/:aid/submission — returns null on 404, throws ApiError otherwise

export async function uploadSubmission(
  aid: string, file: File, onProgress?: (pct: number) => void
): Promise<SubmissionResponse>
  // POST /actividades/:aid/submission — FormData with 'file' field
  // onProgress: called with 0-100 percentage via axios onUploadProgress

export async function replaceSubmission(
  aid: string, file: File, onProgress?: (pct: number) => void
): Promise<SubmissionResponse>
  // PUT /actividades/:aid/submission

export async function deleteSubmission(aid: string): Promise<void>
  // DELETE /actividades/:aid/submission → expects 204

export async function studentDownloadSubmission(aid: string, filename: string): Promise<void>
  // GET /actividades/:aid/submission/download → responseType: 'blob'
  // Creates Object URL, clicks <a download="filename">, revokes URL

// Docente / Co-docente
export async function listSubmissions(cid: string, aid: string): Promise<SubmissionListResponse>
  // GET /cursos/:cid/actividades/:aid/submissions

export async function downloadSubmission(sid: string, filename: string): Promise<void>
  // GET /submissions/:sid/download → blob download (same pattern as studentDownloadSubmission)

export async function listConfigChanges(cid: string, aid: string): Promise<ConfigChangesResponse>
  // GET /cursos/:cid/actividades/:aid/config-changes
```

All functions use the same `toApiError` helper pattern as `admin.ts` and `student.ts`.

### Student `ActividadDetailPage` — submission section

Added below the existing activity info block:

**State**: `submission: SubmissionResponse | null`, `submLoading: boolean`, `submError: string | null`, `selectedFile: File | null`, `integrityChecked: boolean`, `uploading: boolean`, `uploadProgress: number`, `uploadError: string | null`, `showDeleteConfirm: boolean`, `deleting: boolean`

**On mount**: call `getSubmission(aid)` → set `submission` (null if 404).

**Upload form** (shown when `submission === null`):
- `<IntegrityDeclaration checked={integrityChecked} onChange={setIntegrityChecked} />`
- `<FileUpload onFileSelect={setSelectedFile} selectedFile={selectedFile} onClear={() => setSelectedFile(null)} />`
- Client-side size check: if `selectedFile.size > 52_428_800` → set `uploadError = "El archivo supera el límite de 50 MB."` and clear selectedFile
- Upload button disabled when `!integrityChecked || !selectedFile || uploading`
- On submit: `uploadSubmission(aid, selectedFile, setPct)` → set `submission = result`
- Progress bar: `<div style={{ width: \`${uploadProgress}%\` }}>`

**Current submission** (shown when `submission !== null`):
- Filename, size (formatted), submitted-at (formatted), late badge, edit window expiry (if not locked)
- If `submission.is_locked`: "Entrega bloqueada" badge; no action buttons
- If `!submission.is_locked`: "Replace" button → opens file picker inline (reuses `<FileUpload>`); "Delete" button → `<ConfirmModal>`
- Student download link: `<button onClick={() => studentDownloadSubmission(aid, submission.original_filename)}>Descargar mi entrega</button>`

### `src/components/docente/SubmissionTable.tsx`

Props: `submissions: SubmissionListItem[]`, `cursoId: string`, `actividadId: string`, `role: 'docente' | 'co_docente'`

Renders a `<table>` with columns: Student name, Email, Filename, Size, Submitted at, Late, Locked.
Each row is a `<Link>` to `/${role === 'docente' ? 'docente' : 'codocente'}/cursos/${cursoId}/actividades/${actividadId}/submissions/${item.id}`.
Late rows show a `<span className="bg-yellow-500/20 text-yellow-400 ...">Tarde</span>` badge.
Locked rows show a `<span className="text-[#888888]">Bloqueada</span>` indicator.

### `src/pages/docente/SubmissionsPage.tsx`

Params: `cid`, `aid` from `useParams`.
State: `data: SubmissionListResponse | null`, `changes: ConfigChangesResponse | null`, `loading: boolean`, `error: string | null`.
On mount: `Promise.all([listSubmissions(cid, aid), listConfigChanges(cid, aid)])`.
Renders: heading "Entregas", `<SubmissionTable>`, then a "Cambios de Configuración" section below with the audit log.
Back link: `← Actividad` → `/docente/cursos/${cid}/actividades/${aid}`.

### `src/pages/docente/SubmissionDetailPage.tsx`

Props (via route state or prop): none — derives context from URL params.
Params: `cid`, `aid`, `sid`.
State: `item: SubmissionListItem | null` (passed via route or re-derived from list — simplest: passed via location state from list page; if missing, show error + back link).
Renders: student name + email, filename, size, submitted-at, late badge, locked badge, edit window expiry, "Download" button.
Download: `downloadSubmission(sid, item.original_filename)`.
`backPath` prop: `/docente/cursos/${cid}/actividades/${aid}/submissions` (or codocente equivalent).

> **Note on detail data**: The submissions list already contains all metadata needed for the detail page. Passing it via `useLocation` state avoids an extra API call. If accessed directly via bookmark (no state), show a "Volver a la lista" link and render whatever is available from route params.

### Router additions

```ts
// Inside ProtectedRoute allowedRoles={['docente']}
{ path: '/docente/cursos/:cid/actividades/:aid/submissions', element: <SubmissionsPage /> }
{ path: '/docente/cursos/:cid/actividades/:aid/submissions/:sid', element: <SubmissionDetailPage /> }

// Inside ProtectedRoute allowedRoles={['co_docente']}
{ path: '/codocente/cursos/:cid/actividades/:aid/submissions', element: <CodocenteSubmissionsPage /> }
{ path: '/codocente/cursos/:cid/actividades/:aid/submissions/:sid', element: <SubmissionDetailPage /> }
```

### Docente `ActividadDetailPage` — "Ver Entregas" link

Add a `<Link>` button after the activity metadata section:

```tsx
<Link
  to={`/docente/cursos/${cursoId}/actividades/${aid}/submissions`}
  className="inline-flex items-center gap-2 bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg px-4 py-2 text-[#f0f0f0] hover:border-[#E85D24] transition-colors text-sm"
>
  Ver Entregas
</Link>
```

---

## Scope Adjustments (vs. spec)

| Spec claim | Actual behavior | Source |
|-----------|-----------------|--------|
| `examen`/`defensa_oral` don't accept file submissions | All 5 tipos accept file submissions; `invalid_activity_type` only fires for out-of-scope tipos | Research Decision 3 / service.go |
| Upload button blocked by client-side size check | Client-side check is UI feedback only; API always re-validates | Research Decision 6 / Principle VII |

---

## Complexity Tracking

No constitution violations. No complexity justification required.
