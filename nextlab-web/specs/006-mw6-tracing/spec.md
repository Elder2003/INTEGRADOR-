# Feature Specification: MW6 — Tracing

**Feature Branch**: `main`

**Created**: 2026-05-26

**Status**: Draft

**Input**: User description: "lets do mw6"

## Clarifications

### Session 2026-05-26

- Q: How does the docente's trace viewer connect to a specific submission — via activity-level sessions, an implicit session-submission link, or direct coupling? → A: Option B — when a student uploads a file and an active session exists, the session ID is automatically recorded in the submission. The docente accesses the trace from the submission detail page only when that link exists.
- Q: Does the docente trace viewer need to show events in real time (while the student is working), as a historical log, or both? → A: Option C — both. A historical viewer is accessible from the submission detail page (existing US2); additionally, a live session monitor page shows all currently active sessions for an activity and lets the docente watch a student's events stream in real time.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Student Opens a Monitored Session (Priority: P1)

A student navigates to an activity with monitoring enabled (`trace_enabled = true`). Before they can begin working, the system informs them that the session will be recorded and prompts them to start. Once they confirm, monitoring begins and they see a visible panel confirming the session is active. All interaction events (keystrokes, clipboard actions, window focus changes) are streamed to the platform in real time. The student can see the session is running throughout their work.

**Why this priority**: Without this story, the tracing system has no data to display. The session must be opened before any other tracing feature has value.

**Independent Test**: Navigate to an activity with monitoring enabled as a student → confirm session start prompt → start session → verify visible "Sesión activa" indicator → interact with the page → verify that a session record exists in the backend.

**Acceptance Scenarios**:

1. **Given** a student is on an activity detail page where `trace_enabled = true`, **When** the page loads, **Then** a session prompt is shown explaining that the session will be monitored and requiring explicit confirmation to start.
2. **Given** the student confirms the session start, **When** the session opens, **Then** a persistent status panel appears showing "Sesión activa" with the start time.
3. **Given** the session is active, **When** the student interacts with the page (typing, copying, tabbing away), **Then** the interaction events are streamed to the backend with no visible disruption to the student's workflow.
4. **Given** the session is active, **When** the student navigates away from the activity page, **Then** the WebSocket connection is cleanly closed and the session is marked as ended.
5. **Given** a student is on an activity where `trace_enabled = false`, **When** the page loads, **Then** no session prompt appears and no monitoring occurs.

---

### User Story 2 — Docente Views a Student's Trace Log (Priority: P2)

A docente navigates to a student's submission and opens the trace viewer. They see a complete, chronological list of all trace events recorded during the student's session: timestamps, event types (keystroke, copy, paste, focus lost, focus regained), and any associated content. The viewer is read-only. This gives the docente evidence of how the student worked during the activity.

**Why this priority**: This is the primary output of the tracing system. Without a viewer, all captured data is inaccessible. It follows US1 (data must exist before it can be viewed).

**Independent Test**: Navigate to a submission detail page as a docente → click "Ver trazas" → verify chronological event list renders with type, timestamp, and content columns.

**Acceptance Scenarios**:

1. **Given** a docente is on the submission detail page for a student, **When** they click "Ver trazas", **Then** they navigate to the trace viewer page for that submission.
2. **Given** the trace viewer loads, **When** the session has events, **Then** a chronological table is shown with columns: timestamp, event type, and content/detail.
3. **Given** the trace viewer loads, **When** the session has no events (empty trace), **Then** an empty state message is shown ("No hay eventos de trazas registrados.").
4. **Given** the trace viewer is showing events, **When** the docente attempts to edit any event, **Then** no edit controls exist — all data is read-only.
5. **Given** a docente is on the submission list page for an activity, **When** they click a submission row, **Then** the submission detail shows a "Ver trazas" button only for submissions that have a linked session.

---

### User Story 3 — Student Explicitly Ends a Session (Priority: P3)

A student who has finished their work can explicitly end the monitoring session from the activity page. The system closes the WebSocket connection, marks the session as ended, and updates the status panel to reflect the session is no longer active. The student can restart a new session if the edit window is still open.

