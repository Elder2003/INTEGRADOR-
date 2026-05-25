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

## Architecture Rule — Thin Client

> **The frontend has zero business logic. Everything computes in the backend.**

The frontend is a display and input layer only. It renders what the API returns and
sends user input to the API. It does not:

- Calculate or derive grades
- Enforce deadlines or block uploads based on time
- Validate file contents or enforce size limits
- Compute integrity scores, risk levels, or similarity
- Manage activity state transitions
- Check data-level permissions beyond role routing

If a view needs different data, add a backend endpoint. Do not transform API responses
in components.

---

## Milestones

Each milestone aligns to one or more backend milestones and ends with a React Doctor
scan. A milestone is **done** when its deliverable works against the live API.

---

### MW1 — Foundation & Auth
**Backend:** M1  
**Goal:** Running shell, JWT auth flow, role-based redirect.

- [ ] `.env` — `VITE_API_URL=http://localhost:8080`
- [ ] `src/index.css` — `@import "tailwindcss"` + CSS variables (Unifranz palette)
- [ ] `src/lib/api.ts` — axios instance, JWT interceptor (auto-refresh on 401)
- [ ] `src/lib/auth.ts` — `getAccessToken`, `setTokens`, `clearTokens`
- [ ] `src/api/auth.ts` — login, register, refresh, logout
- [ ] `src/context/AuthContext.tsx` — user + role state, login/logout actions
- [ ] `src/components/common/` — Layout, Topbar, Sidebar, ProtectedRoute, Spinner, ErrorMessage
- [ ] `src/router/index.tsx` — all routes, ProtectedRoute guards per role
- [ ] `src/pages/auth/LoginPage.tsx`
- [ ] `src/pages/auth/RegisterPage.tsx`
- [ ] `src/App.tsx` + `src/main.tsx`
- [ ] React Doctor scan — no ERRORs

**Done when:** Login → JWT in memory → role redirect → logout → /login.

---

### MW2 — Docente: Course Management
**Backend:** M2, M3  
**Goal:** Docente can manage courses, groups, students, and activities.

- [ ] `src/api/academic.ts` — cursos CRUD, actividades CRUD, grupos CRUD, grants, students roster
- [ ] `src/types/index.ts` — TS interfaces for all API shapes (add to as milestones progress)
- [ ] `src/pages/docente/DashboardPage.tsx` — list of my courses
- [ ] `src/pages/docente/CursoDetailPage.tsx` — tabs: actividades | students | groups
- [ ] `src/pages/docente/CreateActividadPage.tsx` — form → POST /cursos/:id/actividades
- [ ] `src/pages/docente/ActividadDetailPage.tsx` — activity info + config
- [ ] `src/components/common/ConfirmModal.tsx`
- [ ] React Doctor scan — no ERRORs

**Done when:** Docente creates a course, adds a student, creates an activity.

---

### MW3 — Admin Panel
**Backend:** M3  
**Goal:** Admin manages catalog: carreras, materias, student profiles.

- [ ] `src/api/admin.ts` — carreras CRUD, materias CRUD, student-profiles CRUD
- [ ] `src/pages/admin/DashboardPage.tsx`
- [ ] `src/pages/admin/CarrerasPage.tsx`
- [ ] `src/pages/admin/MateriasPage.tsx`
- [ ] `src/pages/admin/StudentProfilesPage.tsx`
- [ ] React Doctor scan — no ERRORs

**Done when:** Admin creates a carrera, links a materia, creates a student profile.

---

### MW4 — Student Portal
**Backend:** M2, M3  
**Goal:** Student views their assigned courses and activities.

- [ ] `src/api/academic.ts` — student endpoints: GET /me/cursos, GET /me/cursos/:id/actividades
- [ ] `src/components/student/ActivityCard.tsx`
- [ ] `src/components/student/ActivityTypeBadge.tsx`
- [ ] `src/pages/student/DashboardPage.tsx` — my courses
- [ ] `src/pages/student/ActividadesPage.tsx` — activities for a course
- [ ] `src/pages/student/ActividadPage.tsx` — activity detail (read + submit area placeholder)
- [ ] React Doctor scan — no ERRORs

**Done when:** Student logs in, sees their courses, opens an activity detail page.

---

### MW5 — File Submissions
**Backend:** M4  
**Goal:** Student uploads files; docente sees and downloads them.

