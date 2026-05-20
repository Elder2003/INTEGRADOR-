# Tasks: M2 — Authentication and Roles

**Input**: Design documents from `specs/002-auth-roles/`

**Prerequisites**: plan.md ✅ | spec.md ✅ | research.md ✅ | data-model.md ✅ | contracts/auth.md ✅ | quickstart.md ✅

**Tests**: Not requested — no test tasks generated.

---

## Phase 1: Setup

**Purpose**: Add the new dependency, extend config, and prepare the environment before any feature work.

- [X] T001 Add `github.com/golang-jwt/jwt/v5` to `go.mod` require block; run `docker run --rm -v "$(pwd):/workspace" -w /workspace golang:1.22-alpine go mod tidy` to fetch it and update `go.sum`
- [X] T002 [P] Add `JWT_SECRET`, `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL` entries to `.env.example` with values `your-secret-key-min-32-characters-long`, `15m`, `168h`
- [X] T003 [P] Add `JWTSecret string`, `AccessTokenTTL time.Duration`, `RefreshTokenTTL time.Duration` to `internal/config/config.go`; read from env vars `JWT_SECRET`, `ACCESS_TOKEN_TTL` (default `15m`), `REFRESH_TOKEN_TTL` (default `168h`); add startup validation: `if len(cfg.JWTSecret) < 32 { slog.Error(...); os.Exit(1) }`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Database schema, pgxpool upgrade, model structs, repository layer, and module scaffold — all must be complete before any user story can run end-to-end.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T004 Create `migrations/000002_auth.up.sql` with: `CREATE TABLE users` (id UUID PK, email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, role TEXT CHECK IN ('admin','docente','co_docente','estudiante') NOT NULL, is_active BOOLEAN DEFAULT TRUE NOT NULL, created_at/updated_at TIMESTAMPTZ); `CREATE TABLE refresh_tokens` (id UUID PK, user_id UUID FK→users CASCADE, token_hash TEXT UNIQUE NOT NULL, replaced_by UUID FK→refresh_tokens, expires_at TIMESTAMPTZ NOT NULL, revoked_at TIMESTAMPTZ, created_at TIMESTAMPTZ); `CREATE TABLE password_reset_tokens` (id UUID PK, user_id UUID FK→users CASCADE, token_hash TEXT UNIQUE NOT NULL, expires_at TIMESTAMPTZ NOT NULL, used_at TIMESTAMPTZ, created_at TIMESTAMPTZ); plus indexes on `users.email`, `refresh_tokens.token_hash`, `refresh_tokens.user_id`, `password_reset_tokens.token_hash`
- [X] T005 Create `migrations/000002_auth.down.sql` with: `DROP TABLE IF EXISTS password_reset_tokens; DROP TABLE IF EXISTS refresh_tokens; DROP TABLE IF EXISTS users;` (reverse dependency order)
- [X] T006 Upgrade `cmd/api/main.go`: replace `pgx.Connect` + `*pgx.Conn` with `pgxpool.New` + `*pgxpool.Pool`; add import `github.com/jackc/pgx/v5/pgxpool`; update `health.NewChecker` call to pass `pgPool`; remove single-connection import `github.com/jackc/pgx/v5`
- [X] T007 [P] Create `internal/auth/model.go` with Go structs: `User{ID, Email, PasswordHash, Role, IsActive, CreatedAt, UpdatedAt}`, `RefreshToken{ID, UserID, TokenHash, ReplacedBy *uuid.UUID, ExpiresAt, RevokedAt *time.Time, CreatedAt}`, `PasswordResetToken{ID, UserID, TokenHash, ExpiresAt, UsedAt *time.Time, CreatedAt}`, `AccessTokenClaims{Sub, Email, Role, JTI string; jwt.RegisteredClaims}`, `LoginRequest{Email, Password string}`, `LoginResponse{AccessToken, TokenType string; ExpiresIn int; RefreshToken string}`, `RefreshRequest{RefreshToken string}`, `LogoutRequest{RefreshToken string}`, `ForgotPasswordRequest{Email string}`, `ResetPasswordRequest{Token, NewPassword string}`, `ErrorResponse{Error, Message, CorrelationID string; RetryAfter *int}`
- [X] T008 [P] Create `internal/auth/repository.go` with functions using `*pgxpool.Pool`: `GetUserByEmail(ctx, db, email) (*User, error)`, `CreateRefreshToken(ctx, db, userID, tokenHash, replacedBy *uuid.UUID, expiresAt) (*RefreshToken, error)`, `GetRefreshTokenByHash(ctx, db, tokenHash) (*RefreshToken, error)`, `RevokeRefreshToken(ctx, db, id uuid.UUID, replacedBy *uuid.UUID) error`, `RevokeAllUserRefreshTokens(ctx, db, userID) error`, `UpdateUserPassword(ctx, db, userID, newHash) error`, `CreatePasswordResetToken(ctx, db, userID, tokenHash, expiresAt) error`, `GetPasswordResetTokenByHash(ctx, db, tokenHash) (*PasswordResetToken, error)`, `MarkPasswordResetTokenUsed(ctx, db, id uuid.UUID) error`
- [X] T009 Replace stub content of `internal/auth/module.go` with `RegisterRoutes(app *fiber.App, db *pgxpool.Pool, rdb *redis.Client, cfg *config.Config)`; create a `/auth` route group on the fiber app; leave handler registrations as `// TODO: registered in US1–US4` placeholders so the file compiles
- [X] T010 Add call `auth.RegisterRoutes(app, pgPool, redisClient, cfg)` to `cmd/api/main.go` after the health route; add import `github.com/Elder2003/nextlab-api/internal/auth`; verify `docker compose build api` compiles successfully with no errors

