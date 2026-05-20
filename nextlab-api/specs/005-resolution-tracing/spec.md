# Feature Specification: M5 — Resolution & Tracing

**Feature Branch**: `005-resolution-tracing`

**Created**: 2026-05-20

**Status**: Draft

**Input**: User description: "Lets do the milestone 5"

## Clarifications

### Session 2026-05-20

- Q: When the activity deadline passes — especially during a connection loss — what happens to the session and trace events? → A: The session closes automatically at the activity deadline. Events buffered client-side during a connection loss are accepted if they arrive before the deadline closes the session. Events arriving after the session is deadline-closed are rejected. Submission after the deadline is still accepted (marked as late per M4 rules). The valid trace window is strictly exam-open to deadline.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Student Opens an Exam Session (Priority: P1)

A student opens an activity of type `exam` or `defensa_oral` and is informed that an audit trace will be active during their session. The system creates a session record, establishes a real-time connection to receive events, and begins capturing from that moment. For other activity types (`practica`, `tarea`, `informe`) no session is created and the student works without trace capture.

**Why this priority**: Session creation and the consent notice are the prerequisite for everything else in this milestone. Without a live session, no events can be recorded and no jobs can be queued.

**Independent Test**: A student with an active enrollment opens an exam activity → system confirms a session was created, the student sees a notice that audit is active, and the real-time connection is confirmed open. Opening a `tarea` activity does NOT create a session.

**Acceptance Scenarios**:

1. **Given** a student is enrolled in a course with an active `exam` activity, **When** they open the exam, **Then** a new session record is created with status `active`, the student is informed that activity is being monitored, and a real-time connection is established.
2. **Given** a student opens a `tarea` activity, **When** the page loads, **Then** no session record is created and no trace connection is opened.
3. **Given** a student already has an active session for an activity, **When** they attempt to open the same activity again, **Then** the existing session is returned (no duplicate).

---

### User Story 2 — Browser Events Are Captured During the Session (Priority: P2)

While a session is active, browser events — tab focus loss, window visibility changes, clipboard copy/paste actions, and session heartbeats — are transmitted to the server and stored as immutable trace events. Each event receives a sequential number and a hash that chains it to the previous event, guaranteeing the integrity of the sequence.

**Why this priority**: Event capture is the core audit value of this milestone. An open session without events is worthless for later analysis.

**Independent Test**: With an active session open, trigger a tab-blur event (switch to another tab) → the server records a `tab_blur` event with correct timestamp, sequence number, and chained hash. Retrieve the event list and verify hash chain integrity holds.

**Acceptance Scenarios**:

1. **Given** an active exam session, **When** the student switches to another tab, **Then** a `tab_blur` event is appended with `received_at` set to server time, a monotonically increasing `sequence_num`, and a hash derived from the previous event's hash.
2. **Given** an active exam session, **When** the student copies text, **Then** a `clipboard_copy` event is appended with the same chaining rules; the copied content is NOT stored.
3. **Given** an active exam session, **When** a heartbeat interval fires (every 30 seconds), **Then** a `heartbeat` event is recorded to confirm session liveness.
4. **Given** an active session, **When** the student closes the browser without submitting, **Then** the session is marked `abandoned` and no further events are accepted.
5. **Given** a stale session where no heartbeat was received for more than 5 minutes, **When** the system checks, **Then** the session is marked `timed_out` automatically.

---

### User Story 3 — Session Completion Triggers Async Processing (Priority: P3)

When a student submits their work for a traced activity, the session is closed and a processing job is enqueued in the async queue. A background worker picks up the job and stores a basic acknowledgement record within the defined SLA. The worker infrastructure is established in this milestone; deeper analysis (heuristics, AI) is added in later milestones.

**Why this priority**: The worker base must be established before M8 (Detection Engine) and M9 (Format Validation) can add analysis steps. Without the queue and worker, the async pipeline cannot be built.

**Independent Test**: Submit an exam file → session transitions to `completed` → within 3 minutes a worker acknowledgement record exists for that session → the student's file upload response is received in under 2 seconds (confirming the worker does not block the HTTP response).

