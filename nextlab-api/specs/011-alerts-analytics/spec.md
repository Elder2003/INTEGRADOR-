# Feature Specification: Alerts & Analytics

**Feature Branch**: `011-alerts-analytics`

**Created**: 2026-05-23

**Status**: Draft

**Input**: User description: "lets do milestone 11"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Real-Time Event Alerts via SSE (Priority: P1)

A docente is grading submissions when a new student submission arrives. Without refreshing the page, the docente immediately sees a notification appear — "Ana García submitted Stage 2" — and clicks it to jump to the submission. When format validation fails or the detection engine raises an integrity flag, another alert appears labeled clearly as advisory.

**Why this priority**: Real-time awareness is the core deliverable of M11. Docentes currently have no way to know when events happen in their activities without polling. SSE alerts eliminate the need for manual refresh and are the foundation that makes analytics feel alive.

**Independent Test**: Connect a docente SSE stream (`GET /alerts/stream`) → submit a file as a student in that docente's actividad → within 5 seconds the stream delivers an event with `type=submission_received` and correct `actividad_id` and `student_id`. Also verify that a second docente's SSE stream does NOT receive the event (role isolation).

**Acceptance Scenarios**:

1. **Given** a docente is connected to the alert stream, **When** a student in one of the docente's activities submits a file, **Then** the stream delivers a `submission_received` event within 5 seconds containing the activity ID, student ID, and submission timestamp.
2. **Given** a docente is connected to the alert stream, **When** the format validation engine marks a submission as non-compliant, **Then** the stream delivers a `format_violation` event with severity and affected fields, labelled as advisory.
3. **Given** a docente is connected to the alert stream, **When** the detection engine raises an integrity flag on a submission, **Then** the stream delivers a `integrity_flag` event with the flag type and advisory label — no grade or sanction is applied automatically.
4. **Given** a docente is connected to the alert stream, **When** a git analysis job completes, **Then** the stream delivers a `git_analysis_done` event with the student ID and overall flag count.
5. **Given** a docente is connected to the alert stream, **When** a student submits a stage after the deadline, **Then** the stream delivers a `late_submission` event with the stage ID and minutes-late value.
6. **Given** two docentes are connected to their respective streams, **When** an event occurs in Docente A's activity, **Then** only Docente A's stream receives the event; Docente B receives nothing.

---

### User Story 2 - Per-Activity Analytics Dashboard (Priority: P2)

A docente opens the analytics page for a specific activity and sees at a glance: how many students have submitted, how many are pending, how many submitted late, the grade distribution histogram, and a count of integrity flags raised. This data refreshes on demand without requiring a full page reload.

**Why this priority**: Activity-level analytics lets the docente make informed grading decisions and identify struggling students. It is independently useful without real-time alerts.

**Independent Test**: Create an activity with 3 students → 2 submit (1 late, 1 on time) → 1 flagged by detection → GET `/actividades/:id/analytics` → verify `submission_count=2`, `pending_count=1`, `late_count=1`, `flagged_count=1`, `grade_distribution` is empty (no grades yet) → grade one submission → repeat GET → verify `grade_distribution` shows 1 entry.

**Acceptance Scenarios**:

1. **Given** an activity with submissions and grades, **When** the docente requests analytics for that activity, **Then** the response includes submission count, pending count, late count, flagged count, and grade distribution.
2. **Given** an activity with no submissions, **When** the docente requests analytics, **Then** the response returns all counts as zero and an empty grade distribution.
3. **Given** a docente requests analytics for an activity belonging to a different docente, **Then** the system returns a 403 error (role isolation).
4. **Given** an activity with grades already assigned, **When** the docente requests analytics, **Then** grade distribution is grouped in ranges (0–49, 50–69, 70–89, 90–100) with counts per range.

---

### User Story 3 - Course-Level Aggregate Analytics (Priority: P3)

A docente views a summary across all activities in a course: total submissions received, total pending, overall integrity flag rate, and average grade per activity. This gives a bird's-eye view of course health without drilling into each activity.

**Why this priority**: Course-level analytics is a convenience aggregation on top of per-activity data. It is useful but not blocking — docentes can derive this manually from per-activity views.

**Independent Test**: Create a course with 2 activities, each with submissions and grades → GET `/cursos/:id/analytics` → verify the response contains an array of per-activity summaries plus a course-level rollup (`total_submissions`, `avg_grade`, `flag_rate`).

**Acceptance Scenarios**:

1. **Given** a course with multiple activities, **When** the docente requests course analytics, **Then** the response includes a per-activity breakdown and a course-level rollup with total submissions, average grade, and flag rate.
2. **Given** a docente requests analytics for a course they do not own, **Then** the system returns a 403 error.
3. **Given** a course with activities that have no submissions yet, **When** analytics are requested, **Then** the response returns zero counts and null averages where appropriate — no division-by-zero errors.

---

### Edge Cases

- What happens when the docente's SSE connection drops mid-stream? The server cleans up the connection silently; the client reconnects automatically via browser EventSource retry.
- What happens when an alert event is generated but no docente is connected? The event is stored in the `alerts` table with its sequential ID; on next connection the browser sends `Last-Event-ID` and the server replays all missed alerts automatically.
- What happens when an activity has submissions but all grades are null? Grade distribution returns all buckets as zero counts; average grade returns null.
- How does the system handle very large courses (500+ students)? Analytics queries are aggregated at the database level — no in-memory accumulation.
- What happens when two simultaneous events arrive for the same docente stream? Both are delivered in order; SSE is a sequential protocol.