**Why this priority**: US1 handles implicit session end on navigation. This story adds an explicit "Terminar sesión" control that improves clarity for students who want to confirm their session is properly closed.

**Independent Test**: With an active session open, click "Terminar sesión" → confirm the prompt → verify the status panel updates to show the session has ended → verify the WebSocket is disconnected.

**Acceptance Scenarios**:

1. **Given** a session is active, **When** the student clicks "Terminar sesión", **Then** a confirmation prompt appears ("¿Terminar la sesión de monitoreo?").
2. **Given** the student confirms, **When** the session ends, **Then** the status panel updates to "Sesión terminada" with the end time and the WebSocket disconnects.
3. **Given** the session has ended, **When** the student is still within the edit window, **Then** an option to start a new session appears.
4. **Given** the student cancels the termination prompt, **When** they return to the activity, **Then** the session remains active and events continue streaming.

---

### User Story 4 — Docente Monitors a Live Session (Priority: P3)

A docente navigates to a live sessions monitor page for an activity and sees a list of students who currently have an active monitoring session open. Clicking a student's name opens a live event feed that updates automatically as the student's interaction events arrive. The docente can watch events appear in real time without refreshing the page. When the student ends their session, the live feed indicates the session has closed.

**Why this priority**: This fulfills the real-time monitoring capability stated in the project constitution. It complements the historical viewer (US2) by allowing proctoring during an active exam or assignment window.

**Independent Test**: While a student has an active session open, navigate to the live monitor page as a docente → verify the student appears in the active sessions list → click the student → verify new events appear in the feed as the student interacts with the page → wait for the student to end their session → verify the feed shows the session as ended.

**Acceptance Scenarios**:

1. **Given** a docente navigates to the live monitor for an activity, **When** one or more students have active sessions, **Then** a list shows each student's name, session start time, and event count.
2. **Given** the docente clicks a student in the live sessions list, **When** the live feed opens, **Then** events appear as they are received, newest at the bottom, with type and timestamp.
3. **Given** the live feed is open and no new events arrive for 30 seconds, **When** the connection is still active, **Then** the feed shows a "Esperando eventos…" indicator rather than appearing broken.
4. **Given** the student ends their session while the docente is watching, **When** the session closes, **Then** the live feed updates to show "Sesión terminada" and stops receiving new events.
5. **Given** no students have active sessions for an activity, **When** the docente opens the live monitor, **Then** an empty state is shown ("No hay sesiones activas en este momento.").
6. **Given** the docente's connection to the live feed drops, **When** reconnection succeeds, **Then** any events missed during the gap are not retroactively shown, but the feed continues from the reconnection point.

---

### Edge Cases

- What happens when the WebSocket connection drops mid-session? → The panel shows a "Conexión perdida — reconectando…" indicator and attempts automatic reconnection up to 3 times before showing an error state. Session data already sent is preserved.
- What happens if the student closes the browser tab while a session is active? → The backend detects the disconnection and marks the session as ended. No data loss for events already received.
- What happens when a docente tries to access traces for a submission from a different course? → The API returns 403 and the viewer shows an access-denied error message.
- What happens if trace data fails to load? → An error message is shown with a retry button.
- What happens when a student starts a session on an activity they have already submitted and the edit window is closed? → The session prompt is not shown if `is_locked = true` on the submission.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST display a session start prompt on activity detail pages where `trace_enabled = true` and the student has no active session.
- **FR-002**: The session MUST NOT begin without explicit confirmation from the student.
- **FR-003**: The system MUST establish a real-time connection to the monitoring service upon session start and stream interaction events (keystroke, copy, paste, focus change) automatically.
- **FR-004**: A persistent status panel MUST be visible throughout an active session, showing session state and start time.
- **FR-005**: The system MUST cleanly close the real-time connection when the student navigates away from the activity page or explicitly ends the session.
- **FR-006**: The system MUST NOT capture or display any event content in a way that could be intercepted by other students — all data goes directly to the server.
- **FR-007**: Docente MUST be able to access a trace viewer from the submission detail page for any submission with a linked session.
- **FR-008**: The trace viewer MUST display events in chronological order with at minimum: timestamp, event type, and associated content/detail.
- **FR-009**: The trace viewer MUST be entirely read-only — no editing, deletion, or modification of events is possible from the UI.
- **FR-010**: The system MUST show an appropriate empty state when a session has no recorded events.
- **FR-011**: The system MUST handle real-time connection drops gracefully, with automatic reconnection and visible status to the student.
- **FR-012**: The student MUST be able to explicitly end a session via a "Terminar sesión" control, with confirmation.
- **FR-013**: The "Ver trazas" button on the submission detail page MUST only appear when a session ID is recorded on that submission (i.e., an active session was running when the file was uploaded).
- **FR-014**: When a student uploads a file and an active monitoring session exists, the system MUST automatically associate the current session ID with the submission record so the docente can access the trace from the submission detail page.
- **FR-015**: Docente MUST be able to navigate to a live sessions monitor page for an activity that lists all students with currently active sessions.
- **FR-016**: The live sessions list MUST show each active student's name, session start time, and event count, and MUST update automatically without page reload.
- **FR-017**: Docente MUST be able to open a live event feed for any active session, receiving new events in real time as the student generates them.
- **FR-018**: The live event feed MUST show a "Esperando eventos…" state when the session is active but no new events have arrived in the last 30 seconds.
- **FR-019**: The live event feed MUST display "Sesión terminada" and stop receiving new events when the student's session closes.
- **FR-020**: Co-docente MUST have the same access to the live session monitor as docente (read-only).

