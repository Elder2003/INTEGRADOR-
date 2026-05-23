# Quickstart: Alerts & Analytics (M11)

Three independent test flows — one per user story.

---

## Story 1: Real-Time SSE Alert Stream

**Prerequisites**: A docente account, a student account enrolled in an `investigacion` activity.

```bash
BASE=http://localhost:3000

# 1. Login as docente
TOKEN=$(curl -s -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"docente@test.com","password":"pass"}' | jq -r .token)

# 2. Get a short-lived SSE token
SSE_TOKEN=$(curl -s -X POST $BASE/alerts/stream-token \
  -H "Authorization: Bearer $TOKEN" | jq -r .token)

# 3. Open SSE stream in background (prints events as they arrive)
curl -sN "$BASE/alerts/stream?token=$SSE_TOKEN" &
SSE_PID=$!

# 4. Login as student and submit a file
STUDENT_TOKEN=$(curl -s -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"student@test.com","password":"pass"}' | jq -r .token)

curl -s -X POST $BASE/actividades/$ACTIVIDAD_ID/stages/$STAGE_ID/submit \
  -H "Authorization: Bearer $STUDENT_TOKEN" \
  -F "file=@sample.pdf"

# Expected: within 5 seconds, the SSE stream prints:
# id: 1
# event: submission_received
# data: {"alert_id":1,"actividad_id":"...","student_id":"...","advisory":false,...}

# 5. Verify role isolation: open stream as ANOTHER docente — event must NOT appear
OTHER_TOKEN=$(curl -s -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"other_docente@test.com","password":"pass"}' | jq -r .token)

OTHER_SSE=$(curl -s -X POST $BASE/alerts/stream-token \
  -H "Authorization: Bearer $OTHER_TOKEN" | jq -r .token)

curl -sN "$BASE/alerts/stream?token=$OTHER_SSE" &
# Wait 10 s — other docente's stream should remain silent

# 6. Disconnect and reconnect (Last-Event-ID replay)
kill $SSE_PID
# Re-issue token and reconnect with Last-Event-ID header
SSE_TOKEN2=$(curl -s -X POST $BASE/alerts/stream-token \
  -H "Authorization: Bearer $TOKEN" | jq -r .token)

curl -sN "$BASE/alerts/stream?token=$SSE_TOKEN2" \
  -H "Last-Event-ID: 0"
# Expected: replays alert id=1 immediately on connect

# 7. Mark alert as read
curl -s -X PATCH $BASE/alerts/1/read \
  -H "Authorization: Bearer $TOKEN"
# Expected: {"ok":true}
```

---

## Story 2: Per-Activity Analytics

**Prerequisites**: An `investigacion` activity with 3 enrolled students — 2 submitted (1 late, 1 on time), 1 detection flag raised, 1 grade assigned (score: 85).

```bash
# 1. Get analytics for the activity (as docente)
curl -s $BASE/actividades/$ACTIVIDAD_ID/analytics \
  -H "Authorization: Bearer $TOKEN" | jq .

# Expected response:
# {
#   "actividad_id": "...",
#   "submission_count": 2,
#   "pending_count": 1,
#   "late_count": 1,
#   "flagged_count": 1,
#   "grade_distribution": { "range_0_49":0, "range_50_69":0, "range_70_89":1, "range_90_100":0 },
#   "avg_grade": 85.0
# }

# 2. Verify zero state (new activity, no submissions)
curl -s $BASE/actividades/$EMPTY_ACTIVIDAD_ID/analytics \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: all counts 0, avg_grade null

# 3. Verify 403 for wrong docente
curl -s $BASE/actividades/$ACTIVIDAD_ID/analytics \
  -H "Authorization: Bearer $OTHER_TOKEN"
# Expected: 403
```

---

## Story 3: Course-Level Analytics

**Prerequisites**: A course with 2 activities — activity A has 2 submissions (1 graded 90), activity B has 1 submission (1 graded 60, 1 flag).

```bash
# 1. Get course-level analytics
curl -s $BASE/cursos/$CURSO_ID/analytics \
  -H "Authorization: Bearer $TOKEN" | jq .

# Expected response:
# {
#   "curso_id": "...",
#   "activities": [
#     { "actividad_id":"actA", "submission_count":2, "pending_count":0, "late_count":0, "flagged_count":0, "avg_grade":90.0, ... },
#     { "actividad_id":"actB", "submission_count":1, "pending_count":0, "late_count":0, "flagged_count":1, "avg_grade":60.0, ... }
#   ],
#   "total_submissions": 3,
#   "avg_grade": 75.0,
#   "flag_rate": 0.333
# }

# 2. Verify 403 for wrong docente
curl -s $BASE/cursos/$CURSO_ID/analytics \
  -H "Authorization: Bearer $OTHER_TOKEN"
# Expected: 403

# 3. Verify null avg_grade when no grades assigned
curl -s $BASE/cursos/$UNGRADED_CURSO_ID/analytics \
  -H "Authorization: Bearer $TOKEN" | jq .avg_grade
# Expected: null
```
