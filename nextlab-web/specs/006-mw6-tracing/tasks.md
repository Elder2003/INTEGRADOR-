# Tasks: MW6 — Tracing

**Feature**: MW6 — Tracing
**Plan**: [plan.md](plan.md) | **Spec**: [spec.md](spec.md) | **Contracts**: [contracts/tracing.md](contracts/tracing.md)
**Total tasks**: 18 | **Last updated**: 2026-05-26

---

## Phase 1 — Setup

**Goal**: Lay the shared foundation — types, remove old scaffold, create stub pages. All tasks are independent and run in parallel.

- [X] T001 [P] Add 5 new types + update `SubmissionListItem` in `src/types/academic.ts` — append after existing types: `SessionResponse { id: string; actividad_id: string; student_id: string; started_at: string; ended_at: string | null }`, `TraceLogEntry { id: string; session_id: string; type: 'keystroke' | 'copy' | 'paste' | 'focus_in' | 'focus_out'; timestamp: string; content: string | null }`, `TraceLogResponse { session_id: string; events: TraceLogEntry[]; total: number }`, `ActiveSessionEntry { session_id: string; student: StudentInfo; started_at: string; event_count: number }`, `ActiveSessionsResponse { actividad_id: string; sessions: ActiveSessionEntry[] }`. Also update the existing `SubmissionListItem` interface by adding `session_id: string | null` as a new field (nullable — null means no active session was linked at upload time).

- [X] T002 [P] Delete `src/components/student/TracingPanel.tsx` — this is the Bloque 4 scaffold component (default export, inline CSS var styles, props: wpm/activeTime/pasteCount/corrections/activityHistory) that is incompatible with MW6's session-based model. It is dead code (not imported by any live page). Remove the file entirely. Also delete `src/components/student/MiniBarChart.tsx` if it is only referenced by TracingPanel (verify with grep before deleting).

- [X] T003 [P] Create minimal placeholder stubs for 8 new page/component files: `src/components/student/SessionPanel.tsx` (exports named `SessionPanel` returning `<p>SessionPanel</p>`), `src/components/docente/TraceEventTable.tsx` (exports named `TraceEventTable` returning `<p>TraceEventTable</p>`), `src/pages/docente/TraceViewerPage.tsx` (exports named `TraceViewerPage` returning `<p>TraceViewerPage</p>`), `src/pages/docente/LiveSessionsPage.tsx` (exports named `LiveSessionsPage` returning `<p>LiveSessionsPage</p>`), `src/pages/docente/LiveSessionFeedPage.tsx` (exports named `LiveSessionFeedPage` returning `<p>LiveSessionFeedPage</p>`), `src/pages/codocente/TraceViewerPage.tsx` (exports named `CodocenteTraceViewerPage` returning `<p>CodocenteTraceViewerPage</p>`), `src/pages/codocente/LiveSessionsPage.tsx` (exports named `CodocenteLiveSessionsPage` returning `<p>CodocenteLiveSessionsPage</p>`), `src/pages/codocente/LiveSessionFeedPage.tsx` (exports named `CodocenteLiveSessionFeedPage` returning `<p>CodocenteLiveSessionFeedPage</p>`). These stubs are replaced in Phases 3–5.

---

## Phase 2 — Foundational

**Goal**: Wire the API module, hook, and new routes so all paths compile and TypeScript passes. Complete both waves before starting user story work.

**Wave 1 (parallel — different files):**

- [X] T004 [P] Create `src/api/tracing.ts` — implement 5 functions using the same `toApiError` helper pattern as `src/api/student.ts`. Import `api` from `../lib/api`, `ApiError` from `../types`, `AxiosError` from `axios`, and types from `../types/academic`. Functions: `createSession(aid: string): Promise<SessionResponse>` → `POST /actividades/${aid}/sessions`; `endSession(sid: string): Promise<void>` → `POST /sessions/${sid}/end`, expects 204; `getTraceLog(submissionId: string): Promise<TraceLogResponse>` → `GET /submissions/${submissionId}/trace`; `listActiveSessions(cid: string, aid: string): Promise<ActiveSessionsResponse>` → `GET /cursos/${cid}/actividades/${aid}/sessions/active`; `getLiveEvents(sid: string, since?: string): Promise<TraceLogResponse>` → `GET /sessions/${sid}/events${since ? '?since=' + encodeURIComponent(since) : ''}`. All catch blocks call `toApiError(err)`. Depends on T001.

