# Google OAuth Integration — Replacement Guide

**Last updated**: 2026-05-19
**Milestone**: M3 — Academic Management
**Purpose**: Documents the Google OAuth student login integration so that it can be understood, audited, and replaced in the future if needed.

---

## What This Integration Does

Students at Unifranz authenticate using their institutional Google account (`@unifranz.edu.bo`). Google proves the student owns that email address. The system then uses that verified email to:

1. Look up the student's carrera and semester in the admin-provisioned `student_profiles` table.
2. Create or update the student's internal account (`users` row).
3. Automatically enroll the student in all active cursos matching their carrera and semester (`inscripciones`).
4. Issue the same JWT access + refresh token pair used by docentes and admins.

**Google's role is identity verification only.** It does not control carrera or semester data — that is owned by the admin via `student_profiles`. This is an intentional design decision (see `research.md` — Decision 2).

---

## Requirements This Fulfills

| Requirement | Description |
|---|---|
| FR-006 | Auto-associate student with matching cursos on login |
| FR-017 | Read carrera + semester from student's institutional identity at login |
| FR-018 | Authenticate students via institutional Google account; auto-create account on first login |

---

## Current Data Flow

```
Student browser
  │
  ├─ GET /auth/google
  │     └─ Server generates CSRF state (32 bytes), stores in Redis (60s TTL)
  │        Redirects to Google OAuth consent screen
  │
  ├─ [Student approves consent on Google]
  │
  └─ GET /auth/google/callback?code=...&state=...
        ├─ Validate CSRF state (Redis GET + DEL)
        ├─ Exchange authorization code → Google tokens
        ├─ Verify Google ID token (signature + audience + expiry)
        ├─ Extract email from ID token
        ├─ SELECT * FROM student_profiles WHERE email = $email
        │     └─ If not found → 403 "account not provisioned"
        ├─ UPSERT INTO users (email, role='estudiante', google_sub) VALUES (...)
        ├─ EnrollmentSync(userID, carreraID, semester)
        │     ├─ Find active cursos matching carrera+semester
        │     ├─ INSERT missing inscripciones
        │     └─ Revoke stale inscripciones (semester_advance)
        └─ Issue JWT (access token 15m + refresh token 7d)
           Same issueTokenPair() function used by email/password login
```

---

## Code Locations

| File | Responsibility |
|---|---|
| `internal/auth/google.go` | `GoogleLoginHandler` (redirect) + `GoogleCallbackHandler` (token exchange, profile lookup, enrollment sync call, JWT issue) |
| `internal/auth/module.go` | Route registration: `GET /auth/google`, `GET /auth/google/callback` |
| `internal/academic/service.go` | `EnrollmentSync(ctx, userID, carreraID, semester)` — creates/revokes inscripciones |
| `internal/config/config.go` | `GoogleClientID`, `GoogleClientSecret`, `GoogleRedirectURL` fields |
| `migrations/000003_academic.up.sql` | `student_profiles` table + `google_sub` column on `users` + nullable `password_hash` |

**Go libraries used**:
- `golang.org/x/oauth2` — OAuth 2.0 flow (authorization URL, code exchange)
- `golang.org/x/oauth2/google` — Google-specific OAuth endpoints
- `google.golang.org/api/idtoken` — Google ID token verification

---

## What Would Need to Change If You Replace This

The integration has been designed with a single seam: the `GoogleCallbackHandler` function in `internal/auth/google.go`. It is the **only place** that knows about Google. Everything downstream (account creation, enrollment sync, JWT issuance) is provider-agnostic.

To replace Google OAuth with a different identity provider:

### Step 1: Replace the OAuth flow
Delete or rename `internal/auth/google.go`. Write a new handler that:
- Redirects the student to the new provider's auth endpoint.
- Handles the callback and receives a verified identity (email or user ID).

### Step 2: Keep the same downstream contract
The callback handler must produce **one thing**: a verified `email` string (or equivalent unique identifier). Then it calls the same functions:
```go
// These do NOT change regardless of identity provider:
profile := GetStudentProfileByEmail(ctx, db, email)
UpsertUser(ctx, db, email, role="estudiante", providerSub)
EnrollmentSync(ctx, db, userID, profile.CarreraID, profile.Semester)
issueTokenPair(ctx, user, nil)
```

### Step 3: Update config
Replace `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL` in `config.go` with the new provider's config fields.

### Step 4: Update the `users` table
Replace `google_sub` column name with the new provider's identifier (or make it generic, e.g., `oauth_sub TEXT`, `oauth_provider TEXT`).

---

## Possible Future Alternatives

| Alternative | When to use | Notes |
|---|---|---|
| **Microsoft Entra ID (Azure AD)** | If Unifranz migrates email to Microsoft 365 | Change OAuth endpoints + library to `golang.org/x/oauth2/microsoft`; downstream code unchanged |
| **Unifranz internal SSO** | If the university builds its own IdP | Replace the OAuth flow; keep `student_profiles` and `EnrollmentSync` |
| **SAML 2.0** | Enterprise SSO requirement | Add a SAML library; the callback produces email; rest unchanged |
| **Direct carrera/semester from IdP token** | If IT admin configures custom claims in Google Workspace | `student_profiles` lookup can be eliminated; read carrera+semester directly from token claims. `student_profiles` table becomes optional. |
| **Enrollment API integration** | If Unifranz exposes an enrollment API | Replace `student_profiles` admin table with a call to the enrollment API on each login; `EnrollmentSync` stays the same shape |

---

## FAQ

**Q: Why not read carrera + semester directly from Google Workspace?**
A: This requires the Unifranz IT administrator to configure custom attributes in Google Workspace Admin Console and populate them for every student. This was rejected as a hard dependency on external IT configuration. The `student_profiles` approach gives the project team full control. See `research.md` — Decision 2.

**Q: Why is enrollment re-synced on every login instead of just first login?**
A: Students advance one semester per period. Re-syncing on every login keeps enrollment current automatically when a student's `semester` field is updated in `student_profiles`. See `research.md` — Decision 3.

**Q: Do docentes and admins use Google OAuth?**
A: No. Docentes and admins use email + bcrypt password (M2). Google OAuth is exclusively for students with `@unifranz.edu.bo` accounts.

**Q: What happens if a student's Google account changes email?**
A: The `google_sub` column stores Google's stable user ID (not email). If the email changes but `google_sub` stays the same, the system can still identify the user. However, `student_profiles` is keyed by email — an admin update to the profile would be needed. For now this is an edge case flagged for M12 (Admin Modules).
