# Tasks: MW5 — File Submissions

**Feature**: MW5 — File Submissions
**Plan**: [plan.md](plan.md) | **Spec**: [spec.md](spec.md) | **Contracts**: [contracts/submissions.md](contracts/submissions.md)
**Total tasks**: 15 | **Last updated**: 2026-05-26

---

## Phase 1 — Setup

**Goal**: Lay the shared foundation — types, rewritten components, and stub pages — before any user story work begins. All tasks in this phase are independent and can run in parallel.

- [X] T001 [P] Add 6 new submission types to `src/types/academic.ts` — append after the existing `UpdateGrupoRequest` interface: `SubmissionResponse { id: string; actividad_id: string; original_filename: string; file_size_bytes: number; mime_type: string; submitted_at: string; is_late: boolean; edit_window_expires_at: string | null; is_locked: boolean; last_replaced_at: string | null }`, `StudentInfo { id: string; name: string; email: string }`, `SubmissionListItem { id: string; student: StudentInfo; original_filename: string; file_size_bytes: number; mime_type: string; submitted_at: string; is_late: boolean; is_locked: boolean; last_replaced_at: string | null; edit_window_expires_at: string | null }`, `SubmissionListResponse { actividad_id: string; submissions: SubmissionListItem[]; total: number }`, `ConfigChangeActor { id: string; name: string }`, `ConfigChangeResponse { id: string; changed_by: ConfigChangeActor; field_name: string; previous_value: string; new_value: string; changed_at: string }`, `ConfigChangesResponse { actividad_id: string; changes: ConfigChangeResponse[] }`.

- [X] T002 [P] Rewrite `src/components/student/FileUpload.tsx` — convert from default export + inline styles to named export + Tailwind CSS v4 classes. Keep identical props interface and logic: `Props { acceptedTypes: string[]; acceptedLabels: string[]; onFileSelect: (file: File) => void; selectedFile?: File | null; onClear?: () => void }`. Export as `export function FileUpload(...)`. Replace all `style={{...}}` objects with Tailwind classes using the project palette (`border-[#2e2e2e]`, `text-[#f0f0f0]`, `text-[#888888]`, `bg-[#E85D24]`, `border-[#E85D24]` for drag-over state). Keep the drag-and-drop, file-selected, and clear-button behavior unchanged.

- [X] T003 [P] Rewrite `src/components/student/IntegrityDeclaration.tsx` — convert from default export + inline styles to named export + Tailwind CSS v4 classes. Keep identical props interface: `Props { checked: boolean; onChange: (checked: boolean) => void }`. Export as `export function IntegrityDeclaration(...)`. Replace all `style={{...}}` with Tailwind classes: border changes color when checked (`border-[#E85D24]` checked, `border-[#2e2e2e]` unchecked), background `bg-[#E85D24]/10` when checked. Keep the declaration text and checkbox behavior unchanged.

- [X] T004 [P] Create minimal placeholder stub pages for 3 new page files: `src/pages/docente/SubmissionsPage.tsx` (exports named `SubmissionsPage` returning `<p>SubmissionsPage</p>`), `src/pages/docente/SubmissionDetailPage.tsx` (exports named `SubmissionDetailPage` returning `<p>SubmissionDetailPage</p>`), `src/pages/codocente/SubmissionsPage.tsx` (exports named `CodocenteSubmissionsPage` returning `<p>CodocenteSubmissionsPage</p>`). These stubs are replaced in Phases 4–5.

---

## Phase 2 — Foundational

**Goal**: Wire the API module and new routes so all paths compile. Must complete before user story work begins.

