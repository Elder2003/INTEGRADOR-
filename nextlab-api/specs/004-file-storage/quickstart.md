# Quickstart: M4 — File Storage

## Prerequisites

- M1 (Docker Compose), M2 (auth + JWT), and M3 (academic structure) must be fully operational.
- MinIO is already declared in `docker-compose.yml` from M1; verify it starts with the stack.
- A docente account and a student account (enrolled in a course with at least one activity) must exist from the M3 setup.

## 1. Add MinIO environment variables

Add to `.env`:

```env
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=nextlab
MINIO_USE_SSL=false
```

## 2. Rebuild and start the stack

```bash
docker compose up --build
```

The API startup creates the `nextlab` bucket if it does not exist. Check logs for:
```
[minio] bucket "nextlab" ready
```

## 3. Run the migration

```bash
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" up
```

Verify `actividades` now has `late_submission_allowed` and the `submissions` table exists.

## 4. Upload a file as a student

```bash
# Obtain a student JWT from M3 Google OAuth flow first, then:
curl -X POST http://localhost:3000/actividades/{aid}/submission \
  -H "Authorization: Bearer $STUDENT_TOKEN" \
  -F "file=@./sample_report.pdf"
```

Expected response `201`:
```json
{
  "id": "...",
  "original_filename": "sample_report.pdf",
  "submitted_at": "...",
  "is_late": false,
  "edit_window_expires_at": "...",
  "is_locked": false
}
```

## 5. List submissions as docente

```bash
curl http://localhost:3000/cursos/{cid}/actividades/{aid}/submissions \
  -H "Authorization: Bearer $DOCENTE_TOKEN"
```

Verify the student's submission appears with `is_late`, `original_filename`, and `file_size_bytes`.

## 6. Download the file as docente

```bash
curl -O -J http://localhost:3000/submissions/{sid}/download \
  -H "Authorization: Bearer $DOCENTE_TOKEN"
```

Verify the downloaded file matches the uploaded content.

## 7. Test the edit window

```bash
# Within 30 minutes of step 4 — replace the file:
curl -X PUT http://localhost:3000/actividades/{aid}/submission \
  -H "Authorization: Bearer $STUDENT_TOKEN" \
  -F "file=@./sample_report_v2.pdf"
# Expect 200 with updated filename and last_replaced_at set.

# Wait for the edit window to expire (or use an activity with a deadline 1 minute out).
# Then attempt another replace:
curl -X PUT http://localhost:3000/actividades/{aid}/submission \
  -H "Authorization: Bearer $STUDENT_TOKEN" \
  -F "file=@./sample_report_v3.pdf"
# Expect 403 — submission locked.
```

## 8. Test late submission blocking

```bash
# Create an activity with a past deadline and late_submission_allowed = false.
# As docente:
curl -X PUT http://localhost:3000/cursos/{cid}/actividades/{aid} \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"late_submission_allowed": false}'

# As student, attempt to submit after deadline:
curl -X POST http://localhost:3000/actividades/{aid}/submission \
  -H "Authorization: Bearer $STUDENT_TOKEN" \
  -F "file=@./report.pdf"
# Expect 422 with error "late_submissions_disabled".

# Verify audit log recorded the config change:
curl http://localhost:3000/cursos/{cid}/actividades/{aid}/config-changes \
  -H "Authorization: Bearer $DOCENTE_TOKEN"
```

## 9. Verify MinIO isolation

```bash
# Attempt to download another student's file:
curl http://localhost:3000/submissions/{other_student_sid}/download \
  -H "Authorization: Bearer $STUDENT_TOKEN"
# Expect 403.
```

## Verification Checklist

- [ ] Migration `000004` applied cleanly; `submissions` table exists with UNIQUE constraint
- [ ] File upload returns 201 with correct `edit_window_expires_at`
- [ ] Docente submission list returns all students' submissions
- [ ] File download streams the correct content (verify with `md5sum`)
- [ ] Replacement within edit window changes `original_filename` in the DB
- [ ] Replacement after window expiry returns 403
- [ ] Late submission rejected when `late_submission_allowed = false`
- [ ] Config change audit log records each toggle with timestamp and actor
- [ ] Cross-student download returns 403
- [ ] MinIO cleanup: no orphan objects after a failed DB insert (inject a DB error and verify)
