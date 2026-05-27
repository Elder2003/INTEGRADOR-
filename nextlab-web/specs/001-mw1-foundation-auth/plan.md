# Implementation Plan: MW1 — Foundation & Auth

**Branch**: `main` | **Date**: 2026-05-24 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-mw1-foundation-auth/spec.md`

---

## Summary

Build the complete authentication shell for `nextlab-web`: axios client with JWT
interceptor, AuthContext with role-based state, React Router v7 protected routes
with redirect-back, and a login page. No self-registration page — the API has no
register endpoint; users are created by admins. Role is extracted from the JWT
access token claims after login.

---

## Technical Context

**Language/Version**: TypeScript 5.9 (strict mode)

**Primary Dependencies**: React 19, React Router v7, axios, Tailwind CSS v4, Lucide React

**Storage**: `localStorage` for refresh token only (`nextlab_refresh_token`). Access token in memory.

**Testing**: Manual smoke tests via `quickstart.md`. No automated tests for MW1.

**Target Platform**: Browser (Chromium / Firefox). Desktop-first.

**Performance Goals**: Login completes in under 3 seconds (SC-001). 401 silent refresh transparent to user.

**Constraints**: No external UI component library (constitution Principle VI). No `fetch()` — axios only (Principle II).

**Scale/Scope**: Single-page app, 4 roles, ~10 protected routes total at MW1.

---

## Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Role-Gated Rendering | ✅ Pass | ProtectedRoute enforces role on every route |
| II. API Client Isolation | ✅ Pass | All HTTP through `src/lib/api.ts` axios instance |
| III. React Quality Gate | ✅ Pass | React Doctor scan required before MW1 is done |
| IV. Token Security | ✅ Pass | Access token in memory; refresh token in localStorage only |
| V. Read-Only Evidence | ✅ Pass | No evidence data in MW1 |
| VI. Brand Baseline | ✅ Pass | Unifranz palette applied; no component library |
| VII. Thin Client | ✅ Pass | No business logic — only routing and form empty-field checks |

No violations. All gates pass.

---

## Project Structure

### Documentation (this feature)

```text
specs/001-mw1-foundation-auth/
├── plan.md          ← this file
├── research.md      ← Phase 0 output
├── data-model.md    ← Phase 1 output
├── quickstart.md    ← Phase 1 output
├── contracts/
│   └── auth.md      ← Phase 1 output
└── tasks.md         ← Phase 2 output (speckit-tasks)
```

### Source Code

```text
nextlab-web/
├── .env                              # VITE_API_URL=http://localhost:8080
├── index.html                        # already exists
├── vite.config.ts                    # already exists
│
└── src/
    ├── main.tsx                      # ReactDOM.createRoot + RouterProvider
    ├── App.tsx                       # RouterProvider wrapper
    ├── index.css                     # @import "tailwindcss" + CSS variables
    │
    ├── types/
    │   └── index.ts                  # User, Role, AuthState, API request/response types
    │
    ├── lib/
    │   ├── api.ts                    # axios instance + JWT interceptor (refresh race fix)
    │   └── auth.ts                   # getRefreshToken / setTokens / clearTokens (localStorage helpers)
    │
    ├── api/
    │   └── auth.ts                   # login(), logout(), refresh() — call lib/api.ts
    │
    ├── context/
    │   └── AuthContext.tsx           # AuthContextValue provider + useAuth hook
    │
    ├── router/
    │   └── index.tsx                 # createBrowserRouter — all routes + ProtectedRoute guards
    │
    ├── components/
    │   └── common/
    │       ├── ProtectedRoute.tsx    # isLoading → spinner | no user → redirect | wrong role → role home
    │       ├── Layout.tsx            # Topbar + Sidebar + <Outlet />
    │       ├── Topbar.tsx            # Logo + user email + sign-out button
    │       ├── Sidebar.tsx           # Role-specific nav links
    │       ├── LoadingSpinner.tsx    # Full-screen spinner (used during auth init)
    │       └── ErrorMessage.tsx      # Inline error display component
    │
    └── pages/
        ├── auth/
        │   └── LoginPage.tsx         # Email + password form + Google OAuth button
        ├── student/
        │   └── DashboardPage.tsx     # Placeholder: "Student Dashboard"
        ├── docente/
        │   └── DashboardPage.tsx     # Placeholder: "Docente Dashboard"
        ├── codocente/
        │   └── DashboardPage.tsx     # Placeholder: "Co-docente Dashboard"
        ├── admin/
        │   └── DashboardPage.tsx     # Placeholder: "Admin Dashboard"
        └── NotFoundPage.tsx          # 404 with link back to role home
```

---

## Key Implementation Details

### `src/lib/api.ts` — Axios interceptor

```
requestInterceptor:
  attach Authorization: Bearer <accessToken> if present

responseInterceptor on 401:
  if (refreshPromise is null):
    refreshPromise = callRefresh(storedRefreshToken)
      .then(saveNewTokens)
      .catch(clearTokensAndRedirectToLogin)
      .finally(() => refreshPromise = null)
  await refreshPromise
  retry original request with new token
```

### `src/context/AuthContext.tsx` — Init flow

```
mount useEffect:
  const rt = getRefreshToken()   // from localStorage
  if (rt):
    try:
      response = await refresh(rt)
      setUser(decodeJWT(response.access_token))
      setAccessToken(response.access_token)
    catch:
      clearTokens()
  setIsLoading(false)
```

### `src/router/index.tsx` — ProtectedRoute usage

```
/student/*  → <ProtectedRoute allowedRoles={['estudiante']} />
/docente/*  → <ProtectedRoute allowedRoles={['docente']} />
/codocente/* → <ProtectedRoute allowedRoles={['co_docente']} />
/admin/*    → <ProtectedRoute allowedRoles={['admin']} />
```

### `src/pages/auth/LoginPage.tsx` — Post-login redirect

```
const from = location.state?.from?.pathname ?? roleHomeFor(user.role)
navigate(from, { replace: true })
```

---

## Scope Adjustment (vs. original spec)

**Removed:** User Story 3 (Account Registration via email/password form)

**Reason:** The `nextlab-api` has no `POST /auth/register` endpoint. User accounts
are created by admins through `POST /admin/student-profiles` (MW3). This is
documented in `research.md` Decision 1.

**Added:** Google OAuth button on login page (stretch goal — `GET /auth/google` exists).
Full OAuth flow integration deferred until the callback token delivery mechanism is
verified against the running API.

---

## Complexity Tracking

No constitution violations. No complexity justification required.