**Acceptance Scenarios**:

1. **Given** an active exam session and a pending file upload, **When** the student submits the file, **Then** the session is closed with status `completed`, a job is enqueued to the async queue within the same request, and the HTTP response is returned immediately without waiting for processing.
2. **Given** a queued job, **When** the background worker picks it up, **Then** an acknowledgement record is stored and the job is removed from the queue within 3 minutes of enqueueing.
3. **Given** the worker crashes mid-processing, **When** it restarts, **Then** unacknowledged jobs are re-processed without duplication (exactly-once or idempotent handling).

---

### User Story 4 — Docente Reviews Trace Evidence (Priority: P4)

A docente can retrieve the complete sequence of trace events for a specific student submission. They can also request an integrity verification that confirms the hash chain has not been tampered with. The data is read-only; no event can be modified or deleted.

**Why this priority**: Providing docente read access closes the audit loop. However, this is lower priority because it depends on events being stored (US2) and sessions being completed (US3).

**Independent Test**: After a student session with 15 events, the docente calls the trace endpoint → receives all 15 events in sequence order with correct hashes, and the integrity check endpoint returns `valid: true`. Attempting to fetch events for an activity the docente does not own returns 403.

**Acceptance Scenarios**:

1. **Given** a completed session with trace events, **When** the docente requests the trace for that submission, **Then** all events are returned in ascending sequence order, each with type, timestamp, sequence number, and hash.
2. **Given** a completed session, **When** the docente requests hash chain verification, **Then** the system recomputes the chain and returns `valid: true` if unmodified.
3. **Given** a docente who does not own the course, **When** they request trace data for a student in that course, **Then** they receive a 403 response.
4. **Given** a co-docente with an active grant for an activity, **When** they request the trace for a submission in that activity, **Then** they receive the trace data.

---

### Edge Cases

- What happens when the real-time connection drops mid-session before the deadline? The session stays `active`; events are buffered client-side and sent on reconnect. All events received by the server before the deadline are stored normally.
- What happens if the connection is restored after the deadline has already closed the session? The session is already `completed` or `deadline_closed`; any events arriving after closure are rejected — the trace window is immutably bounded by the deadline.
- What happens if two browser tabs open the same exam simultaneously? The second connection reuses the existing session; both connections feed into the same event stream.
- What happens if a heartbeat gap exceeds 5 minutes during an active session (before the deadline)? The session is automatically marked `timed_out`; no further events are accepted; a timeout event is appended as the final record.
- What if the async queue is temporarily unavailable when a session ends? The submission is still accepted (constitution Principle VII). The job enqueue failure is logged; a retry mechanism re-enqueues within 60 seconds.
- What if trace events arrive out of order due to network jitter? The server assigns `received_at` and `sequence_num` at the moment of receipt; client-side timestamps are stored as metadata only and do not affect ordering.
- What if a student submits after the deadline (late submission)? The file is accepted as late (M4 rules). The session was already closed at the deadline; the submission is linked to that closed session with no additional events recorded.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST create an exam session record when a student opens an activity of type `exam` or `defensa_oral`, and MUST NOT create a session for other activity types.
- **FR-002**: System MUST inform the student that activity monitoring is active before they can begin an `exam` or `defensa_oral` activity.
- **FR-003**: System MUST return the existing active session if one already exists for the student and activity (no duplicate sessions).
- **FR-004**: System MUST accept real-time browser event messages during an active session and append them as immutable trace events.
- **FR-005**: Each trace event MUST be assigned a server-side `received_at` timestamp and a monotonically increasing `sequence_num` scoped to the session.
- **FR-006**: Each trace event MUST include a hash computed from its own content combined with the hash of the immediately preceding event, creating a verifiable chain.
- **FR-007**: System MUST accept the following event types: `tab_blur`, `tab_focus`, `window_blur`, `window_focus`, `clipboard_copy`, `clipboard_paste`, `heartbeat`, `session_open`, `session_close`.
- **FR-008**: System MUST NOT store the content of clipboard events — only the occurrence and timestamp.
- **FR-009**: System MUST mark a session as `timed_out` when no heartbeat has been received for more than 5 minutes during an active session before the activity deadline.
- **FR-010**: System MUST mark a session as `abandoned` when the real-time connection closes without a formal session-close event and before the deadline.
- **FR-011**: System MUST automatically close the session when the activity deadline is reached, regardless of whether the student has submitted. The session status is set to `completed` if a submission exists at that moment, or `deadline_closed` if it does not.
- **FR-011b**: System MUST also close the session as `completed` when the student submits their file before the deadline.
- **FR-011c**: Any trace events received after the session is closed (by deadline or submission) MUST be rejected; no events may be appended to a closed session.
- **FR-012**: System MUST enqueue an async processing job immediately upon session closure when a submission is linked; this enqueue MUST NOT block the file upload HTTP response.
- **FR-013**: The async worker MUST consume jobs from the queue, store an acknowledgement record, and acknowledge the job within 3 minutes of enqueueing.
- **FR-014**: The async worker MUST handle restarts gracefully — previously unacknowledged jobs MUST be reprocessed without losing data.
- **FR-015**: System MUST expose an endpoint for docentes to retrieve all trace events for a specific submission, ordered by sequence number.
- **FR-016**: System MUST expose an endpoint that recomputes and returns the hash chain integrity status (`valid` / `invalid`) for a given session.
- **FR-017**: Trace events MUST be read-only after creation; no `UPDATE` or `DELETE` operations are permitted on trace event records.
- **FR-018**: Access to trace data MUST be restricted to: the owning docente, a co-docente with an active grant for the activity, and the system admin.
- **FR-019**: The student MUST NOT have access to their own trace event records or analysis results.
- **FR-020**: Trace capture applies only to `exam` and `defensa_oral` activity types; it MUST NOT activate for `practica`, `tarea`, or `informe`.

