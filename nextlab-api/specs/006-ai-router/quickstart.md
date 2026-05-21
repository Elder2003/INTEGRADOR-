# Quickstart: M6 — AI Router Verification

End-to-end verification steps for the AI Router milestone.

## Prerequisites

- `docker compose up --build` with M6 code deployed
- Ollama running on `AI_PRIMARY_URL` with `AI_PRIMARY_MODEL` loaded
- Admin JWT (`role=admin`) available as `$ADMIN_TOKEN`
- Student JWT (`role=estudiante`) available as `$STUDENT_TOKEN`
- `$BASE=http://localhost:8080`

---

## Step 1 — Migration applied

```sql
\dt  -- should list router_configs and ai_request_logs
SELECT mode, changed_by, changed_at FROM router_configs ORDER BY changed_at DESC LIMIT 1;
-- Expected: mode='local', changed_by=NULL (system seed)
```

---

## Step 2 — GET current config (any authenticated user)

```bash
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" $BASE/ai/config
# Expected: {"mode":"local","prev_mode":null,"changed_by":null,"changed_at":"..."}
```

---

## Step 3 — POST /ai/complete in local mode

```bash
curl -s -X POST $BASE/ai/complete \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"Say hello."}]}'
# Expected: 200 with choices[0].message.content non-empty
# Expected headers: X-AI-Backend: local, X-AI-Mode: local
```

---

## Step 4 — Verify ai_request_logs entry

```sql
SELECT mode, backend_used, fallback_used, latency_ms, status
FROM ai_request_logs ORDER BY logged_at DESC LIMIT 1;
-- Expected: mode=local, backend_used=local, fallback_used=false, status=success
-- Confirm: no prompt_content or completion_content column exists
```

---

## Step 5 — Non-admin cannot change mode

```bash
curl -s -X PUT $BASE/ai/config \
  -H "Authorization: Bearer $STUDENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"mode":"cloud"}'
# Expected: 403 {"error":"forbidden"}
```

---

## Step 6 — Admin changes mode to auto

```bash
curl -s -X PUT $BASE/ai/config \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"mode":"auto"}'
# Expected: 200 {"mode":"auto","prev_mode":"local","changed_by":"<admin-uuid>","changed_at":"..."}
```

Verify DB:
```sql
SELECT mode, prev_mode FROM router_configs ORDER BY changed_at DESC LIMIT 1;
-- Expected: mode=auto, prev_mode=local
SELECT COUNT(*) FROM router_configs;
-- Expected: 2 (seed row + this change) — NO rows were updated/deleted
```

---

## Step 7 — Mode takes effect within 5 seconds

Wait 6 seconds, then:

```bash
curl -s -X POST $BASE/ai/complete \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"Say hello."}]}'
# Expected: X-AI-Mode: auto, X-AI-Backend: local (primary healthy)
```

---

## Step 8 — Auto mode fallback (primary unavailable)

Stop Ollama (`docker compose stop ollama` or set `AI_PRIMARY_URL` to unreachable).

```bash
curl -s -X POST $BASE/ai/complete \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"Say hello."}]}'
# Expected: 200 response with X-AI-Backend: cloud
# Expected latency <= 10 seconds
```

Verify fallback logged:
```sql
SELECT backend_used, fallback_used, fallback_reason FROM ai_request_logs ORDER BY logged_at DESC LIMIT 1;
-- Expected: backend_used=cloud, fallback_used=true, fallback_reason=primary_error or primary_timeout
```

---

## Step 9 — local mode: no fallback on primary failure

```bash
# Set mode to local
curl -s -X PUT $BASE/ai/config \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"mode":"local"}' \
  -H "Content-Type: application/json"

# Wait 6 seconds, then try with Ollama still stopped:
curl -s -X POST $BASE/ai/complete \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"Say hello."}]}'
# Expected: 503 {"error":"backend_unavailable","message":"primary backend failed"}
# Expected: NO fallback attempt, response within a few seconds
```

---

## Step 10 — Invalid mode rejected

```bash
curl -s -X PUT $BASE/ai/config \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"mode":"turbo"}'
# Expected: 400 {"error":"invalid_mode"}
# Verify: router_configs row count unchanged
```

---

## Step 11 — Config history endpoint

```bash
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" $BASE/ai/config/history
# Expected: entries array with all changes in reverse chronological order
# Verify: no row is missing (total = number of PUT /ai/config calls + 1 seed)
```

---

## Step 12 — Empty messages rejected

```bash
curl -s -X POST $BASE/ai/complete \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"messages":[]}'
# Expected: 400 {"error":"invalid_request","message":"messages required"}
# Verify: NO ai_request_logs entry created for this failed validation
```

---

## Step 13 — Constitution compliance spot-check

```sql
-- Verify no UPDATE or DELETE statements were ever issued
-- (check application logs for any unexpected DML)
SELECT COUNT(*) FROM router_configs;     -- must equal number of mode changes + 1 seed
SELECT COUNT(*) FROM ai_request_logs;   -- must equal number of successful POST /ai/complete calls

-- Verify prompt content is absent
\d ai_request_logs  -- confirm no prompt_content, completion_content, or message columns
```
