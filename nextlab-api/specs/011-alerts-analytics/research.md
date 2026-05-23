# Research: Alerts & Analytics (M11)

## Decision 1: SSE Streaming in Fiber v2

**Decision**: Use `c.Context().SetBodyStreamWriter(func(w *bufio.Writer))` with manual `fmt.Fprintf` to write SSE-formatted lines (`id:`, `event:`, `data:`, blank line).

**Rationale**: Fiber v2 has no built-in SSE helper, but its `fasthttp` context exposes `SetBodyStreamWriter` which keeps the connection open for streaming. This is the established pattern in the Fiber ecosystem. The writer blocks until the connection closes, making it suitable for a goroutine-per-connection model.

**Implementation sketch**:
```go
c.Set("Content-Type", "text/event-stream")
c.Set("Cache-Control", "no-cache")
c.Set("Connection", "keep-alive")
lastEventID := c.Get("Last-Event-ID") // sent automatically by browser EventSource
c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
    // 1. replay backlog using lastEventID
    // 2. subscribe to Redis channel
    // 3. forward messages until ctx.Done()
})
```

**Alternatives considered**: `gofiber/contrib/sse` package — not yet stable for this Fiber version; avoided to reduce dependency risk.

---

## Decision 2: Redis Pub/Sub Fan-Out

**Decision**: Use `go-redis/v9` `Subscribe` (not Streams) on per-docente channels named `alerts:<docente_id>`. The alert writer calls `rdb.Publish(ctx, "alerts:"+docenteID, jsonPayload)` after inserting the DB row. The SSE goroutine calls `sub := rdb.Subscribe(ctx, "alerts:"+docenteID)` and reads from `sub.Channel()`.

**Rationale**: Pub/sub is ephemeral (no consumer group management needed) and perfectly suited to fan-out to a live SSE connection. The DB row is the durable record; Redis is only the delivery vehicle. Channel per docente keeps isolation simple — no topic filtering needed on the subscriber side.

**Alternatives considered**:
- Redis Streams (XADD/XREAD): overkill — persistence and consumer groups already covered by the DB; pub/sub suffices.
- PostgreSQL LISTEN/NOTIFY: would tie up one dedicated PG connection per connected docente — not scalable.

---

## Decision 3: Last-Event-ID Replay

**Decision**: `alerts` table has a `BIGSERIAL` primary key `id`. Every SSE event is sent with `id: <alerts.id>`. On reconnect, the browser sends `Last-Event-ID: <last_id>` in the request header. The SSE handler reads this header, queries `SELECT * FROM alerts WHERE docente_id=$1 AND id > $2 ORDER BY id ASC` and streams those rows before subscribing to the Redis channel.

**Rationale**: Standard SSE protocol — browsers implement this automatically. The sequential integer ID is monotonically increasing, making the "replay since" query trivially correct and index-friendly.

**Race condition handling**: The handler subscribes to Redis **before** querying the backlog. Any events published between the backlog query and subscription completion will be caught because the subscription is already active when the query runs. Duplicates are impossible because each alert has a unique ID that the client tracks.

---

## Decision 4: Short-Lived SSE Token

**Decision**: `POST /alerts/stream-token` (requires standard JWT auth) generates a UUID v4, stores it in Redis as `sse_token:<uuid>` → `<docente_id>` with a 60 s TTL, and returns `{"token": "<uuid>"}`. The SSE endpoint reads `?token=<uuid>`, looks up the key in Redis, deletes it immediately (one-time use), and proceeds with the resolved `docente_id`.

**Rationale**: Short-lived + one-time use prevents token reuse if the URL is logged. 60 s window is sufficient for the frontend to open the EventSource immediately after receiving the token. Redis TTL handles cleanup automatically.

**Alternatives considered**: JWT in query param — exposes the long-lived JWT in server access logs for the duration of the connection.

---

## Decision 5: Alert Writer Interface

**Decision**: Expose a lightweight `AlertWriter` interface from `internal/alerts`:
```go
type AlertWriter interface {
    WriteAlert(ctx context.Context, a Alert) error
}
```
The concrete `*Service` implements it. This interface is injected into submission, format, detection, and git worker so they can write alerts without importing the full alerts module.

**Rationale**: Avoids circular imports. The interface lives in `internal/alerts/model.go`; consumers only import the interface type, not the full service. The concrete service is wired in `cmd/api/main.go`.

---

## Decision 6: Analytics Query Strategy

**Decision**: Analytics are computed with a single SQL query per request using `COUNT`, `AVG`, and conditional aggregation (`COUNT(*) FILTER (WHERE ...)`). No materialized views or caches — data is always live.

**Rationale**: For the stated scale (500 students, 20 activities), a well-indexed aggregate query returns in well under 2 s. Adding a cache layer introduces staleness complexity without measurable benefit at this scale.

**Grade distribution query** (example for `investigacion` submissions):
```sql
SELECT
  COUNT(*) FILTER (WHERE grade IS NOT NULL)                              AS graded_count,
  COUNT(*) FILTER (WHERE grade BETWEEN 0 AND 49)                        AS range_0_49,
  COUNT(*) FILTER (WHERE grade BETWEEN 50 AND 69)                       AS range_50_69,
  COUNT(*) FILTER (WHERE grade BETWEEN 70 AND 89)                       AS range_70_89,
  COUNT(*) FILTER (WHERE grade BETWEEN 90 AND 100)                      AS range_90_100,
  AVG(grade)                                                             AS avg_grade
FROM stage_submissions ss
JOIN investigacion_stages s ON ss.stage_id = s.id
WHERE s.actividad_id = $1
```

**Multi-tipo support**: The analytics query must union across `submissions` (tarea/informe/examen), `stage_submissions` (investigacion), `git_analyses` (git_analysis), and `defense_appointments` (defensa_oral) since different activity types store grades in different tables.
