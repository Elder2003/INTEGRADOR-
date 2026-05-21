# Tasks: M6 — AI Router

**Input**: Design documents from `specs/006-ai-router/`

**Prerequisites**: plan.md ✅, spec.md ✅, data-model.md ✅, contracts/ai-router.md ✅, research.md ✅, quickstart.md ✅

**Tests**: Not requested — no test tasks generated.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)
- Exact file paths are included in every description

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Extend config and create DB migration — no `internal/ai/` package yet.

- [X] T001 Extend `internal/config/config.go` to add AIPrimaryURL (required), AIPrimaryModel (default "llama3"), AICloudURL, AICloudAPIKey, AIPrimaryTimeoutSecs (default 30); validate AIPrimaryURL at startup; warn (not error) if cloud vars missing when mode is local
- [X] T002 [P] Create `migrations/000006_ai_router.up.sql` with `router_configs` (id UUID PK, mode TEXT CHECK IN ('local','auto','cloud'), prev_mode TEXT nullable, changed_by UUID nullable FK users(id), changed_at TIMESTAMPTZ DEFAULT now()) and `ai_request_logs` (id, request_id UUID NOT NULL, mode, backend_used TEXT CHECK IN ('local','cloud'), fallback_used BOOLEAN DEFAULT false, fallback_reason TEXT, latency_ms INT, status TEXT CHECK IN ('success','failed'), error_detail TEXT, logged_at TIMESTAMPTZ DEFAULT now()) — both with DESC indexes; NO prompt_content or completion_content columns
- [X] T003 [P] Create `migrations/000006_ai_router.down.sql` with `DROP TABLE IF EXISTS ai_request_logs; DROP TABLE IF EXISTS router_configs;`

**Checkpoint**: Config fields available; migration files ready to apply.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core `internal/ai/` types, HTTP client, and DB repository that all user stories depend on. No routing or HTTP handlers yet.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T004 Create `internal/ai/model.go` with: RouterConfig struct (ID, Mode, PrevMode *string, ChangedBy *uuid.UUID, ChangedAt), AIRequestLog struct (ID, RequestID, Mode, BackendUsed, FallbackUsed bool, FallbackReason *string, LatencyMs int, Status, ErrorDetail *string, LoggedAt), ChatMessage struct (Role, Content string), ChatRequest struct (Model, Messages []ChatMessage, MaxTokens int json tags), ChatResponse struct (ID, Object, Created int64, Model, Choices []ChatChoice, Usage *ChatUsage), ChatChoice struct (Index int, Message ChatMessage, FinishReason string), ChatUsage struct (PromptTokens, CompletionTokens, TotalTokens int), DispatchRequest struct (RequestID uuid.UUID, Model string, Messages []ChatMessage, MaxTokens int), ErrModeUnchanged sentinel error
- [X] T005 [P] Create `internal/ai/client.go` with BackendClient struct (BaseURL, APIKey string, HTTPClient *http.Client) and Complete(ctx, ChatRequest) (*ChatResponse, error) method that POSTs to BaseURL+"/v1/chat/completions", sets Content-Type and Authorization header only when APIKey non-empty, decodes JSON response, and returns a descriptive error on non-2xx status
- [X] T006 [P] Create `internal/ai/repository.go` with Repository struct (db *pgxpool.Pool), NewRepository constructor, GetCurrentMode(ctx) (string, error) — SELECT mode ORDER BY changed_at DESC LIMIT 1; returns "local" on pgx.ErrNoRows, GetCurrentConfig(ctx) (*RouterConfig, error) — same query returning full row, InsertModeChange(ctx, mode, prevMode string, changedBy *uuid.UUID) (*RouterConfig, error) — INSERT RETURNING, LogRequest(ctx, *AIRequestLog) error — INSERT with all metadata fields (NO prompt content), GetConfigHistory(ctx, limit, offset int) ([]*RouterConfig, int, error) — SELECT ORDER BY changed_at DESC with COUNT
- [X] T007 Create `internal/ai/service.go` with: Service struct (repo *Repository, primaryClient *BackendClient, cloudClient *BackendClient, modeCache atomic.Pointer[cachedMode]), cachedMode struct (mode string, fetchedAt time.Time), NewService(repo *Repository, cfg *config.Config) *Service — constructs both clients with appropriate timeouts, SeedIfEmpty(ctx context.Context) — inserts mode='local' prev_mode=NULL changed_by=NULL if router_configs is empty, StartCacheRefresh(ctx context.Context) — starts background goroutine that ticks every 3s and atomically stores current mode from DB, GetCurrentConfig(ctx) (*RouterConfig, error) stub

**Checkpoint**: Foundation ready — package compiles; no HTTP handlers yet.

---

## Phase 3: User Story 1 — Route an AI Request (Priority: P1) 🎯 MVP

**Goal**: POST /ai/complete accepts a prompt, routes it to the configured backend (local or cloud mode), and returns a valid OpenAI-format completion with X-AI-Backend, X-AI-Mode, X-Request-ID headers.