### Key Entities

- **Session**: A monitoring period linked to a student + activity combination. Has a start time, optional end time, a session ID, and connection state (connecting, active, ended, error).
- **TraceEvent**: A single recorded interaction event within a session. Has a type (keystroke, copy, paste, focus_in, focus_out), a timestamp, and optional content detail. Read-only after creation.
- **Submission ↔ Session link**: When a student uploads a file while a session is active, the submission record stores the session ID. If no session is active at upload time, the submission has no linked session and "Ver trazas" is hidden.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Session start-to-active state transition completes in under 2 seconds on a standard connection.
- **SC-002**: Trace events are displayed in the docente viewer within 1 second of the page load for sessions with up to 500 events.
- **SC-003**: The reconnection mechanism recovers a dropped session within 5 seconds of connection loss, with no visible interruption to the student's workflow beyond the status indicator.
- **SC-004**: 100% of trace events sent before a clean session close are visible in the docente viewer — no event loss on normal termination.
- **SC-005**: The session status panel correctly reflects the connection state (active / reconnecting / ended) at all times with no stale state shown to the student.
- **SC-006**: Live events appear in the docente's feed within 2 seconds of being generated by the student on a standard connection.
- **SC-007**: The live sessions list refreshes to show newly started or ended sessions within 5 seconds of the state change.

## Assumptions

- `trace_enabled` is a property already returned by the existing student activity detail API — no new student-facing API endpoint is needed to determine if monitoring applies.
- The real-time event connection uses the WebSocket URL pattern already defined in the constitution: `ws://[API_HOST]/sessions/:sid/ws?token=<jwt>`.
- The backend (M5) provides: POST to create a session, a WebSocket endpoint to stream events, and GET `/submissions/:sid/trace` to retrieve the recorded event list for the docente viewer.
- Session reconnection is the frontend's responsibility up to 3 attempts; after 3 failures the session is shown as errored but not automatically ended on the server.
- Co-docente has the same read-only access to trace logs as docente (same authorization scope as submission downloads from MW5).
- A student may have at most one active session per activity at a time; starting a new session while one is active is not possible.
- Trace events captured on the student side are sent immediately via WebSocket and are not buffered or retried locally — if the connection is lost, events during the gap are not recovered.
- Sessions are linked to the activity, not the submission. However, when a file is uploaded while a session is active, the session ID is captured in the submission record (see FR-014). A student can open a session without having submitted a file.
- The backend (M5) provides an endpoint to list all active sessions for an activity (e.g., GET `/cursos/:cid/actividades/:aid/sessions/active`) and a mechanism for the docente to receive live events — either by subscribing to the session's WebSocket or via a server-side push channel.
- The live session monitor (US4) is accessible from the docente's activity detail page alongside "Ver Entregas".
