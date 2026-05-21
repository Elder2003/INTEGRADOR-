# Research: M6 — AI Router

## Decision 1: HTTP client for OpenAI-compatible backends

**Decision**: Use Go's standard `net/http` + `encoding/json` directly; do NOT add `sashabaranov/go-openai` or any third-party AI SDK.

**Rationale**: Both Ollama and Vercel AI Gateway expose identical `/v1/chat/completions` endpoints that accept and return the OpenAI Chat Completions JSON schema. The request body has four fields (`model`, `messages`, `max_tokens`, `stream`) and the response has one primary field (`choices[0].message.content`). A custom thin client defined in `internal/ai/client.go` is 60–80 lines and avoids a large transitive dependency tree. The constitution locks Go + Fiber; adding an AI SDK that pulls in `github.com/sashabaranov/go-openai` adds ~150KB and a maintenance surface for a feature set we don't need.

**Alternatives considered**:
- `sashabaranov/go-openai` — mature, well-tested, but oversized for our use (streaming, fine-tuning, embeddings, function calling endpoints all unused in M6–M9).
- `openai/openai-go` (official, newer) — even heavier.

---

## Decision 2: In-process mode cache implementation

**Decision**: Use `sync/atomic` storing a pointer to a `cachedMode` struct (mode string + fetched_at time). A single background goroutine polls the DB every 3 seconds and swaps the pointer atomically. Cache is read lock-free on every dispatch.

**Rationale**: The 5-second SLA is the hard constraint (FR-003, SC-002). A 3-second poll interval provides a safety margin. `sync/atomic` pointer swaps are contention-free for the very hot path (every AI dispatch reads the mode). A mutex-based approach would add lock contention under concurrent AI requests. The background goroutine is started once in `NewService` and respects context cancellation for clean shutdown.

**Alternatives considered**:
- Lazy refresh (read from cache; if > 5s stale, hit DB inline): adds latency to dispatch on every cache miss; unacceptable for the hot path.
- Redis pub/sub for instant propagation: adds a Redis dependency on mode changes; overkill when a 3-second poll meets the SLA with margin.

---

## Decision 3: router_configs table design (append-only history)

**Decision**: `router_configs` is an INSERT-only table. Each row captures: `id`, `mode`, `prev_mode`, `changed_by` (FK to users), `changed_at`. Current mode = `SELECT mode FROM router_configs ORDER BY changed_at DESC LIMIT 1`. On API startup, if the table is empty, the service seeds it with `mode='local'` and `changed_by` set to a system sentinel UUID.

**Rationale**: Constitution Principle I mandates append-only records for all decision-related data. A single overwrite row would make it impossible to audit "what mode was active when this AI request was processed?" The `changed_at` timestamp cross-references with `ai_request_logs.logged_at` to reconstruct the exact mode active at any point in history. Seeding with `local` on first startup is safe — it is the most conservative mode (no cloud calls).

**Alternatives considered**:
- Single overwrite row: simpler query but violates Principle I.
- Shared `decision_log` table (constitution's Option C): suitable for M13 Appeals, but `decision_log` doesn't exist yet and coupling M6 to a M13 schema creates premature abstraction.

---

## Decision 4: ai_request_logs content policy

**Decision**: `ai_request_logs` stores ONLY metadata — `request_id`, `mode`, `backend_used`, `fallback_used`, `fallback_reason`, `latency_ms`, `status`, `error_detail`, `logged_at`. Prompt content and completion content are NEVER stored.

**Rationale**: FR-011 is explicit. The audit log's purpose is operational traceability (which backend, when, outcome) and constitutional traceability (Principle III: AI output is advisory, not autonomous). Storing prompt/completion would create a privacy surface (student exam content, docente rubrics) with no corresponding benefit for the audit use case. The `error_detail` field stores only the error message from the backend, not any portion of the prompt.

**Alternatives considered**:
- Store prompt hash only (for deduplication detection): rejected — adds complexity, no clear M6 requirement.
- Store full prompt + completion: rejected per FR-011 and data-minimisation principle.

---

## Decision 5: Cloud backend 429 (rate-limit) handling

**Decision**: When the cloud backend returns HTTP 429, the router returns a structured error to the caller immediately. No internal retry, no queue.

**Rationale**: The router is a thin dispatch layer. Rate-limit backoff logic belongs to the consumer (detection engine, worker) that decides whether to retry, delay, or degrade gracefully. The router logging the 429 as `status=failed, error_detail="cloud_rate_limited"` gives the consumer all the information it needs. Adding retry-with-backoff to the router would couple it to the retry semantics of each consumer.

**Alternatives considered**:
- Router retries once after `Retry-After` delay: makes the router stateful and long-running for a single request; blocks the goroutine.
- Queue the request for later: out of scope for M6; would require a new async mechanism.

---

## Decision 6: New environment variables

| Variable | Default | Purpose |
|---|---|---|
| `AI_PRIMARY_URL` | (required) | Ollama base URL, e.g. `http://ollama:11434` |
| `AI_PRIMARY_MODEL` | `llama3` | Default model name sent to primary backend |
| `AI_CLOUD_URL` | (required if mode=cloud/auto) | Vercel AI Gateway base URL |
| `AI_CLOUD_API_KEY` | (required if mode=cloud/auto) | Bearer token for cloud backend |
| `AI_PRIMARY_TIMEOUT_SECS` | `30` | Seconds before primary backend timeout triggers fallback |

`AI_CLOUD_URL` and `AI_CLOUD_API_KEY` are validated at startup only when the initial mode is `cloud` or `auto`. If initial mode is `local`, missing cloud vars are logged as warnings but do not prevent startup (operator may switch to `auto` later without restart).

---

## Decision 7: Startup mode seed

**Decision**: On startup, if `router_configs` is empty, insert a row with `mode='local'`, `prev_mode=NULL`, `changed_by=<system_user_id>`, `changed_at=NOW()`.

**Rationale**: The system must always have a defined mode. `local` is the safest default — no cloud API keys required, no external calls. The constitution requires append-only history; a synthetic "initial" row is consistent with that model and makes the "current mode" query reliable from day one.

**System user ID**: Use a fixed well-known UUID (e.g., `00000000-0000-0000-0000-000000000000`) for the system sentinel. This UUID need not exist in the `users` table; the FK is nullable for the `changed_by` field when `changed_by IS NULL` indicates a system-generated entry.

---

## Decision 8: Module placement

**Decision**: New package `internal/ai/` with files: `module.go`, `model.go`, `repository.go`, `service.go`, `client.go`, `handler_admin.go`, `handler_route.go`.

**Rationale**: Consistent with the existing module pattern (`internal/auth/`, `internal/tracing/`, etc.). The AI router is logically independent of tracing and submissions. `internal/ai/client.go` contains the thin HTTP client used to talk to Ollama and Vercel. `handler_route.go` exposes `POST /ai/complete`; `handler_admin.go` exposes `GET|PUT /ai/config`.
