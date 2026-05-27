# Implementation Plan: MW2 — Docente Course Management

**Branch**: `013-appeals-final-audit` | **Date**: 2026-05-25 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/002-mw2-docente-courses/spec.md`  
**Research**: `specs/002-mw2-docente-courses/research.md`  
**Data Model**: `specs/002-mw2-docente-courses/data-model.md`  
**Contracts**: `specs/002-mw2-docente-courses/contracts/academic.md`

---

## Summary

Build the docente course management area: a course list dashboard, tabbed course detail
screen, activity CRUD, group management, student roster view, and co-docente grant controls.
All data is fetched live from the `nextlab-api`; no mocks remain after MW2. The Bloque 4
prototype `ActividadTipo` is corrected to the 5-value API enum.

Key discoveries from research (see research.md for full rationale):
- `actividad_tipo` has 5 values, not 7
- `GET /admin/materias` is admin-only; docentes fall back to manual UUID input
- Title and `tipo` are editable until first submission (`has_submissions = false`)
- Course deletion is blocked by submissions, not enrolled students
- No `GET /actividades/:aid/grants` — grant list lives in React state only

---

## Technical Context

**Language/Version**: TypeScript 5.9 (strict mode)

**Primary Dependencies**: React 19, React Router v7, axios, Tailwind CSS v4, Lucide React

**New API module**: `src/api/academic.ts` — all course, activity, group, roster, and grant calls

**State management**: React local state (`useState`, `useEffect`) — no global store for MW2 data

**Routing**: `useSearchParams()` for tab persistence; React Router v7 `useParams()` for IDs

**Form inputs**: `<input type="datetime-local">` for deadline; convert to ISO 8601 before POST

**Testing**: Manual smoke tests via `quickstart.md`. No automated tests for MW2.

**Constraints**: No business logic on frontend (constitution Principle VII). No external libs.
Co-docente grant list is client-state only — lost on page refresh (no GET endpoint).

---

## Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Role-Gated Rendering | ✅ Pass | All new routes inside docente ProtectedRoute |
| II. API Client Isolation | ✅ Pass | All calls through `src/api/academic.ts` → `src/lib/api.ts` |
| III. React Quality Gate | ✅ Pass | React Doctor scan required before MW2 is done |
| IV. Token Security | ✅ Pass | No token handling changes in MW2 |
| V. Read-Only Evidence | ✅ Pass | No evidence data in MW2 |
| VI. Brand Baseline | ✅ Pass | Unifranz palette; no component library |
| VII. Thin Client | ✅ Pass | No grade computation, no deadline enforcement, no permission logic |

One deliberate thin-client exception: `getMaterias()` silently returns `[]` on 403
instead of throwing. This is error-handling (graceful degradation on a known API limitation),
not permission logic — the frontend does not derive access rights from it.

No violations. All gates pass.

---

## Project Structure

### Documentation (this feature)

```text
specs/002-mw2-docente-courses/
├── plan.md          ← this file
├── research.md      ← Phase 0 output (7 decisions)
├── data-model.md    ← Phase 1 output
├── quickstart.md    ← Phase 1 output
├── contracts/
│   └── academic.md  ← Phase 1 output
├── tasks.md         ← Phase 2 output (speckit-tasks)
└── checklists/
    └── requirements.md
```

### Source Code Changes

Files to **create** (new):

```text
src/
├── api/
│   └── academic.ts                            # All academic domain API calls
│
├── pages/
│   └── docente/
│       ├── CursoDetailPage.tsx                # Tabbed course detail
│       ├── CreateCursoPage.tsx                # Course creation form
│       ├── EditCursoPage.tsx                  # Edit semester/section
│       ├── CreateActividadPage.tsx            # Activity creation form
│       ├── ActividadDetailPage.tsx            # Activity detail + grants
│       └── EditActividadPage.tsx              # Activity edit form
│
└── components/
    ├── common/
    │   └── ConfirmModal.tsx                   # Shared delete confirmation modal
    └── docente/
        ├── CourseCard.tsx                     # Course summary card
        └── ActividadTypeSelect.tsx            # <select> for 5 actividad tipos
```

Files to **update** (existing):

```text
src/
├── types/
│   └── index.ts           # Add MW2 types; fix ActividadTipo to 5 values
│
├── pages/
│   └── docente/
│       └── DashboardPage.tsx   # Replace placeholder with real course list
│
└── router/
    └── index.tsx              # Add 6 new docente routes
