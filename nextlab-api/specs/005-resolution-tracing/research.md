# Research: M5 — Resolution & Tracing

## Decision 1: WebSocket Library

**Decision**: Use `github.com/gofiber/websocket/v2` — the official Fiber WebSocket adapter.

**Rationale**: Already consistent with the locked stack (Fiber v2). Wraps gorilla/websocket. The upgrade pattern integrates cleanly with Fiber middleware for auth checking before the upgrade. Not in go.mod yet — must be added.

**Alternatives considered**:
- `nhooyr.io/websocket` — better API but not Fiber-native; would require manual HTTP hijacking.
- Raw net/http WebSocket — incompatible with Fiber's fasthttp engine.

---

## Decision 2: JWT Authentication Over WebSocket

**Decision**: Pass the JWT as a URL query parameter `?token=<jwt>` on the WebSocket upgrade request.

**Rationale**: Browser WebSocket APIs (`new WebSocket(url)`) cannot set custom HTTP headers during the upgrade handshake. The standard workaround is a query parameter. The token is validated by a Fiber middleware **before** the upgrade occurs, so the WebSocket handler receives only authenticated connections.

**Alternatives considered**:
- First-message auth token: client sends token as first WebSocket message. Rejected — the connection is open before auth is confirmed, creating a window for unauthenticated connections.
- Cookie auth: viable but requires same-origin setup; query param is simpler for the API-first architecture.

**Security note**: TLS in production (MINIO_USE_SSL pattern applies) ensures the query param is not visible in transit. Token must not be logged server-side.

---

## Decision 3: Redis Streams Consumer Group Pattern

**Decision**: Use Redis Streams with a named consumer group (`analysis-workers`) and XACK-based acknowledgement.

**Pattern**:
- Enqueue: `XADD nextlab:sessions:jobs * session_id <uuid> submission_id <uuid> enqueued_at <ts>`
- Consume: `XREADGROUP GROUP analysis-workers worker-<hostname> COUNT 1 BLOCK 5000 STREAMS nextlab:sessions:jobs >`
- Acknowledge: `XACK nextlab:sessions:jobs analysis-workers <stream-id>`
- Crash recovery: On worker start, call `XPENDING nextlab:sessions:jobs analysis-workers - + 10` and `XCLAIM` any messages idle > 30 seconds.

**Rationale**: Redis Streams provide durable, ordered, exactly-once-deliverable messages with built-in consumer group semantics. The Pending Entry List (PEL) ensures messages survive worker crashes. `go-redis/v9` (already in go.mod) has full Streams support — no new library needed.

**Alternatives considered**:
- Redis List (BLPOP): no acknowledgement, no crash recovery. Rejected.
- Separate message broker (RabbitMQ, NATS): not in the locked stack. Rejected.

---

## Decision 4: DB Durability for Worker Jobs

**Decision**: Write a `worker_jobs` row to PostgreSQL **before** the XADD, then store the Redis stream message ID back into the row on success.

**Rationale**: If Redis is temporarily unavailable at session close time, the job exists in the DB with status `pending` and no `stream_id`. The worker's startup sweep can detect these orphaned rows and re-enqueue them. This gives two layers of durability without distributed transactions.

**Pattern**:
```
1. INSERT INTO worker_jobs (session_id, submission_id, status='pending') RETURNING id
2. XADD nextlab:sessions:jobs * job_id <id> session_id <sid> ...
3. UPDATE worker_jobs SET stream_id=<stream-id>, status='queued' WHERE id=<id>
```

**Alternatives considered**:
- Redis-only: no DB mirror. Rejected — Redis persistence (AOF/RDB) is not guaranteed in this Docker Compose setup.
- Transactional outbox pattern: too complex for this stage; the two-step above is sufficient.

---

## Decision 5: Session Deadline Auto-Close

**Decision**: Two-layer approach: lazy close on event receipt + periodic sweep in the worker.

**Lazy close**: When any event arrives for a session, the handler first checks `actividades.deadline`. If `NOW() > deadline`, it rejects the event, marks the session `deadline_closed`, and appends a final `session_close` trace event.

**Periodic sweep**: The worker runs a ticker every 60 seconds querying:
```sql
SELECT es.id FROM exam_sessions es
JOIN actividades a ON a.id = es.actividad_id
WHERE es.status = 'active' AND a.deadline < NOW()
```
For each result: close the session as `deadline_closed` and enqueue a worker job if a submission is linked.

**Rationale**: Lazy close handles the common case (student sends an event just after deadline) with zero background goroutines in the API. The sweep handles sessions where the deadline passed but no events arrived (student disconnected before deadline).

**Alternatives considered**:
- API-side goroutine per session: would require in-memory state and is fragile across restarts. Rejected.
- Database trigger: would work but adds hidden logic. Rejected for observability.

---

## Decision 6: Trace Event Hash Chaining

**Decision**: SHA-256 of a canonical string: `<session_id>|<sequence_num>|<event_type>|<received_at_unix_nano>|<metadata_json_canonical>|<prev_hash>`.

**First event seed**: The first event's `prev_hash` input is `SHA-256(<session_id>)` (the session ID itself hashed), ensuring the chain is rooted in an unforgeable value.

**Rationale**: SHA-256 is collision-resistant and fast. Canonical JSON (keys sorted, no extra whitespace) ensures deterministic hashing of the metadata field. The chain is self-verifiable: any party with the session ID can recompute the entire chain without trusting the stored hashes.

**Alternatives considered**:
- HMAC-SHA256 with a server secret: adds a server key dependency; verification requires the key. Rejected — the spec requires independent verifiability (no key needed).
- SHA-3: stronger but unnecessary overhead for this use case.

---

## Decision 7: Event Rate Limiting

**Decision**: 30 events per minute per session, enforced in-memory per WebSocket connection.

**Rationale**: Real browser events (tab focus/blur, clipboard, heartbeat every 30s) generate at most 10–15 events per minute under normal exam use. 30/min gives a 2× headroom for bursts while blocking flood attacks. Excess events return a WebSocket error frame without closing the connection (the student continues the exam).

**Implementation**: Each connection goroutine maintains a `rateCount int` and `rateReset time.Time` field. On each event: if `rateReset` is past, reset counter; if counter > 30, send error frame and skip.

---

## Decision 8: Multi-Tab Support

**Decision**: In-memory connection hub — a `sync.Map` mapping `sessionID → []*websocket.Conn`.

**Rationale**: The spec says multiple tabs reuse the same session. All connections feed events into the same session. The hub is local to the API process; since this is a single-instance deployment (Docker Compose, one API container), no distributed pub/sub is needed for M5.

**Alternatives considered**:
- Redis pub/sub for multi-instance: needed at scale but out of scope for M5 single-instance topology.

---

## Decision 9: M4 Integration Point

**Decision**: Extend `internal/submission/service.go` `UploadFile` and `ReplaceFile` to detect `exam`/`defensa_oral` activity type and close the associated session within the same operation.

**Pattern in UploadFile** (after DB insert):
```
if act.Tipo ∈ {exam, defensa_oral}:
    session = repo.GetActiveSession(studentID, actividadID)
    if session != nil:
        repo.CloseSessionWithSubmission(session.ID, submission.ID)
        worker_repo.EnqueueJob(session.ID, submission.ID)
```

**Rationale**: Keeps the session-close atomic with the submission insert. No separate endpoint needed. If no session exists (e.g., student accessed via raw API), the upload proceeds normally (constitution Principle VII) — session remains nil and no job is queued.

**Alternatives considered**:
- Separate session-close endpoint called by frontend: race condition between close and submission. Rejected.
