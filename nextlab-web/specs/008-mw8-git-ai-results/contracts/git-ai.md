# API Contracts: MW8 — Git & AI Results

All calls go through `src/lib/api.ts` (axios instance with auth interceptor).
Module: `src/api/gitAnalysis.ts`

---

## Git Submission

### POST /actividades/:aid/git-submission
Submit a git repository URL for an activity.

**Request body**:
```json
{ "repo_url": "https://github.com/student/repo" }
```

**Success response** (200 / 201): `SubmissionResponse` with `repo_url` populated.

**Error responses**:
- 400 `invalid_url` — URL failed backend validation
- 403 `submission_locked` — submission window closed
- 409 `already_submitted` — use PATCH to update

---

### PATCH /actividades/:aid/git-submission
Update an existing git repository URL (within edit window).

**Request body**:
```json
{ "repo_url": "https://github.com/student/new-repo" }
```

**Success response** (200): `SubmissionResponse` with updated `repo_url`.
Backend automatically invalidates all three analysis reports.

**Error responses**:
- 400 `invalid_url`
- 403 `edit_window_expired` — window closed, update not allowed
- 404 `not_found` — no git submission exists yet; use POST

---

## Analysis Report Endpoints

### GET /submissions/:sid/git-analysis
Retrieve the git analysis report for a submission.

**Success response** (200): `GitAnalysisReport`
```json
{
  "status": "done",
  "commit_count": 23,
  "authors": ["alice@example.com"],
  "date_from": "2026-03-01T10:00:00Z",
  "date_to": "2026-05-15T18:30:00Z",
  "files": [
    { "path": "src/main.py", "additions": 150, "deletions": 20 }
  ]
}
```

**Pending response** (200):
```json
{ "status": "pending" }
```

**Error response** (200 — status field, not HTTP error):
```json
{ "status": "error", "error_message": "Repository not accessible" }
```

---

### GET /submissions/:sid/ai-report
Retrieve the AI-generated analysis report.

**Success response** (200): `AIReport`
```json
{
  "status": "done",
  "sections": [
    { "title": "Resumen", "content": "El código demuestra..." },
    { "title": "Hallazgos", "content": "Se identificaron..." },
    { "title": "Nivel de riesgo", "content": "Bajo" }
  ]
}
```

**Pending / Error**: same pattern as git-analysis.

---

### GET /submissions/:sid/detection-report
Retrieve the similarity/detection report.

**Success response** (200): `DetectionReport`
```json
{
  "status": "done",
  "similarity_score": 12,
  "flagged_segments": [
    {
      "source": "github.com/other/repo",
      "match_percentage": 85,
      "content": "def calculate_average(..."
    }
  ]
}
```

**Pending / Error**: same pattern.

---

## Error handling

All functions use the local `toApiError` pattern (matching `src/api/grading.ts`):

```ts
function toApiError(err: unknown): never {
  const axiosErr = err as AxiosError<{ error: string; message: string }>
  if (axiosErr.response) {
    const { error, message } = axiosErr.response.data ?? {}
    throw new ApiError(message ?? 'An error occurred', error ?? 'unknown_error')
  }
  if (axiosErr.request) {
    throw new ApiError('Unable to reach the server. Check your connection.', 'network_error')
  }
  throw new ApiError('An unexpected error occurred', 'unknown_error')
}
```
