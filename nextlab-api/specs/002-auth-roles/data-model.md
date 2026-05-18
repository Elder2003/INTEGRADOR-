# Data Model: M2 — Authentication and Roles

## New Migration

`migrations/000002_auth.up.sql` / `000002_auth.down.sql`

---

## Tables

### `users`

Central identity table. One row per person with system access.

```sql
CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL CHECK (role IN ('admin', 'docente', 'co_docente', 'estudiante')),
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

**Constraints**:
- `email` is the login identifier; must be unique across all roles.
- `password_hash` stores the bcrypt hash (cost 12); plaintext is never written.
- `role` is an enum-like TEXT with a CHECK constraint — no separate roles table needed at M2 scale.
- `is_active = FALSE` disables login and invalidates refresh tokens; access tokens expire naturally (≤15 min window).
- `updated_at` is set by the application on every UPDATE, not via trigger, to keep the migration simple.

---

### `refresh_tokens`

Tracks all issued refresh tokens for revocation and reuse detection.

```sql
CREATE TABLE refresh_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    replaced_by UUID        REFERENCES refresh_tokens(id),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_token_hash  ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_user_id     ON refresh_tokens(user_id);
```

**Constraints**:
- `token_hash` is the SHA-256 hex digest of the raw token value; the raw value is only held in memory and returned to the client once.
- `replaced_by` points to the successor token created during rotation. A non-null `replaced_by` combined with a non-null `revoked_at` signals a completed (normal) rotation. A non-null `revoked_at` with a null `replaced_by` signals a logout or forced revocation.
- Reuse detection: if a request arrives for a token where `revoked_at IS NOT NULL`, follow `replaced_by` and revoke that token as well.
- `ON DELETE CASCADE` — deleting a user removes all their refresh tokens automatically.

**State transitions**:

```
ISSUED → (used) → ROTATED (revoked_at set, replaced_by set)
ISSUED → (logout) → REVOKED (revoked_at set, replaced_by null)
ISSUED → (account deactivated) → REVOKED (revoked_at set, replaced_by null)
ROTATED → (replaced_by reused) → CASCADE REVOKE (replaced_by.revoked_at set)
```

---

### `password_reset_tokens`

Single-use, time-limited tokens for the forgotten-password flow.

```sql
CREATE TABLE password_reset_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prt_token_hash ON password_reset_tokens(token_hash);
```

**Constraints**:
- `token_hash` is the SHA-256 hex digest of the raw reset token.
- A token is invalid if `expires_at < NOW()` OR `used_at IS NOT NULL`.
- Multiple outstanding reset tokens for the same user are allowed; only the most recent is communicated to the user (older ones remain valid until they expire, but this is not a security concern since the password change invalidates all of them).
- `ON DELETE CASCADE` — deleting a user removes all their reset tokens.

---

## Application-Layer Entities (Go structs, no DB table)

### `AccessTokenClaims`

Embedded in signed JWT. Extracted by `AuthMiddleware` on every protected request.

| Field | Type | Description |
|---|---|---|
| `sub` | string (UUID) | User ID |
| `email` | string | User email |
| `role` | string | One of: `admin`, `docente`, `co_docente`, `estudiante` |
| `jti` | string (UUID) | Unique token identifier |
| `iat` | int64 (Unix) | Issued-at timestamp |
| `exp` | int64 (Unix) | Expiry timestamp (iat + 15 min) |

### `LoginRequest` / `LoginResponse`

See `contracts/auth.md` for the full HTTP contract.

---

## Entity Relationships

```
users (1) ──< refresh_tokens (N)      [user owns many sessions]
users (1) ──< password_reset_tokens (N) [user owns many reset requests]
refresh_tokens (1) ──> refresh_tokens (0..1)  [replaced_by self-reference]
```

---

## Seed Data

A seed admin account must be inserted manually (or via a seed migration) before the system can be used:

```sql
-- Insert after running migrations. password = 'admin123' (bcrypt hash, cost 12)
INSERT INTO users (email, password_hash, role)
VALUES (
    'admin@unifranz.edu.bo',
    '$2a$12$QKpjYMvQIXgcZZaPam8Mw.BVmKi2WRuGw/G0xdUAtugAmYwTdOmgm', -- password: admin123, cost 12
    'admin'
);
```

The seed hash must be generated outside the application and inserted directly. The application never stores or logs plaintext passwords.
