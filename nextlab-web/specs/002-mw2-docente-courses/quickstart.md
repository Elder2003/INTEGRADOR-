# Quickstart / Test Scenarios: MW2 — Docente Course Management

Prerequisites:
- `nextlab-api` running on `http://localhost:8080`
- A docente user account (email/password login — students use Google OAuth only)
- At least one materia record visible to admin (or known materia UUID for fallback input)

---

## Scenario 1: View empty course list

```
1. Log in as docente
2. Navigate to /docente/dashboard
3. Expected: "No courses yet" empty state message is shown
   A "New Course" CTA button is visible
```

---

## Scenario 2: Create a course (materia dropdown available)

```
1. From the dashboard, click "New Course"
2. Expected: navigated to /docente/cursos/new
3. If materia dropdown is populated (admin account exists with materias):
   a. Select a materia from the dropdown
   b. Enter semester: "2026-1"
   c. Enter section: "A"
   d. Click Create
4. Expected: redirected to /docente/cursos/<new-id>?tab=actividades
   Course detail page shows the new course name and "0 students"
```

---

## Scenario 3: Create a course (materia UUID fallback)

```
1. From the dashboard, click "New Course"
2. If no materias loaded (docente gets 403 on /admin/materias):
   Expected: text input appears with message "Contact your admin to get the materia ID"
   Dropdown is replaced by a plain UUID text field
3. Enter a valid materia UUID
4. Enter semester and section
5. Create — course appears in the list
```

---

## Scenario 4: Course detail — Actividades tab

```
1. Click any course from the dashboard
2. Expected: /docente/cursos/:id?tab=actividades
   "Actividades" tab is active
   Activity list is empty (or shows existing activities)
3. Click "New Activity"
4. Expected: navigated to /docente/cursos/:id/actividades/new
```

---

## Scenario 5: Tab URL persistence and back-navigation

```
1. On the course detail page, click the "Grupos" tab
2. Expected: URL changes to /docente/cursos/:id?tab=grupos
3. Click "New Activity" → create an activity → redirected back
4. Expected: URL is /docente/cursos/:id?tab=actividades
5. Press browser back button
6. Expected: returns to /docente/cursos/:id?tab=grupos (correct tab restored)
```

---

## Scenario 6: Create an activity

```
1. Navigate to /docente/cursos/:id/actividades/new
2. Fill: title="Tarea 1", tipo="tarea", deadline=<next week datetime>, instructions="..."
3. Submit
4. Expected: redirected to /docente/cursos/:id?tab=actividades
   New activity "Tarea 1" appears in the list with a "tarea" badge
```

---

## Scenario 7: Activity tipo — only 5 values shown

```
1. Open the activity creation form
2. Expand the tipo select
3. Expected: exactly 5 options: practica, tarea, informe, examen, defensa_oral
   No "investigacion" or "git_analysis" options
```

---

## Scenario 8: Edit activity — field locking

```
1. Open an existing activity with has_submissions = false
2. Click Edit
3. Expected: title and tipo fields are editable
4. Edit an activity with has_submissions = true
5. Expected: title and tipo inputs are disabled
   A note "Locked: submissions exist" is shown near those fields
```

---

## Scenario 9: Delete activity — confirmation required

```
1. On the Actividades tab, click Delete on any activity
2. Expected: ConfirmModal appears with "Delete this activity?" text
3. Click Cancel
4. Expected: modal closes, activity still present
5. Click Delete again → confirm
6. Expected: activity removed from list
```

---

## Scenario 10: Delete course — API constraint

```
1. Attempt to delete a course that has submission records
2. Expected: ConfirmModal appears
3. Confirm
4. Expected: modal closes; API error message shown inline on the course detail page
   (e.g., "Cannot delete course: submissions exist")
   Course is NOT deleted
```

---

## Scenario 11: View student roster

```
1. On a course detail page, click the "Estudiantes" tab
2. Expected: URL = /docente/cursos/:id?tab=estudiantes
   Roster table shows enrolled students (email, enrollment date, group)
   No "enroll" button — roster is read-only
```

---

## Scenario 12: Create and manage grupos

```
1. On the "Grupos" tab, click "New Group"
2. Enter nombre="Grupo A", capacity=4
3. Save
4. Expected: new group appears with "0/4 members" display
5. From the Estudiantes tab, assign a student to "Grupo A"
6. Expected: student row shows "Grupo A"; group shows "1/4 members"
```

---

## Scenario 13: Co-docente grant

```
1. Navigate to an activity detail page
2. Enter a valid co-docente user ID in the grant form
3. Click "Grant Access"
4. Expected: co-docente entry appears in the list
5. Click "Revoke" on that entry
6. Expected: entry removed immediately from the list
7. Refresh the page
8. Expected: grant list is empty (no GET endpoint — state is lost on reload)
   Note "Grants refresh on reload" is shown
```

---

## Scenario 14: Toggle late submissions

```
1. On an activity detail page, the "Late submissions allowed" toggle shows the current state
2. Toggle it
3. Expected: API is called (PUT /actividades/:id with late_submission_allowed toggled)
   UI reflects the new state
```

---

## Scenario 15: React Doctor gate

```
1. After completing all above scenarios, run:
   npx react-doctor@latest .
2. Expected: 0 findings with severity ERROR
   Any WARNINGs reviewed and justified
```