## Clarifications

### Session 2026-05-23

- Q: When a submission event occurs, which component inserts the Alert row? → A: API handler writes `submission_received` alerts synchronously on submission; each async worker (detection, format, git) writes its own alert after completing its analysis.
- Q: How should the SSE endpoint authenticate the docente given EventSource cannot send headers? → A: A dedicated `POST /alerts/stream-token` endpoint issues a short-lived one-time token (valid 60 s); the SSE stream is opened as `GET /alerts/stream?token=<value>`.
- Q: How should new alert events be fanned out to connected SSE goroutines? → A: Redis pub/sub — alert writer publishes to `alerts:<docente_id>`; SSE goroutine subscribes and pushes each message to the client.
- Q: On SSE reconnect, should missed alerts be replayed server-side or fetched separately by the client? → A: Server-side replay using the standard SSE `Last-Event-ID` header — the server assigns a sequential numeric ID to every SSE event; on reconnect the browser sends `Last-Event-ID` automatically; the server replays all alerts with ID > last_seen_id then switches to live pub/sub delivery. No separate backlog endpoint needed.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a `POST /alerts/stream-token` endpoint that issues a short-lived one-time token (valid 60 seconds) for authenticated docentes. The SSE stream endpoint `GET /alerts/stream?token=<value>` MUST validate this token and reject expired or reused tokens with 401.
- **FR-002**: The SSE stream MUST deliver events only for activities belonging to the authenticated docente (Principle IV — role isolation).
- **FR-003**: Alert events MUST be generated for at least the following triggers: student file submission, format validation failure, detection engine flag, git analysis completion, and late stage submission. The submission API handler MUST write `submission_received` alerts synchronously; each async worker (detection, format, git) MUST write its own alert upon completing analysis.
- **FR-004**: Every alert MUST be persisted in the database with a sequential numeric ID used as the SSE event `id` field. On reconnect, the server MUST use the `Last-Event-ID` header sent automatically by the browser to replay all alerts with ID greater than the last received — eliminating any race condition between disconnect and reconnect.
- **FR-005**: All AI-generated alerts (format violations, integrity flags) MUST be labelled as advisory in the event payload — the system MUST NOT imply automated grade changes or sanctions (Principle III).
- **FR-006**: The system MUST provide a per-activity analytics endpoint (`GET /actividades/:id/analytics`) returning submission count, pending count, late count, flagged count, and grade distribution.
- **FR-007**: The system MUST provide a course-level analytics endpoint (`GET /cursos/:id/analytics`) returning a per-activity breakdown and a course-level rollup.
- **FR-008**: Analytics endpoints MUST enforce docente ownership — requests for activities or courses not belonging to the caller MUST return 403.
- **FR-009**: Docentes MUST be able to mark individual alerts as read (`PATCH /alerts/:id/read`); read status MUST be persisted.
- **FR-010**: Analytics data MUST be computed from live database state — no stale cache older than the current request.
- **FR-011**: The alert fan-out MUST use Redis pub/sub: the alert writer publishes a JSON payload to the `alerts:<docente_id>` channel; each connected SSE goroutine subscribes to its docente's channel and forwards received messages to the HTTP stream.

### Key Entities

- **Alert**: A persisted record of a system event relevant to a docente. Attributes: ID, docente_id, actividad_id, student_id (nullable), type (enum: submission_received, format_violation, integrity_flag, git_analysis_done, late_submission), payload (structured metadata), is_read, created_at.
- **ActivityAnalytics**: An on-demand aggregate view of an activity. Attributes: actividad_id, submission_count, pending_count, late_count, flagged_count, grade_distribution (bucketed), avg_grade (nullable).
- **CourseAnalytics**: An on-demand aggregate across all activities in a course. Attributes: curso_id, per_activity summaries, total_submissions, avg_grade (nullable), flag_rate (percentage).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Alert events are delivered to connected docentes within 5 seconds of the triggering action occurring.
- **SC-002**: A docente who reconnects after being offline receives all missed alerts generated during the disconnection period — zero alert loss.
- **SC-003**: Analytics endpoints respond in under 2 seconds for courses with up to 500 students and 20 activities.
- **SC-004**: 100% of AI-generated alert events in the SSE stream are labelled as advisory — no alert implies automatic grade change.
- **SC-005**: Role isolation is enforced for all alert and analytics endpoints — a docente can never see alerts or analytics for another docente's activities (verified by automated test).

## Assumptions

- The SSE stream uses the standard browser `EventSource` API; clients are expected to handle automatic reconnection via the built-in retry mechanism.
- Alert backlog is bounded to the last 30 days; older alerts are not fetched via the `since` parameter.
- Grade distribution is computed at request time from existing `submissions` / `stage_submissions` / `git_analyses` / `defense_appointments` grade columns — no separate grade table is introduced.
- The detection engine, format validator, git worker, and stage submission service already emit the data needed to generate alerts — M11 adds the alert-writing side only, not new analysis logic.
- Co-docentes with an explicit grant on a single activity receive alerts only for that activity.
- Students do not have access to any alert or analytics endpoint.
