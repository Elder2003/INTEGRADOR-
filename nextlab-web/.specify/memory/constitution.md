<!--
SYNC IMPACT REPORT
==================
Version change: 1.0.0 → 1.2.0
Type of bump: MINOR — Principle VI reworked from design lock to iterative baseline;
             Principle VII added (Thin Client); milestones table updated.
Rationale: UX must be adjustable based on user feedback without governance overhead.
           Only the no-component-library rule remains as a hard constraint.
           Business logic principle added per explicit project requirement.

Modified principles:
  VI. Brand & Design System Lock → VI. Brand Baseline & UX Iteration
Added principles:
  VII. Thin Client — No Business Logic

Modified sections:
  - Implementation Phases renamed to Implementation Milestones; restructured as MW1–MW10.

Templates requiring updates:
  ⚠ .specify/templates/ — not yet initialized in nextlab-web.

Deferred TODOs: none.
-->

# nextlab-web — Frontend Constitution

**Universidad Privada Franz Tamayo — Unifranz | Proyecto Integrador, Ingeniería de Sistemas**  
**Companion to:** `nextlab-api` constitution (`.specify/memory/constitution.md` in nextlab-api)

---

## Core Principles

### I. Role-Gated Rendering

Every route, page, and data-access call MUST be protected by the authenticated user's role
as stored in `AuthContext`. There are four valid roles: `estudiante`, `docente`,
`co_docente`, `admin`.

- The `ProtectedRoute` component MUST wrap every authenticated route and redirect to `/login`
  if no valid session exists.
- If the authenticated role does not match the required role for a route, the user MUST be
  redirected to their own role's dashboard — never shown a blank page or a raw 403.
- UI elements (buttons, tabs, menu items) that trigger role-restricted actions MUST be
  conditionally hidden, not merely disabled, to avoid exposing available operations to
  unauthorized users.
- Role MUST be derived exclusively from the JWT claims returned by the API — never hardcoded
  or inferred from URL structure.

**Rationale:** The role isolation defined in the `nextlab-api` constitution (Principle IV)
MUST be mirrored on the client. Showing a student the docente grading panel — even
read-only — would expose analysis reports they are explicitly forbidden to see.

---

### II. API Client Isolation

ALL HTTP calls to the nextlab-api MUST be made through the centralized axios instance
defined in `src/lib/api.ts`. Direct use of `fetch()`, `XMLHttpRequest`, or any other
HTTP mechanism anywhere in components, pages, or hooks is forbidden.

- The axios instance MUST attach the `Authorization: Bearer <access_token>` header on
  every outgoing request automatically via an interceptor.
- On receiving a `401` response, the interceptor MUST attempt one silent token refresh
  (POST `/auth/refresh`) and retry the original request before propagating the error.
- On refresh failure, the interceptor MUST clear all tokens and redirect to `/login`.
- Domain-specific API functions MUST live in `src/api/<domain>.ts` files — no inline
  `api.get(...)` calls inside component bodies.

**Rationale:** A single HTTP gateway ensures every request is authenticated, every 401
is handled consistently, and token rotation logic lives in exactly one place. Scattered
`fetch()` calls would inevitably miss the interceptor and silently expose unauthenticated
requests or broken token refresh.

---

### III. React Quality Gate (React Doctor)

Every implementation phase MUST pass a React Doctor scan before that phase is considered
complete and before the next phase begins.

- React Doctor findings of severity **ERROR** are blocking — the phase MUST NOT be marked
  done until all errors are resolved.
- React Doctor findings of severity **WARNING** MUST be reviewed. Warnings may be deferred
  only with an explicit written justification added as a code comment at the affected site.
- The scan MUST target the full `src/` directory, not individual files.
- Acceptable React Doctor categories to enforce: state & effects, performance, architecture,
  security, accessibility.

This principle exists because speckit-implement generates React code phase by phase.
Anti-patterns introduced early (e.g., stale closure bugs, missing dependency arrays,
direct DOM manipulation) compound across later phases and become expensive to fix.

**Rationale:** React Doctor provides deterministic, automated enforcement of React best
practices that would otherwise require manual review per PR. Gating each phase prevents
technical debt accumulation before the next phase builds on top of it.

---

### IV. Token Security

Access tokens MUST be stored exclusively in memory (inside `AuthContext` state).
They MUST NOT be written to `localStorage`, `sessionStorage`, `cookies`, or any
browser-persistent storage.

- Refresh tokens MAY be stored in `localStorage` solely to support page-reload persistence.
- On logout (explicit or triggered by refresh failure), ALL tokens MUST be cleared from
  both memory and `localStorage` before the redirect to `/login`.
