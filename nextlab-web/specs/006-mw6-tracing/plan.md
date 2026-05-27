# Implementation Plan: MW6 — Tracing

**Branch**: `main` | **Date**: 2026-05-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/006-mw6-tracing/spec.md`
**Research**: `specs/006-mw6-tracing/research.md`
**Data Model**: `specs/006-mw6-tracing/data-model.md`
**Contracts**: `specs/006-mw6-tracing/contracts/tracing.md`

---

## Summary

Build real-time session tracing across three roles. Student sees a session prompt on monitored activities, starts monitoring, and sees a persistent `SessionPanel` while working. Docente gets a historical trace viewer (from submission detail) and a live sessions monitor page. Co-docente has the same read-only access as docente.

Key decisions from research:
- WebSocket managed by a custom hook `useTracingSession.ts` (native API, no library)
- `TracingPanel.tsx` (Bloque 4 scaffold) fully replaced by `SessionPanel.tsx` (Tailwind + named export)
- Docente live feed uses 2-second polling (`GET /sessions/:sid/events?since=`) not a second WebSocket
- Session-submission link is backend-automated; frontend does not pass session ID on upload
- `SubmissionListItem.session_id: string | null` added to expose the link to the detail page

---

## Technical Context

**Language/Version**: TypeScript 5.9 (strict mode)

**Primary Dependencies**: React 19, React Router v7, axios, Tailwind CSS v4, Lucide React

**Storage**: N/A — frontend only; session data stored by nextlab-api

**Testing**: Manual smoke tests via `quickstart.md`. No automated tests for MW6.

**Target Platform**: Browser (React SPA at `http://localhost:5173`)

**Performance Goals**: Session start-to-active < 2 s (SC-001); live events visible within 2 s of generation (SC-006, achievable with 2-second poll interval)

**Constraints**: No external UI component library. No business logic on the frontend. WebSocket token via query param is the only permitted non-header auth exception (constitution IV).

**Scale/Scope**: 1 custom hook, 1 new API module, 5 new types + 1 type update, 1 replaced component, 1 new shared component, 5 new pages, 2 updated pages, 6 new routes.

---

## Constitution Check

*GATE: Must pass before implementation.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Role-Gated Rendering | ✅ Pass | SessionPanel under `ProtectedRoute allowedRoles={['estudiante']}`. Trace viewer, live sessions, live feed under docente + co_docente ProtectedRoutes. "Ver trazas" and "Ver Sesiones en Vivo" buttons hidden when not applicable. |
| II. API Client Isolation | ✅ Pass | All REST calls (createSession, endSession, getTraceLog, listActiveSessions, getLiveEvents) through `api` from `../lib/api`. WebSocket uses native API — the ONLY permitted exception (constitution IV). No raw `fetch()`. |
| III. React Quality Gate | ✅ Pass | React Doctor scan required after all MW6 pages complete. |
| IV. Token Security | ✅ Pass | Access token passed as `?token=<jwt>` to WebSocket only — explicitly permitted by constitution. All REST calls use the axios interceptor header. |
| V. Read-Only Evidence | ✅ Pass | Trace events displayed read-only in both historical viewer and live feed. No edit controls. |
| VI. Brand Baseline | ✅ Pass | Tailwind CSS v4 throughout. TracingPanel (inline styles, default export) fully replaced by SessionPanel (Tailwind, named export). |
| VII. Thin Client | ✅ Pass | Session state machine is local UI state reflecting backend state. No event content analysis. No reconnection logic that overrides server authority. Polling stops on unmount — server decides session lifetime. |

No violations. All gates pass.

---

## Project Structure

### Documentation (this feature)

```text
specs/006-mw6-tracing/
├── plan.md            ← this file
├── research.md        ← Phase 0 output (6 decisions)
├── data-model.md      ← Phase 1 output
├── quickstart.md      ← Phase 1 output
├── contracts/
│   └── tracing.md     ← Phase 1 output
└── tasks.md           ← Phase 2 output (speckit-tasks)
```

### Source Code Changes

Files to **create** (new):

```text
src/
├── hooks/
│   └── useTracingSession.ts       # WebSocket lifecycle: connect, send events, reconnect, cleanup
│
├── api/
│   └── tracing.ts                 # 5 functions: createSession, endSession, getTraceLog,
│                                  #              listActiveSessions, getLiveEvents
│
├── components/
│   ├── student/
│   │   └── SessionPanel.tsx       # Session status panel: state indicator, start time, terminate button
│   └── docente/
│       └── TraceEventTable.tsx    # Shared read-only table for both historical viewer + live feed
│
└── pages/
    ├── docente/
    │   ├── TraceViewerPage.tsx    # GET /submissions/:subid/trace — historical log
    │   ├── LiveSessionsPage.tsx   # GET /cursos/:cid/actividades/:aid/sessions/active
    │   └── LiveSessionFeedPage.tsx # Polls GET /sessions/:sid/events?since= every 2s
    └── codocente/
        ├── LiveSessionsPage.tsx   # Thin wrapper: mounts LiveSessionsPage under co-docente route
        └── LiveSessionFeedPage.tsx # Thin wrapper: mounts LiveSessionFeedPage under co-docente route
```

