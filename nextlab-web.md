# nextlab-web — Frontend Implementation Plan

**Project:** Unifranz Academic Audit System — Web Frontend  
**Stack:** React 19 + Vite 8 + TypeScript 5.9 + Tailwind CSS v4 + React Router v7  
**API:** nextlab-api (M1–M13, running on localhost:8080)  
**Author:** Elder Omar Gutierrez Rosales

---

## Overview

Full frontend for the nextlab platform. Connects to the real nextlab-api. Covers four user roles:
`estudiante`, `docente`, `co_docente`, `admin`.

---

## Tech Stack

| Layer | Choice | Notes |
|-------|--------|-------|
| Framework | React 19 | Already installed |
| Build | Vite 8 | Already configured |
| Language | TypeScript 5.9 | Already configured |
| Styling | Tailwind CSS v4 | Already configured via `@tailwindcss/vite` |
| Routing | React Router v7 | Already installed |
| Icons | Lucide React | Already installed |
| HTTP client | axios | **To install** |
| State | React Context + useReducer | No external lib needed |

**Install command:**
```bash
pnpm add axios
```

---

## Design System

| Token | Value | Usage |
|-------|-------|-------|
| `--color-accent` | `#E85D24` | Buttons, active states, badges |
| `--color-bg` | `#111111` | Page background |
| `--color-surface` | `#1a1a1a` | Cards, panels |
| `--color-surface-2` | `#242424` | Nested surfaces, inputs |
| `--color-border` | `#2e2e2e` | Dividers, outlines |
| `--color-text` | `#f0f0f0` | Primary text |
| `--color-muted` | `#888888` | Secondary/placeholder text |
| Font | Inter | Body text |
| Font mono | JetBrains Mono | Code, trace output, hashes |

Left-border accent stripe (`border-l-4 border-[#E85D24]`) applied to all content sections.

---

## Project Structure

```
src/
├── main.tsx
├── App.tsx
├── index.css                    # Tailwind imports + CSS variables
│
├── lib/
│   ├── api.ts                   # axios instance + JWT interceptor (auto-refresh on 401)
│   └── auth.ts                  # token storage helpers (access = memory, refresh = localStorage)
│
├── api/                         # One file per API domain
│   ├── auth.ts                  # POST /auth/login, /auth/register, /auth/refresh, /auth/logout
│   ├── academic.ts              # cursos, actividades, grupos, grants — M2/M3
│   ├── admin.ts                 # carreras, materias, student-profiles — M3
│   ├── submissions.ts           # upload, replace, delete, list, download — M4
│   ├── tracing.ts               # sessions, trace, verify — M5
│   ├── git.ts                   # git submissions + analyses — M6
│   ├── ai.ts                    # AI analysis endpoints — M7
│   ├── detection.ts             # plagiarism detection — M8
│   ├── alerts.ts                # alert list + dismiss — M9/M10
│   ├── projects.ts              # investigación stages — M11
│   ├── analytics.ts             # activity + course analytics — M12
│   └── appeals.ts               # create + list appeals — M13
│
├── context/
│   └── AuthContext.tsx          # { user, role, accessToken } + login/logout actions
│
├── router/
│   └── index.tsx                # All routes; ProtectedRoute enforces role
│
├── components/
│   ├── common/
│   │   ├── Layout.tsx           # Shell: Topbar + Sidebar + <Outlet />
│   │   ├── Topbar.tsx           # Logo, user avatar, logout button
│   │   ├── Sidebar.tsx          # Role-specific nav links
│   │   ├── ProtectedRoute.tsx   # Redirects if unauthenticated or wrong role
│   │   ├── LoadingSpinner.tsx
│   │   ├── ErrorMessage.tsx
│   │   └── ConfirmModal.tsx
│   │
│   ├── student/
│   │   ├── ActivityCard.tsx     # Card shown in student dashboard per activity
│   │   ├── ActivityTypeBadge.tsx # tarea | examen | informe | git_analysis | …
│   │   ├── FileUpload.tsx       # Drag-and-drop file submission
│   │   ├── TimerDisplay.tsx     # Countdown; orange <10 min, red <5 min, auto-submit at 0
│   │   ├── TracingPanel.tsx     # Live event stream from WebSocket session
│   │   ├── RiskBadge.tsx        # integrity_flag / format_violation badge
│   │   └── IntegrityDeclaration.tsx # Checkbox required before submit
│   │
│   └── docente/
│       ├── SubmissionTable.tsx  # Paginated list of student submissions
│       ├── TraceViewer.tsx      # Renders trace events timeline
│       ├── GradePanel.tsx       # Exam session + grade_decisions form
│       ├── AnalyticsBar.tsx     # Grade distribution bar chart (no lib, CSS)
│       ├── AlertsTable.tsx      # List of integrity alerts
│       └── AppealCard.tsx       # Single appeal with accept/reject action
│
└── pages/
    ├── auth/
    │   ├── LoginPage.tsx
    │   └── RegisterPage.tsx
    │
    ├── student/
    │   ├── DashboardPage.tsx        # GET /me/cursos
    │   ├── ActividadesPage.tsx      # GET /me/cursos/:id/actividades
    │   ├── ActividadPage.tsx        # Activity detail: submit file, open session, timer
    │   ├── GradesPage.tsx           # Aggregate grade view (derived from activity list)
    │   └── AppealPage.tsx           # POST /actividades/:aid/appeals
    │
    ├── docente/
    │   ├── DashboardPage.tsx        # GET /cursos (list all my courses)
    │   ├── CursoDetailPage.tsx      # GET /cursos/:id — tabs: actividades, students, groups, analytics
    │   ├── CreateActividadPage.tsx  # POST /cursos/:id/actividades
    │   ├── ActividadDetailPage.tsx  # GET /cursos/:id/actividades/:aid
    │   ├── SubmissionsPage.tsx      # GET /cursos/:cid/actividades/:aid/submissions
    │   ├── SubmissionDetailPage.tsx # Trace + grade for a single submission
    │   ├── ActivityAnalyticsPage.tsx # GET /actividades/:aid/analytics
    │   └── CourseAnalyticsPage.tsx  # GET /cursos/:id/analytics
    │
    ├── codocente/
    │   ├── ActividadViewPage.tsx    # GET /grants/actividades/:aid
    │   └── SubmissionsPage.tsx      # GET /cursos/:cid/actividades/:aid/submissions
    │
    └── admin/
        ├── DashboardPage.tsx
        ├── CarrerasPage.tsx         # CRUD /admin/carreras
        ├── MateriasPage.tsx         # CRUD /admin/materias
        └── StudentProfilesPage.tsx  # CRUD /admin/student-profiles
```

