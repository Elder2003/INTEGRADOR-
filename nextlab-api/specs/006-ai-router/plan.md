# Implementation Plan: M6 — AI Router

**Branch**: `006-ai-router` | **Date**: 2026-05-20 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/006-ai-router/spec.md`

---

## Summary

Implement a three-mode AI routing layer for nextlab-api: a thin HTTP dispatcher that
forwards OpenAI-compatible prompt/completion requests to either the primary backend
(Ollama, self-hosted GPU) or the cloud fallback (Vercel AI Gateway), based on a runtime-
configurable mode (`local` / `auto` / `cloud`). Mode is persisted as an append-only
history in PostgreSQL and cached in-process for sub-5-second hot-swap. Every dispatch
produces an immutable audit log entry (metadata only — no prompt content). A management
API allows admins to read and change the mode without restarting the service.

**Technical approach**: New `internal/ai/` package · custom thin HTTP client (no SDK
dependency) · `sync/atomic` pointer for lock-free mode cache · 3-second background poll ·
`router_configs` append-only table (current mode = latest row) · `ai_request_logs`
append-only table (metadata only) · two management endpoints (`GET|PUT /ai/config`,
`GET /ai/config/history`) · one dispatch endpoint (`POST /ai/complete`) · four new
environment variables · migration `000006_ai_router`

---

## Technical Context

**Language/Version**: Go 1.22 (module declares `go 1.25.0`)

**Primary Dependencies**:
- `gofiber/fiber/v2` — HTTP framework (locked by constitution)
- `jackc/pgx/v5/pgxpool` — PostgreSQL pool (from M1, no change)
- `golang-jwt/jwt/v5` — JWT parsing (from M2, no change)
- `google/uuid` — UUID generation (already in go.sum)
- `net/http` + `encoding/json` — thin AI backend HTTP client (stdlib, no new dependency)
- `sync/atomic` — lock-free mode cache (stdlib)

**No new go.mod dependencies** for M6.

**Storage**:
- PostgreSQL 16: 2 new tables (`router_configs`, `ai_request_logs`)

**New environment variables**:

| Variable | Required | Default | Notes |
|---|---|---|---|
| `AI_PRIMARY_URL` | Yes | — | Ollama base URL, e.g. `http://ollama:11434` |
| `AI_PRIMARY_MODEL` | No | `llama3` | Default model sent to primary backend |
| `AI_CLOUD_URL` | Conditional | — | Required when initial mode is `auto` or `cloud` |
| `AI_CLOUD_API_KEY` | Conditional | — | Bearer token for Vercel AI Gateway |
| `AI_PRIMARY_TIMEOUT_SECS` | No | `30` | Primary backend timeout before fallback |

`AI_CLOUD_URL` and `AI_CLOUD_API_KEY` are validated at startup only when mode is `auto`
or `cloud`. Missing cloud vars in `local` mode are logged as warnings, not startup failures.

**Target Platform**: Linux/AMD64 (Docker Compose, same as M1–M5)

**Performance Goals**:
- Router dispatch overhead ≤ 50ms (SC-001, excluding backend inference time)
- Mode change propagation ≤ 5 seconds (SC-002, FR-003)
- Auto-mode fallback total response ≤ 10 seconds (SC-003)

**Constraints**:
- `router_configs` and `ai_request_logs`: no `UPDATE` or `DELETE` ever (Constitution Principle I)
- Prompt and completion content MUST NOT be stored anywhere (FR-011)
- AI output is advisory only — no autonomous decisions (Constitution Principle III)
- Mode management write restricted to `admin` role (Constitution Principle IV)
- Async analysis work (M8 detection engine) calls this router; router itself is synchronous

**Scale/Scope**: ~100–500 AI requests/day in peak usage; `ai_request_logs` rows grow at
~150K/year at peak — negligible storage.

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Immutable Evidence | ✅ **PRIMARY** | `router_configs` and `ai_request_logs` are INSERT-only. Repository layer exposes no update or delete methods for these tables. Append-only history satisfies the audit requirement. |
| II. Asynchronous Analysis Isolation | ✅ Maintained | The AI router is a synchronous dispatch layer. It does NOT call the router from an HTTP handler in response to a student action — it is called by the async worker (M8+). `POST /ai/complete` is an internal endpoint. M6 itself adds no inline analysis to any student-facing handler. |
| III. Human Authority Over AI | ✅ **PRIMARY** | The router is infrastructure, not a decision engine. It returns raw completions; the calling service (M8 detection, M7 RAG) is responsible for labelling output as advisory. FR-011 ensures no prompt content is retained. No score, grade, or sanction is produced by this module. |
| IV. Strict Role-Based Data Isolation | ✅ **GATE** | `PUT /ai/config` and `GET /ai/config/history` require `role=admin`. `GET /ai/config` (read-only) is accessible to any authenticated user. `POST /ai/complete` is accessible to any authenticated user (internal service calls). Students cannot access management endpoints. |
| V. Dual RAG Independence | ✅ N/A | No RAG in M6. The router is the transport layer; RAG filters are M7's concern. |
| VI. Non-Invasive Auditing | ✅ N/A | No session or trace involvement in M6. |
| VII. Submission Always Accepted | ✅ N/A | No submission flow involvement in M6. |