Files to **update** (existing):

```text
src/
├── types/
│   └── academic.ts        # Add 5 types (SessionResponse, TraceLogEntry, TraceLogResponse,
│                          #              ActiveSessionEntry, ActiveSessionsResponse)
│                          # Update SubmissionListItem: add session_id: string | null
│
├── components/
│   └── student/
│       └── TracingPanel.tsx  # REPLACE entirely → becomes SessionPanel.tsx (different file)
│                             # Old file can be deleted; new component is SessionPanel.tsx
│
├── pages/
│   ├── student/
│   │   └── ActividadDetailPage.tsx  # Add session section: prompt + SessionPanel + gates upload form
│   └── docente/
│       ├── ActividadDetailPage.tsx  # Add "Ver Sesiones en Vivo" link
│       └── SubmissionDetailPage.tsx # Add "Ver trazas" button (conditional on session_id)
│
└── router/
    └── index.tsx  # Add 6 new routes (3 docente, 3 co-docente)
```

---

## Key Implementation Details

### `src/hooks/useTracingSession.ts`

```ts
type SessionStatus = 'idle' | 'starting' | 'connecting' | 'active' | 'reconnecting' | 'ended' | 'error'

export function useTracingSession(aid: string, traceEnabled: boolean) {
  // Returns:
  //   status: SessionStatus
  //   sessionId: string | null
  //   startTime: string | null
  //   startSession(): void  — calls createSession(aid), then opens WebSocket
  //   endSession(): void    — calls endSession(sessionId), closes WebSocket
}
```

Internal WebSocket URL: `` `${import.meta.env.VITE_API_URL.replace('http', 'ws')}/sessions/${sessionId}/ws?token=${accessToken}` ``

DOM event listeners added on `active` state (removed on cleanup):
- `document.addEventListener('copy', ...)` → send `{ type: 'copy', content: e.clipboardData.getData('text') }`
- `document.addEventListener('paste', ...)` → send `{ type: 'paste', content: e.clipboardData.getData('text') }`
- `window.addEventListener('blur', ...)` → send `{ type: 'focus_out' }`
- `window.addEventListener('focus', ...)` → send `{ type: 'focus_in' }`
- `document.addEventListener('keydown', ...)` → send `{ type: 'keystroke' }` (no content — key value not sent for privacy)

Reconnection: exponential backoff — 1s, 2s, 4s (max 3 attempts). After 3 failures: set status to `error`.

Pong response: WebSocket `onmessage` checks for `{ type: 'ping' }` → sends `{ type: 'pong' }` back.

Access token: obtained from `useAuth()` context inside the hook.

### `src/api/tracing.ts`

```ts
export async function createSession(aid: string): Promise<SessionResponse>
  // POST /actividades/${aid}/sessions

export async function endSession(sid: string): Promise<void>
  // POST /sessions/${sid}/end → expects 204

export async function getTraceLog(submissionId: string): Promise<TraceLogResponse>
  // GET /submissions/${submissionId}/trace

export async function listActiveSessions(cid: string, aid: string): Promise<ActiveSessionsResponse>
  // GET /cursos/${cid}/actividades/${aid}/sessions/active

export async function getLiveEvents(sid: string, since?: string): Promise<TraceLogResponse>
  // GET /sessions/${sid}/events?since=${since}
```

All use the same `toApiError` pattern as `admin.ts` and `student.ts`.

### `src/components/student/SessionPanel.tsx`

Props: `status: SessionStatus`, `startTime: string | null`, `onStart: () => void`, `onEnd: () => void`

Renders based on status:
- `idle`: "Esta actividad requiere monitoreo de trazabilidad." + "Iniciar sesión de monitoreo" button
- `starting` / `connecting`: spinner + "Conectando..."
- `active`: green pulsing dot + "Sesión activa" + formatted start time + "Terminar sesión" button
- `reconnecting`: yellow dot + "Conexión perdida — reconectando…"
- `ended`: grey dot + "Sesión terminada" + end time
- `error`: red badge "Error de conexión — sesión interrumpida" + option to start new session

### Student `ActividadDetailPage` — session section

Added **above** the submission section (and before the upload form):

```tsx
{actividad.trace_enabled && (
  <SessionPanel
    status={sessionStatus}
    startTime={sessionStartTime}
    onStart={startSession}
    onEnd={() => setShowEndConfirm(true)}
  />
)}
```