### Key Entities

- **ExamSession**: Represents one student's work session for a traced activity. Links to student, activity, submission. Carries status (`active`, `completed`, `abandoned`, `timed_out`, `deadline_closed`), start time, end time, and the hash of the last event (used to chain the next). The session is strictly bounded by the activity deadline — no events are accepted after the session closes.
- **TraceEvent**: An individual immutable event within a session. Carries session reference, event type, `sequence_num`, `received_at` (server time), optional metadata (e.g., window title — no sensitive content), and `event_hash` (chained).
- **WorkerJob** (queue entry): A job record linking to a completed session, carrying enqueue time and status (`pending`, `processing`, `done`, `failed`). Stored in the async queue and mirrored in the database for durability.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Students receive a session-active confirmation within 1 second of opening a traced activity.
- **SC-002**: 100% of trace events received during an active session are stored; no events are silently dropped.
- **SC-003**: File upload responses for exam submissions are returned within 2 seconds regardless of worker processing load, confirming asynchronous isolation.
- **SC-004**: Worker acknowledgement records are created within 3 minutes for 100% of completed sessions under normal operating conditions.
- **SC-005**: The hash chain integrity check returns `valid: true` for any unmodified session trace.
- **SC-006**: Docentes can retrieve the full trace for any submission within their courses; cross-course access is rejected 100% of the time.

## Assumptions

- The student's browser must support WebSockets (all modern browsers do); no fallback to polling is required for this milestone.
- Trace activation is the client's responsibility to initiate on page load for `exam`/`defensa_oral` activities; the API validates the session state on every event.
- The Redis Streams instance is the same Redis 7 deployment already used for OAuth state tokens (M2); no new infrastructure is added.
- The async worker runs as a separate process (cmd/worker) within the same Docker Compose stack.
- Client-side event buffering (for reconnection) is handled by the frontend; the backend only receives and stores events that arrive.
- Heartbeat intervals are driven by the client (every 30 seconds); the server only checks the last-received timestamp.
- Hash algorithm is SHA-256; the first event in a session uses the session ID as the seed for the chain.
- No camera, microphone, or local filesystem access is used — only browser-level JavaScript events.
- The oral defense (`defensa_oral`) session model is identical to the exam session model for tracing purposes.
