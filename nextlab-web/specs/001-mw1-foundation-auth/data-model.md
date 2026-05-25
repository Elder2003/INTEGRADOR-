# Data Model: MW1 — Foundation & Auth

Frontend-only types. These are NOT database entities — they are TypeScript interfaces
that represent the shape of data the frontend works with.

---

## AuthState

The central auth state held in `AuthContext`.

| Field | Type | Notes |
|-------|------|-------|
| `user` | `User \| null` | null when unauthenticated |
| `accessToken` | `string \| null` | In memory only — never persisted |
| `isLoading` | `boolean` | true during initial auth check on page load |

---

## User

Derived from JWT access token claims after login. Not fetched from an endpoint.

| Field | Type | Source |
|-------|------|--------|
| `id` | `string` (UUID) | JWT claim `sub` |
| `email` | `string` | JWT claim `email` |
| `role` | `Role` | JWT claim `role` |

---

## Role

```ts
type Role = 'estudiante' | 'docente' | 'co_docente' | 'admin'
```

Maps to these home routes:

| Role | Home route |
|------|-----------|
| `estudiante` | `/student/dashboard` |
| `docente` | `/docente/dashboard` |
| `co_docente` | `/codocente/dashboard` |
| `admin` | `/admin/dashboard` |

---

## API Request / Response Shapes

### POST /auth/login

**Request:**
```ts
interface LoginRequest {
  email: string
  password: string
}
```

**Response (200):**
```ts
interface LoginResponse {
  access_token: string   // JWT — decode payload to get User
  token_type: string     // "Bearer"
  expires_in: number     // seconds until access token expires
  refresh_token: string  // store in localStorage
}
```

**Error responses:**
- `401 { error: "invalid_credentials", message: "Invalid email or password" }`
- `429 { error: "rate_limit_exceeded", message: "...", retry_after: number }`
- `403 { error: "account_inactive", message: "This account has been deactivated" }`

---

### POST /auth/refresh

**Request:**
```ts
interface RefreshRequest {
  refresh_token: string
}
```

**Response (200):** Same shape as `LoginResponse` — new token pair.

**Error responses:**
- `401 { error: "invalid_token" }` — expired or already used
- `401 { error: "token_reuse_detected" }` — security event; session terminated

---

### POST /auth/logout

Requires `Authorization: Bearer <access_token>` header.

**Request:**
```ts
interface LogoutRequest {
  refresh_token: string
}
```

**Response (200):** `{ message: "Logged out successfully" }`

---

### GET /auth/me

Requires `Authorization: Bearer <access_token>` header.

**Response (200):**
```ts
interface MeResponse {
  id: string
  email: string
  role: Role
}
```

---

## LocalStorage Keys

| Key | Value | Notes |
|-----|-------|-------|
| `nextlab_refresh_token` | `string` | Cleared on logout or refresh failure |

Access token is NEVER written to localStorage (constitution Principle IV).

---

## JWT Claims Shape

Access token payload (base64url-decoded middle segment):

```ts
interface JWTClaims {
  sub: string    // user UUID
  email: string
  role: Role
  jti: string    // token ID
  iat: number    // issued at (unix timestamp)
  exp: number    // expires at (unix timestamp)
}
```