---

## Auth Flow

```
1. User submits login form
   → POST /auth/login → { access_token, refresh_token, user }
   → access_token stored in AuthContext (memory only, cleared on tab close)
   → refresh_token stored in localStorage
   → user.role determines redirect: /student/dashboard | /docente/dashboard | /admin/dashboard

2. axios interceptor on every request:
   → Sets Authorization: Bearer <access_token>
   → On 401 response: calls POST /auth/refresh with refresh_token
   → Retries original request with new access_token
   → On refresh failure: clears tokens, redirects to /login

3. Logout:
   → POST /auth/logout
   → Clears AuthContext + localStorage
   → Redirect to /login
```

---

## Routing Map

```
/                        → redirect based on role (or /login if unauthenticated)
/login
/register

/student/
  dashboard
  cursos/:id/actividades
  actividades/:aid
  grades
  actividades/:aid/appeal

/docente/
  dashboard
  cursos/new
  cursos/:id                          (tabs: actividades | students | groups | analytics)
  cursos/:id/actividades/new
  cursos/:id/actividades/:aid
  cursos/:id/actividades/:aid/submissions
  cursos/:id/actividades/:aid/analytics
  cursos/:id/analytics
  submissions/:sid                    (trace + grade)

/codocente/
  actividades/:aid
  cursos/:cid/actividades/:aid/submissions

/admin/
  dashboard
  carreras
  materias
  student-profiles
```

---

## Implementation Phases

### Phase 1 — Foundation
**Goal:** Running app shell, auth flow, axios client, route skeleton.

- [ ] `src/index.css` — CSS variables + Tailwind base
- [ ] `src/lib/api.ts` — axios instance, baseURL from `VITE_API_URL`, JWT interceptor
- [ ] `src/lib/auth.ts` — `getAccessToken`, `setTokens`, `clearTokens`
- [ ] `src/context/AuthContext.tsx` — user state, login/logout functions
- [ ] `src/api/auth.ts` — login, register, refresh, logout API calls
- [ ] `src/components/common/` — Layout, Topbar, Sidebar, ProtectedRoute, Spinner
- [ ] `src/router/index.tsx` — all routes with ProtectedRoute guards
- [ ] `src/pages/auth/LoginPage.tsx`
- [ ] `src/pages/auth/RegisterPage.tsx`
- [ ] `src/App.tsx` + `src/main.tsx`
- [ ] `.env` — `VITE_API_URL=http://localhost:8080`

**Done when:** Login → JWT stored → redirect to role dashboard → logout → back to /login.

---

### Phase 2 — Student Surface
**Goal:** Full student workflow: browse activities, upload submission, live session, grades, appeal.

