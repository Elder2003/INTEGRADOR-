# UI Contracts: MW1 — Foundation & Auth

Interface contracts for the components and modules built in MW1.

---

## AuthContext

**File:** `src/context/AuthContext.tsx`

```ts
interface AuthContextValue {
  user: User | null
  accessToken: string | null
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
}
```

**Guarantees:**
- `user` is non-null if and only if `accessToken` is non-null
- `isLoading` is `true` only during the initial page-load auth check — never during user-triggered actions
- `login()` throws with an `ApiError` on failure — callers are responsible for catching
- `logout()` clears both in-memory token and `localStorage` before resolving, regardless of API response

---

## ProtectedRoute

**File:** `src/components/common/ProtectedRoute.tsx`

```ts
interface ProtectedRouteProps {
  allowedRoles: Role[]
}
```

**Behaviour:**
- If `isLoading` → render `<LoadingSpinner />`
- If `user === null` → `<Navigate to="/login" state={{ from: location }} replace />`
- If `user.role` not in `allowedRoles` → `<Navigate to={roleHomeFor(user.role)} replace />`
- Otherwise → `<Outlet />`

---

## LoginPage

**File:** `src/pages/auth/LoginPage.tsx`

**Inputs:** email (text), password (password)  
**Validation (UI only, before submit):** both fields non-empty; email contains `@`  
**On submit:**
1. Call `AuthContext.login(email, password)`
2. On success → navigate to `location.state?.from?.pathname ?? roleHomeFor(user.role)`
3. On `invalid_credentials` → show "Invalid email or password" inline below form
4. On `rate_limit_exceeded` → show "Too many attempts. Try again in {retry_after}s"
5. On `account_inactive` → show "This account has been deactivated"
6. On network error → show "Unable to reach the server. Check your connection."

**Loading state:** submit button shows spinner and is disabled while request is in flight

---

## axios instance (lib/api.ts)

**Guarantees:**
- Every request includes `Authorization: Bearer <accessToken>` if a token exists in context
- On `401` response: triggers one silent refresh attempt before retrying
- If refresh succeeds: retries original request with new token, transparent to caller
- If refresh fails: clears tokens, redirects to `/login`, rejects the original promise
- Concurrent `401`s share a single refresh promise (no token reuse — see research Decision 3)

---

## Router

**File:** `src/router/index.tsx`

| Path | Component | Allowed roles |
|------|-----------|--------------|
| `/` | Redirect → role home or `/login` | any |
| `/login` | LoginPage | unauthenticated only |
| `/student/*` | student pages | `estudiante` |
| `/docente/*` | docente pages | `docente` |
| `/codocente/*` | codocente pages | `co_docente` |
| `/admin/*` | admin pages | `admin` |
| `*` | NotFoundPage | any |

Authenticated users who navigate to `/login` are immediately redirected to their role home.
