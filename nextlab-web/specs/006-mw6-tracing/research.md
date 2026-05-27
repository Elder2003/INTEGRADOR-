# Research: MW6 — Tracing

**Feature**: MW6 — Tracing
**Date**: 2026-05-26

---

## Decision 1: WebSocket Client — Native API via Custom Hook

**Decision**: Use the native browser `WebSocket` API wrapped in a custom React hook `src/hooks/useTracingSession.ts`.

**Rationale**: The constitution explicitly mandates the WebSocket URL pattern `ws://[API_HOST]/sessions/:sid/ws?token=<jwt>` and names WebSocket as the student-side transport. No third-party WebSocket library (socket.io, etc.) is needed or permitted — the hook encapsulates connection lifecycle, DOM event capture, reconnection logic, and cleanup, keeping the component thin. The hook is the appropriate boundary between React state and the browser WebSocket API.

**Alternatives considered**:
- `socket.io-client`: Rejected — external dependency, not permitted by Principle VI; adds unnecessary abstraction over a plain WebSocket server.
- Inline WebSocket in component: Rejected — violates single-responsibility and makes reconnection/cleanup error-prone.

---

## Decision 2: TracingPanel.tsx — Full Replacement

**Decision**: Replace `src/components/student/TracingPanel.tsx` entirely. The existing file is Bloque 4 scaffold (default export, inline CSS variable styles, different data interface: WPM/corrections/activityHistory). MW6 needs a session status panel (connection state, start time, terminate button) — a fundamentally different component.

**Rationale**: The old interface accepts pre-computed analytics metrics that the MW6 API does not provide. Adapting it would require a breaking props change anyway. A clean replacement named `SessionPanel` avoids confusion and follows the Tailwind + named export convention established in MW5.

**Alternatives considered**:
- Adapting existing TracingPanel: Rejected — props interface is incompatible; would result in dead props and misleading naming.

---

## Decision 3: Docente Live Feed — 2-Second Polling (not WebSocket)

**Decision**: The docente live event feed uses interval polling (`GET /sessions/:sessionId/events?since=<iso>` every 2 seconds) rather than a second WebSocket connection.

**Rationale**: The backend session WebSocket is designed for the student to *send* events. Connecting a docente as a read-only subscriber to the same WebSocket endpoint may not be supported (Go gorilla/websocket hubs require explicit multi-listener setup). Polling with a `since` timestamp parameter is universally supported, delivers events within the 2-second SC-006 target, and avoids the need to negotiate a secondary WebSocket lifecycle. The polling stops when the docente leaves the page (component unmount cleanup).

**Alternatives considered**:
- Docente WebSocket on same endpoint: Rejected — cannot confirm backend supports multiple listeners on a single session; risky assumption.
- Server-Sent Events (SSE): Rejected — requires additional backend endpoint; polling achieves same SLA with less coordination.

---

## Decision 4: Session-Submission Link — Backend Auto-Associates

**Decision**: The session ID is automatically associated with the submission by the backend (M5). When a student uploads a file, the backend checks for an active session for that student+activity and stores the link. The frontend does NOT pass the session ID to the upload endpoint.

**Rationale**: This keeps the frontend thin (Principle VII) and avoids race conditions where the frontend might pass a stale or incorrect session ID. The backend is authoritative about which sessions are active. The `SubmissionListItem` returned by `GET /cursos/:cid/actividades/:aid/submissions` will include `session_id: string | null` — null means no session was active at upload time.

**Alternatives considered**:
- Frontend sends session ID in upload FormData: Rejected — business logic on the frontend (deciding which session is "current"); violates Principle VII.

---

## Decision 5: `SubmissionListItem` Type Extension

**Decision**: Add `session_id: string | null` to the existing `SubmissionListItem` interface in `src/types/academic.ts`. The `SubmissionDetailPage` (MW5) already receives the full `SubmissionListItem` via `useLocation().state`, so adding this field gives it enough information to conditionally show "Ver trazas" without an additional API call.

**Rationale**: Minimal change to the existing type; backward compatible (field is nullable); consistent with how the detail page already works in MW5. No new API endpoint needed for the student-facing submission shape (`SubmissionResponse`) since students navigate to tracing via the activity page, not the submission.

**Alternatives considered**:
- Separate `GET /submissions/:sid/has-trace` endpoint: Rejected — unnecessary round trip; the list endpoint already has all the data.

---

## Decision 6: Existing Student Activity Page — Session Prompt Placement

**Decision**: The session prompt (confirmation to start monitoring) and the `SessionPanel` (active session indicator + terminate button) are added as a new section in the existing `src/pages/student/ActividadDetailPage.tsx`, rendered only when `actividad.trace_enabled === true`. The prompt gates the display of the upload form — the student must start a session before submitting a file.

**Rationale**: Keeps all activity-related student state on one page, consistent with how the file submission section was added in MW5. The session is scoped to the activity, so placing it on the activity detail page is the natural location. Gating the upload form behind session start aligns with FR-001 and FR-002 (explicit confirmation required before monitoring begins).

**Alternatives considered**:
- Separate `/student/cursos/:id/actividades/:aid/session` route: Rejected — extra navigation step with no user value; the session is an enhancement to the activity detail, not a standalone page.