- [X] T005 Create `src/api/submissions.ts` — implement 8 functions using the same `toApiError` helper pattern as `src/api/admin.ts`. Import `api` from `../lib/api`, `ApiError` from `../types`, `AxiosError` from `axios`, and types from `../types/academic`. Functions:
  1. `getSubmission(aid: string): Promise<SubmissionResponse | null>` → `GET /actividades/${aid}/submission`; return `null` on 404, throw ApiError on other errors.
  2. `uploadSubmission(aid: string, file: File, onProgress?: (pct: number) => void): Promise<SubmissionResponse>` → `POST /actividades/${aid}/submission` with `FormData` (`formData.append('file', file)`); pass `{ onUploadProgress: (e) => onProgress?.(Math.round((e.loaded * 100) / (e.total ?? 1))) }` as axios config.
  3. `replaceSubmission(aid: string, file: File, onProgress?: (pct: number) => void): Promise<SubmissionResponse>` → `PUT /actividades/${aid}/submission` with same FormData + progress pattern.
  4. `deleteSubmission(aid: string): Promise<void>` → `DELETE /actividades/${aid}/submission`; expects 204.
  5. `studentDownloadSubmission(aid: string, filename: string): Promise<void>` → `GET /actividades/${aid}/submission/download` with `{ responseType: 'blob' }`; create `URL.createObjectURL(new Blob([res.data]))`, create a temp `<a>` element with `download={filename}`, click it, then `URL.revokeObjectURL`.
  6. `listSubmissions(cid: string, aid: string): Promise<SubmissionListResponse>` → `GET /cursos/${cid}/actividades/${aid}/submissions`.
  7. `downloadSubmission(sid: string, filename: string): Promise<void>` → `GET /submissions/${sid}/download` with `{ responseType: 'blob' }`; same blob-download pattern as #5.
  8. `listConfigChanges(cid: string, aid: string): Promise<ConfigChangesResponse>` → `GET /cursos/${cid}/actividades/${aid}/config-changes`.
  All catch blocks call `toApiError(err)`. Depends on T001.

- [X] T006 Update `src/router/index.tsx` — import `SubmissionsPage` and `SubmissionDetailPage` from `../pages/docente/SubmissionsPage` and `../pages/docente/SubmissionDetailPage`; import `CodocenteSubmissionsPage` from `../pages/codocente/SubmissionsPage`. Add 2 routes inside the existing `ProtectedRoute allowedRoles={['docente']}` children array (after the existing actividad routes): `{ path: '/docente/cursos/:cid/actividades/:aid/submissions', element: <SubmissionsPage /> }` and `{ path: '/docente/cursos/:cid/actividades/:aid/submissions/:sid', element: <SubmissionDetailPage /> }`. Add 2 routes inside the existing `ProtectedRoute allowedRoles={['co_docente']}` children array: `{ path: '/codocente/cursos/:cid/actividades/:aid/submissions', element: <CodocenteSubmissionsPage /> }` and `{ path: '/codocente/cursos/:cid/actividades/:aid/submissions/:sid', element: <SubmissionDetailPage /> }`. Depends on T004.

---

## Phase 3 — US1: Student File Upload (P1) 🎯 MVP

**Story goal**: Student navigates to an activity detail page, sees their submission status, uploads a file (with integrity declaration), sees upload progress, and gets Replace + Delete controls while the edit window is open. When locked, submission shown read-only.

**Independent test criteria**: Navigate to `/student/cursos/:id/actividades/:aid` as a student → upload form visible with locked integrity declaration checkbox; check checkbox + select file → Upload button activates; submit → submission card shown with filename, size, submitted-at, edit window expiry, Replace and Delete buttons.

