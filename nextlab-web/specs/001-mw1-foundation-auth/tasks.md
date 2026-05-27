# Tasks: MW1 â€” Foundation & Auth

**Input**: Design documents from `specs/001-mw1-foundation-auth/`

**Prerequisites**: plan.md âœ… | spec.md âœ… | research.md âœ… | data-model.md âœ… | contracts/auth.md âœ…

**Tests**: Not requested â€” no test tasks included.

**Scope note**: User Story 3 (Registration) removed â€” no `/auth/register` endpoint exists in the API. See research.md Decision 1.

---

## Phase 1: Setup

**Purpose**: Environment, dependencies, and shared types. No logic yet.

- [X] T001 Create `nextlab-web/.env` with content `VITE_API_URL=http://localhost:8080`
- [X] T002 Install axios by running `pnpm add axios` in `nextlab-web/`
- [X] T003 Create `src/index.css` with `@import "tailwindcss"` and CSS custom properties: `--color-accent: #E85D24`, `--color-bg: #111111`, `--color-surface: #1a1a1a`, `--color-surface-2: #242424`, `--color-border: #2e2e2e`, `--color-text: #f0f0f0`, `--color-muted: #888888`; import Inter and JetBrains Mono from Google Fonts via `@import url(...)`
- [X] T004 Create `src/types/index.ts` with: `Role` union type (`'estudiante' | 'docente' | 'co_docente' | 'admin'`), `User` interface (`id`, `email`, `role`), `AuthState` interface (`user`, `accessToken`, `isLoading`), `JWTClaims` interface (`sub`, `email`, `role`, `jti`, `iat`, `exp`), `LoginRequest`, `LoginResponse`, `RefreshRequest`, `LogoutRequest` matching shapes in data-model.md, `ApiError` class extending Error with `code: string` and optional `retryAfter: number`

---

## Phase 2: Foundation

**Purpose**: HTTP client, token helpers, auth API functions, and AuthContext. ALL user stories depend on this phase.

**âš ï¸ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T005 [P] Create `src/lib/auth.ts` with four functions: `getRefreshToken()` reads `localStorage.getItem('nextlab_refresh_token')`; `setTokens(accessToken, refreshToken)` stores only the refresh token to localStorage; `clearTokens()` removes `nextlab_refresh_token` from localStorage; `decodeJWT(token: string): JWTClaims` base64url-decodes `token.split('.')[1]` and JSON.parses the result â€” no signature verification
- [X] T006 Create `src/lib/api.ts` with: axios instance with `baseURL: import.meta.env.VITE_API_URL`; request interceptor that attaches `Authorization: Bearer <token>` header when a token is available (token must be read from a getter function, not captured at module init); response interceptor that on 401 â€” uses a module-level `refreshPromise` variable to serialize concurrent refresh attempts, calls `/auth/refresh` once, retries the original request with the new token, and on refresh failure calls `clearTokens()` then `window.location.href = '/login'`; export the instance as default
- [X] T007 Create `src/api/auth.ts` with three exported async functions that call `src/lib/api.ts`: `login(email, password)` â†’ POST `/auth/login` â†’ returns `LoginResponse`; `refreshToken(rt)` â†’ POST `/auth/refresh` â†’ returns `LoginResponse`; `logout(refreshToken)` â†’ POST `/auth/logout` with Authorization header â†’ returns void; each function throws `ApiError` on non-2xx responses, extracting `error` and `message` from the response body; the 429 case must also extract `retry_after` from the response and set it on `ApiError.retryAfter`
- [X] T008 Create `src/context/AuthContext.tsx` with: `AuthContextValue` interface matching contracts/auth.md; `AuthProvider` component that holds `user: User | null`, `accessToken: string | null`, `isLoading: boolean` in state; `login(email, password)` function that calls `src/api/auth.ts` login, decodes the JWT with `decodeJWT` to extract `User`, sets state, stores refresh token via `setTokens`; `logout()` function that calls `src/api/auth.ts` logout then calls `clearTokens()` and clears state; `useAuth()` hook that throws if used outside provider; export both `AuthProvider` and `useAuth` â€” **do NOT add the init useEffect yet, that is Task T021**
- [X] T009 [P] Create `src/components/common/LoadingSpinner.tsx` â€” full-screen centered spinner using a Tailwind CSS animated `border-t-4 border-[#E85D24]` circle, accepts optional `message?: string` prop displayed below the spinner
- [X] T010 [P] Create `src/components/common/ErrorMessage.tsx` â€” displays an error string in a dark red surface box with an orange left border stripe; accepts `message: string` and optional `onDismiss?: () => void` prop; renders nothing if message is empty