**Independent Test**: Send `POST /ai/complete` with a messages array in local mode; confirm 200 response with `choices[0].message.content` non-empty and correct response headers.

- [X] T008 [US1] Implement local and cloud Dispatch(ctx, DispatchRequest) (*ChatResponse, *AIRequestLog, error) branches in `internal/ai/service.go`: read mode from atomic cache, call appropriate client.Complete with timeout context derived from cfg.AIPrimaryTimeoutSecs, populate AIRequestLog fields (RequestID, Mode, BackendUsed, LatencyMs measured from start, Status "success"/"failed", ErrorDetail on failure), call repo.LogRequest, return response and log entry
- [X] T009 [US1] Create `internal/ai/handler_route.go` with routeHandler(svc *Service) fiber.Handler: parse body into DispatchRequest, return 400 {"error":"invalid_request","message":"messages required"} if messages empty or missing WITHOUT calling Dispatch, generate request_id if omitted, call svc.Dispatch, set response headers X-AI-Backend (backend_used), X-AI-Mode (mode), X-Request-ID (request_id), return 503 {"error":"backend_unavailable","message":"primary backend failed"} on local-mode failure and 503 {"error":"backend_unavailable","message":"cloud backend failed"} on cloud-mode failure
- [X] T010 [US1] Create `internal/ai/module.go` with RegisterRoutes(app *fiber.App, db *pgxpool.Pool, cfg *config.Config, authMW fiber.Handler, roleGuard func(...string) fiber.Handler): construct repo and service, call SeedIfEmpty, call StartCacheRefresh(context.Background()), mount POST /ai/complete and GET /ai/config under authMW group, mount PUT /ai/config and GET /ai/config/history under authMW+roleGuard("admin") group
- [X] T011 [US1] Extend `cmd/api/main.go` to import `github.com/Elder2003/nextlab-api/internal/ai` and call `ai.RegisterRoutes(app, pgPool, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard)` after existing module registrations

**Checkpoint**: POST /ai/complete functional in local and cloud modes; quickstart steps 2–4 pass.

---

## Phase 4: User Story 2 — Administrator Changes Mode (Priority: P2)

**Goal**: PUT /ai/config changes routing mode at runtime; change propagates within 5 seconds; full history preserved append-only.

**Independent Test**: PUT /ai/config {"mode":"auto"} as admin → 200 with new config; GET /ai/config confirms new mode; GET /ai/config/history shows both seed and new entry; non-admin PUT returns 403; duplicate mode returns 409.