- [X] T004b [P] Update `src/router/index.tsx` — import stubs: `TraceViewerPage`, `LiveSessionsPage`, `LiveSessionFeedPage` from `../pages/docente/...`; `CodocenteTraceViewerPage`, `CodocenteLiveSessionsPage`, `CodocenteLiveSessionFeedPage` from `../pages/codocente/...`. Add 3 routes inside `ProtectedRoute allowedRoles={['docente']}` children: `{ path: '/docente/cursos/:cid/actividades/:aid/sessions', element: <LiveSessionsPage /> }`, `{ path: '/docente/cursos/:cid/actividades/:aid/sessions/:sid/live', element: <LiveSessionFeedPage /> }`, `{ path: '/docente/cursos/:cid/actividades/:aid/submissions/:subid/trace', element: <TraceViewerPage /> }`. Add 3 routes inside `ProtectedRoute allowedRoles={['co_docente']}` children: same patterns with `CodocenteLive*` and `CodocenteTraceViewerPage` components and `/codocente/` prefix. Depends on T003.

**Wave 2 (sequential — depends on T004):**

- [X] T005 Create `src/hooks/useTracingSession.ts` — custom hook that manages the full WebSocket session lifecycle. Import `useEffect`, `useRef`, `useState`, `useCallback` from react; `createSession`, `endSession` from `../api/tracing`; `useAuth` from `../context/AuthContext`. Type: `type SessionStatus = 'idle' | 'starting' | 'connecting' | 'active' | 'reconnecting' | 'ended' | 'error'`. Hook signature: `export function useTracingSession(aid: string, traceEnabled: boolean)` returning `{ status: SessionStatus; sessionId: string | null; startTime: string | null; startSession: () => Promise<void>; terminateSession: () => Promise<void> }`. Implementation: `startSession` calls `createSession(aid)` → sets `sessionId` and `startTime` → opens WebSocket at `` `${import.meta.env.VITE_API_URL.replace('http', 'ws')}/sessions/${sessionId}/ws?token=${accessToken}` `` → on `ws.onopen`: set status to `'active'`; add DOM event listeners (`document` for `copy`, `paste`, `keydown`; `window` for `blur`, `focus`) each sending JSON `{ type: '...', timestamp: new Date().toISOString(), content?: string }` via `ws.send(JSON.stringify(...))` — keydown sends `{ type: 'keystroke' }` with NO content; copy/paste include `e.clipboardData?.getData('text')` truncated to 500 chars; blur → `focus_out`; focus → `focus_in`; on `ws.onmessage`: if `{ type: 'ping' }` respond with `ws.send(JSON.stringify({ type: 'pong' }))`; on `ws.onclose`/`ws.onerror`: if `reconnectAttempts < 3` set status to `'reconnecting'`, wait `1000 * 2^attempt` ms, retry WebSocket connection; after 3 failures set status `'error'`. `terminateSession` calls `endSession(sessionId)` → closes WebSocket → sets status `'ended'`. `useEffect` cleanup: close WebSocket and remove DOM event listeners on unmount. If `!traceEnabled`: return `{ status: 'idle', ... }` immediately (no-op). Depends on T004.

---

## Phase 3 — US1 + US3: Student Session (P1)

**Story goal (US1)**: Student navigates to a monitored activity, sees a session prompt, starts monitoring, sees the active status panel, and can interact normally. **Story goal (US3)**: Student can explicitly end the session via a confirm modal. Both stories share the same two files.

**Independent test criteria**: Navigate to an activity with `trace_enabled=true` as a student → session prompt visible (upload form hidden) → click "Iniciar sesión" → "Sesión activa" panel appears with green dot + start time → copy some text → events stream (verify via docente live monitor) → click "Terminar sesión" → confirm modal → panel shows "Sesión terminada".

- [X] T006 [US1] Implement `src/components/student/SessionPanel.tsx` (replace stub) — named export `SessionPanel`. Type: `type SessionStatus = 'idle' | 'starting' | 'connecting' | 'active' | 'reconnecting' | 'ended' | 'error'`. Props: `{ status: SessionStatus; startTime: string | null; onStart: () => void; onEnd: () => void }`. Render by status: `idle` → `bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4` with text "Esta actividad requiere monitoreo de trazabilidad." and `<button onClick={onStart} className="mt-3 px-4 py-2 bg-[#E85D24] text-white text-sm rounded-lg hover:bg-[#d4521f] transition-colors">Iniciar sesión de monitoreo</button>`; `starting`/`connecting` → spinner + "Conectando..."; `active` → green pulsing dot (`w-2 h-2 rounded-full bg-green-500 animate-pulse`) + `<span className="text-green-400 text-sm font-medium">Sesión activa</span>` + formatted `startTime` + `<button onClick={onEnd} className="text-xs border border-[#2e2e2e] text-[#888888] px-3 py-1 rounded hover:border-red-400 hover:text-red-400 transition-colors">Terminar sesión</button>`; `reconnecting` → yellow dot + "Conexión perdida — reconectando…"; `ended` → grey dot + "Sesión terminada" + formatted end time (use Date.now() at render) + "Iniciar nueva sesión" button that calls `onStart`; `error` → red badge "Error de conexión — sesión interrumpida" + "Reintentar" button calling `onStart`. Depends on T001.