**Result: PASS** — Principles I and III are the primary enforcement drivers. No gate violations.
No complexity-tracking entries required.

---

## Project Structure

### Documentation (this feature)

```text
specs/006-ai-router/
├── plan.md                   ← this file
├── research.md               ← Phase 0 output
├── data-model.md             ← Phase 1 output
├── quickstart.md             ← Phase 1 output
├── contracts/
│   └── ai-router.md          ← Phase 1 output
├── checklists/
│   └── requirements.md
└── tasks.md                  ← Phase 2 output (/speckit-tasks)
```

### Source Code Changes

```text
nextlab-api/
├── cmd/
│   └── api/
│       └── main.go                       # EXTEND: add AI_PRIMARY_URL etc. to config.Load();
│                                         #   call ai.RegisterRoutes(app, pgPool, cfg, authMW, roleGuard)
├── internal/
│   ├── ai/                               # NEW PACKAGE
│   │   ├── module.go                     # RegisterRoutes: /ai/complete + /ai/config endpoints
│   │   ├── model.go                      # RouterConfig, AIRequestLog, request/response types
│   │   ├── client.go                     # Thin HTTP client for OpenAI-compatible backends
│   │   ├── repository.go                 # INSERT router_configs + ai_request_logs; read current mode
│   │   ├── service.go                    # Dispatch(), SetMode(), GetCurrentMode(), cache goroutine
│   │   ├── handler_route.go              # POST /ai/complete
│   │   └── handler_admin.go              # GET|PUT /ai/config, GET /ai/config/history
│   └── config/
│       └── config.go                     # EXTEND: add AI_PRIMARY_URL, AI_PRIMARY_MODEL,
│                                         #   AI_CLOUD_URL, AI_CLOUD_API_KEY, AI_PRIMARY_TIMEOUT_SECS
├── migrations/
│   ├── 000006_ai_router.up.sql           # router_configs, ai_request_logs tables + indexes
│   └── 000006_ai_router.down.sql
└── go.sum                                # Unchanged (no new dependencies)
```

**Structure Decision**: All AI routing logic lives in `internal/ai/`. The `config` package
gets new fields for backend URLs/keys. No new binaries — the router runs inside the existing
`cmd/api` process. The worker (`cmd/worker`) will call `POST /ai/complete` over HTTP in M8.

---

## Phase 0: Research Summary

See [research.md](research.md). Key decisions:

| Topic | Decision |
|---|---|
| AI backend client | `net/http` + `encoding/json` stdlib — no third-party AI SDK |
| Mode cache | `sync/atomic` pointer + 3-second background poll goroutine |
| RouterConfig persistence | Append-only `router_configs` table; current mode = latest row |
| Startup seed | Insert `mode='local'` if table empty; `changed_by=NULL` for system entry |
| ai_request_logs content | Metadata only — no prompt, no completion (FR-011) |
| Cloud 429 handling | Return 503 to caller immediately; no internal retry |
| New go.mod deps | None — stdlib only |
| Module placement | `internal/ai/` consistent with existing package pattern |

---

## Phase 1: Design Summary

### Data Model

See [data-model.md](data-model.md). Migration `000006_ai_router`:

| Table | Description |
|---|---|
| `router_configs` | Append-only mode change history; current mode = `ORDER BY changed_at DESC LIMIT 1` |
| `ai_request_logs` | Append-only per-request audit; metadata only, no prompt/completion content |

### API Contract

See [contracts/ai-router.md](contracts/ai-router.md). Full endpoint list:

| Method | Path | Role | Description |
|---|---|---|---|
| `POST` | `/ai/complete` | any authenticated | Route prompt to configured backend |
| `GET` | `/ai/config` | any authenticated | Read current mode |
| `PUT` | `/ai/config` | admin | Change routing mode (inserts history row) |
| `GET` | `/ai/config/history` | admin | List full mode change history |