**Checkpoint**: `docker compose build api` succeeds. Migration 000002 applies on next startup. All auth structs and repository functions exist and compile.

---

## Phase 3: User Story 1 — Credential-Based Login (Priority: P1) 🎯 MVP

**Goal**: `POST /auth/login` accepts email + password and returns a working JWT access token and refresh token.

**Independent Test**: Seed admin user via psql → `curl -X POST /auth/login` with valid credentials → 200 with `access_token` and `refresh_token`. Wrong password → 401. 6th rapid attempt → 429.

- [X] T011 [P] [US1] Create `internal/auth/service.go` with `Authenticate(ctx, db, email, password string) (*LoginResponse, error)`: call `GetUserByEmail`; if not found run `bcrypt.CompareHashAndPassword` against a pre-computed dummy hash and return `ErrInvalidCredentials` (constant-time anti-enumeration); if found and `!user.IsActive` return `ErrAccountInactive`; call `bcrypt.CompareHashAndPassword(user.PasswordHash, password)`; on match generate HS256 JWT with claims `{sub: userID, email, role, jti: newUUID, iat, exp: now+AccessTokenTTL}`; generate crypto-random 32-byte refresh token UUID; hash it with `sha256`; call `CreateRefreshToken`; return `LoginResponse`
- [X] T012 [P] [US1] Add rate limiter helpers to `internal/auth/service.go`: `CheckRateLimit(ctx, rdb, ip, email string) (blocked bool, retryAfter int, err error)` — key `rate_limit:<hex(sha256(ip+":"+email))[:16]>`, cooldown key `rate_cool:<same>`: if cooldown key exists return blocked=true with TTL; `RecordFailedAttempt(ctx, rdb, ip, email string) error` — INCR attempt key, EXPIRE 60s on first increment, if count > 5 SET cooldown key EX 60
- [X] T013 [US1] Create `internal/auth/handler.go` with `LoginHandler(svc *Service) fiber.Handler`: parse `LoginRequest`; extract client IP from `c.IP()`; call `CheckRateLimit` → 429 if blocked; call `Authenticate` → 401 on `ErrInvalidCredentials`, 403 on `ErrAccountInactive`; on success return `LoginResponse`; call `RecordFailedAttempt` on any credential error; register `POST /auth/login` in `internal/auth/module.go`
- [X] T014 [US1] Verify US1 end-to-end: `docker compose up --build -d`, seed admin user per `quickstart.md` step 2, run curl steps 3 and 6-attempt rate limit test from `quickstart.md`; confirm 200 on valid credentials, 401 on wrong password, 429 after 5 failures within 60 seconds

**Checkpoint**: Login works. Tokens are returned. Rate limiter fires at 5 failures.

---

## Phase 4: User Story 2 — Token Refresh Without Re-Login (Priority: P2)

**Goal**: `POST /auth/refresh` rotates the refresh token and returns a new access token. `POST /auth/logout` invalidates the session. Reuse of a rotated refresh token triggers cascade revocation.

**Independent Test**: Login → refresh with returned RT → verify new AT and new RT → try old RT → 401. Login → logout → try refresh → 401. Login → manually revoke RT in DB → refresh → 401.