- [X] T007 [US1] Extend `src/pages/student/ActividadDetailPage.tsx` — add tracing session section. Import `useTracingSession` from `../../hooks/useTracingSession`; `SessionPanel` from `../../components/student/SessionPanel`; `ConfirmModal` from `../../components/common/ConfirmModal`. Add state: `showEndConfirm: boolean`. Add hook call (after existing state): `const { status: sessionStatus, startTime: sessionStartTime, startSession, terminateSession } = useTracingSession(aid!, actividad?.trace_enabled ?? false)`. Add session section **above** the "Mi Entrega" heading (inside the outer `<div>`, after the instructions block): `{actividad.trace_enabled && (<div className="space-y-1"><h2 className="text-lg font-semibold text-[#f0f0f0]">Sesión de monitoreo</h2><SessionPanel status={sessionStatus} startTime={sessionStartTime} onStart={startSession} onEnd={() => setShowEndConfirm(true)} /></div>)}`. Gate the upload form: wrap the entire upload form (the block rendered when `submission === null`) with `{(!actividad.trace_enabled || sessionStatus === 'active') ? (<>...upload form...</>) : (<p className="text-[#888888] text-sm">Inicia la sesión de monitoreo para poder subir tu entrega.</p>)}`. Add `<ConfirmModal isOpen={showEndConfirm} title="Terminar sesión" message="¿Seguro que deseas terminar la sesión de monitoreo?" confirmLabel="Terminar" isLoading={false} onConfirm={() => { terminateSession(); setShowEndConfirm(false); }} onCancel={() => setShowEndConfirm(false)} />` at the end of the JSX. Depends on T005, T006.

---

## Phase 4 — US2: Docente Historical Trace Viewer (P2)

**Story goal**: Docente navigates from a submission's detail page to a read-only trace log showing all recorded events in chronological order.

**Independent test criteria**: From submission detail page with `session_id !== null`, click "Ver trazas" → trace viewer page loads with chronological table of events showing type badges (copy/paste highlighted yellow, focus_* blue, keystroke grey) and timestamps.

- [X] T008 [P] [US2] Implement `src/components/docente/TraceEventTable.tsx` (replace stub) — named export `TraceEventTable`. Props: `{ events: TraceLogEntry[] }`. If `events.length === 0`: render `<p className="text-[#888888] py-4">No hay eventos de trazas registrados.</p>`. Otherwise render `<div className="overflow-x-auto"><table className="w-full text-sm">` with header: Tipo, Timestamp, Contenido. For each event: `<tr key={event.id} className="border-b border-[#2e2e2e]">`. Type badge: `keystroke` → `bg-[#2e2e2e] text-[#888888]`; `copy`/`paste` → `bg-yellow-400/10 text-yellow-400`; `focus_in`/`focus_out` → `bg-blue-400/10 text-blue-400`. Timestamp: `new Date(event.timestamp).toLocaleString('es-BO')`. Content: `event.content ?? <span className="text-[#888888] italic">—</span>`. Depends on T001.

- [X] T009 [P] [US2] Update `src/pages/docente/SubmissionDetailPage.tsx` — add "Ver trazas" button. The page already imports `useLocation`, `useParams`, `Link`. Add: `import { useLocation } from 'react-router-dom'` (already imported). After the download button block, add: `{item.session_id && (<Link to={\`\${location.pathname}/trace\`} className="inline-flex items-center gap-2 bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg px-4 py-2 text-sm text-[#f0f0f0] hover:border-[#E85D24] transition-colors">Ver trazas</Link>)}`. This uses `location.pathname` so it works for both `/docente/...` and `/codocente/...` routes (the component is shared per MW5 plan). Depends on T001 (SubmissionListItem now has session_id).

