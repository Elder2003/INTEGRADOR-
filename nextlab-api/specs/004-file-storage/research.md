# Research: M4 — File Storage

## Decision Log

---

### 1. MinIO Go Client Library

**Decision**: `minio/minio-go/v7`

**Rationale**: Official MinIO SDK. Directly mirrors the MinIO/S3 API with idiomatic Go wrappers (`PutObject`, `GetObject`, `RemoveObject`, `PresignedGetObject`). Actively maintained by the MinIO team. `aws/aws-sdk-go-v2` would also work but is significantly more complex and targets AWS — not worth the extra surface area for self-hosted MinIO.

**Alternatives considered**:
- `aws/aws-sdk-go-v2` (S3-compatible, heavier, AWS-centric)
- Direct HTTP calls to MinIO (no SDK — maintenance burden, not worth it)

---

### 2. Upload Flow: Client → API → MinIO

**Decision**: Student uploads multipart form to the API; API streams the body to MinIO via `PutObject`. The API controls the storage key; MinIO is never addressed directly by the client.

**Rationale**: FR-016 requires all file access to be mediated through the API. Presigned PUT URLs would let the client write directly to MinIO — while time-limited and signed, this bypasses the API's ability to enforce the 50 MB limit, validate enrollment, and ensure atomicity (FR-014). API-proxied upload is slightly heavier but gives full control at the scale of this project (~1500 students, 50 MB max).

**Fiber multipart handling**: `c.FormFile("file")` reads the file part into a `*multipart.FileHeader`. Size can be checked against the 50 MB limit before streaming to MinIO. The file is opened as a stream: `file.Open()` → pass reader to `minio.PutObject(ctx, bucket, key, reader, size, opts)`.

**Alternatives considered**:
- Presigned PUT URLs (client uploads directly) — rejected because it bypasses API-layer validation and mediation
- Chunked / streaming upload via custom protocol — overkill for 50 MB max; standard multipart is sufficient

---

### 3. Download Flow: API-Proxied Streaming

**Decision**: API validates authorization, calls `minio.GetObject(ctx, bucket, key, opts)`, and streams the response body directly to the client using `c.SendStream(reader, size)` with appropriate `Content-Disposition` and `Content-Type` headers.

**Rationale**: Fully satisfies FR-016 (mediated access). No direct or presigned URLs are ever sent to the client. At Unifranz's scale, streaming 50 MB files through the Go API server is well within reason. Fiber's `SendStream` avoids buffering the full file in memory.

**Alternatives considered**:
- Short-lived presigned GET URL (API validates auth, returns temporary MinIO URL for client to fetch directly) — lighter on server but the URL leaks the MinIO endpoint and could be forwarded. Rejected for cleaner security posture.

---

### 4. Atomicity: File-First, Record-Second

**Decision**: Upload to MinIO first. If `PutObject` succeeds, insert the submission record in PostgreSQL. If the DB insert fails (unique constraint violation, DB down), call `minio.RemoveObject` to clean up the orphaned file. Log the cleanup attempt; if RemoveObject also fails, record an orphan-cleanup-needed log entry for manual resolution.

**Rationale**: FR-014 requires that no submission record is created unless the file is confirmed stored. The reverse order (DB first, then MinIO) would leave an orphan record pointing to a non-existent file. Best-effort cleanup of MinIO on DB failure is the correct approach.

**Alternatives considered**:
- Two-phase commit across DB + MinIO — not feasible; MinIO has no distributed transaction protocol
- DB-first with rollback if MinIO fails — rejected because a rollback gap could leave a phantom record if the process crashes between DB commit and MinIO upload

---

### 5. Concurrent Upload Prevention

**Decision**: PostgreSQL `UNIQUE (student_id, actividad_id)` constraint on the `submissions` table prevents duplicate first-time submissions. When a race produces two simultaneous first uploads for the same student+activity, the second DB insert will raise a unique constraint violation; the Go handler catches this, deletes the MinIO object it just uploaded, and returns HTTP 409.

**Rationale**: No Redis lock or advisory lock is needed. The DB constraint is atomic and reliable. The MinIO cleanup on 409 ensures no orphan objects. For the replacement case (student replaces within edit window), the existing submission row is UPDATEd — not inserted — so no unique constraint is involved. The service wraps this in a DB transaction to avoid partial updates.