- [X] T007 [US1] Extend `src/pages/student/ActividadDetailPage.tsx` — add full submission section below the instructions block. Add state: `submission: SubmissionResponse | null`, `submLoading: boolean`, `submError: string | null`, `selectedFile: File | null`, `integrityChecked: boolean`, `uploading: boolean`, `uploadProgress: number`, `uploadError: string | null`, `showDeleteConfirm: boolean`, `deleting: boolean`, `showReplaceForm: boolean`. On mount (parallel with existing actividad fetch): call `getSubmission(aid!)` → set `submission` (null on 404, set `submError` on other errors). Section heading: "Mi Entrega". **Upload form** (rendered when `submission === null`): `<IntegrityDeclaration checked={integrityChecked} onChange={setIntegrityChecked} />`; `<FileUpload onFileSelect={f => { if (f.size > 52428800) { setUploadError('El archivo supera el límite de 50 MB.'); return; } setSelectedFile(f); setUploadError(null); }} selectedFile={selectedFile} onClear={() => setSelectedFile(null)} acceptedTypes={[]} acceptedLabels={['Cualquier formato']} />`; progress bar: `<div className="w-full bg-[#2e2e2e] rounded-full h-1.5"><div className="bg-[#E85D24] h-1.5 rounded-full transition-all" style={{ width: \`${uploadProgress}%\` }} /></div>` (shown when `uploading`); upload button disabled when `!integrityChecked || !selectedFile || uploading`; on click: `uploadSubmission(aid!, selectedFile, pct => setUploadProgress(pct))` → on success set `submission = result`, reset form state. **Current submission card** (rendered when `submission !== null`): `bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4`; show `original_filename`, `(file_size_bytes / 1024).toFixed(0) + ' KB'`, formatted `submitted_at`, `is_late` badge (`text-yellow-400 bg-yellow-400/10`), edit window expiry formatted or "Entrega bloqueada" badge when `is_locked`. If `!submission.is_locked`: "Reemplazar" button (sets `showReplaceForm = true`) + "Eliminar" button (sets `showDeleteConfirm = true`). Replace inline form: `<FileUpload .../>` + "Confirmar reemplazo" button → `replaceSubmission(aid!, selectedFile, pct => setUploadProgress(pct))` → update `submission`, hide form. Delete: `<ConfirmModal title="Eliminar entrega" message="¿Seguro que deseas eliminar tu entrega?" confirmLabel="Eliminar" onConfirm={() => { deleteSubmission(aid!).then(() => setSubmission(null)) }} onCancel={() => setShowDeleteConfirm(false)} />`. Student download link: `<button onClick={() => studentDownloadSubmission(aid!, submission.original_filename)}>Descargar mi entrega</button>`. Error display: `submError && <ErrorMessage message={submError} />`, `uploadError && <p className="text-red-400 text-sm">{uploadError}</p>`. Depends on T002, T003, T005.

---

## Phase 4 — US2: Docente Submissions List (P2)

**Story goal**: Docente navigates to the submissions page for an activity and sees a table of all student submissions with names, filenames, dates, and badges. Docente can click any row to go to the detail page. Co-docente sees the same list. Config-change audit log shown below the table.

**Independent test criteria**: Navigate to `/docente/cursos/:cid/actividades/:aid/submissions` as a docente with student submissions → table renders with student name, filename, late badge, locked status; click a row → navigate to submission detail URL.

- [X] T008 [P] [US2] Implement `src/components/docente/SubmissionTable.tsx` (replace stub) — props: `submissions: SubmissionListItem[]`, `cursoId: string`, `actividadId: string`, `role: 'docente' | 'co_docente'`. Render a `<table className="w-full text-sm">` with header row: Student, Email, Archivo, Tamaño, Enviado, Estado. For each item render a `<tr>` that is a `<Link>` (use `<tr onClick={() => navigate(...)}>` with `cursor-pointer hover:bg-[#1a1a1a]`) to `/${role === 'docente' ? 'docente' : 'codocente'}/cursos/${cursoId}/actividades/${actividadId}/submissions/${item.id}`. Columns: `item.student.name`, `item.student.email`, `item.original_filename`, `(item.file_size_bytes / 1024).toFixed(0) + ' KB'`, formatted `item.submitted_at` (toLocaleString es-BO), badges for `is_late` (`bg-yellow-400/10 text-yellow-400 text-xs px-2 py-0.5 rounded`) and `is_locked` (`bg-[#2e2e2e] text-[#888888] text-xs px-2 py-0.5 rounded`). Empty state when `submissions.length === 0`: `<p className="text-[#888888] py-4">Ningún estudiante ha enviado archivos aún.</p>`. Requires `useNavigate` from react-router-dom.

