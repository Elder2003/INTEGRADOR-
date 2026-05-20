# Feature Specification: M4 — File Storage

**Feature Branch**: `004-file-storage`

**Created**: 2026-05-19

**Status**: Draft

**Input**: User description: "Lets do the milestone 4"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Student Submits a File for an Activity (Priority: P1)

A student uploads a file (PDF, Word document, or zip archive) for an assigned activity. The system records the submission immediately with an exact timestamp and stores the file. If the submission arrives after the activity deadline, it is automatically flagged as late — but the upload completes regardless. The student sees a confirmation that their submission was received.

**Why this priority**: File submission is the central transaction of the academic workflow. Every other M4 story depends on at least one submission existing. Late marking and versioning are meaningless without a first upload.

**Independent Test**: A student uploads a PDF for an activity with a past deadline. The file is stored. The submission record shows the timestamp and is marked as late. A second student submits before the deadline and their record is NOT marked as late.

**Acceptance Scenarios**:

1. **Given** an authenticated student enrolled in a course with an open activity, **When** they upload a file within the deadline, **Then** the submission is recorded with the exact upload timestamp and `is_late = false`.
2. **Given** an authenticated student, an activity whose deadline has passed, and `late_submission_allowed = true` on that activity, **When** they upload a file, **Then** the submission is accepted, recorded with the exact upload timestamp, and `is_late = true`.
3. **Given** an authenticated student, an activity whose deadline has passed, and `late_submission_allowed = false` on that activity, **When** they attempt to upload a file, **Then** the upload is rejected with a clear error message explaining that late submissions are not accepted for this activity.
4. **Given** an authenticated student, **When** their upload completes, **Then** they receive a confirmation containing the submission ID, timestamp, and late status.
5. **Given** a student who is not enrolled in a course, **When** they attempt to submit a file for an activity in that course, **Then** the upload is rejected with an authorization error.

---

### User Story 2 — Student Replaces or Removes Their Submission Within the Edit Window (Priority: P2)

After submitting on time, a student notices a mistake and wants to fix it. They have a 30-minute window from the moment of their first successful upload to either replace the file with a corrected version or delete the submission and start fresh. Once the window closes — or if the submission was already late — the submission is locked and no further changes are possible.