- [X] T012 [US2] Implement SetMode(ctx, mode string, callerID uuid.UUID) (*RouterConfig, error) in `internal/ai/service.go`: read current mode via repo.GetCurrentMode, return ErrModeUnchanged if mode unchanged, call repo.InsertModeChange with prev_mode=current, atomically update modeCache immediately after insert (don't wait for 3s poll), return new RouterConfig
- [X] T013 [US2] Implement GetConfigHistory(ctx, limit, offset int) ([]*RouterConfig, int, error) in `internal/ai/repository.go`: SELECT id, mode, prev_mode, changed_by, changed_at FROM router_configs ORDER BY changed_at DESC LIMIT $1 OFFSET $2 plus SELECT COUNT(*) FROM router_configs in same transaction
- [X] T014 [US2] Create `internal/ai/handler_admin.go` with: getConfigHandler(svc) — GET /ai/config returns current RouterConfig as JSON; putConfigHandler(svc) — PUT /ai/config validates mode IN ('local','auto','cloud') returning 400 {"error":"invalid_mode"}, calls svc.SetMode returning 409 {"error":"no_change"} on ErrModeUnchanged or 200 on success; getConfigHistoryHandler(svc) — GET /ai/config/history parses limit (default 20) and offset (default 0) query params, returns {"entries":[...],"total":N}

**Checkpoint**: All quickstart steps 5–11 pass; router_configs row count increments correctly; no UPDATE/DELETE in DB.

---

## Phase 5: User Story 3 — Automatic Fallback in Auto Mode (Priority: P3)

**Goal**: In auto mode, when the primary backend fails or times out, the router transparently falls back to cloud and logs the fallback event.

**Independent Test**: Set mode to auto, stop Ollama, POST /ai/complete — expect 200 from cloud backend with X-AI-Backend: cloud; ai_request_logs shows fallback_used=true, fallback_reason=primary_error or primary_timeout.

- [X] T015 [US3] Implement auto mode branch in Dispatch() in `internal/ai/service.go`: create a derived context with cfg.AIPrimaryTimeoutSecs deadline for primary call; on any error (timeout or connection error) set logEntry.FallbackUsed=true, logEntry.FallbackReason=classifyPrimaryError(err) (returns "primary_timeout" if errors.Is(err, context.DeadlineExceeded), "primary_error" otherwise), attempt cloud client with full parent context; if cloud also fails set status="failed" with both errors; always call repo.LogRequest regardless of outcome
- [X] T016 [US3] Handle all-backends-failed case in `internal/ai/handler_route.go`: when Dispatch returns error in auto mode return 503 {"error":"backend_unavailable","message":"all backends failed","backends":["local","cloud"]}

**Checkpoint**: Quickstart steps 7–9 pass; fallback correctly logged; local mode does NOT fall back on primary failure.

---

## Phase 6: User Story 4 — Audit Logging Completeness (Priority: P4)

**Goal**: Every AI request produces exactly one log entry with required metadata; validation failures produce no log entry; no prompt/completion content in logs.

**Independent Test**: Send requests in all modes (success, fallback, failure, validation error); query ai_request_logs and confirm: one entry per successful/failed dispatch, zero entries for 400 validation rejections, no prompt_content column exists.

- [X] T017 [US4] Confirm in `internal/ai/handler_route.go` that the empty-messages 400 response is returned BEFORE any call to svc.Dispatch (so no log entry is created for invalid requests); add inline comment referencing FR-011 explaining why validation precedes dispatch
- [X] T018 [US4] Verify in `internal/ai/service.go` Dispatch() that error_detail is set to a non-nil pointer only when status="failed" (it remains nil for status="success") and that ErrorDetail contains only backend error message with no portion of the prompt or request body

**Checkpoint**: Quickstart steps 12–13 pass; constitution compliance confirmed.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Build verification, environment wiring, and end-to-end quickstart validation.

- [X] T019 [P] Run `go build ./...` from repo root and fix any compilation errors in `internal/ai/` or `cmd/api/main.go`
- [X] T020 [P] Verify `docker-compose.yml` (or `.env.example`) has placeholder entries for AI_PRIMARY_URL, AI_PRIMARY_MODEL, AI_CLOUD_URL, AI_CLOUD_API_KEY, AI_PRIMARY_TIMEOUT_SECS
- [ ] T021 Run quickstart.md verification steps 1–13 against a live docker compose stack and confirm all expected responses

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on T001 (config fields needed for NewService) — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 completion
- **US2 (Phase 4)**: Depends on Phase 2 completion; can start in parallel with US1 after Phase 2
- **US3 (Phase 5)**: Depends on US1 (Dispatch() exists); modifies Dispatch() auto branch
- **US4 (Phase 6)**: Depends on US1 and US3; verification tasks
- **Polish (Phase 7)**: Depends on all phases complete

### User Story Dependencies

- **US1 (P1)**: After Phase 2 — no dependency on US2/US3/US4
- **US2 (P2)**: After Phase 2 — can work in parallel with US1 (different files: handler_admin.go vs handler_route.go); requires service.go skeleton from T007 and T008
- **US3 (P3)**: After US1 (T008 must exist before modifying auto branch in T015)
- **US4 (P4)**: After US1 and US3 (verification of completed dispatch paths)

### Within Each User Story

- T004 → (T005 [P] with T006) → T007 → T008 → T009 → T010 → T011
- T007, T008 → T012 → T014 (T013 [P] with T012)
- T008 → T015 → T016
- T009, T015 → T017 → T018

### Parallel Opportunities

| Phase | Parallel tasks |
|---|---|
| Phase 1 | T002, T003 (both create new files) |
| Phase 2 | T005, T006 (both depend on T004 only; write different files) |
| Phase 4 | T012, T013 (different files/concerns within service and repository) |
| Phase 7 | T019, T020 (independent verification tasks) |

---

## Parallel Example: US1 Implementation

```
After T007 completes:
  └─ T008 (service Dispatch local/cloud) → T009 (handler_route.go) → T010 (module.go) → T011 (main.go)

After T007 completes (parallel track for US2):
  └─ T012 (SetMode) + T013 [P] (GetConfigHistory) → T014 (handler_admin.go)
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Setup (T001–T003)
2. Complete Phase 2: Foundational (T004–T007)
3. Complete Phase 3: US1 (T008–T011)
4. **STOP and VALIDATE**: POST /ai/complete works in local mode; quickstart steps 2–4 pass
5. Proceed to US2, US3, US4 in priority order

### Incremental Delivery

1. T001–T007 → Package compiles, migration ready
2. T008–T011 → POST /ai/complete working (MVP)
3. T012–T014 → Admin mode management working
4. T015–T016 → Auto fallback working
5. T017–T018 → Audit completeness confirmed
6. T019–T021 → Full quickstart validation

---

## Notes

- No new `go.mod` dependencies — stdlib `net/http`/`encoding/json`/`sync/atomic` only
- Constitution Principle I: `router_configs` and `ai_request_logs` are INSERT-only — no UPDATE or DELETE methods exist in repository.go
- Constitution Principle III: FR-011 — `error_detail` contains only backend error text, never prompt or completion content
- The `gracefulClose` pattern from M5 WS handler is NOT needed here — this is synchronous HTTP only
- `StartCacheRefresh(context.Background())` runs for the lifetime of the process; no shutdown hook needed at this scale
- Ollama does not require an Authorization header; Vercel AI Gateway requires `Bearer <AICloudAPIKey>`