**Checkpoint**: `lib/api.ts`, `api/auth.ts`, `AuthContext` complete. Foundation ready.

---

## Phase 3: User Story 1 â€” Sign In and Role Redirect (Priority: P1) ðŸŽ¯ MVP

**Goal**: Any user can sign in with email and password and land on their role-specific home screen.

**Independent Test**: Quickstart Scenario 1 â€” sign in as each of the four roles and confirm correct home screen.

- [X] T011 [P] [US1] Create `src/pages/student/DashboardPage.tsx` â€” placeholder page that renders a heading "Student Dashboard" and the signed-in user's email from `useAuth()`
- [X] T012 [P] [US1] Create `src/pages/docente/DashboardPage.tsx` â€” placeholder page that renders "Docente Dashboard" and user email
- [X] T013 [P] [US1] Create `src/pages/codocente/DashboardPage.tsx` â€” placeholder page that renders "Co-docente Dashboard" and user email
- [X] T014 [P] [US1] Create `src/pages/admin/DashboardPage.tsx` â€” placeholder page that renders "Admin Dashboard" and user email
- [X] T015 [P] [US1] Create `src/pages/NotFoundPage.tsx` â€” 404 page with message "Page not found" and a link back to `/`
- [X] T016 [US1] Create `src/pages/auth/LoginPage.tsx` with: email + password form inputs; client-side validation â€” show inline error if either field is empty or email has no `@` symbol, validated on submit only; calls `useAuth().login()`; on success reads `location.state?.from?.pathname` â€” if present navigates there, else navigates to the role home path (`/student/dashboard`, `/docente/dashboard`, `/codocente/dashboard`, `/admin/dashboard`); on `ApiError` displays the API `message` field via `<ErrorMessage />`; the 429 case must show `"Too many attempts. Try again in {retryAfter} seconds"`; submit button is disabled and shows a spinner while the request is in flight; if user is already authenticated redirect immediately to their role home
- [X] T017 [US1] Create `src/router/index.tsx` using `createBrowserRouter`: route `/` redirects to `/login` for unauthenticated (handled later by ProtectedRoute); `/login` â†’ LoginPage; `/student/dashboard` â†’ student DashboardPage; `/docente/dashboard` â†’ docente DashboardPage; `/codocente/dashboard` â†’ codocente DashboardPage; `/admin/dashboard` â†’ admin DashboardPage; `*` â†’ NotFoundPage; export the router as default and also export a `roleHomeFor(role: Role): string` helper function
- [X] T018 [US1] Create `src/main.tsx` rendering `<AuthProvider><RouterProvider router={router} /></AuthProvider>` inside `React.StrictMode`; delete or leave `src/App.tsx` empty â€” the RouterProvider is the app root

**Checkpoint**: `pnpm dev` â†’ navigate to `/login` â†’ sign in as docente â†’ lands on `/docente/dashboard` showing "Docente Dashboard" and email.

---

## Phase 4: User Story 2 â€” Protected Navigation (Priority: P2)

**Goal**: Unauthenticated users cannot reach protected screens. Wrong-role users are redirected to their own home.

