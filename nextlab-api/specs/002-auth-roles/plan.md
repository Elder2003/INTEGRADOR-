# Implementation Plan: M2 — Authentication and Roles

**Branch**: `002-auth-roles` | **Date**: 2026-05-17 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/002-auth-roles/spec.md`

---

## Summary

Implement a complete identity layer for the nextlab-api: credential-based login
issuing short-lived JWT access tokens (15 min) and long-lived refresh tokens (7
days), token rotation with reuse detection, logout, role-based access middleware
enforcing four roles, and a password recovery flow. All token state is stored in
PostgreSQL; rate limiting counters live in Redis (already available from M1).

**Technical approach**: `golang-jwt/jwt/v5` for HS256 signing · `bcrypt` cost 12
for password hashing · stateful refresh tokens stored as SHA-256 hashes ·
single-level reuse detection via `replaced_by` column · Redis fixed-window counter
for login rate limiting · two Fiber middlewares (`AuthMiddleware` + `RoleGuard`) ·
one new migration (`000002_auth`) · three new DB tables.

---

## Technical Context

**Language/Version**: Go 1.22

**Primary Dependencies**:
- `gofiber/fiber/v2` — HTTP framework (locked by constitution)
- `golang-jwt/jwt/v5` — JWT generation and validation *(new in M2)*
- `jackc/pgx/v5` — PostgreSQL driver (from M1)
- `redis/go-redis/v9` — Redis client, used for rate limiting (from M1)
- `golang.org/x/crypto/bcrypt` — password hashing (stdlib extension, already in go.sum)
- `crypto/sha256` + `encoding/hex` — refresh token hashing (stdlib)
- `google/uuid` — token ID generation (already in go.sum via minio-go)

**Storage**:
- PostgreSQL 16: `users`, `refresh_tokens`, `password_reset_tokens` tables
- Redis 7: rate limit counters (keys expire in 60s, no persistence needed)

**Testing**: `go test` stdlib

**Target Platform**: Linux/AMD64 (Docker Compose, same as M1)

**Performance Goals**:
- Login response ≤ 2s including bcrypt (SC-001)
- JWT middleware overhead ≤ 5ms per request (pure in-memory validation)
- Rate limit check ≤ 2ms (single Redis INCR + GET round-trip)

**Constraints**:
- Access tokens are stateless JWTs — no DB lookup on protected routes
- Refresh tokens are stateful (DB row) — required for revocation (FR-004, FR-013)
- `JWT_SECRET` must be at least 32 bytes; validated at startup with `os.Exit(1)` on failure
- bcrypt cost factor fixed at 12; not configurable (prevents accidental weakening)
- No plaintext passwords in logs, DB, or error messages anywhere in the codebase

**Scale/Scope**: ~800–1500 active students + faculty, 2 carreras (Unifranz)

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Immutable Evidence | ✅ N/A | `users` and `refresh_tokens` rows are mutable by design (account management). Append-only tables are M5+. |
| II. Async Analysis Isolation | ✅ N/A | No analysis jobs in M2. |
| III. Human Authority Over AI | ✅ N/A | No AI calls in M2. |
| IV. Strict Role-Based Data Isolation | ✅ **PRIMARY** | Role middleware is the core deliverable of M2. Four roles enforced via `RoleGuard`. `co_docente` scoped to explicit grants in later milestones; M2 establishes the identity foundation. |
| V. Dual RAG Independence | ✅ N/A | No RAG in M2. |
| VI. Non-Invasive Auditing | ✅ N/A | No audit capture in M2. |
| VII. Submission Always Accepted | ✅ N/A | No submission endpoints in M2. |

**Result: PASS** — No violations. Constitution Principle IV is the direct driver of this milestone.

---

## Project Structure

### Documentation (this feature)

```text
specs/002-auth-roles/
├── plan.md                  ← this file
├── research.md              ← Phase 0 output
├── data-model.md            ← Phase 1 output
├── quickstart.md            ← Phase 1 output
├── contracts/
│   └── auth.md              ← Phase 1 output
└── tasks.md                 ← Phase 2 output (/speckit-tasks)
```

### Source Code Changes

```text
nextlab-api/
├── cmd/
│   └── api/
│       └── main.go              # Upgrade pgx.Conn → pgxpool.Pool; call auth.RegisterRoutes
├── internal/
│   ├── auth/                    # Expand from stub to full module
│   │   ├── module.go            # RegisterRoutes(app, db, redis, cfg) — replaces stub
│   │   ├── handler.go           # HTTP handlers: Login, Refresh, Logout, ForgotPassword, ResetPassword
│   │   ├── service.go           # Business logic: authenticate, rotateToken, revokeToken, resetPassword
│   │   ├── repository.go        # DB queries: GetUserByEmail, CreateRefreshToken, GetRefreshToken, etc.
│   │   ├── middleware.go        # AuthMiddleware (JWT validate), RoleGuard(roles ...string)
│   │   └── model.go             # Go structs: User, RefreshToken, PasswordResetToken, AccessTokenClaims
│   └── config/
│       └── config.go            # Add JWT_SECRET, ACCESS_TOKEN_TTL, REFRESH_TOKEN_TTL fields
├── migrations/
│   ├── 000002_auth.up.sql       # CREATE users, refresh_tokens, password_reset_tokens + indexes
│   └── 000002_auth.down.sql     # DROP tables in reverse dependency order
├── go.mod                       # Add golang-jwt/jwt/v5
├── go.sum                       # Updated by go mod tidy
└── .env.example                 # Add JWT_SECRET, ACCESS_TOKEN_TTL, REFRESH_TOKEN_TTL
```

**Structure Decision**: Auth logic lives entirely in `internal/auth/`. The module
exposes a single `RegisterRoutes` function called from `cmd/api/main.go`. Adding
M2 touches only `cmd/api/main.go` (two changes) and creates new files in
`internal/auth/` — zero changes to any other `internal/` package.

---

## Phase 0: Research Summary

See [research.md](research.md) for full decision log. Key decisions:

| Topic | Decision |
|---|---|
| JWT library | `golang-jwt/jwt/v5` — maintained, HS256 |
| Password hashing | `bcrypt` cost 12 (~250ms, GPU-resistant) |
| Rate limiting | Redis fixed-window counter; key: `rate_limit:{hash(ip+email)}` |
| Refresh token storage | PostgreSQL table; stored as SHA-256 hash |
| Reuse detection | Single-level: `replaced_by` column + cascade revoke on reuse |
| Access token payload | `sub`, `email`, `role`, `jti`, `iat`, `exp` |
| New config fields | `JWT_SECRET` (required, min 32 bytes), `ACCESS_TOKEN_TTL` (default 15m), `REFRESH_TOKEN_TTL` (default 168h) |
| pgx upgrade | `pgx.Conn` → `pgxpool.Pool` to handle concurrent bcrypt-heavy requests |

---

## Phase 1: Design Summary

### Data Model

See [data-model.md](data-model.md). Three new tables in migration 000002:

| Table | Purpose |
|---|---|
| `users` | Identity: email, bcrypt hash, role, active flag |
| `refresh_tokens` | Session tokens: hash, expiry, revocation, rotation chain |
| `password_reset_tokens` | Single-use reset tokens: hash, expiry, used flag |

### API Contract

See [contracts/auth.md](contracts/auth.md). Five endpoints + two middlewares:

| Endpoint | Auth | Purpose |
|---|---|---|
| `POST /auth/login` | No | Issue access + refresh tokens |
| `POST /auth/refresh` | No (RT) | Rotate refresh token, issue new access token |
| `POST /auth/logout` | Yes | Revoke refresh token |
| `POST /auth/forgot-password` | No | Generate password reset token |
| `POST /auth/reset-password` | No | Apply new password, revoke all refresh tokens |
| `AuthMiddleware` | — | Validate JWT on protected routes |
| `RoleGuard(roles)` | — | Enforce role-based access per endpoint |

### Quickstart

See [quickstart.md](quickstart.md). After `docker compose up --build`:
1. Seed admin user via `psql`
2. `POST /auth/login` → access token + refresh token
3. Use `Authorization: Bearer` header on protected routes
4. `POST /auth/refresh` → rotate tokens
5. `POST /auth/logout` → invalidate session

---

## Key Implementation Notes

### Module registration pattern

`internal/auth/module.go` exposes:

```go
func RegisterRoutes(app *fiber.App, db *pgxpool.Pool, rdb *redis.Client, cfg *config.Config)
```

`cmd/api/main.go` calls this after existing health route. No other
files change. This is the pattern all future modules (M3, M4, …) will follow.

### pgxpool upgrade

M1 used `pgx.Conn` (single connection). M2 upgrades to `pgxpool.Pool`
because bcrypt calls block the goroutine for ~250ms — a single connection would
serialize all concurrent login requests. `pgxpool.Pool` implements the same
`Ping(ctx)` interface used by `health.Checker`, so the health check requires no changes.

```go
// cmd/api/main.go — replace:
pgConn, err := pgx.Connect(ctx, cfg.DatabaseURL)
// with:
pgPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
```

### JWT signing key validation at startup

`config.go` enforces minimum key length:

```
if len(cfg.JWTSecret) < 32 → slog.Error("JWT_SECRET too short") + os.Exit(1)
```

### Rate limiting flow

```
On each POST /auth/login:
  key  = "rate_limit:"  + hex(sha256(clientIP + ":" + email))[:16]
  cool = "rate_cool:"   + hex(sha256(clientIP + ":" + email))[:16]

  if GET cool → non-nil: return 429 with TTL(cool) as retry_after
  count = INCR key
  if count == 1: EXPIRE key 60
  if count > 5: SET cool "1" EX 60; return 429
  → proceed to credential check
```

### Anti-enumeration in login and forgot-password

- Login: always run `bcrypt.CompareHashAndPassword` even when the email is not
  found (compare against a pre-computed dummy hash) so response time is constant.
- Forgot-password: always return HTTP 200; only include `reset_token` in the
  response body when the email exists.