### Quickstart

See [quickstart.md](quickstart.md). 13-step verification covering: migration, GET config,
POST complete in local mode, log audit, non-admin rejection, mode change to auto, 5-second
propagation, fallback test, local-mode no-fallback, invalid mode rejection, history endpoint,
empty messages rejection, constitution compliance spot-check.

---

## Key Implementation Notes

### config.go: new AI fields

```go
type Config struct {
    // ... existing fields ...
    AIPrimaryURL         string
    AIPrimaryModel       string
    AICloudURL           string
    AICloudAPIKey        string
    AIPrimaryTimeoutSecs int
}
```

In `Load()`:
```go
cfg.AIPrimaryURL         = getenv("AI_PRIMARY_URL", "")
cfg.AIPrimaryModel       = getenv("AI_PRIMARY_MODEL", "llama3")
cfg.AICloudURL           = getenv("AI_CLOUD_URL", "")
cfg.AICloudAPIKey        = getenv("AI_CLOUD_API_KEY", "")
cfg.AIPrimaryTimeoutSecs = parseInt(getenv("AI_PRIMARY_TIMEOUT_SECS", "30"))
```

Validate `AIPrimaryURL` as required. `AICloudURL`/`AICloudAPIKey` validated only when
current DB mode is `auto` or `cloud` at startup.

---

### client.go: thin OpenAI-compatible HTTP client

```go
type BackendClient struct {
    BaseURL    string
    APIKey     string  // empty for Ollama (no auth)
    HTTPClient *http.Client
}

type ChatRequest struct {
    Model     string        `json:"model"`
    Messages  []ChatMessage `json:"messages"`
    MaxTokens int           `json:"max_tokens,omitempty"`
}

type ChatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ChatResponse struct {
    ID      string       `json:"id"`
    Object  string       `json:"object"`
    Created int64        `json:"created"`
    Model   string       `json:"model"`
    Choices []ChatChoice `json:"choices"`
    Usage   *ChatUsage   `json:"usage,omitempty"`
}

func (c *BackendClient) Complete(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST",
        c.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")
    if c.APIKey != "" {
        httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
    }
    resp, err := c.HTTPClient.Do(httpReq)
    // ... error handling, JSON decode ...
}
```

---

### service.go: mode cache + dispatch

```go
type cachedMode struct {
    mode      string
    fetchedAt time.Time
}

type Service struct {
    repo          *Repository
    primaryClient *BackendClient
    cloudClient   *BackendClient
    modeCache     atomic.Pointer[cachedMode]
}

// Start background cache refresh goroutine
func (s *Service) StartCacheRefresh(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(3 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done(): return
            case <-ticker.C:
                mode, _ := s.repo.GetCurrentMode(ctx)
                s.modeCache.Store(&cachedMode{mode: mode, fetchedAt: time.Now()})
            }
        }
    }()
}

func (s *Service) Dispatch(ctx context.Context, req DispatchRequest) (*ChatResponse, *AIRequestLog, error) {
    mode := s.modeCache.Load().mode
    start := time.Now()
    logEntry := &AIRequestLog{
        RequestID: req.RequestID,
        Mode:      mode,
    }

    switch mode {
    case "local":
        resp, err := s.primaryClient.Complete(ctx, req.toChatRequest())
        logEntry.BackendUsed = "local"
        logEntry.LatencyMs = int(time.Since(start).Milliseconds())
        if err != nil {
            logEntry.Status, logEntry.ErrorDetail = "failed", ptr(err.Error())
        } else {
            logEntry.Status = "success"
        }
        s.repo.LogRequest(ctx, logEntry)
        return resp, logEntry, err

    case "cloud":
        // symmetric to local

    case "auto":
        resp, err := s.primaryClient.Complete(ctx, req.toChatRequest())
        if err != nil {
            // fallback
            logEntry.FallbackUsed = true
            logEntry.FallbackReason = ptr(classifyPrimaryError(err))
            resp, err = s.cloudClient.Complete(ctx, req.toChatRequest())
            logEntry.BackendUsed = "cloud"
        } else {
            logEntry.BackendUsed = "local"
        }
        logEntry.LatencyMs = int(time.Since(start).Milliseconds())
        // ... set status, log, return
    }
}
```

---

### repository.go: append-only writes

