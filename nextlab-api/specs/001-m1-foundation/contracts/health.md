# Contract: GET /health

**Milestone**: M1 — Fundación del Proyecto
**Module**: `internal/health`

---

## Endpoint

```
GET /health
```

No authentication required. No request body. No query parameters.

---

## Responses

### 200 OK — All dependencies healthy

```json
{
  "status": "ok",
  "dependencies": {
    "database": { "status": "ok", "latency_ms": 3 },
    "cache":    { "status": "ok", "latency_ms": 1 },
    "storage":  { "status": "ok", "latency_ms": 5 }
  }
}
```

### 503 Service Unavailable — One or more dependencies unhealthy

```json
{
  "status": "degraded",
  "dependencies": {
    "database": { "status": "ok", "latency_ms": 2 },
    "cache":    { "status": "down", "error": "dial tcp: connection refused" },
    "storage":  { "status": "ok", "latency_ms": 4 }
  }
}
```

---

## Field Definitions

| Field | Type | Always present | Description |
|---|---|---|---|
| `status` | string | yes | Aggregate status: `"ok"` all healthy / `"degraded"` some down / `"down"` all down |
| `dependencies` | object | yes | Per-dependency breakdown |
| `dependencies.<name>.status` | string | yes | `"ok"` \| `"down"` |
| `dependencies.<name>.latency_ms` | int | when `"ok"` | Round-trip ping duration in milliseconds |
| `dependencies.<name>.error` | string | when `"down"` | Raw error message from the failed ping |

**Dependency names**: `database`, `cache`, `storage` (fixed set for M1).

---

## Behaviour Rules

1. All three pings run **in parallel** with a shared 400ms deadline.
2. If any dependency times out, its status is `"down"` and `error` is `"timeout"`.
3. The endpoint responds with **HTTP 200** if and only if all three statuses are `"ok"`.
4. Any other combination returns **HTTP 503**.
5. The endpoint itself MUST respond within **500ms** end-to-end (SC-002).
6. The API MUST return a response even if all three dependencies are down — the
   endpoint never hangs or panics.

---

## SLA

| Metric | Target |
|---|---|
| Response time (all deps healthy) | ≤ 500ms |
| Response time (deps down, timeout path) | ≤ 500ms (400ms timeout + overhead) |

---

## Example `curl`

```bash
curl -s http://localhost:8080/health | jq .
```
