# Research: MW5 — File Submissions

**Date**: 2026-05-26
**Feature**: `specs/005-mw5-file-submissions/`

---

## Decision 1 — File upload transport via axios FormData

**Decision**: Use `api.post('/actividades/:aid/submission', formData, { onUploadProgress })` where `api` is the centralized axios instance from `src/lib/api.ts`. Pass a `FormData` object with the `file` field — axios automatically sets `Content-Type: multipart/form-data`.

**Rationale**: Constitution Principle II requires ALL HTTP calls through the centralized axios instance. Axios supports `FormData` natively and exposes `onUploadProgress` for progress tracking. No `fetch()` or `XMLHttpRequest` needed.

**Alternatives considered**: `XMLHttpRequest` (supports progress natively but forbidden by constitution); raw `fetch()` (forbidden by constitution); no progress tracking (violates FR-013).

---

## Decision 2 — File download via axios blob + URL.createObjectURL

**Decision**: Download endpoints (`GET /actividades/:aid/submission/download`, `GET /submissions/:sid/download`) are called via `api.get(url, { responseType: 'blob' })`. The blob is wrapped in `URL.createObjectURL`, a temporary `<a>` element is clicked programmatically, then the URL is revoked.

**Rationale**: The API streams the raw file — no JSON response. Axios `responseType: 'blob'` captures the binary data. This is a standard browser pattern for programmatic downloads without leaving the SPA. The original filename is passed separately (from the submission metadata already loaded on the page).

**Alternatives considered**: Opening download URL directly in a new tab (bypasses the axios interceptor — auth header would be missing, violating Principle II); a server-generated signed URL (not available in this API).

---

## Decision 3 — Activity types that accept file submissions

**Decision**: All five activity types (`practica`, `tarea`, `informe`, `examen`, `defensa_oral`) accept file submissions. The `invalid_activity_type` error from the backend only fires for types not in its `validTipos` set — none of these five trigger it. The student upload form is shown for all five types.

**Rationale**: Source code of `internal/submission/service.go` confirms `validTipos = map[string]bool{"practica": true, "tarea": true, "informe": true, "examen": true, "defensa_oral": true}`. The spec assumption that `examen`/`defensa_oral` are invalid was incorrect. Future types (`git_analysis`, `investigacion`) are handled by MW7/MW8 and are out of scope here.

**Alternatives considered**: Hiding upload for `examen`/`defensa_oral` — rejected; backend confirms all 5 tipos are valid for file submission.

---

## Decision 4 — FileUpload and IntegrityDeclaration: rewrite to Tailwind + named exports

**Decision**: Both existing components (`src/components/student/FileUpload.tsx`, `src/components/student/IntegrityDeclaration.tsx`) use inline `style` objects with CSS variables. They will be rewritten to use Tailwind CSS v4 classes (matching the MW3/MW4 code style) and converted from default exports to named exports.

**Rationale**: The existing components were Bloque 4 scaffold code written before the Tailwind-first convention was established. All MW3/MW4 pages use Tailwind classes exclusively. Keeping inline-style components creates two styling systems in one codebase and makes future edits inconsistent. The functional logic of both components is kept; only styling and export style change.

**Alternatives considered**: Keep inline styles and default exports (faster, but creates inconsistency that React Doctor may flag as architecture warnings); create new versions alongside old (dead code, confusing).

---

## Decision 5 — Submission detail route structure

**Decision**: Docente submission detail is at `/docente/cursos/:cid/actividades/:aid/submissions/:sid`. Co-docente uses `/codocente/cursos/:cid/actividades/:aid/submissions/:sid`. Both use the same `SubmissionDetailPage` component with a `backPath` prop.

**Rationale**: Keeping `:cid` and `:aid` in the detail URL preserves navigation context (the "back" link to the list page) without needing `useLocation` state, which breaks on direct URL access. A single `SubmissionDetailPage` component is reused by both roles via prop injection — avoids duplicating identical view code.

**Alternatives considered**: `/docente/submissions/:sid` (loses course/activity context for the back link); separate SubmissionDetailPage for each role (unnecessary code duplication).

---

## Decision 6 — 50 MB client-side file size check

**Decision**: Show a "file too large" error immediately when the user selects a file over 50 MB — before sending the request. This is classified as a "basic format check for immediate UI feedback only" (constitution Principle VII permitted exception). The API also enforces the 50 MB limit independently.

**Rationale**: Uploading a 60 MB file only to receive a 400 error after a full upload attempt is a poor UX. The backend is the source of truth; the client check is a UX shortcut only. Both must agree on the limit (50 MB = `52_428_800` bytes, confirmed from `service.go`).

**Alternatives considered**: No client check (poor UX for large files); enforce only via API (user must wait for upload to complete before seeing error).

---

## Decision 7 — Edit window display

**Decision**: Show the edit window expiry as a human-readable string (e.g., "Editable hasta: 1 jun 2026 14:30") when the submission is not locked. Do NOT render a countdown timer. When `is_locked = true`, show a "Entrega bloqueada" badge and hide Replace/Delete buttons.

**Rationale**: Constitution Principle VII forbids deadline enforcement logic on the frontend. A countdown timer that dynamically hides buttons when it reaches zero would duplicate the backend's lock logic on the client. The page re-fetches on mount; `is_locked` from the API is the authoritative signal. The expiry timestamp is purely informational display.

**Alternatives considered**: Live countdown with auto-hide controls (violates Principle VII — frontend enforcing lock state); no expiry display (leaves users confused about when their edit window closes).

---

## Decision 8 — Co-docente SubmissionsPage: separate component

**Decision**: Create `src/pages/codocente/SubmissionsPage.tsx` as a thin wrapper that renders the same inner logic as the docente page. Both share `SubmissionDetailPage` via the same component file under different routes.

**Rationale**: Co-docente has identical read/download permissions for submissions. Reusing logic via a shared component avoids prop-drilling the role through multiple layers. The docente `SubmissionDetailPage` accepts a `backPath` prop that varies by role.

**Alternatives considered**: Single page with role-conditional logic inside (harder to test independently per role); fully duplicated co-docente pages (unnecessary maintenance overhead).