**Alternatives considered**:
- Redis lock `SET "upload_lock:{student_id}:{actividad_id}" EX 30 NX` — would work but adds Redis as a coordination point for a case already handled by the DB constraint. Simpler to rely on the DB.

---

### 6. Edit Window Lockedness: Derived, Not Stored

**Decision**: `is_locked` is NOT stored as a column. It is computed at API response time: `is_locked = (edit_window_expires_at IS NULL) OR (NOW() > edit_window_expires_at)`. The service layer enforces this check before allowing any replacement or deletion.

**Rationale**: Storing `is_locked` would require a background job to flip it when the window expires, adding operational complexity. Computing it on read is trivially cheap. `edit_window_expires_at` IS stored (it's part of the submission record per FR-006) and is immutable after creation — no staleness risk.

**Alternatives considered**:
- PostgreSQL generated column for `is_locked` — not feasible, generated columns cannot reference `now()`
- PostgreSQL trigger to flip `is_locked` — complex, requires a background pg_cron or pg_agent setup not in the locked stack

---

### 7. File Replacement Within Edit Window

**Decision**: When a student replaces a file within the edit window:
1. Upload new file to MinIO with a fresh UUID key.
2. In a DB transaction: `UPDATE submissions SET storage_key=..., original_filename=..., file_size_bytes=..., mime_type=..., submitted_at=now(), is_late=(now() > deadline), last_replaced_at=now() WHERE student_id=... AND actividad_id=...`.
3. After the transaction commits, delete the old MinIO object.

The `edit_window_expires_at` is NOT updated. The DB transaction ensures the record is consistent before the old file is removed.

**Rationale**: Keeping the edit window fixed from first-upload time prevents abuse (student could keep resetting the clock by replacing every 29 minutes). Updating `is_late` on replacement correctly reflects whether the new active file is before or after the deadline.

---

### 8. Storage Key Structure

**Decision**: `submissions/{activity_id}/{student_id}/{uuid}{ext}`

- `activity_id` and `student_id` provide namespace isolation and make it easy to identify orphan objects.
- `uuid` (random v4) ensures no collisions on replacement.
- `ext` is derived from the original filename (`filepath.Ext(header.Filename)`).
- The key is stored in `submissions.storage_key` and is never exposed to the client.

---

### 9. MinIO Bucket Initialization

**Decision**: A single bucket named `nextlab` (or configured via `MINIO_BUCKET` env var). Bucket creation is idempotent at API startup: `minioClient.MakeBucket(ctx, bucket, opts)` with `IfNotExists` semantics. Bucket is private (no public read policy).

**Rationale**: Matches the single-bucket pattern common for small platforms. Objects are namespaced by prefix (`submissions/`), leaving room for future prefixes (`rag/`, `resources/`) without a new bucket.

---

### 10. Migration Strategy

**Decision**: Single migration `000004_file_storage`:
- `ALTER TABLE actividades ADD COLUMN late_submission_allowed BOOLEAN NOT NULL DEFAULT true` — extends the M3 table
- `CREATE TABLE actividad_config_changes` — immutable audit log for config flag changes
- `CREATE TABLE submissions` — core new table with `UNIQUE(student_id, actividad_id)`

No changes to other M3 tables. The `has_submissions` column on `actividades` (set by M3 to `false`) is flipped to `true` by M4 on the first successful submission for an activity.

---

### 11. Module Patterns

**Decision**: New `internal/submission/` module following the identical `RegisterRoutes(app, db, minioClient, cfg)` pattern from M2/M3. The academic module (`internal/academic/`) is extended minimally — the existing `handler_actividad.go` is updated to accept `late_submission_allowed` in the `PUT /cursos/:cid/actividades/:aid` request body, and `service.go` records the audit log entry when the value changes.

**Rationale**: Keeps modules single-purpose. Submission logic (upload, replace, delete, download) lives in `internal/submission/`. Activity config changes are a property of the academic module and stay there.

---

## Config additions required

| Env var | Purpose | Required |
|---|---|---|
| `MINIO_ENDPOINT` | MinIO server address (e.g. `minio:9000`) | Yes |
| `MINIO_ACCESS_KEY` | MinIO root user / access key | Yes |
| `MINIO_SECRET_KEY` | MinIO root password / secret key | Yes |
| `MINIO_BUCKET` | Bucket name (default: `nextlab`) | No (defaults) |
| `MINIO_USE_SSL` | Enable TLS for MinIO connection (default: `false` for internal Docker network) | No (defaults) |