- [X] T015 [P] [US2] Add `RotateRefreshToken(ctx, db, rawToken string) (*LoginResponse, error)` to `internal/auth/service.go`: hash rawToken with sha256; `GetRefreshTokenByHash`; if not found → `ErrInvalidToken`; if `RevokedAt != nil` AND `ReplacedBy != nil` → reuse detected: call `RevokeRefreshToken(ReplacedBy)`, log security event, return `ErrTokenReuseDetected`; if `RevokedAt != nil` → `ErrInvalidToken`; if `ExpiresAt < now` → `ErrInvalidToken`; get user, check `IsActive`; generate new token pair; call `CreateRefreshToken(newToken)`; call `RevokeRefreshToken(oldID, replacedBy=newTokenID)`; return `LoginResponse`
- [X] T016 [P] [US2] Add `RevokeSession(ctx, db, userID uuid.UUID, rawToken string) error` to `internal/auth/service.go`: hash rawToken; `GetRefreshTokenByHash`; verify `token.UserID == userID`; if already revoked return nil (idempotent); call `RevokeRefreshToken(token.ID, nil)`
- [X] T017 [US2] Add `RefreshHandler` and `LogoutHandler` to `internal/auth/handler.go`; `LogoutHandler` requires the request to include `refresh_token` in body (access token verified by AuthMiddleware applied in module.go); register `POST /auth/refresh` (no auth) and `POST /auth/logout` (with AuthMiddleware, added as route-level middleware) in `internal/auth/module.go`
- [X] T018 [US2] Verify US2 end-to-end per `quickstart.md` steps 5–6: refresh returns new tokens; old refresh token rejected; logout invalidates session; reuse of a rotated token (submit old RT after refresh) returns 401 with `token_reuse_detected` error code

**Checkpoint**: Session rotation works. Logout invalidates. Reuse detection fires.

---

## Phase 5: User Story 3 — Role-Based Access Control (Priority: P3)

**Goal**: Every protected endpoint in the system can declare required roles. Requests without a valid token get 401. Requests with insufficient role get 403. `admin` passes all guards. `co_docente` passes `docente`-level guards.

**Independent Test**: Obtain tokens for users with different roles (seed docente, estudiante users) → call `GET /auth/me` with each → 200 for all authenticated; call a docente-only route as estudiante → 403; call without token → 401.

- [X] T019 [US3] Create `internal/auth/middleware.go` with `AuthMiddleware(cfg *config.Config) fiber.Handler`: extract `Authorization: Bearer <token>` header; parse JWT with `golang-jwt/jwt/v5` using `cfg.JWTSecret`; on invalid/expired/missing token return 401 `ErrorResponse{Error:"unauthorized"}`; on success set `fiber.Locals`: `user_id`, `user_email`, `user_role`; pass correlation_id from existing locals through to error response
- [X] T020 [US3] Add `RoleGuard(roles ...string) fiber.Handler` to `internal/auth/middleware.go`: read `user_role` from `fiber.Locals`; `admin` always passes; `co_docente` passes any guard that includes `docente`; if role not in allowed list return 403 `ErrorResponse{Error:"forbidden"}`
- [X] T021 [US3] Register `GET /auth/me` in `internal/auth/module.go` behind `AuthMiddleware` (returns `{id, email, role}` from fiber.Locals — no DB call); verify role enforcement per `quickstart.md` step 7 using tokens from seeded users with `docente`, `estudiante`, and `admin` roles; confirm `co_docente` passes docente-level guard

**Checkpoint**: Auth middleware and role guard work. `GET /auth/me` returns current user. Wrong role → 403.

---

## Phase 6: User Story 4 — Password Recovery (Priority: P4)

**Goal**: `POST /auth/forgot-password` generates a single-use reset token (returned in response body for M2). `POST /auth/reset-password` consumes the token, updates the password, and revokes all active sessions.

**Independent Test**: Request reset for known email → get `reset_token` in response → submit token with new password → 200 → login with new password succeeds → login with old password fails → submit same reset token again → 400.

- [X] T022 [P] [US4] Add `GeneratePasswordResetToken(ctx, db, email string) (rawToken string, err error)` to `internal/auth/service.go`: `GetUserByEmail`; if not found return empty string, nil (no-op, no enumeration); generate `crypto/rand` 32-byte token; base64url-encode as raw token; hash with sha256; call `CreatePasswordResetToken(userID, hash, expiresAt=now+1h)`; return rawToken
- [X] T023 [P] [US4] Add `ResetPassword(ctx, db, rawToken, newPassword string) error` to `internal/auth/service.go`: validate `len(newPassword) >= 8`; hash rawToken with sha256; `GetPasswordResetTokenByHash`; check `UsedAt == nil` and `ExpiresAt > now`; hash newPassword with `bcrypt.GenerateFromPassword(cost=12)`; call `UpdateUserPassword`; call `MarkPasswordResetTokenUsed`; call `RevokeAllUserRefreshTokens` (forces re-login after reset)
- [X] T024 [US4] Add `ForgotPasswordHandler` and `ResetPasswordHandler` to `internal/auth/handler.go`; register `POST /auth/forgot-password` and `POST /auth/reset-password` (no auth required) in `internal/auth/module.go`; verify per `quickstart.md` step 8: reset token returned in response, password updated, old password rejected, re-use of token returns 400