```

---

## Key Implementation Details

### `src/api/academic.ts` — Error handling

All functions import `api` from `../lib/api` and throw `ApiError` (same pattern as
`src/api/auth.ts`). Error extraction helper `toApiError(err)` is copied from auth.ts.

Exception: `getMaterias()` catches 403 and returns `[]`:

```ts
export async function getMaterias(): Promise<Materia[]> {
  try {
    const res = await api.get<Materia[]>('/admin/materias')
    return res.data
  } catch (err) {
    const axiosErr = err as AxiosError
    if (axiosErr.response?.status === 403) return []
    throw toApiError(err)
  }
}
```

### `src/router/index.tsx` — Route ordering constraint

`/docente/cursos/new` MUST be declared before `/docente/cursos/:id`. React Router v7
matches routes in definition order; if `:id` comes first, `"new"` is treated as a UUID
param and renders the wrong page.

```ts
// Inside the docente ProtectedRoute children array:
{ path: 'cursos/new', element: <CreateCursoPage /> },
{ path: 'cursos/:id', element: <CursoDetailPage /> },
{ path: 'cursos/:id/edit', element: <EditCursoPage /> },
{ path: 'cursos/:id/actividades/new', element: <CreateActividadPage /> },
{ path: 'cursos/:id/actividades/:aid', element: <ActividadDetailPage /> },
{ path: 'cursos/:id/actividades/:aid/edit', element: <EditActividadPage /> },
```

### `CursoDetailPage` — Tab URL encoding

```ts
const [searchParams, setSearchParams] = useSearchParams()
const activeTab = searchParams.get('tab') ?? 'actividades'

function setTab(tab: string) {
  setSearchParams({ tab }, { replace: false })
}
```

`replace: false` is intentional — each tab click adds a history entry so browser back
navigation restores the previous tab (spec US3 AC5).

### `CreateActividadPage` — Deadline conversion

```ts
// deadline from <input type="datetime-local"> is "YYYY-MM-DDTHH:mm"
// API needs ISO 8601 with seconds and timezone
const isoDeadline = deadlineLocal + ':00Z'
```

The `:00` appends seconds; `Z` means UTC. Acceptable because the form does not expose
timezone selection — UTC is the consistent fallback (research Decision 6).

### `EditActividadPage` — Field locking

```ts
const locked = actividad.has_submissions

// In JSX:
<input
  disabled={locked}
  value={title}
  onChange={e => setTitle(e.target.value)}
/>
{locked && <p className="text-[#888888] text-sm">Locked: submissions exist</p>}
```

The API enforces this; the disabled state is purely UX. If the API returns `field_locked`
(422), `<ErrorMessage>` displays the message as received.

### `ConfirmModal` — Error display pattern

The modal does NOT display errors. Parent page pattern:

```ts
const [deleteError, setDeleteError] = useState<string | null>(null)
const [isDeleting, setIsDeleting] = useState(false)
const [showConfirm, setShowConfirm] = useState(false)

async function handleDeleteConfirm() {
  setIsDeleting(true)
  try {
    await deleteCurso(id)
    navigate('/docente/dashboard')
  } catch (err) {
    setDeleteError((err as ApiError).message)
  } finally {
    setIsDeleting(false)
    setShowConfirm(false)   // close modal in both success and failure cases
  }
}
```

Then below the page header: `{deleteError && <ErrorMessage message={deleteError} />}`

### Co-docente grants in `ActividadDetailPage`

```ts
const [grants, setGrants] = useState<CodocenteGrant[]>([])
const [grantUserId, setGrantUserId] = useState('')

async function handleGrant() {
  const grant = await grantAccess(aid, grantUserId)
  setGrants(prev => [...prev, grant])
  setGrantUserId('')
}

async function handleRevoke(userId: string) {
  await revokeAccess(aid, userId)
  setGrants(prev => prev.filter(g => g.user_id !== userId))
}
```

State is not restored on refresh — this is documented to the user via a note in the UI
("Co-docente list refreshes on page reload"). Backend tracking is in the DB; the UI
just cannot read it back (research Decision 5).

---

## Scope Adjustments (vs. original spec)

| Spec claim | Actual behavior | Source |
|-----------|-----------------|--------|
| Activity tipo: 7 values (incl. investigacion, git_analysis) | 5 values only | Research Decision 1 |
| Materia dropdown for docentes | Falls back to manual UUID input on 403 | Research Decision 2 |
| Activity tipo immutable after creation | Mutable until `has_submissions = true` | Research Decision 3 |
| Delete course: blocked by enrolled students | Blocked by any submission record | Research Decision 4 |
| Grant list always visible | Lost on page refresh (no GET endpoint) | Research Decision 5 |

---

## Complexity Tracking

No constitution violations. No complexity justification required.

Deliberate design choice: `getMaterias()` swallows 403 and returns `[]`. This is the
minimal viable response to a known API gap (admin-only endpoint) that avoids breaking
the course creation form for docentes.
