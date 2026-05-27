# Quickstart: Appeals & Final Audit (M13)

Assumes a running stack. Set env vars: `ADMIN_TOKEN`, `DOCENTE_TOKEN`, `STUDENT_TOKEN`, `SESSION_ID`.

## Story 1 — Student Files an Appeal

```bash
# 1. Student files analysis_dispute appeal
APPEAL_ID=$(curl -s -X POST http://localhost:3000/appeals \
  -H "Authorization: Bearer $STUDENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\":\"$SESSION_ID\",\"type\":\"analysis_dispute\",\"reason\":\"The AI flagged paste but this is my original work\"}" \
  | jq -r '.id')
echo "Appeal: $APPEAL_ID"

# 2. Verify appeal is visible to docente
curl -s http://localhost:3000/appeals/$APPEAL_ID \
  -H "Authorization: Bearer $DOCENTE_TOKEN" | jq '{id:.id, status:.status}'
# Expect status: "pending"

# 3. Student cannot file duplicate
curl -s -X POST http://localhost:3000/appeals \
  -H "Authorization: Bearer $STUDENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\":\"$SESSION_ID\",\"type\":\"analysis_dispute\",\"reason\":\"Again\"}" \
  | jq '.error'
# Expect "conflict"
```

## Story 2 — Grade Log

```bash
# Get grade decision with override reason
curl -s http://localhost:3000/sessions/$SESSION_ID/grade-log \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  | jq '{final_grade:.final_grade, suggested:.suggested_grade_range, reason:.override_reason}'
```

## Story 3 — Docente Reviews Appeal

```bash
# 1. Docente rejects the appeal
curl -s -X PUT http://localhost:3000/appeals/$APPEAL_ID/review \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"decision":"rejected","justification":"Heuristic evidence confirmed pattern; no original work demonstrated"}' \
  | jq '{status:.status}'
# Expect status: "resolved"

# 2. Try re-review — expect 409
curl -s -X PUT http://localhost:3000/appeals/$APPEAL_ID/review \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"decision":"upheld","justification":"Actually changing my mind"}' \
  | jq '.error'
# Expect "already_resolved"
```

## Story 3b — Docente Escalates to Admin

```bash
# File a new appeal on a different session
APPEAL2=$(curl -s -X POST http://localhost:3000/appeals \
  -H "Authorization: Bearer $STUDENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\":\"$SESSION_ID2\",\"type\":\"grade_dispute\",\"reason\":\"Grade seems unfair\"}" \
  | jq -r '.id')

# Docente escalates
curl -s -X PUT http://localhost:3000/appeals/$APPEAL2/escalate \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"notes":"Conflict of interest — I assigned this grade"}' \
  | jq '.status'
# Expect "escalated"

# Admin resolves
curl -s -X PUT http://localhost:3000/appeals/$APPEAL2/admin-review \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"decision":"upheld","justification":"Evidence shows original work; grade adjusted to 85"}' \
  | jq '.status'
# Expect "admin_resolved"
```

## Story 4 — Evidence Chain Verification

```bash
# Get full evidence chain
curl -s http://localhost:3000/appeals/$APPEAL_ID/evidence \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '{valid:.valid, count:(.events|length)}'
# Expect valid: true, count: 4

# Verify first event has null prev_hash
curl -s http://localhost:3000/appeals/$APPEAL_ID/evidence \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  | jq '.events[0].prev_hash'
# Expect null

# Student cannot access evidence
curl -s http://localhost:3000/appeals/$APPEAL_ID/evidence \
  -H "Authorization: Bearer $STUDENT_TOKEN" | jq '.error'
# Expect "forbidden"
```
