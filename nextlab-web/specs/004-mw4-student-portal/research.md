# Research: MW4 — Student Portal

**Feature**: MW4 — Student Portal
**Date**: 2026-05-26

---

## Decision 1: StudentCursoEntry and StudentActividadEntry types go in `src/types/academic.ts`

**Decision**: Add both new response shapes to the existing `src/types/academic.ts` file.

**Rationale**: MW3 set the pattern — all academic domain types (Carrera, StudentProfile, Materia, etc.) live in `src/types/academic.ts`. Keeping all academic shapes in one file avoids fragmentation and keeps imports predictable.

**Alternatives considered**: New `src/types/student.ts` file — rejected. Unnecessary splitting; the academic.ts file is not large and these types are clearly in the academic domain.

**API shapes confirmed from source code** (`internal/academic/model.go`):
```go
type StudentCursoEntry struct {
  ID            uuid.UUID `json:"id"`
  MateriaName   string    `json:"materia_name"`
  DocenteEmail  string    `json:"docente_email"`
  SemesterLabel string    `json:"semester_label"`
  Section       string    `json:"section"`
  Grupo         *GrupoRef `json:"grupo"`  // nullable
}

type GrupoRef struct {
  ID   uuid.UUID `json:"id"`
  Name string    `json:"name"`
}

type StudentActividadEntry struct {
  ID           uuid.UUID `json:"id"`
  Title        string    `json:"title"`
  Tipo         string    `json:"tipo"`
  Deadline     time.Time `json:"deadline"`
  Instructions string    `json:"instructions"`
  TraceEnabled bool      `json:"trace_enabled"`
  PostedAt     time.Time `json:"posted_at"`
  LastEditedAt time.Time `json:"last_edited_at"`
}
```

---

## Decision 2: Student API functions go in new `src/api/student.ts`

**Decision**: Create a new `src/api/student.ts` module for all student-facing API calls.

**Rationale**: Mirrors the separation established in MW3: `admin.ts` for admin endpoints, `academic.ts` for docente endpoints. A `student.ts` module completes the role-to-module mapping and makes it immediately clear which API surface each role consumes. The constitution (Principle II) requires domain-specific API functions in `src/api/<domain>.ts`.

**Alternatives considered**: Adding to `academic.ts` — rejected. That file is the docente module; mixing student and docente calls would violate the role-separation intent and make future audits harder.

**Functions needed**:
- `getStudentCursos()` → `GET /me/cursos` (returns `StudentCursoEntry[]`)
- `getStudentActividades(cursoId)` → `GET /me/cursos/:id/actividades` (returns `StudentActividadEntry[]`)

---

## Decision 3: Student dashboard IS the course list (no separate `/student/cursos` route)

**Decision**: Replace `src/pages/student/DashboardPage.tsx` with the course list. The route `/student/dashboard` shows enrolled courses — no redirect, no separate `/student/cursos` page.

**Rationale**: The spec defines US1 as "A student logs in and is taken to their dashboard. The dashboard shows all courses..." — the course list IS the dashboard. Adding a separate route for the course list would require an intermediate redirect and add unnecessary complexity.

**Alternatives considered**: `/student/cursos` as a separate page with the dashboard being a welcome screen — rejected. Extra page + redirect adds friction without delivering any value.

---

## Decision 4: Activity list is a new page at `/student/cursos/:id/actividades`

**Decision**: Add a new page `src/pages/student/ActividadesPage.tsx` at the path `/student/cursos/:id/actividades`.

**Rationale**: US2 requires a distinct page for the activity list, not an expansion of the dashboard. The student navigates into a course context — a separate URL makes the navigation history sensible (back button works, shareable URL).

**Router addition**: New route added inside the existing `ProtectedRoute allowedRoles={['estudiante']}` block, alongside the dashboard route.

---

## Decision 5: Activity detail re-fetches the list and finds by `:aid` (confirmed in clarification)

**Decision**: `src/pages/student/ActividadDetailPage.tsx` calls `getStudentActividades(cursoId)` on mount, then filters by the `:aid` URL parameter.

**Rationale**: There is no `GET /me/cursos/:id/actividades/:aid` endpoint in the API (confirmed from `internal/academic/module.go`). Re-fetching the full list is the only server-side option. The list response is small (one course's activities), so the performance cost is negligible. This approach ensures direct URL navigation (bookmarks, shared links) works without relying on React Router `state`.

**Fallback**: If `:aid` does not match any activity in the list (activity deleted, wrong URL), the page shows a not-found message with a link back to the activity list.

---

## Decision 6: Old Bloque 4 mocked student pages are deleted

**Decision**: Delete all 10 mocked student pages from `src/pages/student/` that are NOT imported by the router: `Dashboard.tsx`, `Exam.tsx`, `Practice.tsx`, `Task.tsx`, `Report.tsx`, `Project.tsx`, `Defense.tsx`, `Grades.tsx`, `Appeal.tsx`, `ProjectStage.tsx`.

**Rationale**: These were the original Bloque 4 scaffold with hardcoded mock data. They are not imported by the router, they use mock data patterns that violate the constitution (no API calls), and keeping them creates confusion about what is real code versus prototype. MW4 replaces them with API-connected pages.

**Files to keep**: `DashboardPage.tsx` (already in router, will be replaced with real content).

**Verification**: Confirmed from router read — only `DashboardPage.tsx` is imported for the student role; all other student page files are unused dead code.