- Tokens MUST NOT be logged, appended to URLs as query parameters, or exposed in error
  messages rendered to the user.
- The WebSocket connection to `/sessions/:sid/ws` MUST pass the access token as a query
  parameter (`?token=<jwt>`) solely because the WebSocket protocol does not support
  custom headers — this is the only permitted exception to the header-only rule.

**Rationale:** Storing access tokens in `localStorage` exposes them to XSS attacks.
Keeping them in memory limits the blast radius of a script injection to the current
tab session. The refresh token in `localStorage` is an accepted tradeoff for usability,
consistent with the `httpOnly` cookie alternative being unavailable in this deployment.

---

### V. Read-Only Evidence Display

Evidence data returned by the API MUST be rendered as read-only in the UI. The following
data types are evidence and MUST NEVER be editable by any client:

- Trace events (`/submissions/:sid/trace`)
- Analysis results (AI, detection, format validation)
- Appeal records once submitted
- Submission timestamps

Grade decisions (`grade_decisions`) are editable by the docente only, exclusively through
the GradePanel component, and MUST be submitted via the API — never patched locally in
client state before the API confirms the write.

**Rationale:** This mirrors the Immutable Evidence principle in the `nextlab-api`
constitution (Principle I). Allowing client-side editing of evidence — even cosmetically —
creates the appearance of tampering and undermines the legal defensibility of the appeal
process.

---

### VI. Brand Baseline & UX Iteration

The Unifranz visual identity provides the **starting baseline** for the frontend.
UX adjustments based on user feedback are expected, encouraged, and do not require
a constitution amendment. The design is meant to evolve.

The following tokens are the initial defaults — they MAY be adjusted in response
to user feedback without any governance process:

| Token | Default value | Usage |
|-------|--------------|-------|
| Accent | `#E85D24` | Buttons, active states, badges, left-border stripes |
| Background | `#111111` | Page background |
| Surface | `#1a1a1a` | Cards, panels |
| Surface-2 | `#242424` | Nested surfaces, inputs |
| Border | `#2e2e2e` | Dividers |
| Text | `#f0f0f0` | Primary text |
| Muted | `#888888` | Secondary / placeholder text |
| Font body | Inter | All prose |
| Font mono | JetBrains Mono | Trace output, hashes, code |

Layouts, spacing, page structure, interactions, animations, navigation patterns,
and component design are all freely changeable at any time based on feedback.

The **one hard constraint** in this principle: no external UI component library
(e.g., shadcn, MUI, Chakra) may be introduced. All components are built from scratch
with Tailwind CSS v4. This rule exists because a component library imposes its own
design system on top of the palette, making future adjustments harder, not easier.

**Rationale:** A frontend that cannot respond to user feedback is not useful. The
Unifranz palette is a starting point for brand consistency, not a cage. Keeping
components hand-built ensures that any UX change — however large — is always one
CSS edit away.

---

### VII. Thin Client — No Business Logic

The frontend MUST be a pure display and input layer. All computation, enforcement,
validation, and decision-making belongs exclusively to `nextlab-api`. The following
are FORBIDDEN in `nextlab-web` source code:

- **Grade computation** — do not calculate, average, or derive grades from raw data.
  Display exactly what the API returns.
- **Deadline enforcement** — do not compute whether a submission is late, block uploads
  based on time, or show countdown timers that affect submission acceptance. The backend
  marks `is_late`; the frontend displays it.
- **Format validation** — do not inspect file contents, MIME types beyond basic browser
  accept attributes, or enforce file size limits. The backend rejects invalid files.
- **Permission logic** — do not derive access rights from data fields (e.g., checking
  `actividad.docente_id === user.id` in a component). Role routing (Principle I) is the
  only authorization the frontend performs; all data-level authorization lives in the API.
- **Detection / AI heuristics** — do not compute integrity scores, similarity ratios,
  or risk levels. Render what the API provides.
- **State machines** — do not manage activity state transitions (open → closed → graded)
  on the client. Query the API for current state and render accordingly.
- **Data transformation** — do not reshape, aggregate, or filter API responses before
  display unless it is purely cosmetic (e.g., sorting a list for UI order). If a view
  needs different data, add an API endpoint, do not massage data in the component.

The only logic permitted on the frontend:

- Routing and role-based redirect (Principle I)
- Empty-field and basic format checks in forms (e.g., "email must contain @") —
  for immediate UI feedback only; the API always re-validates
