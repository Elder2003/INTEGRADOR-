# Feature Specification: MW1 — Foundation & Auth

**Feature Branch**: `001-mw1-foundation-auth`

**Created**: 2026-05-24

**Status**: Draft

**Input**: User description: "Lets do the mw1"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Sign In and Role Redirect (Priority: P1)

Any platform user — student, teacher, co-teacher, or administrator — can sign in
with their registered email and password. After a successful sign-in, the system
immediately sends them to the home screen that corresponds to their role.
A student lands on their student dashboard; a teacher lands on their course list;
an administrator lands on the admin panel.

**Why this priority**: Sign-in is the entry point to the entire platform. Without
it, no other milestone can be used or tested. It is the foundation on which every
other screen depends.

**Independent Test**: Can be fully tested by signing in as each of the four roles
and confirming each one lands on the correct home screen.

**Acceptance Scenarios**:

1. **Given** a registered user with role `estudiante`, **When** they submit valid credentials, **Then** they land on the student home screen.
2. **Given** a registered user with role `docente`, **When** they submit valid credentials, **Then** they land on the teacher course list.
3. **Given** a registered user with role `co_docente`, **When** they submit valid credentials, **Then** they land on the co-teacher view.
4. **Given** a registered user with role `admin`, **When** they submit valid credentials, **Then** they land on the admin panel.
5. **Given** any user, **When** they submit incorrect credentials, **Then** a clear error message appears and they remain on the sign-in screen.
6. **Given** any user, **When** a network error occurs during sign-in, **Then** a descriptive error message appears and no redirect happens.

---

### User Story 2 — Protected Navigation (Priority: P2)

Any screen beyond the sign-in and registration pages is protected. An unauthenticated
visitor who tries to open a protected URL is redirected to sign-in. A signed-in user
who navigates to a screen intended for a different role is redirected to their own
home screen rather than seeing an error or a blank page.

**Why this priority**: Without access control, every protected page is publicly
accessible. This must be in place before any other screens are built.

**Independent Test**: Can be fully tested by visiting protected URLs while signed
out, and by visiting cross-role URLs while signed in as a mismatched role.

**Acceptance Scenarios**:

1. **Given** an unauthenticated user, **When** they navigate directly to any protected URL, **Then** they are redirected to the sign-in screen.
2. **Given** a signed-in student, **When** they navigate to a teacher-only URL, **Then** they are redirected to the student home screen.
3. **Given** a signed-in teacher, **When** they navigate to the admin panel URL, **Then** they are redirected to the teacher home screen.
4. **Given** an unauthenticated user redirected to sign-in from a protected URL, **When** they sign in successfully, **Then** they are sent back to the URL they originally tried to visit.
5. **Given** a user who navigates directly to the sign-in screen (no prior URL), **When** they sign in successfully, **Then** they land on their role-specific home screen.

---

### User Story 3 — Account Registration (Priority: P3)

A new user can create an account by providing their name, email, and password.
After successful registration, they are immediately signed in and redirected to
the home screen corresponding to their assigned role.

**Why this priority**: Registration is needed to create test accounts for all
roles, but sign-in and access control are prerequisite to registration being useful.

**Independent Test**: Can be tested by registering a new email and confirming
the user lands on the correct dashboard and can navigate protected screens.

**Acceptance Scenarios**:

1. **Given** a new email address, **When** a user submits valid registration details, **Then** their account is created and they are signed in and redirected to their role home screen.
2. **Given** an already-registered email address, **When** a user tries to register with it, **Then** a clear error message appears and no account is created.
3. **Given** a registration form with missing required fields, **When** the user attempts to submit, **Then** the missing fields are highlighted before the form is submitted.

---

### User Story 4 — Session Persistence (Priority: P4)

A signed-in user who refreshes the page or closes and reopens the browser tab
returns to the same screen they were on, still signed in, without being asked
to sign in again.

**Why this priority**: Without session persistence, every page refresh forces a
sign-in, making the platform unusable during development and testing.

**Independent Test**: Can be tested by signing in, refreshing the page, and
confirming the user is still on the same screen with no sign-in prompt.

**Acceptance Scenarios**:

1. **Given** a signed-in user on any protected screen, **When** they refresh the page, **Then** they remain signed in and see the same screen.
2. **Given** a signed-in user, **When** they close and reopen the browser tab, **Then** they remain signed in.

---

### User Story 5 — Sign Out (Priority: P5)

A signed-in user can explicitly sign out from any screen. After signing out,
their session ends immediately and they are redirected to the sign-in screen.
Any subsequent attempt to access a protected screen redirects them to sign-in.

**Why this priority**: Sign-out is a basic security requirement but depends on
sign-in already working.

**Independent Test**: Can be tested by signing out and confirming that protected
URLs redirect to sign-in.

**Acceptance Scenarios**:

1. **Given** a signed-in user, **When** they click sign out, **Then** their session ends and they are redirected to the sign-in screen.
2. **Given** a user who just signed out, **When** they navigate to a protected URL, **Then** they are redirected to the sign-in screen.
3. **Given** a user who just signed out, **When** they press the browser back button, **Then** they are redirected to the sign-in screen (not back into a protected screen).

---

### User Story 6 — Automatic Session Expiry Handling (Priority: P6)

When a signed-in user's session expires while they are actively using the platform,
the system attempts to renew the session silently in the background. If renewal
fails, the user is redirected to the sign-in screen with a clear message explaining
that their session ended.

**Why this priority**: Session expiry is an edge case that can be deferred until
the core auth flow is solid.

**Independent Test**: Can be tested by letting a short-lived session expire and
confirming the outcome is either a seamless continuation or a graceful redirect.

**Acceptance Scenarios**:

1. **Given** a signed-in user whose session has expired, **When** they take any action that requires authentication, **Then** the session is renewed silently and the action completes without interruption.
2. **Given** a signed-in user whose session cannot be renewed, **When** they take any authenticated action, **Then** they are redirected to sign-in with a message indicating their session ended.

---

### Edge Cases

- What happens when the sign-in service is unavailable? → Show a descriptive error; do not redirect.
- What happens if a user manually edits the URL to a non-existent route? → Show a 404 screen with a link back to their home screen.
- What happens if a user's role changes server-side while they are signed in? → On next route navigation or page refresh, they are redirected to the correct home screen for their updated role.
- What happens if a user opens the app on two tabs and signs out on one? → The other tab redirects to sign-in on the next navigation or action.

## Clarifications

### Session 2026-05-24

- Q: After an unauthenticated user is redirected to sign-in, where do they land after successful sign-in? → A: Back to the URL they originally tried to visit (redirect-back / deep linking).
- Q: Should sign-in error messages be generic or specific (email-not-found vs wrong-password)? → A: Generic — "Invalid email or password" — to prevent email enumeration.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to sign in with their registered email and password.
- **FR-002**: System MUST redirect users back to the URL they originally tried to visit after successful sign-in. If no prior URL exists (direct sign-in), the system MUST redirect to their role-specific home screen.
- **FR-003**: System MUST display a single generic message "Invalid email or password" for any sign-in failure, regardless of whether the email exists or the password is wrong. The message MUST NOT reveal which field caused the failure.
- **FR-004**: System MUST allow new users to register an account with their name, email, and password.
- **FR-005**: System MUST reject registration attempts that use an already-registered email address and inform the user.
- **FR-006**: System MUST prevent unauthenticated users from accessing any screen other than sign-in and registration.
- **FR-007**: System MUST redirect authenticated users away from screens intended for a different role, to their own role's home screen.
- **FR-008**: System MUST maintain the user's signed-in state across page refreshes without requiring re-authentication.
- **FR-009**: System MUST attempt to silently renew an expired session before interrupting the user.
- **FR-010**: System MUST redirect the user to sign-in with an explanatory message when session renewal fails.
- **FR-011**: System MUST allow users to sign out explicitly from any screen.
- **FR-012**: System MUST end the session immediately upon sign-out and prevent back-navigation into protected screens.
- **FR-013**: System MUST display inline validation feedback on registration and sign-in forms before submission.

### Key Entities

- **User Session**: Represents an authenticated user's active identity on the platform. Holds the user's identity, assigned role, and validity period. Expires after a period of inactivity.
- **User**: A registered platform participant. Has a name, email address, and an assigned role. Role is assigned by the system and cannot be self-selected during registration.
- **Role**: Determines which screens and actions are accessible. Four values: `estudiante` (student), `docente` (teacher), `co_docente` (co-teacher), `admin` (administrator).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users complete the sign-in flow in under 3 seconds on a standard connection.
- **SC-002**: Incorrect credential errors appear within 2 seconds of form submission.
- **SC-003**: 100% of successful sign-ins result in a redirect to the correct role-specific home screen.
- **SC-004**: 0% of unauthenticated navigation attempts reach a protected screen.
- **SC-005**: 100% of cross-role navigation attempts redirect to the user's own home screen.
- **SC-006**: Users remain signed in after a page refresh without any re-authentication prompt.
- **SC-007**: 100% of explicit sign-out attempts end the session and redirect to sign-in immediately.

## Assumptions

- The authentication backend (sign-in, registration, session renewal, sign-out) is fully operational as part of the nextlab-api M1 milestone.
- Four roles exist in the system: `estudiante`, `docente`, `co_docente`, `admin`. No other roles are in scope.
- Role is assigned server-side at registration or by an administrator; users cannot choose their role.
- Email and password are the only credentials required. No social login, SSO, or two-factor authentication is in scope for MW1.
- The platform is used in a browser environment; mobile native apps are out of scope.
- A signed-out user who navigates to `/` is redirected to the sign-in screen.
- A signed-in user who navigates to `/` is redirected to their role-specific home screen.