- [ ] `src/api/academic.ts` — student endpoints (me/cursos, me/cursos/:id/actividades)
- [ ] `src/api/submissions.ts` — upload, replace, delete, download
- [ ] `src/api/tracing.ts` — open session, WebSocket connect
- [ ] `src/api/appeals.ts` — create appeal
- [ ] `src/types/index.ts` — TS interfaces for all API response shapes
- [ ] `src/components/student/` — all 7 student components
- [ ] `src/pages/student/DashboardPage.tsx`
- [ ] `src/pages/student/ActividadesPage.tsx`
- [ ] `src/pages/student/ActividadPage.tsx` (file upload + timer + tracing)
- [ ] `src/pages/student/GradesPage.tsx`
- [ ] `src/pages/student/AppealPage.tsx`

**Done when:** Student can log in, see their courses, open an activity, upload a file, see the tracing panel update via WebSocket, and submit an appeal.

---

### Phase 3 — Docente Core
**Goal:** Course/activity management and submission review.

- [ ] `src/api/academic.ts` — docente endpoints (cursos CRUD, actividades CRUD, grupos, grants)
- [ ] `src/components/docente/SubmissionTable.tsx`
- [ ] `src/components/docente/TraceViewer.tsx`
- [ ] `src/pages/docente/DashboardPage.tsx`
- [ ] `src/pages/docente/CursoDetailPage.tsx` (with tabs)
- [ ] `src/pages/docente/CreateActividadPage.tsx`
- [ ] `src/pages/docente/ActividadDetailPage.tsx`
- [ ] `src/pages/docente/SubmissionsPage.tsx`
- [ ] `src/pages/docente/SubmissionDetailPage.tsx` (trace timeline + basic info)

**Done when:** Docente can create a course, add an activity, see student submission list, and view a submission's trace.

---

### Phase 4 — Grading
**Goal:** Exam sessions, grade decisions, investigación stages, defensa oral.

- [ ] `src/api/tracing.ts` — grade decision endpoints (exam sessions)
- [ ] `src/api/projects.ts` — investigación stages + stage submissions
- [ ] `src/api/git.ts` — git submissions + analyses
- [ ] `src/components/docente/GradePanel.tsx`
- [ ] Grade form in `SubmissionDetailPage.tsx` (tarea/examen/informe/practica)
- [ ] Investigación stage grading sub-page
- [ ] Defensa oral scheduling + grading sub-page

**Done when:** Docente can assign a final grade to a submission and it appears in the student's grades view.

---

### Phase 5 — Analytics + Alerts
**Goal:** Visual analytics and alerts management.

- [ ] `src/api/analytics.ts` — activity analytics, course analytics
- [ ] `src/api/alerts.ts` — list + dismiss alerts
- [ ] `src/components/docente/AnalyticsBar.tsx` — CSS-only bar chart (no chart lib)
- [ ] `src/components/docente/AlertsTable.tsx`
- [ ] `src/pages/docente/ActivityAnalyticsPage.tsx`
- [ ] `src/pages/docente/CourseAnalyticsPage.tsx`
- [ ] Alerts section in `ActividadDetailPage.tsx`

**Done when:** Docente can see grade distribution bars and pending/flagged counts per activity.

---

### Phase 6 — Admin + Co-docente
**Goal:** Admin CRUD surfaces and co-docente granted access.

- [ ] `src/api/admin.ts` — carreras, materias, student-profiles
- [ ] `src/pages/admin/CarrerasPage.tsx`
- [ ] `src/pages/admin/MateriasPage.tsx`
- [ ] `src/pages/admin/StudentProfilesPage.tsx`
- [ ] `src/pages/codocente/ActividadViewPage.tsx`
- [ ] `src/pages/codocente/SubmissionsPage.tsx`
- [ ] Sidebar nav differentiation for co_docente role

**Done when:** Admin can manage catalog data; co_docente can view submissions on granted activities.

---

## API Environment

```env
VITE_API_URL=http://localhost:8080
```

The API runs on port 8080 inside Docker (`docker compose up`).
WebSocket sessions connect to `ws://localhost:8080/sessions/:sid/ws?token=<jwt>`.

---

## Activity Types Reference

| tipo | Submission table | Grade table |
|------|-----------------|-------------|
| `tarea` | `submissions` | `grade_decisions.final_grade` via `exam_sessions` |
| `examen` | `submissions` | `grade_decisions.final_grade` via `exam_sessions` |
| `informe` | `submissions` | `grade_decisions.final_grade` via `exam_sessions` |
| `practica` | `submissions` | `grade_decisions.final_grade` via `exam_sessions` |
| `investigacion` | `stage_submissions` | `stage_submissions.grade` |
| `git_analysis` | `git_submissions` | `git_analyses.grade` |
| `defensa_oral` | `defense_appointments` | `defense_appointments.grade` |

---

## Notes

- The `src/` directory is currently empty — all files must be created.
- `vite.config.ts` already has Tailwind and React plugins configured.
- `pnpm` is the package manager used in this project.
- Tailwind CSS v4 uses `@import "tailwindcss"` syntax (not `@tailwind base/components/utilities`).
- React Router v7 uses `createBrowserRouter` + `RouterProvider` (not `BrowserRouter`).
