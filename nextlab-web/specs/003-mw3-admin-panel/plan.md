# Implementation Plan: MW3 — Admin Panel

**Branch**: `013-appeals-final-audit` | **Date**: 2026-05-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-mw3-admin-panel/spec.md`  
**Research**: `specs/003-mw3-admin-panel/research.md`  
**Data Model**: `specs/003-mw3-admin-panel/data-model.md`  
**Contracts**: `specs/003-mw3-admin-panel/contracts/admin.md`

---

## Summary

Build the admin catalog management area: CRUD pages for carreras, materias, and student profiles. All three entities are required before any other platform activity can occur. Data is fetched live from `nextlab-api`; no mocks. Key discovery: student profile "delete" is a soft-deactivation via `PUT` (no DELETE endpoint exists). Carrera and materia deactivation is triggered via `DELETE` which the server handles as soft-delete, returning 204.

Key decisions from research (see research.md for full rationale):
- Student profiles deactivated via `PUT { is_active: false }` — no DELETE endpoint
- `Carrera` and `StudentProfile` types added to `src/types/academic.ts` (not a new file)
- New `src/api/admin.ts` for all admin CRUD (separate from `academic.ts` which serves docentes)
- Materia form has 4 required fields: code, name, semester (1–10), carrera dropdown
- `@unifranz.edu.bo` email validation is server-side only; UI displays server error

---

## Technical Context

**Language/Version**: TypeScript 5.9 (strict mode)

**Primary Dependencies**: React 19, React Router v7, axios, Tailwind CSS v4, Lucide React

**Storage**: N/A — frontend only; all data fetched from `nextlab-api` via HTTP

**Testing**: Manual smoke tests via `quickstart.md`. No automated tests for MW3.

**Target Platform**: Browser (React SPA at `http://localhost:5173`)

**Project Type**: Web application — frontend display layer

**Performance Goals**: List pages load in under 3 seconds (SC-002)

**Constraints**: No external UI component library. No business logic on the frontend (constitution Principle VII). No email domain validation in the frontend — display whatever the server returns.

**Scale/Scope**: ~4 pages, 1 new API module, ~6 new types. Small, self-contained milestone.

---

## Constitution Check

*GATE: Must pass before implementation. Re-checked after design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Role-Gated Rendering | ✅ Pass | All 3 new admin routes inside the existing admin `ProtectedRoute` block. Role derived from JWT via `AuthContext`. Non-admin users redirected to their own dashboard, never shown a blank page or 403. |
| II. API Client Isolation | ✅ Pass | `src/api/admin.ts` imports `api` from `../lib/api`. No raw `fetch()`. All admin API functions defined in that module; no inline `api.get(...)` in component bodies. |
| III. React Quality Gate | ✅ Pass | React Doctor scan required after all MW3 pages are complete, before milestone is marked done. |
| IV. Token Security | ✅ Pass | No token handling changes. Access token stays in `AuthContext` memory. Axios interceptor handles auth automatically. |
| V. Read-Only Evidence | ✅ Pass | No evidence data in MW3. Not applicable. |
| VI. Brand Baseline | ✅ Pass | Unifranz palette throughout. No shadcn, MUI, or any component library. Tables and forms built with Tailwind CSS v4. |
| VII. Thin Client | ✅ Pass | No business logic. Frontend only checks for empty required fields before submit. All real validation (email domain, code uniqueness, active-dependency blocking) is server-side. Frontend displays whatever error the API returns via `<ErrorMessage>`. |

No violations. All gates pass.

---

## Project Structure

### Documentation (this feature)

```text
specs/003-mw3-admin-panel/
├── plan.md           ← this file
├── research.md       ← Phase 0 output (7 decisions)
├── data-model.md     ← Phase 1 output
├── quickstart.md     ← Phase 1 output
├── contracts/
│   └── admin.md      ← Phase 1 output
├── tasks.md          ← Phase 2 output (speckit-tasks — NOT created by speckit-plan)
└── checklists/
    └── requirements.md
```

### Source Code Changes

Files to **create** (new):

```text
src/
├── api/
│   └── admin.ts               # All admin CRUD: carreras, materias, student-profiles
│
└── pages/
    └── admin/
        ├── CarrerasPage.tsx       # List + inline create + inline edit + deactivate
        ├── MateriasPage.tsx       # List + inline create + inline edit + deactivate
        └── StudentProfilesPage.tsx  # List + inline create + inline edit + deactivate
```

Files to **update** (existing):

```text
src/
├── types/
│   └── academic.ts    # Add Carrera, StudentProfile + 6 new request interfaces
│
├── pages/
│   └── admin/
│       └── DashboardPage.tsx  # Replace placeholder with nav cards to 3 pages
│
└── router/
    └── index.tsx      # Add 3 new admin child routes
```

---

## Key Implementation Details

### `src/api/admin.ts` — Error handling

Follows the same `toApiError` helper pattern as `academic.ts`. Copy helper locally or import from a shared location if one exists:

```ts
export async function deactivateStudentProfile(id: string): Promise<StudentProfile> {
  try {
    const res = await api.put<StudentProfile>(`/admin/student-profiles/${id}`, { is_active: false })
    return res.data
  } catch (err) {
    toApiError(err)
  }
}
```

### Page pattern — inline CRUD (no sub-routes)

All CRUD for each entity is on a single page. No separate `/new` or `/:id/edit` routes. Pattern per row:

```tsx
{editingId === item.id ? (
  <EditRow ... />
) : (
  <DisplayRow onEdit={() => setEditingId(item.id)} onDeactivate={() => { setSelected(item); setShowConfirm(true) }} />
)}
```

Action errors are shown via `<ErrorMessage>` below the page heading — **not** inside `<ConfirmModal>` (MW2 established pattern).

### `MateriasPage` — semester select

Hardcoded `<select>` with 10 options (1–10). Server validates the value.

### `MateriasPage` — carrera dependency

On mount: fetches `getCarreras()` for the dropdown. If empty: shows message + disabled submit ("Crea una carrera primero").

### `StudentProfilesPage` — deactivation wording

No DELETE endpoint exists. Confirm modal:
- Title: `"Desactivar perfil"`
- Message: `"¿Desactivar el perfil de {profile.email}? El estudiante no podrá acceder hasta que sea reactivado."`
- Confirm label: `"Desactivar"`

### `src/router/index.tsx` — admin routes

Inside the existing admin `ProtectedRoute` children array:

```ts
{ path: 'carreras', element: <CarrerasPage /> },
{ path: 'materias', element: <MateriasPage /> },
{ path: 'student-profiles', element: <StudentProfilesPage /> },
```

---

## Scope Adjustments (vs. original nextlab-web.md plan)

| Spec claim | Actual behavior | Source |
|-----------|-----------------|--------|
| Student profiles CRUD (implied DELETE) | No DELETE endpoint; deactivation via PUT `{ is_active: false }` | Research Decision 1 |
| Admin dashboard shows entity counts | Counts not returned by list endpoints; dashboard shows nav links only | Research Decision 7 |

---

## Complexity Tracking

No constitution violations. No complexity justification required.