**Why this priority**: A fixed editing window balances academic integrity (submissions become immutable records) with practical fairness (students shouldn't lose points for minor upload errors). It also prevents concurrent upload races by making only one active submission per student per activity valid at any time.

**Independent Test**: A student submits on time. Within 30 minutes they upload a replacement file — the new file becomes their active submission. They then try to replace again after the window closes and are rejected. A second student submits late and immediately receives a locked submission with no editing window.

**Acceptance Scenarios**:

1. **Given** an on-time submission and an edit window still open, **When** the student uploads a replacement file, **Then** the new file becomes the active submission, the previous file is replaced, and the edit window timer continues from the original submission time.
2. **Given** an on-time submission and an edit window still open, **When** the student deletes their submission, **Then** the submission is removed and the student can start fresh with a new upload (which starts a new 30-minute window).
3. **Given** an on-time submission whose 30-minute edit window has expired, **When** the student attempts to replace or delete, **Then** the request is rejected with a clear message that the submission is locked.
4. **Given** a late submission (marked `is_late = true`), **When** the student attempts to replace or delete immediately after upload, **Then** the request is rejected — late submissions have no editing window and are locked upon receipt.
5. **Given** two simultaneous upload requests for the same student and activity, **When** one is already being processed, **Then** the second request is rejected until the first completes.

---

### User Story 3 — Docente Reviews and Downloads Submissions (Priority: P3)

A docente opens an activity and sees the full list of student submissions: who submitted, when, whether it was late, and how many versions each student has uploaded. The docente can download any individual submission file to review its content.

**Why this priority**: Without download access for docentes, the submission pipeline produces evidence that can never be used. This story closes the loop between student upload and docente review, which is the primary value of M4.

**Independent Test**: Three students submit files for the same activity (one late, one on time, one with two versions). The docente's submission list shows all four submission records. The docente downloads each file and receives the correct content.

**Acceptance Scenarios**:

1. **Given** an authenticated docente who owns a course, **When** they request the submission list for one of their activities, **Then** they see all submissions with student name, submission timestamp, late flag, version number, filename, and file size.
2. **Given** a submission record, **When** the docente requests its download URL or file stream, **Then** they receive the exact file that was uploaded by the student.
3. **Given** an activity in a course owned by another docente, **When** a docente attempts to list or download its submissions, **Then** the request is denied.
4. **Given** an activity with no submissions, **When** the docente requests the submission list, **Then** they receive an empty list — not an error.

---

### User Story 4 — Student Views and Re-Downloads Their Own Submissions (Priority: P4)

A student wants to verify what they submitted. They can see a list of all their own submission versions for any activity they are enrolled in and download any of their prior files. They cannot see other students' files.

**Why this priority**: Student access to their own submission history is a basic transparency requirement that reduces support requests and builds trust in the system. It has no security risk as long as cross-student access is properly blocked.

**Independent Test**: Student A submits on time and replaces their file within the window. Student B submits once and their window expires. Each student sees only their own active submission. Neither student can access the other's file.

**Acceptance Scenarios**:

1. **Given** an authenticated student, **When** they request their submission list for an activity, **Then** only their own submissions are returned.
2. **Given** a student's own submission, **When** they request its download, **Then** they receive the file they uploaded.
3. **Given** another student's submission ID, **When** a student attempts to download it, **Then** the request is denied.

---

### Edge Cases

- What happens when a student uploads a file with the same filename as a prior version? (Both are kept — MinIO objects use unique keys; the filename in the record is informational only.)
- What happens if a file upload is interrupted mid-transfer? (The incomplete object must not produce a submission record. The operation is atomic: file stored first, record created only on confirmed storage success.)
- What happens if MinIO is temporarily unavailable during an upload? (The system returns a 503 error; no partial submission record is created. The student can retry.)
- What is the maximum allowed file size? (Up to 50 MB per file, rejected with a clear error if exceeded — this is a precondition check before transfer begins.)
- What file types are accepted? (Any file type is accepted at the storage layer. Format validation is a separate concern handled in M9; M4 does not block on content type.)
- What happens if the activity is not of a submittable type? (Only `práctica`, `tarea`, `informe`, `examen`, and `defensa_oral` accept file submissions. A submission attempt against an unsupported type is rejected.)
- What happens if a student submits after the deadline when late submissions are disabled? (`late_submission_allowed = false` + past deadline → upload rejected with a clear error. The student is informed that late submissions are not permitted for this activity.)
- What happens if a student submits after the deadline when late submissions are enabled? (`late_submission_allowed = true` + past deadline → upload accepted and `is_late = true` is set. The docente decides the grade consequence.)
- What happens if the docente changes `late_submission_allowed` while some students have already submitted late? (Previously accepted late submissions remain in the record unchanged. The flag change only affects future upload attempts.)
- What happens when a student submits 5 minutes before the deadline and `late_submission_allowed = false`? (Their edit window closes at the deadline — 5 minutes after upload — not at 30 minutes. The confirmation message must clearly show the actual expiry time.)
- What happens when a student replaces their file after the deadline within the extended 30-minute window (`late_submission_allowed = true`)? (The replacement is accepted. The submission's `is_late` flag is updated to `true` because the active file's timestamp is now after the deadline.)
- What happens if a student submits after the course is deactivated? (Course deactivation always blocks new submissions regardless of the `late_submission_allowed` flag.)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A student MUST be able to upload a file for any activity in a course in which they are enrolled.
- **FR-002**: A submission received before or at the activity deadline MUST always be accepted. A submission received after the deadline MUST be accepted only when `late_submission_allowed = true` on that activity; when `late_submission_allowed = false`, the upload MUST be rejected with a clear error message. Within-deadline submissions are accepted unconditionally.
- **FR-003**: The system MUST automatically set `is_late = true` on any submission whose recorded timestamp is after the activity's deadline, and `is_late = false` otherwise. This flag MUST be set at the moment of receipt and MUST NOT be alterable after the fact.
- **FR-004**: The submission timestamp MUST be recorded at the moment the file is confirmed stored — not at the moment the upload is initiated.
- **FR-005**: Only one active submission is permitted per student per activity at any time. A student MUST NOT be able to start a new upload while one is already in progress for the same activity.
- **FR-005b**: On-time submissions (where the original upload is before or at the deadline) MUST be given an editing window starting from the moment the file is confirmed stored. The window duration depends on `late_submission_allowed`:
  - When `late_submission_allowed = false`: the window closes at `min(upload_time + 30 minutes, activity deadline)`. A student who submits 5 minutes before the deadline has only 5 minutes to edit.
  - When `late_submission_allowed = true`: the window always runs the full 30 minutes from upload, even if it crosses the deadline.
  The edit window expiry timestamp MUST be stored on the submission record and returned to the student in the upload confirmation.
- **FR-005c**: Late submissions (`is_late = true`) are locked immediately upon receipt. No editing window is granted.
- **FR-005d**: Once an on-time submission's editing window has expired, the submission is locked. Any replacement or deletion attempt MUST be rejected with a clear error message.
- **FR-005e**: When a student replaces their submission within the edit window, the previous file is superseded. The new file and its storage timestamp become the active submission. The edit window expiry is NOT reset — it was calculated at first upload and does not change. If the replacement is uploaded after the deadline (possible only when `late_submission_allowed = true` and the window extends past the deadline), the submission's `is_late` flag MUST be updated to `true` to reflect the actual timestamp of the active file.
- **FR-005f**: When a student deletes their submission within the edit window, the active submission is removed. The student may then upload a new file, which starts a fresh 30-minute window.
- **FR-006**: Every submission record MUST include: student reference, activity reference, storage object key, original filename, file size in bytes, MIME type (as reported by the client), submission timestamp, `is_late` flag, edit window expiry timestamp (null for late submissions), and locked status.
- **FR-007**: A docente MUST be able to retrieve the full list of submissions for any activity in their own courses, including student identity, timestamp, late flag, version number, filename, and file size.
- **FR-008**: A docente MUST be able to download any individual submission file from any of their own activities.
- **FR-009**: A student MUST be able to retrieve the list of their own submissions for any activity in a course they are enrolled in.
- **FR-010**: A student MUST be able to download any of their own submitted files.
- **FR-011**: A student MUST NOT be able to access, list, or download another student's submission.
- **FR-012**: A docente MUST NOT be able to access or download submissions belonging to an activity in another docente's course.
- **FR-013**: File uploads MUST be rejected if the file exceeds 50 MB, with a clear error returned before the transfer completes.
- **FR-014**: A submission record MUST NOT be created unless the file has been successfully persisted to storage. The operation MUST be atomic: file first, record only on confirmed success.
- **FR-015**: If the storage backend is unavailable during an upload, the system MUST return an error to the client and leave no partial submission record in the database.
- **FR-016**: Submission file access MUST be mediated through the API — direct, unauthenticated storage URLs MUST NOT be exposed to clients.
- **FR-017**: Each activity MUST have a `late_submission_allowed` boolean flag, defaulting to `true`. A docente MUST be able to toggle this flag on any of their activities at any time, including after the deadline has passed.
- **FR-018**: Every change to an activity's `late_submission_allowed` flag MUST be recorded in an immutable audit log entry containing: activity reference, previous value, new value, changed-by (docente reference), and change timestamp. This log MUST be accessible to the owning docente and to admins.
- **FR-019**: When a docente re-enables `late_submission_allowed` (false → true), the change applies only to future upload attempts — submissions that were previously rejected are NOT retroactively created or recovered.
- **FR-020**: A co-docente with an active grant for a specific activity MUST be able to retrieve the full submission list for that activity, including student identity, submission timestamp, late flag, version number, filename, and file size.
- **FR-021**: A co-docente with an active grant for a specific activity MUST be able to download any individual submission file for that activity.
- **FR-022**: A co-docente MUST NOT be able to list or download submissions belonging to any activity for which they do not hold an active grant, including other activities in the same course.

### Key Entities

- **Submission**: The active file upload by a student for a specific activity. Only one submission exists per student per activity at any point. Key attributes: student reference, activity reference, storage object key (internal), original filename, file size (bytes), MIME type, submitted_at timestamp, `is_late` flag, `edit_window_expires_at` (null when late), `is_locked` (boolean, true once window expires or submission is late), created_at, last_replaced_at (null if never replaced within the window).
- **Activity** *(existing from M3)*: Extended in M4 with a `late_submission_allowed` flag (boolean, default true). This flag is docente-configurable at any time; every change is recorded in the Activity Config Change Log.
- **Activity Config Change Log**: Immutable record of each change to an activity's `late_submission_allowed` flag. Key attributes: activity reference, previous value, new value, changed_by (docente reference), changed_at timestamp.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A student can complete a file upload and receive a submission confirmation in under 10 seconds for files up to 10 MB on a standard connection.
- **SC-002**: Every submission in the database has a non-null timestamp that matches the actual moment of storage confirmation — zero submissions with retroactively altered timestamps.
- **SC-003**: After a replacement within the edit window, the active submission reflects the new file. No replacement is possible once the window has expired — zero successful replacements outside the window across all test scenarios.
- **SC-004**: Every submission's `is_late` flag correctly reflects the timestamp of the active file versus the deadline. Edit window expiry timestamps are computed as `min(upload_time + 30 min, deadline)` when `late_submission_allowed = false`, and `upload_time + 30 min` when `late_submission_allowed = true` — zero incorrect window expiry values or late flags across all test scenarios.
- **SC-005**: No cross-student or cross-docente file access succeeds during security testing — zero unauthorized download events.
- **SC-006**: Zero submissions are permanently lost due to a mid-upload failure; partial records are never left in the database after a storage error.
- **SC-007**: Every change to an activity's `late_submission_allowed` flag produces exactly one audit log entry — zero unlogged changes across all test scenarios.

## Clarifications

### Session 2026-05-19

- Q: Can a submission be rejected after the deadline, and who controls it? → A: Activities have a docente-configurable `late_submission_allowed` flag (default: true). When true, late submissions are accepted and marked as such; when false, post-deadline uploads are rejected with a clear error. Every change to this flag is recorded in an immutable audit log with timestamp, previous value, new value, and the docente who made the change.
- Q: Can a co-docente with an activity grant download submission files? → A: Yes. A co-docente with an active grant can list submissions, download files, and see late status and version history — scoped exclusively to their granted activity. Grade assignment is deferred to the milestone that handles grade workflows.
- Q: How are concurrent uploads and versioning handled? → A: Only one active submission per student per activity at any time; concurrent uploads are rejected. On-time submissions receive a 30-minute editing window during which the student can replace the file or delete and re-upload; late submissions are locked immediately upon receipt. After the window expires, the submission is immutable. There is no multi-version history — only one active submission exists per student per activity.
- Q: Does the edit window always run 30 minutes, even when close to the deadline? → A: No. When `late_submission_allowed = false`, the window closes at `min(upload_time + 30 min, deadline)` — a student submitting 5 minutes before the deadline gets 5 minutes to edit. When `late_submission_allowed = true`, the full 30 minutes always runs; a replacement uploaded after the deadline within that window sets `is_late = true` on the final submission.

## Assumptions

- The MinIO instance is running and reachable from the API server at the time of M4 development (configured via Docker Compose as established in M1).
- Authentication and role middleware from M2 are operational. All M4 endpoints enforce role-based access using the existing JWT middleware.
- The academic structure from M3 (courses, activities, student enrollment) is in place and working. M4 adds submission endpoints on top of existing M3 entities.
- File format validation (PDF/Word structure checks, compliance reports) is out of scope for M4 — that is handled in M9. M4 stores all file types without content inspection.
- The `is_late` flag is computed entirely from the server-side clock at storage-confirmation time versus the activity's stored deadline. The client clock is never trusted.
- Storage object keys are generated server-side (e.g., `submissions/{activity_id}/{student_id}/{version}/{uuid}.{ext}`) and are never exposed directly to the client.
- Files are served to authenticated clients via signed or proxied URLs with short expiry — never via permanent public URLs.
- The 50 MB per-file size limit is a reasonable default for academic document submissions at this stage; it can be adjusted by an admin in a future milestone.
- Students may submit to activities of type `práctica`, `tarea`, `informe`, `examen`, and `defensa_oral`. Submission behavior for oral defenses is limited to file attachments (e.g., slide decks); real-time defense recording is out of scope for M4.
- Co-docentes with an active grant can list submissions, download files, and view late status and version history for their granted activity only. They cannot upload submissions or manage activity settings. Grade assignment by co-docentes is deferred to the milestone that handles grade workflows.