- WebSocket lifecycle management (connect on session open, disconnect on close)
- Loading, error, and empty-state rendering

**Rationale:** Duplicating business logic across two codebases creates divergence bugs
where the UI and API disagree on what is allowed. Keeping all logic in the backend means
a single source of truth, consistent behavior across any future client (mobile, CLI), and
a frontend that is trivially replaceable.

---

## Technology & Architecture Constraints

The following stack decisions are locked and MUST NOT be swapped without a
constitution amendment:

| Layer | Technology | Notes |
|-------|-----------|-------|
| Framework | React 19 | Concurrent features available |
| Build tool | Vite 8 | `@tailwindcss/vite` plugin already configured |
| Language | TypeScript 5.9 | Strict mode; no `any` without justification |
| Styling | Tailwind CSS v4 | `@import "tailwindcss"` syntax (not v3 directives) |
| Routing | React Router v7 | `createBrowserRouter` + `RouterProvider` |
| HTTP | axios | Centralized instance; no raw fetch |
| Icons | Lucide React | Consistent icon set |
| State | React Context + useReducer | No external state manager |
| Package manager | pnpm | Already used in project |
| Quality gate | React Doctor (skill) | Run after each phase |

**API base URL:** Configured via `VITE_API_URL` environment variable.  
Default: `http://localhost:8080` (nextlab-api Docker container).

**WebSocket URL:** `ws://localhost:8080/sessions/:sid/ws?token=<jwt>`

No additional runtime dependencies may be added without updating this table.

---

## Implementation Milestones

Development is organized into 10 milestones (MW1–MW10), each aligned to one or more
`nextlab-api` backend milestones. A milestone is DONE when its deliverable works
end-to-end against the live API AND passes a React Doctor scan (Principle III).

| Milestone | Frontend Scope | Backend APIs | Deliverable |
|-----------|---------------|-------------|-------------|
| MW1 — Foundation & Auth | Axios client, AuthContext, router, login, register | M1 | Login → role dashboard → logout works |
| MW2 — Docente: Course Management | Courses CRUD, actividades CRUD, grupos, students roster | M2, M3 | Docente can create a course and add an activity |
| MW3 — Admin Panel | Carreras, materias, student-profiles CRUD | M3 | Admin can manage catalog entities |
| MW4 — Student Portal | Student course list, activity list, activity detail page | M2, M3 | Student sees their assigned activities |
| MW5 — File Submissions | Student upload/replace/delete; docente submission list + download | M4 | Student uploads a file; docente sees it in the list |
| MW6 — Tracing | Student session open + WebSocket trace panel; docente trace viewer | M5 | Docente sees real-time trace events for a session |
| MW7 — Grading | Exam session form, grade decision panel; student grades view; investigación stages; defensa oral scheduling | M5, M11 | Docente grades a submission; student sees the grade |
| MW8 — Git & AI Results | Student git submission form; docente git analysis + AI/detection report display | M6, M7, M8 | Docente can read AI and detection reports |
| MW9 — Alerts & Analytics | Docente alerts list + dismiss; activity analytics page; course analytics page | M9, M10, M12 | Analytics page renders grade distribution |
| MW10 — Appeals | Student appeal form; docente appeal list + resolve | M13 | Student files appeal; docente sees it |

**React Doctor gate:** MUST run after every milestone before marking it complete.
**Thin Client gate:** Every milestone review MUST confirm that no business logic
was added to the frontend (Principle VII).

---

## Governance

**Authority:** This constitution supersedes all README conventions, inline comments,
and verbal agreements for the `nextlab-web` project.

**Companion authority:** The `nextlab-api` constitution governs all server-side behavior.
In case of conflict between the two constitutions, API-side rules take precedence for
data integrity and security concerns; frontend-side rules take precedence for rendering
and UX concerns.

**Amendment procedure:**
1. Open a PR against `main` that modifies `.specify/memory/constitution.md`.
2. State the version bump type (MAJOR / MINOR / PATCH) and rationale in the PR description.
3. Update the Sync Impact Report comment at the top of this file.
4. At least one reviewer familiar with the affected phase MUST approve.

**Versioning policy:**
- **MAJOR** — backward-incompatible removal or redefinition of a principle.
- **MINOR** — new principle or section added; material expansion of guidance.
- **PATCH** — clarification, wording, typo fix; no semantic change.

**Compliance review:** Constitution gates MUST be checked at the start of each
implementation phase. Any phase task that would violate a principle MUST be redesigned
before implementation begins.

**Version**: 1.2.0 | **Ratified**: 2026-05-24 | **Last Amended**: 2026-05-24