**Independent Test**: Quickstart Scenarios 3 and 4 â€” direct URL access while signed out and cross-role navigation.

- [X] T019 [US2] Create `src/components/common/ProtectedRoute.tsx` matching contracts/auth.md: if `isLoading` is true render `<LoadingSpinner />`; if `user` is null render `<Navigate to="/login" state={{ from: location }} replace />`; if `user.role` is not in `allowedRoles` prop render `<Navigate to={roleHomeFor(user.role)} replace />`; otherwise render `<Outlet />`; accepts `allowedRoles: Role[]` prop
- [X] T020 [US2] Update `src/router/index.tsx` to wrap each role's routes with `<ProtectedRoute allowedRoles={[...]} />`; the `/login` route must redirect already-authenticated users to their role home using a small `PublicOnlyRoute` component (redirect to role home if user exists); the `/` route redirects to role home if authenticated or to `/login` if not

**Checkpoint**: Navigate to `/docente/dashboard` while signed out â†’ redirected to `/login`. Sign in as `estudiante`, navigate to `/docente/dashboard` â†’ redirected to `/student/dashboard`.

---

## Phase 5: User Story 4 â€” Session Persistence (Priority: P4)

**Goal**: Signed-in users survive page refresh without re-authenticating.

**Independent Test**: Quickstart Scenario 5 â€” sign in, press F5, still signed in.

- [X] T021 [US4] Update `src/context/AuthContext.tsx` to add a `useEffect` that runs once on mount: reads `getRefreshToken()` from localStorage; if a token exists calls `refreshToken(rt)` from `src/api/auth.ts`, on success decodes the new JWT and sets `user` and `accessToken` in state; on failure calls `clearTokens()`; sets `isLoading: false` in both cases; sets `isLoading: true` as the initial state value; `lib/api.ts` must be able to read the current access token â€” pass a getter function into the axios instance (or use a module-level setter called from AuthContext after each token update)

**Checkpoint**: Sign in â†’ press F5 â†’ page loads with spinner briefly â†’ still shows dashboard, still signed in.

---

## Phase 6: User Story 5 â€” Sign Out (Priority: P5)

**Goal**: Users can sign out from any screen. Back button after sign-out does not return to protected screen.

**Independent Test**: Quickstart Scenarios 6 and 3 (sign-out flow).

- [X] T022 [P] [US5] Create `src/components/common/Topbar.tsx` with: Unifranz logo text "NextLab" in accent color `#E85D24`; signed-in user's email (from `useAuth()`) on the right; a sign-out button that calls `useAuth().logout()` then navigates to `/login`; Tailwind classes for dark background `bg-[#1a1a1a]` and bottom border `border-b border-[#2e2e2e]`; sign-out button is disabled while logout is in flight
- [X] T023 [P] [US5] Create `src/components/common/Sidebar.tsx` with: role-specific nav links derived from `useAuth().user.role`; student links: "My Courses"; docente links: "My Courses", "Analytics"; admin links: "Carreras", "Materias", "Student Profiles"; co-docente links: "Granted Activities"; each link uses React Router `<NavLink>` with active state highlighted in `#E85D24`; left border stripe `border-l-4 border-[#E85D24]` on the active item
- [X] T024 [US5] Create `src/components/common/Layout.tsx` that renders `<Topbar />` at top, `<Sidebar />` on the left, and `<Outlet />` as main content; uses CSS grid or flex layout; background `bg-[#111111]`, minimum full screen height
- [X] T025 [US5] Update `src/router/index.tsx` to nest all authenticated role routes under `<Layout />` so every protected page has the Topbar and Sidebar; the Layout wrapper goes inside the ProtectedRoute wrappers

**Checkpoint**: Sign in â†’ see Topbar with email and Sidebar with role links â†’ click sign out â†’ at `/login` â†’ press back â†’ still at `/login`.

---

## Phase 7: User Story 6 â€” Session Expiry Handling (Priority: P6)

