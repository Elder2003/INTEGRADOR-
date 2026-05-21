# API Contract: M6 — AI Router

Base path: `/ai`

All endpoints require a valid JWT (`Authorization: Bearer <token>`).
Mode management write requires `role=admin`.

---

## Endpoints

### POST /ai/complete

Route a prompt/completion request to the currently configured backend.

**Auth**: Any authenticated user (internal service-to-service calls)

**Request body** (OpenAI Chat Completions format):
```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "model":      "llama3",
  "messages": [
    { "role": "system",  "content": "You are a helpful assistant." },
    { "role": "user",    "content": "Summarise the following text..." }
  ],
  "max_tokens": 512
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `request_id` | UUID string | No | If omitted, router generates one |
| `model` | string | No | Overrides `AI_PRIMARY_MODEL` env default |
| `messages` | array | Yes | At least one message required |
| `max_tokens` | int | No | Default: 512 |

**Response 200** (OpenAI Chat Completions format + router metadata headers):
```json
{
  "id":      "chatcmpl-abc123",
  "object":  "chat.completion",
  "created": 1716220800,
  "model":   "llama3",
  "choices": [
    {
      "index": 0,
      "message": {
        "role":    "assistant",
        "content": "The text discusses..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens":     42,
    "completion_tokens": 87,
    "total_tokens":      129
  }
}
```

Response headers:
```
X-AI-Backend: local | cloud
X-AI-Mode:    local | auto | cloud
X-Request-ID: <uuid>
```

**Error responses**:

| Status | Body | When |
|---|---|---|
| 400 | `{"error":"invalid_request","message":"messages required"}` | Empty or missing messages array |
| 503 | `{"error":"backend_unavailable","message":"all backends failed","backends":["local","cloud"]}` | All backends failed (auto mode) |
| 503 | `{"error":"backend_unavailable","message":"primary backend failed"}` | Primary failed in local mode |
| 503 | `{"error":"backend_unavailable","message":"cloud backend failed"}` | Cloud failed in cloud mode |
| 401 | `{"error":"unauthorized"}` | Missing or invalid JWT |

---

### GET /ai/config

Read the current routing mode and last change metadata.

**Auth**: Any authenticated user

**Response 200**:
```json
{
  "mode":       "auto",
  "prev_mode":  "local",
  "changed_by": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "changed_at": "2026-05-20T10:05:00Z"
}
```

| Field | Notes |
|---|---|
| `mode` | Current active mode: `local`, `auto`, or `cloud` |
| `prev_mode` | Previous mode; `null` on first seed entry |
| `changed_by` | UUID of the admin who made the change; `null` for system seed |
| `changed_at` | RFC3339 timestamp of the last change |

---

### PUT /ai/config

Update the routing mode. Inserts an append-only history entry.

**Auth**: `role=admin` required

**Request body**:
```json
{
  "mode": "cloud"
}
```

| Field | Type | Required | Allowed values |
|---|---|---|---|
| `mode` | string | Yes | `local`, `auto`, `cloud` |

**Response 200** — new config entry:
```json
{
  "mode":       "cloud",
  "prev_mode":  "auto",
  "changed_by": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "changed_at": "2026-05-20T10:30:00Z"
}
```

**Error responses**:

| Status | Body | When |
|---|---|---|
| 400 | `{"error":"invalid_mode","message":"must be local, auto, or cloud"}` | Unknown mode value |
| 403 | `{"error":"forbidden"}` | Non-admin user |
| 409 | `{"error":"no_change","message":"mode is already cloud"}` | Mode unchanged |

---

### GET /ai/config/history

List the full append-only mode change history, most recent first.

**Auth**: `role=admin` required

**Query params**:

| Param | Type | Default | Notes |
|---|---|---|---|
| `limit` | int | 20 | Max entries to return |
| `offset` | int | 0 | Pagination offset |

**Response 200**:
```json
{
  "entries": [
    {
      "id":         "aabbccdd-...",
      "mode":       "cloud",
      "prev_mode":  "auto",
      "changed_by": "f47ac10b-...",
      "changed_at": "2026-05-20T10:30:00Z"
    },
    {
      "id":         "11223344-...",
      "mode":       "auto",
      "prev_mode":  "local",
      "changed_by": "f47ac10b-...",
      "changed_at": "2026-05-20T10:00:00Z"
    }
  ],
  "total": 3
}
```

---

## WebSocket / Streaming

M6 does not support streaming completions. All requests are synchronous request/response. Streaming support (for future RAG or chat interfaces) is deferred to a later milestone.

---

## Internal Dispatch Flow

```
Caller → POST /ai/complete
         ↓
    [Router reads cached mode]
         ↓
    mode = local  ──→  Ollama /v1/chat/completions
    mode = cloud  ──→  Vercel AI /v1/chat/completions
    mode = auto   ──→  Ollama first; on error/timeout → Vercel AI
         ↓
    [Log AIRequestLog (metadata only)]
         ↓
    Return OpenAI-format response + X-AI-Backend header
```

---

## Rate Limiting

The router does not implement its own rate limiting in M6. Rate limits enforced by the upstream backends (especially Vercel AI Gateway) are propagated as `503 backend_unavailable` errors to the caller. Consumer services are responsible for their own retry logic.