**Checkpoint**: All 5 auth endpoints work. Full login → refresh → logout → reset cycle functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Security event logging, env documentation, and final validation sweep.

- [X] T025 [P] Add structured `slog.Warn` entries to `internal/auth/service.go` for each security event: failed login attempt (`event:"login_failed"`, `email`, `ip`), rate limit activation (`event:"rate_limit_hit"`, `ip`), refresh token reuse detection (`event:"token_reuse_detected"`, `user_id`), logout (`event:"session_revoked"`, `user_id`); include `correlation_id` from ctx in each entry (per FR-015)
- [X] T026 [P] Update `.env.example`: add comment above `JWT_SECRET` explaining minimum 32-char requirement; add `ACCESS_TOKEN_TTL=15m` and `REFRESH_TOKEN_TTL=168h`; run `docker compose up --build -d` and verify `docker compose logs api` shows `"version":2` in migrations log line
- [X] T027 Run full validation per `quickstart.md` SC checklist: SC-001 (login < 2s), SC-002 (expired token rejected), SC-003 (role guard 100%), SC-004 (logout invalidates RT), SC-005 (reset token single-use), SC-006 (no enumeration — same 401 for bad email vs bad password), SC-007 (50 concurrent via `ab`); all pass before marking M2 complete

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — T001/T002/T003 can run immediately, T002 and T003 in parallel
- **Foundational (Phase 2)**: Depends on Setup — T004–T010 must complete before any US phase; T007 and T008 can run in parallel after T004
- **US1 (Phase 3)**: Depends on Foundational — T011 and T012 can run in parallel; T013 depends on both
- **US2 (Phase 4)**: Depends on Foundational + US1 (uses Authenticate + service pattern) — T015 and T016 in parallel; T017 depends on both
- **US3 (Phase 5)**: Depends on Foundational — can start in parallel with US1/US2 after Foundational; T019 and T020 in parallel; T021 depends on both
- **US4 (Phase 6)**: Depends on Foundational — T022 and T023 in parallel; T024 depends on both
- **Polish (Phase 7)**: Depends on all US phases complete; T025 and T026 in parallel

### User Story Dependencies

- **US1 (P1)**: No dependency on other user stories — implement first
- **US2 (P2)**: Depends on US1 (uses `Authenticate` service, `LoginResponse` shape, same repository layer)
- **US3 (P3)**: Independent of US1/US2 — middleware reads JWT claims set by login; can be developed in parallel with US2
- **US4 (P4)**: Independent of US1–US3 — only requires repository and config from Foundational

### Parallel Opportunities

```
Phase 1:  T001 → (T002 ∥ T003)
Phase 2:  T004 → T005 → (T006 ∥ T007 ∥ T008) → T009 → T010
Phase 3:  (T011 ∥ T012) → T013 → T014
Phase 4:  (T015 ∥ T016) → T017 → T018
Phase 5:  (T019 ∥ T020) → T021
Phase 6:  (T022 ∥ T023) → T024
Phase 7:  (T025 ∥ T026) → T027
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1 (Setup) — 3 tasks
2. Complete Phase 2 (Foundational) — 7 tasks
3. Complete Phase 3 (US1: Login) — 4 tasks
4. **STOP and VALIDATE**: login returns tokens, rate limiter fires, wrong password → 401
5. Proceed to US2–US4 incrementally

### Incremental Delivery

| After phase | What works |
|---|---|
| Phase 1–2 | Project compiles with auth scaffolding; migration 000002 applies |
| + Phase 3 | Login, tokens issued, rate limiting |
| + Phase 4 | Refresh, logout, reuse detection |
| + Phase 5 | Role enforcement on all current + future routes |
| + Phase 6 | Password recovery |
| + Phase 7 | Security logging, fully validated |

---

## Notes

- `[P]` = parallelizable (different files, no shared state dependency)
- `[USN]` = maps task to user story for traceability
- No test tasks — add via `/speckit-tasks` re-run with TDD flag if desired
- Commit after each checkpoint (end of each phase minimum)
- The `pgxpool` upgrade in T006 is the only change to a pre-existing file outside `internal/auth/`
- The anti-enumeration dummy-hash compare (T011) is mandatory — do not skip it