- [X] T010 [US2] Implement `src/pages/docente/TraceViewerPage.tsx` (replace stub) — get params `{ cid, aid, subid }` from `useParams<{ cid: string; aid: string; subid: string }>()`. Derive `backPath` from `window.location.pathname.startsWith('/docente')` → `/docente/cursos/${cid}/actividades/${aid}/submissions/${subid}` else codocente equivalent. State: `data: TraceLogResponse | null`, `loading: boolean`, `error: string | null`. On mount: `getTraceLog(subid!).then(setData).catch(e => setError(e.message)).finally(() => setLoading(false))`. Show `<LoadingSpinner />` while loading. On error: `<ErrorMessage message={error} />` + back link. Main render: `<Link to={backPath}>← Entrega</Link>`, heading "Trazas de entrega", `<TraceEventTable events={data?.events ?? []} />`. Depends on T008.

---

## Phase 5 — US4: Docente Live Session Monitor (P3)

**Story goal**: Docente sees a list of all students with active sessions for an activity and can open a live feed that auto-updates with new events every 2 seconds while the student works.

**Independent test criteria**: Click "Ver Sesiones en Vivo" on a docente activity detail page → LiveSessionsPage shows active student(s) → click a student → LiveSessionFeedPage opens and new events appear within ~2 seconds of the student generating them → student ends session → feed shows "Sesión terminada".

- [X] T011 [P] [US4] Update `src/pages/docente/ActividadDetailPage.tsx` — add "Ver Sesiones en Vivo" link alongside the existing "Ver Entregas" link. Add: `<Link to={\`/docente/cursos/${cursoId}/actividades/${aid}/sessions\`} className="inline-flex items-center gap-2 bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg px-4 py-2 text-sm text-[#f0f0f0] hover:border-[#E85D24] transition-colors">Ver Sesiones en Vivo</Link>` in the same "Ver Entregas" div block. The `Link` import is already present.

- [X] T012 [US4] Implement `src/pages/docente/LiveSessionsPage.tsx` (replace stub) — get params `{ cid, aid }` from `useParams`. State: `data: ActiveSessionsResponse | null`, `loading: boolean`, `error: string | null`. On mount: `listActiveSessions(cid!, aid!)` initial fetch. Set up `setInterval(() => listActiveSessions(cid!, aid!).then(setData).catch(() => {}), 5000)` — clear interval on unmount via `useEffect` cleanup. Show `<LoadingSpinner />` on initial loading. Back link: `<Link to={\`/docente/cursos/${cid}/actividades/${aid}\`}>← Actividad</Link>`. Heading: "Sesiones en Vivo". If `data?.sessions.length === 0`: `<p className="text-[#888888]">No hay sesiones activas en este momento.</p>`. Otherwise: table with columns: Estudiante, Email, Inicio, Eventos; each row `onClick={() => navigate(\`/docente/cursos/${cid}/actividades/${aid}/sessions/${entry.session_id}/live\`)}` with `cursor-pointer hover:bg-[#1a1a1a]`. Depends on T004.

- [X] T013 [US4] Implement `src/pages/docente/LiveSessionFeedPage.tsx` (replace stub) — get params `{ cid, aid, sid }` from `useParams`. State: `events: TraceLogEntry[]`, `loading: boolean`, `error: string | null`, `sessionEnded: boolean`. On mount: initial `getLiveEvents(sid!)` → set `events`. Set up 2-second polling: `setInterval(async () => { const last = events[events.length - 1]?.timestamp; const res = await getLiveEvents(sid!, last); if (res.events.length > 0) setEvents(prev => [...prev, ...res.events]); }, 2000)` — clear on unmount. If `sessionEnded` (detect via empty responses after active period, or API returns 404): stop polling, show "Sesión terminada" banner. Back link: `<Link to={\`/docente/cursos/${cid}/actividades/${aid}/sessions\`}>← Sesiones en vivo</Link>`. Heading: session student name (if passed via location.state) or "Sesión en vivo". Render `<TraceEventTable events={events} />`. When `events.length === 0 && !loading`: show "Esperando eventos…" indicator. Depends on T008, T004.

