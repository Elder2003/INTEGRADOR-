# Research: M2 — Authentication and Roles

## Decision Log

---

### 1. JWT Library

**Decision**: `golang-jwt/jwt/v5`

**Rationale**: The maintained fork of the original `dgrijalva/jwt-go`. It is the de-facto standard for JWT in Go as of 2024+, with active maintenance, CVE patches, and Go 1.18+ generics support for type-safe claims parsing. Uses HMAC-SHA256 (HS256) with a symmetric secret key — appropriate for a single-service API where the same server that signs tokens also validates them.

**Alternatives considered**:
- `lestrrat-go/jwx` — feature-rich but heavier API surface; better suited for multi-service JWKS scenarios (M2 has no public key distribution need).
- `cristalhq/jwt` — minimal but lacks the community adoption and documentation density of `golang-jwt/jwt/v5`.

---

### 2. Password Hashing

**Decision**: `golang.org/x/crypto/bcrypt` at cost factor 12

**Rationale**: bcrypt is the established standard for password hashing (designed for slow, GPU-resistant hashing). Cost factor 12 produces ~250ms hash time on modern hardware — fast enough for user experience, slow enough to make brute-force attacks impractical. Argon2id is theoretically stronger but adds an external dependency and is overkill for the current threat model.

**Alternatives considered**:
- `argon2id` — better memory-hardness; deferred to a future security hardening pass if threat model escalates.
- Cost factor 10 — too fast (~60ms); easily brute-forced offline.
- Cost factor 14 — ~1s; degrades login UX noticeably under load.

---

### 3. Rate Limiting Backend

**Decision**: Redis-backed fixed-window counter using `go-redis/v9` (already in `go.sum`)

**Rationale**: Redis is already a required dependency (M1). A fixed-window counter is implemented with two Redis commands: `INCR key` + `EXPIRE key 60`. Key schema: `rate_limit:{ip}:{email}`. This approach is O(1) per check, requires no additional library, and persists across API pod restarts. A sliding-window is more accurate but requires a sorted set and is not worth the complexity for a 60-second window.

**Alternatives considered**:
- In-memory map with `sync.Map` — does not survive pod restarts; fails in multi-replica deployments.
- `ulule/limiter` library — well-designed but adds a dependency for functionality achievable with 15 lines of Redis code.
- Token bucket — more accurate for bursty traffic; overkill for a simple 5-attempt / 60s rule.

---

### 4. Refresh Token Storage

**Decision**: Stateful `refresh_tokens` table in PostgreSQL; token value stored as SHA-256 hash

**Rationale**: Refresh tokens must be revocable (logout, account deactivation, reuse detection). This requires server-side state — a DB table is the natural fit given PostgreSQL is already the primary store. Storing the hash of the token (not the raw value) means a DB breach does not expose replayable tokens. The `replaced_by` column links successive tokens for reuse detection without requiring family-tree traversal.

**Alternatives considered**:
- Redis-only storage — fast but Redis is not configured for persistence in M1/M2; token loss on restart is unacceptable.
- Opaque token in DB without hashing — simpler but leaves tokens exposed in a DB dump.
- JWT refresh tokens (stateless) — eliminates revocation capability; incompatible with FR-004 and FR-013.

---

### 5. Reuse Detection Mechanism

**Decision**: Single-level reuse detection via `replaced_by` + `revoked_at` columns

**Rationale**: When a refresh token RT-1 is used, it is marked `revoked_at = NOW()` and `replaced_by = RT-2`. If RT-1 is submitted again:
1. Lookup by `token_hash` finds RT-1.
2. RT-1 is already revoked → this is a reuse attempt.
3. Follow `replaced_by` to RT-2 and revoke it immediately.
4. Return 401 to both parties.

This requires no recursive queries or family-tree traversal. It handles the most common theft scenario (attacker uses stolen token before the victim notices) without the complexity of full family tracking.

**Alternatives considered**:
- Full family tracking — each token knows all ancestors; recursive revocation of the entire chain. Rejected: significant data model complexity, rarely needed at academic scale.
- No reuse detection (plain rotation) — simpler but leaves stolen tokens silently usable once the attacker refreshes ahead of the victim.

---

### 6. Access Token Format

**Decision**: Signed JWT (HS256) with claims: `sub`, `email`, `role`, `jti`, `iat`, `exp`

**Rationale**: Including `email` and `role` in the token payload avoids a DB lookup on every protected request — the middleware can extract the role directly from the token. `jti` (JWT ID, a UUID) enables future revocation if needed without changing the data model. Access tokens are intentionally stateless and short-lived (15 min).

**Signing key**: read from `JWT_SECRET` environment variable at startup; minimum 32 bytes enforced at config load.

---

### 7. New Config Fields

Three new environment variables added to `internal/config/config.go`:

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_SECRET` | yes | — | HS256 signing key (min 32 bytes) |
| `ACCESS_TOKEN_TTL` | no | `15m` | Access token validity duration |
| `REFRESH_TOKEN_TTL` | no | `168h` (7 days) | Refresh token validity duration |

---

### 8. New Go Dependency

`github.com/golang-jwt/jwt/v5` — only new dependency added in M2. All other requirements (`bcrypt`, `crypto/sha256`, `google/uuid`) are either stdlib or already in `go.sum`.