- [X] T009 [P] [US2] Add "Ver Entregas" button to `src/pages/docente/ActividadDetailPage.tsx` — import `Link` from react-router-dom (already imported via navigate). Add a `<Link to={\`/docente/cursos/${cursoId}/actividades/${aid}/submissions\`} className="inline-flex items-center gap-2 bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg px-4 py-2 text-sm text-[#f0f0f0] hover:border-[#E85D24] transition-colors">Ver Entregas</Link>` after the activity metadata section (e.g., after the late-submissions toggle). This provides the docente navigation entry point into the submissions list.

- [X] T010 [US2] Implement `src/pages/docente/SubmissionsPage.tsx` (replace stub) — get params `{ cid, aid }` from `useParams<{ cid: string; aid: string }>()`. State: `data: SubmissionListResponse | null`, `changes: ConfigChangesResponse | null`, `loading: boolean`, `error: string | null`. On mount: `Promise.all([listSubmissions(cid!, aid!), listConfigChanges(cid!, aid!)])` → set both. Show `<LoadingSpinner />` while loading. Show `<ErrorMessage />` on error. Page heading: "Entregas". Back link: `<Link to={\`/docente/cursos/${cid}/actividades/${aid}\`}>← Actividad</Link>`. Render `<SubmissionTable submissions={data?.submissions ?? []} cursoId={cid!} actividadId={aid!} role="docente" />`. Below: "Cambios de Configuración" section — if `changes?.changes.length`: `<table>` with columns: Quién, Campo, Anterior, Nuevo, Cuándo — one row per `ConfigChangeResponse`. Empty state if no changes: `<p className="text-[#888888]">Sin cambios de configuración registrados.</p>`. Depends on T008.

- [X] T011 [US2] Implement `src/pages/codocente/SubmissionsPage.tsx` (replace stub) — same structure as docente SubmissionsPage but with co-docente back link and role prop. Get params `{ cid, aid }`. State and fetch identical: `Promise.all([listSubmissions(cid!, aid!), listConfigChanges(cid!, aid!)])`. Back link: `<Link to="/codocente/dashboard">← Dashboard</Link>` (co-docente has no per-actividad page in MW5). Render `<SubmissionTable submissions={data?.submissions ?? []} cursoId={cid!} actividadId={aid!} role="co_docente" />`. Omit config-changes section (docente-only concern). Depends on T008.

---

## Phase 5 — US3: Docente Downloads a Submission (P3)

**Story goal**: Docente (or co-docente) opens the submission detail page, sees all submission metadata, and downloads the file with the original filename and correct MIME type.

**Independent test criteria**: Navigate to `/docente/cursos/:cid/actividades/:aid/submissions/:sid` (from list page) as a docente → student name, filename, size, submitted-at, late/locked badges visible; click "Descargar" → file saved to device with original filename.