```go
// GetCurrentMode returns the active mode from the most recent router_configs row.
func (r *Repository) GetCurrentMode(ctx context.Context) (string, error) {
    var mode string
    err := r.db.QueryRow(ctx,
        `SELECT mode FROM router_configs ORDER BY changed_at DESC LIMIT 1`,
    ).Scan(&mode)
    if errors.Is(err, pgx.ErrNoRows) {
        return "local", nil // seed not yet written; caller seeds
    }
    return mode, err
}

// InsertModeChange inserts a new append-only router_configs row.
func (r *Repository) InsertModeChange(ctx context.Context, mode, prevMode string, changedBy *uuid.UUID) (*RouterConfig, error) {
    var cfg RouterConfig
    err := r.db.QueryRow(ctx, `
        INSERT INTO router_configs (mode, prev_mode, changed_by)
        VALUES ($1, $2, $3)
        RETURNING id, mode, prev_mode, changed_by, changed_at
    `, mode, nullableStr(prevMode), changedBy).Scan(
        &cfg.ID, &cfg.Mode, &cfg.PrevMode, &cfg.ChangedBy, &cfg.ChangedAt,
    )
    return &cfg, err
}

// LogRequest inserts an append-only ai_request_logs row.
func (r *Repository) LogRequest(ctx context.Context, log *AIRequestLog) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO ai_request_logs
          (request_id, mode, backend_used, fallback_used, fallback_reason,
           latency_ms, status, error_detail)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
    `, log.RequestID, log.Mode, log.BackendUsed, log.FallbackUsed,
       log.FallbackReason, log.LatencyMs, log.Status, log.ErrorDetail)
    return err
}
```

---

### handler_admin.go: mode change with 409 guard

```go
func putConfigHandler(svc *Service) fiber.Handler {
    return func(c *fiber.Ctx) error {
        var body struct{ Mode string `json:"mode"` }
        if err := c.BodyParser(&body); err != nil || body.Mode == "" {
            return c.Status(400).JSON(fiber.Map{"error": "invalid_mode"})
        }
        if body.Mode != "local" && body.Mode != "auto" && body.Mode != "cloud" {
            return c.Status(400).JSON(fiber.Map{"error": "invalid_mode",
                "message": "must be local, auto, or cloud"})
        }
        callerID, _ := parseUserID(c)
        cfg, err := svc.SetMode(c.Context(), body.Mode, callerID)
        if errors.Is(err, ErrModeUnchanged) {
            return c.Status(409).JSON(fiber.Map{"error": "no_change",
                "message": "mode is already " + body.Mode})
        }
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": "internal_error"})
        }
        return c.JSON(toConfigResponse(cfg))
    }
}
```

---

### module.go: RegisterRoutes + cache start

```go
func RegisterRoutes(
    app *fiber.App,
    db *pgxpool.Pool,
    cfg *config.Config,
    authMW fiber.Handler,
    roleGuard func(roles ...string) fiber.Handler,
) {
    repo := NewRepository(db)
    svc  := NewService(repo, cfg)

    // Seed mode on startup if table is empty
    svc.SeedIfEmpty(context.Background())

    // Start background cache refresh (3-second poll)
    svc.StartCacheRefresh(context.Background())

    // Internal: any authenticated user
    internal := app.Group("/ai", authMW)
    internal.Post("/complete", routeHandler(svc))
    internal.Get("/config", getConfigHandler(svc))

    // Admin-only: mode management
    admin := app.Group("/ai", authMW, roleGuard("admin"))
    admin.Put("/config", putConfigHandler(svc))
    admin.Get("/config/history", getConfigHistoryHandler(svc))
}
```

---

## Key Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Background cache goroutine leaks on shutdown | `StartCacheRefresh` accepts `context.Context`; cancel propagates to goroutine via `ctx.Done()` |
| Both backends unavailable in `auto` mode | Returns 503 with `{"backends":["local","cloud"]}` after both attempts; no retry in router |
| Cloud 429 propagated as generic 503 | `error_detail` in log captures HTTP status; caller can inspect response body for retry guidance |
| Concurrent mode changes race condition | `InsertModeChange` is a single INSERT; PostgreSQL serialises concurrent INSERTs; last writer wins by `changed_at DESC` |
| `AI_CLOUD_URL`/`AI_CLOUD_API_KEY` missing when mode switches to `auto` | Service validates cloud config on `SetMode` call; returns error if cloud vars absent |
| Startup seed races with first mode change | `SeedIfEmpty` uses `INSERT … ON CONFLICT DO NOTHING` on a unique seed row, or simply checks row count before inserting |