The upload form section is **gated** behind session state:
- If `trace_enabled && sessionStatus !== 'active'`: hide upload form, show "Inicia la sesión de monitoreo para poder subir tu entrega."
- If `!trace_enabled || sessionStatus === 'active'`: show upload form as normal

### `src/components/docente/TraceEventTable.tsx`

Props: `events: TraceLogEntry[]`

Renders `<table>` with columns: Tipo, Timestamp, Contenido. Tipo displayed as a badge:
- `keystroke` → `bg-[#2e2e2e] text-[#888888]`
- `copy` / `paste` → `bg-yellow-400/10 text-yellow-400`
- `focus_in` / `focus_out` → `bg-blue-400/10 text-blue-400`

Empty state: `<p className="text-[#888888] py-4">No hay eventos de trazas registrados.</p>`

### `src/pages/docente/TraceViewerPage.tsx`

Params: `cid`, `aid`, `subid` (submission UUID) from `useParams`.
On mount: `getTraceLog(subid)`.
Renders: heading "Trazas de entrega", `<TraceEventTable events={data.events} />`.
Back link: `← Entrega` → `/docente/cursos/${cid}/actividades/${aid}/submissions/${subid}`.

### `src/pages/docente/LiveSessionsPage.tsx`

Params: `cid`, `aid`. Polls `listActiveSessions(cid, aid)` every 5 seconds (interval cleared on unmount).
Each row navigates to `/${role}/cursos/${cid}/actividades/${aid}/sessions/${entry.session_id}/live`.
Back link: `← Actividad` → `/docente/cursos/${cid}/actividades/${aid}`.

### `src/pages/docente/LiveSessionFeedPage.tsx`

Params: `cid`, `aid`, `sid`. 
Initial load: `getLiveEvents(sid)` → store events.
Polling: every 2 seconds, `getLiveEvents(sid, lastEventTimestamp)` → append new events.
When `session.ended_at !== null` (from list page state): stop polling, show "Sesión terminada".
Back link: `← Sesiones en vivo` → `/docente/cursos/${cid}/actividades/${aid}/sessions`.

### Router additions

```ts
// Inside ProtectedRoute allowedRoles={['docente']}
{ path: '/docente/cursos/:cid/actividades/:aid/sessions', element: <LiveSessionsPage /> }
{ path: '/docente/cursos/:cid/actividades/:aid/sessions/:sid/live', element: <LiveSessionFeedPage /> }
{ path: '/docente/cursos/:cid/actividades/:aid/submissions/:subid/trace', element: <TraceViewerPage /> }

// Inside ProtectedRoute allowedRoles={['co_docente']}
{ path: '/codocente/cursos/:cid/actividades/:aid/sessions', element: <CodocenteLiveSessionsPage /> }
{ path: '/codocente/cursos/:cid/actividades/:aid/sessions/:sid/live', element: <CodocenteLiveSessionFeedPage /> }
{ path: '/codocente/cursos/:cid/actividades/:aid/submissions/:subid/trace', element: <CodocenteTraceViewerPage /> }
```

### Docente `ActividadDetailPage` updates

Add "Ver Sesiones en Vivo" link alongside "Ver Entregas":
```tsx
<Link to={`/docente/cursos/${cursoId}/actividades/${aid}/sessions`} className="...">
  Ver Sesiones en Vivo
</Link>
```

### Docente `SubmissionDetailPage` updates

Existing page receives `SubmissionListItem` via `useLocation().state`. Add conditional "Ver trazas" button:
```tsx
{item.session_id && (
  <Link to={`${backPath.replace('/submissions', '/submissions')}/${sid}/trace`} ...>
    Ver trazas
  </Link>
)}
```

Wait — the trace route uses `subid` not `sid`. The submission's UUID is already in the URL as `sid` param. Trace route: `/docente/cursos/:cid/actividades/:aid/submissions/:subid/trace`. Since `sid` in the existing route IS the submission UUID, the link is:
```tsx
{item.session_id && (
  <Link
    to={`${location.pathname}/trace`}
    className="inline-flex items-center gap-2 bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg px-4 py-2 text-sm text-[#f0f0f0] hover:border-[#E85D24] transition-colors"
  >
    Ver trazas
  </Link>
)}
```

---

## Scope Adjustments (vs. spec)

| Spec claim | Actual behavior | Source |
|---|---|---|
| Session prompt blocks "working" | Upload form hidden when session not active; activity instructions still visible | Research Decision 6 / Principle VII |
| "Co-docente thin wrappers" | Co-docente pages mount docente page components via shared implementation | Architecture decision — avoids code duplication |

---

## Complexity Tracking

**`useTracingSession` hook**: Justified complexity. The WebSocket lifecycle (connect, ping/pong, reconnect with backoff, DOM event capture, cleanup) cannot be cleanly split across multiple components. A single hook is the React-idiomatic solution. No constitution violation.
