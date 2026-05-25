# Research: MW1 — Foundation & Auth

## Decision 1: No Self-Registration Endpoint

**Decision:** The spec's User Story 3 (Account Registration via email/password form) is removed from MW1 scope.

**Finding:** The `nextlab-api` auth module exposes no `POST /auth/register` endpoint. Users are created exclusively through:
- `POST /admin/student-profiles` (admin creates student accounts — M3)
- `GET /auth/google` + `GET /auth/google/callback` (Google OAuth for students — exists in M1)
- Admin creates docente/co_docente accounts directly in the database at setup time

**Impact on spec:** US3 is descoped. The login page will include a "Sign in with Google" button as the student entry point for users without a pre-assigned password. No separate registration page is needed.

**Alternatives considered:**
- Add a register endpoint to the API — rejected; out of scope for MW1, requires admin workflow design
- Keep the register page as a placeholder — rejected; would be dead UI with no backing endpoint

---

## Decision 2: User Info Extraction After Login

**Decision:** Decode the JWT access token payload on the client to extract `role`, `email`, and `sub` (user ID). Do NOT call `GET /auth/me` after every login.

**Finding:** `LoginResponse` returns `{ access_token, token_type, expires_in, refresh_token }` — no user fields. The access token is a standard JWT whose middle segment (base64url) contains `{ sub, email, role, jti, iat, exp }`.

**Implementation:** After login, base64url-decode `access_token.split('.')[1]` and `JSON.parse` the result. This is safe on the client — the client does not need to verify the signature; the API does that on every authenticated request.

**Alternatives considered:**
- Call `GET /auth/me` after login — works but adds a round trip; rejected in favor of the zero-cost JWT decode
- Store role in localStorage — rejected; constitution Principle IV forbids storing access token data persistently

---

## Decision 3: Axios Interceptor — Refresh Race Condition

**Decision:** Use a single shared Promise to serialize concurrent refresh attempts. If a refresh is already in flight, queue subsequent 401 retries onto it instead of triggering a new `/auth/refresh` call.

**Rationale:** Multiple simultaneous expired requests (e.g., parallel API calls on a dashboard) would each trigger a refresh. The API uses refresh token rotation — the second refresh call would receive a revoked token and return `token_reuse_detected`, terminating the session unexpectedly.

**Implementation pattern:**
```
let refreshPromise: Promise<string> | null = null

interceptor on 401:
  if (!refreshPromise) {
    refreshPromise = callRefresh().finally(() => refreshPromise = null)
  }
  newToken = await refreshPromise
  retry original request with newToken
```

---

## Decision 4: Auth State Initialization (Avoiding FOUC)

**Decision:** On app load, check `localStorage` for a stored refresh token. If present, call `POST /auth/refresh` silently before rendering any routes. Show a full-screen loading spinner during this check. Only render routes after the check resolves (success or failure).

**Rationale:** Without this, a signed-in user who refreshes the page sees a flash of the login page before being redirected. This is jarring and breaks US4 (Session Persistence).

**Implementation:** `AuthContext` initializes with `isLoading: true`. The context's `useEffect` runs on mount, attempts refresh if a stored token exists, then sets `isLoading: false`. The router renders a `<LoadingSpinner />` while `isLoading` is true.

---

## Decision 5: Redirect-Back Implementation

**Decision:** Use React Router v7's `location.state` to pass the intended URL through the login redirect, then redirect to it after successful sign-in.

**Implementation:**
```
// ProtectedRoute — before redirecting unauthenticated user:
<Navigate to="/login" state={{ from: location }} replace />

// LoginPage — after successful login:
const from = location.state?.from?.pathname ?? roleHomeFor(user.role)
navigate(from, { replace: true })
```

**Alternatives considered:**
- Store intended URL in `sessionStorage` — works but adds complexity; state passing is the React Router idiomatic approach
- Always redirect to role home — rejected; spec clarification Q1 explicitly chose redirect-back

---

## Decision 6: Rate Limiting Display

**Decision:** Display the API's `retry_after` value directly when a 429 response is received. Show: "Too many sign-in attempts. Try again in {retry_after} seconds."

**Finding:** The backend already handles rate limiting (5 failed attempts per IP/email within a window). The `ErrorResponse` includes `retry_after: number` in 429 responses. The frontend only needs to display this value — no client-side rate limit logic.

**Constitution check:** Displaying `retry_after` from the API response is display-only. This does NOT violate Principle VII (Thin Client) — the frontend is not enforcing the rate limit, only showing what the API reported.

---

## Decision 7: Google OAuth Flow

**Decision:** The login page includes a "Sign in with Google" button that redirects to `GET /auth/google`. The OAuth callback is handled server-side; the API redirects back to the frontend with tokens in query params or a redirect URL.

**Note:** Google OAuth flow details (redirect URL, token delivery mechanism) depend on the `nextlab-api` implementation of `GoogleCallbackHandler`. This should be verified against the running API before implementing the Google button. If the callback delivers tokens differently than expected, this is a Day 1 integration task.

**Scope for MW1:** Add the Google button to the login page. Full Google OAuth integration is a stretch goal — the primary flow is email/password login.