- [X] T012 [US3] Implement `src/pages/docente/SubmissionDetailPage.tsx` (replace stub) — get params `{ cid, aid, sid }` from `useParams<{ cid: string; aid: string; sid: string }>()`. Get `item` from `useLocation().state as SubmissionListItem | undefined`. Derive `backPath` from the current URL prefix: if `window.location.pathname.startsWith('/docente')` → `/docente/cursos/${cid}/actividades/${aid}/submissions`, else `/codocente/cursos/${cid}/actividades/${aid}/submissions`. State: `downloading: boolean`, `downloadError: string | null`. If no `item` (direct URL access): show `<ErrorMessage message="Navega desde la lista de entregas para ver el detalle." />` + `<Link to={backPath}>← Volver a la lista</Link>`. Detail view (when `item` exists): page heading `item.original_filename`; metadata block with `bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4` showing: `item.student.name` + `item.student.email`; filename, size (KB); formatted submitted-at; `is_late` badge; `is_locked` badge or edit window expiry. Download button: `<button onClick={() => { setDownloading(true); downloadSubmission(sid!, item.original_filename).catch(e => setDownloadError(e.message)).finally(() => setDownloading(false)); }}>{downloading ? 'Descargando...' : 'Descargar'}</button>` styled `bg-[#E85D24] hover:bg-[#E85D24]/80 text-white px-4 py-2 rounded-lg text-sm transition-colors`. `downloadError && <ErrorMessage message={downloadError} />`. Back link: `<Link to={backPath}>← Volver a la lista</Link>`. Depends on T005.

---

## Phase 6 — Polish & Quality Gate

**Goal**: Ensure quality before marking MW5 complete.

- [X] T013 Run React Doctor scan from repo root (`npx react-doctor@latest .`) — resolve ALL findings with ERROR severity before proceeding; for WARNING findings add a one-line code comment with justification if deferring.

- [X] T014 Audit all affected pages — verify each of `ActividadDetailPage` (student), `SubmissionsPage` (docente + co-docente), and `SubmissionDetailPage` shows: (a) a loading indicator during fetch, (b) an error message on API failure, (c) an appropriate empty state when no submissions exist or no submission found. Update any missing states.

- [X] T015 Verify role guards — sign in as a docente and navigate to `/student/cursos/any-uuid/actividades/any-uuid` → must be redirected to `/docente/dashboard`; sign in as a student and navigate to `/docente/cursos/any-uuid/actividades/any-uuid/submissions` → must be redirected to `/student/dashboard`.

---

## Dependencies

```
Phase 1 (Setup) → Phase 2 (Foundational) → Phases 3–5 (User Stories)

T001 [P]  → T005  ──────────────────────┐
T002 [P]  ────────── → T007 [US1]       │
T003 [P]  ────────── → T007 [US1]       ├→ Phase 6
T004 [P]  → T006  ──────────────────────┤
                        T008 [US2] [P]  │
                        T009 [US2] [P]  │
                        T010 [US2] → depends on T008
                        T011 [US2] → depends on T008
                        T012 [US3] → depends on T005
```

Within Phase 1:
```
T001 [P] + T002 [P] + T003 [P] + T004 [P]  →  T005 + T006  →  US phases
```

---

## Parallel Execution Examples

**Phase 1 first wave** (fully independent):
- T001 (types in academic.ts)
- T002 (rewrite FileUpload)
- T003 (rewrite IntegrityDeclaration)
- T004 (stub pages)

**Phase 2 second wave** (after wave 1):
- T005 (submissions.ts — after T001)
- T006 (router — after T004)

**Phase 4 US2** (after T005 + T006):
- T008 (SubmissionTable component — no page deps)
- T009 (Ver Entregas link in docente ActividadDetailPage — independent)

Then sequentially:
- T010 (docente SubmissionsPage — after T008)
- T011 (co-docente SubmissionsPage — after T008)

**Phases 3 and 5** (US1 and US3 are independent from each other, both depend only on T005):
- T007 (US1 — student upload)
- T012 (US3 — detail page)

---

## Implementation Strategy

**MVP scope (US1 only):** Complete Phase 1 + Phase 2 + Phase 3 (T001–T007). Students can upload, replace, and delete submissions. Docente can navigate to the (stub) submissions page.

**Incremental delivery order:**
1. Foundation (T001–T006) — unblocks all further work
2. US1 Student upload (T007) — core value delivered; students can submit files
3. US2 Docente list (T008–T011) — docente can see all submissions
4. US3 Docente download (T012) — docente can retrieve submitted files
5. Polish (T013–T015) — quality gate before marking MW5 done