- [X] T014 [US4] Implement co-docente wrappers — implement 3 files under `src/pages/codocente/`: `LiveSessionsPage.tsx` (same structure as docente's but with back link to `/codocente/dashboard` and navigate to `/codocente/cursos/${cid}/actividades/${aid}/sessions/${entry.session_id}/live`), `LiveSessionFeedPage.tsx` (same structure as docente's but back link to `/codocente/cursos/${cid}/actividades/${aid}/sessions`), `TraceViewerPage.tsx` (same structure as docente's `TraceViewerPage` but derives backPath with `/codocente/` prefix). All three use the same `listActiveSessions`, `getLiveEvents`, `getTraceLog` API functions and `TraceEventTable` component. Depends on T008, T012, T013.

---

## Phase 6 — Polish & Quality Gate

**Goal**: Ensure quality before marking MW6 complete.

- [X] T015 Run React Doctor scan from repo root (`npx react-doctor@latest .`) — resolve ALL findings with ERROR severity before proceeding. The `useTracingSession` hook uses `useEffect` with complex dependencies — verify no stale closure warnings appear for the reconnect logic. Add a one-line justification comment for any WARNING-severity findings that are deferred.

- [X] T016 Audit all affected pages — verify each of `ActividadDetailPage` (student), `TraceViewerPage`, `LiveSessionsPage`, `LiveSessionFeedPage` (docente + co-docente equivalents) shows: (a) loading indicator during initial fetch, (b) error message on API failure, (c) empty state when no data. Specifically: `LiveSessionFeedPage` must show "Esperando eventos…" when active session has no events yet; `TraceViewerPage` must show `<TraceEventTable>` empty state ("No hay eventos…") when session has no events.

- [X] T017 Verify role guards — sign in as student → navigate to `/docente/cursos/:cid/actividades/:aid/sessions` → must redirect to `/student/dashboard`. Sign in as admin → navigate to `/codocente/cursos/:cid/actividades/:aid/sessions/:sid/live` → must redirect to `/admin/dashboard`. Verify `trace_enabled=false` activity page shows NO session prompt and NO SessionPanel for student.

- [X] T018 Verify TypeScript compiles clean (`npx tsc --noEmit`) with 0 errors — in particular confirm: `useTracingSession` return type, `SessionStatus` usage in `SessionPanel` props, `session_id: string | null` on `SubmissionListItem` does not break existing MW5 code in `SubmissionsPage` or `SubmissionDetailPage`.

---

## Dependencies

```
Phase 1 (Setup) → Phase 2 (Foundational) → Phases 3–5 (User Stories)

T001 [P]  ───────────────────────────────────────────────────────┐
T002 [P]  (no downstream deps — cleanup only)                    │
T003 [P]  → T004b [P] (router stubs) ──────────────────────────┤
                                                                   │
           T004 [P] (api) ──→ T005 (hook) ──→ T006+T007 [US1]   │
           T004b [P] (router)                                      │
                                                                   ├→ Phase 6
           T008 [P] [US2] + T009 [P] [US2] → T010 [US2]         │
                                                                   │
           T011 [P] [US4] + T012 [US4] → T013 [US4] → T014 [US4]│
```

Phase 1 all-parallel:
```
T001 [P] + T002 [P] + T003 [P]
```

Phase 2 two waves:
```
T004 [P] + T004b [P]  →  T005
```

Phase 3 sequential (US1+US3):
```
T006 [US1] + T005 (done) → T007 [US1]
```

Phase 4 partial-parallel (US2):
```
T008 [P] [US2] + T009 [P] [US2] → T010 [US2]
```

Phase 5 sequential (US4):
```
T011 [P] [US4]  (independent)
T012 [US4] → T013 [US4] → T014 [US4]
```

---

## Parallel Execution Examples

**Phase 1 first wave** (fully independent):
- T001 (types), T002 (delete old file), T003 (stub pages)

**Phase 2 first wave** (after Phase 1):
- T004 (tracing.ts API module), T004b (router additions)

**Phase 2 second wave** (after T004):
- T005 (useTracingSession hook)

**Phase 4 first wave** (after T005, independent of each other):
- T008 (TraceEventTable component), T009 (SubmissionDetailPage "Ver trazas" button)

**Phase 5 partial-parallel** (T011 is independent; T012→T013→T014 sequential):
- T011 ("Ver Sesiones en Vivo" link — one-liner addition)

---

## Implementation Strategy

**MVP scope (US1 only):** Complete Phase 1 + Phase 2 + Phase 3 (T001–T007). Student can start and end a monitoring session on traced activities. Session panel visible with status. Upload form gated behind active session.

**Incremental delivery order:**
1. Foundation (T001–T005) — types, hook, API; TypeScript compiles clean
2. US1+US3 Student session (T006–T007) — session prompt, panel, terminate flow
3. US2 Historical trace viewer (T008–T010) — docente reads completed trace logs
4. US4 Live monitor (T011–T014) — docente watches active sessions in real time
5. Polish (T015–T018) — quality gate before marking MW6 done
