# Quickstart / Test Scenarios: MW1 — Foundation & Auth

Prerequisites: `nextlab-api` running on `http://localhost:8080`.
Use the smoke test seed users from the API (created via pgcrypto bcrypt seeding).

---

## Scenario 1: Login as each role → correct home screen

```
1. Navigate to http://localhost:5173/login
2. Enter credentials for a docente user
3. Submit
4. Expected: redirected to /docente/dashboard
   Page shows "Docente Dashboard" (placeholder)

Repeat for:
  - estudiante  → /student/dashboard
  - co_docente  → /codocente/dashboard
  - admin       → /admin/dashboard
```

---

## Scenario 2: Wrong credentials → generic error

```
1. Navigate to /login
2. Enter email: "test@unifranz.edu.bo", password: "wrong"
3. Submit
4. Expected: error message "Invalid email or password" appears below form
   User stays on /login
   No redirect
```

---

## Scenario 3: Redirect-back after login

```
1. While signed out, navigate directly to /docente/dashboard
2. Expected: redirected to /login
3. Sign in as a docente
4. Expected: redirected back to /docente/dashboard (not /login or /)
```

---

## Scenario 4: Cross-role redirect

```
1. Sign in as estudiante
2. Manually navigate to /docente/dashboard
3. Expected: immediately redirected to /student/dashboard
```

---

## Scenario 5: Session persistence across refresh

```
1. Sign in as any user
2. Note current URL (e.g., /docente/dashboard)
3. Press F5 (hard refresh)
4. Expected: still on /docente/dashboard, still signed in
   No login prompt
```

---

## Scenario 6: Explicit sign-out

```
1. Sign in as any user
2. Click the sign-out button in the Topbar
3. Expected: redirected to /login
4. Press browser back button
5. Expected: still on /login (cannot go back into protected screen)
```

---

## Scenario 7: Rate limiting

```
1. Navigate to /login
2. Submit wrong credentials 6+ times in quick succession
3. Expected: error message includes "Too many attempts. Try again in X seconds"
   Submit button remains disabled or shows the message clearly
```

---

## Scenario 8: Network error

```
1. Stop the nextlab-api Docker container
2. Navigate to /login and attempt sign-in
3. Expected: "Unable to reach the server. Check your connection." message
   No redirect, no crash
```