**Goal**: If a session expires mid-use, the user gets a clear message and lands on the login screen.

**Independent Test**: Quickstart Scenario 8 (session expiry path: let access token expire, trigger any API call).

- [X] T026 [US6] Update `src/pages/auth/LoginPage.tsx` to detect `location.state?.sessionExpired === true` (set by the axios interceptor redirect) and display an informational banner "Your session has ended. Please sign in again." above the form; update `src/lib/api.ts` redirect on refresh failure to use `window.location.href = '/login?session_expired=1'` and update LoginPage to read the `session_expired` query param as an alternative detection method

**Checkpoint**: Simulate expired token by clearing the access token in memory (dev tools â†’ React context) and triggering an API call â†’ redirected to `/login` â†’ session-ended message is visible.

---

## Phase 8: Polish

**Purpose**: UX refinements and quality gate.

- [X] T027 Update `src/pages/auth/LoginPage.tsx` to add a "Sign in with Google" button below the form that navigates to `${import.meta.env.VITE_API_URL}/auth/google`; button uses Google's brand color `#4285F4` with a Google icon from Lucide React or an inline SVG; add a note below: "Google sign-in is available for students"
- [X] T028 Run `npx react-doctor@latest ./src` from `nextlab-web/` and resolve all ERROR-severity findings before marking MW1 complete

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies â€” start immediately
- **Phase 2 (Foundation)**: Depends on Phase 1 â€” **blocks all user stories**
- **Phase 3 (US1)**: Depends on Phase 2 â€” first deliverable
- **Phase 4 (US2)**: Depends on Phase 3 (needs router and pages to exist)
- **Phase 5 (US4)**: Depends on Phase 2 (extends AuthContext)
- **Phase 6 (US5)**: Depends on Phase 4 (Layout wraps protected routes)
- **Phase 7 (US6)**: Depends on Phase 2 (extends lib/api.ts and LoginPage)
- **Phase 8 (Polish)**: Depends on all phases complete

### Within Phase 2

```
T005 (lib/auth.ts)     â† no deps, run first
T006 (lib/api.ts)      â† depends on T005 (uses clearTokens)
T007 (api/auth.ts)     â† depends on T006 (uses axios instance)
T008 (AuthContext)     â† depends on T005 + T007
T009, T010             â† [P] independent, run anytime after Phase 1
```

### Parallel Opportunities

```
Phase 1:    T001, T002, T003, T004 â€” all independent

Phase 2:    T005 â†’ T006 â†’ T007 â†’ T008 (sequential)
            T009, T010 â€” parallel with any Phase 2 task

Phase 3:    T011, T012, T013, T014, T015 â€” [P] all independent
            T016 â†’ T017 â†’ T018 (sequential, each builds on previous)

Phase 6:    T022, T023 â€” [P] independent
            T024 â†’ T025 (sequential)
```

---

## Implementation Strategy

### MVP (Phase 1â€“3 only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundation
3. Complete Phase 3: US1 (login + role redirect)
4. **STOP and validate**: Quickstart Scenario 1 â€” sign in as each role

At this point you have a working login with role redirect. No protection yet.

### Full MW1

Continue Phase 4 â†’ 5 â†’ 6 â†’ 7 â†’ 8 in order.
Each phase adds one behavior layer. No phase breaks the previous one.

---

## Notes

- `[P]` = parallelizable (different files, no dependency on incomplete tasks)
- `[USn]` = maps task to user story for traceability
- No test tasks â€” tests are manual via `quickstart.md`
- T006 (`lib/api.ts`) uses `window.location.href` for the logout redirect because axios interceptors run outside React Router context â€” this is intentional
- T021 requires coordinating token updates between AuthContext and the axios interceptor â€” the cleanest solution is a module-level setter in `lib/api.ts` (`export function setAccessTokenGetter(fn: () => string | null)`) called from AuthContext after every token update