- [ ] `src/api/submissions.ts` — upload, replace, delete, download, list
- [ ] `src/components/student/FileUpload.tsx` — drag-and-drop, sends to API
- [ ] `src/components/student/IntegrityDeclaration.tsx` — checkbox before upload
- [ ] `src/components/docente/SubmissionTable.tsx` — paginated list
- [ ] `src/pages/student/ActividadPage.tsx` — wire FileUpload + current submission display
- [ ] `src/pages/docente/SubmissionsPage.tsx` — GET /cursos/:cid/actividades/:aid/submissions
- [ ] `src/pages/docente/SubmissionDetailPage.tsx` — submission metadata + download
- [ ] Co-docente submission view — `src/pages/codocente/SubmissionsPage.tsx`
- [ ] React Doctor scan — no ERRORs

**Done when:** Student uploads a file; docente downloads it from the submissions list.

---

### MW6 — Session Tracing
**Backend:** M5  
**Goal:** Student opens a session; docente reads the trace.

- [ ] `src/api/tracing.ts` — open session, get session
- [ ] `src/components/student/TracingPanel.tsx` — WebSocket connect, display incoming events
- [ ] `src/components/docente/TraceViewer.tsx` — read-only timeline of trace events
- [ ] `src/pages/student/ActividadPage.tsx` — wire TracingPanel on session open
- [ ] `src/pages/docente/SubmissionDetailPage.tsx` — add TraceViewer + verify-hash button
- [ ] React Doctor scan — no ERRORs

**Done when:** Student opens session, events appear in TracingPanel; docente sees full timeline.

---

### MW7 — Grading
**Backend:** M5, M11  
**Goal:** Docente grades submissions; student sees grades.

- [ ] `src/api/tracing.ts` — exam sessions + grade decisions
- [ ] `src/api/projects.ts` — investigación stages, stage submissions
- [ ] `src/components/docente/GradePanel.tsx` — grade form, sends to API, displays API response
- [ ] `src/pages/docente/SubmissionDetailPage.tsx` — wire GradePanel (tarea/examen/informe/practica)
- [ ] `src/pages/docente/InvestigacionPage.tsx` — stage list + stage grade form
- [ ] `src/pages/docente/DefensaPage.tsx` — schedule + grade oral defense
- [ ] `src/pages/student/GradesPage.tsx` — display grades returned by API
- [ ] React Doctor scan — no ERRORs

**Done when:** Docente submits a grade; student sees it on the grades page.

---

### MW8 — Git & AI Results
**Backend:** M6, M7, M8  
**Goal:** Student submits a git repo; docente reads analysis and detection reports.

- [ ] `src/api/git.ts` — git submission form, GET git analysis
- [ ] `src/api/ai.ts` — GET AI analysis result
- [ ] `src/api/detection.ts` — GET detection report
- [ ] `src/pages/student/ActividadPage.tsx` — git URL input form (tipo === git_analysis)
- [ ] `src/pages/docente/SubmissionDetailPage.tsx` — display git analysis + AI result + detection report (all read-only)
- [ ] `src/components/student/RiskBadge.tsx` — display flag returned by API
- [ ] React Doctor scan — no ERRORs

**Done when:** Docente opens a git_analysis submission and reads the AI and detection reports.

---

### MW9 — Alerts & Analytics
**Backend:** M9, M10, M12  
**Goal:** Docente sees alerts and grade analytics.

- [ ] `src/api/alerts.ts` — list alerts, dismiss
- [ ] `src/api/analytics.ts` — activity analytics, course analytics
- [ ] `src/components/docente/AlertsTable.tsx`
- [ ] `src/components/docente/AnalyticsBar.tsx` — CSS-only bar chart (no chart library)
- [ ] `src/pages/docente/ActividadDetailPage.tsx` — add alerts section
- [ ] `src/pages/docente/ActivityAnalyticsPage.tsx`
- [ ] `src/pages/docente/CourseAnalyticsPage.tsx`
- [ ] `src/pages/docente/CursoDetailPage.tsx` — add analytics tab
- [ ] React Doctor scan — no ERRORs

**Done when:** Docente sees grade distribution bars and the flagged-submissions count.

---

### MW10 — Appeals
**Backend:** M13  
**Goal:** Student files an appeal; docente resolves it.

- [ ] `src/api/appeals.ts` — create appeal, list appeals, resolve appeal
- [ ] `src/pages/student/AppealPage.tsx` — appeal form → POST /actividades/:aid/appeals
- [ ] `src/pages/docente/AppealsPage.tsx` — list appeals for my activities + resolve action
- [ ] `src/components/docente/AppealCard.tsx`
- [ ] React Doctor scan — no ERRORs

**Done when:** Student submits an appeal; docente sees it and marks it resolved.

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
