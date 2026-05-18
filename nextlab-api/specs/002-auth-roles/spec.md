# Feature Specification: M2 — Authentication and Roles

**Feature Branch**: `002-auth-roles`

**Created**: 2026-05-17

**Status**: Draft

**Input**: User description: "I want you to specify the milestone 2"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Credential-Based Login (Priority: P1)

A user with a registered account can enter their email and password to gain access to the system. Upon successful authentication, they receive a time-limited session that identifies who they are and what role they hold. The session remains valid for a short period; when it expires, the user must either refresh it or log in again.

**Why this priority**: Without a working login, no other part of the system is accessible. Every other milestone depends on knowing the identity and role of the caller.

**Independent Test**: Can be tested by registering a user in the database directly, calling the login endpoint with valid credentials, and verifying that the response contains a valid access token and a refresh token.

**Acceptance Scenarios**:

1. **Given** a registered user with email `docente@unifranz.edu.bo` and a valid password, **When** they submit their credentials to the login endpoint, **Then** they receive an access token (valid for 15 minutes) and a refresh token.
2. **Given** a registered user, **When** they submit an incorrect password, **Then** the system returns an authentication error with no token issued.
3. **Given** a registered user, **When** they submit a non-existent email, **Then** the system returns the same generic error as a wrong password (no account enumeration).
4. **Given** an access token that has expired, **When** a protected endpoint is called with that token, **Then** the request is rejected with an authentication error.

---

### User Story 2 - Token Refresh Without Re-Login (Priority: P2)

A logged-in user can exchange their refresh token for a new access token without re-entering their password. This allows uninterrupted use of the system across multiple hours or days without storing long-lived credentials. When a refresh token is used, it is immediately replaced with a new one (rotation), invalidating the previous one.

**Why this priority**: Short-lived access tokens (15 min) require a refresh mechanism to keep users logged in. Without refresh, users would be forced to re-enter credentials every 15 minutes, which is unusable.

**Independent Test**: Can be tested by obtaining a valid refresh token via login, waiting for the access token to expire, then calling the refresh endpoint and verifying a new valid access token is returned and the old refresh token is no longer accepted.

**Acceptance Scenarios**:

1. **Given** a valid refresh token, **When** the user calls the refresh endpoint, **Then** they receive a new access token and a new refresh token; the old refresh token is invalidated.
2. **Given** an already-used refresh token, **When** it is submitted to the refresh endpoint again, **Then** the request is rejected (replay prevention).
3. **Given** a logged-in user, **When** they call the logout endpoint, **Then** their refresh token is invalidated and subsequent refresh attempts are rejected.

---

### User Story 3 - Role-Based Access Control (Priority: P3)

The system supports four distinct user roles: administrator, teacher, co-teacher, and student. Each role can only access endpoints appropriate to their responsibilities. Attempting to access a route outside one's role results in a clear denial. This is enforced automatically for every protected endpoint without requiring callers to declare their own role.

**Why this priority**: The academic audit system has strict information boundaries — students must not access teacher views, teachers must not access admin configuration. Role enforcement is a prerequisite for all subsequent milestones.

**Independent Test**: Can be tested by obtaining valid tokens for users of different roles and attempting cross-role requests; verifying that forbidden responses are returned when role requirements are not met.

**Acceptance Scenarios**:

1. **Given** a user with the `student` role, **When** they call an endpoint designated for teachers only, **Then** they receive an authorization error.
2. **Given** a user with the `docente` role, **When** they call an endpoint designated for `admin` only, **Then** they receive an authorization error.
3. **Given** a user with the `admin` role, **When** they call any endpoint, **Then** access is granted (admin has full access).
4. **Given** a user with the `co_docente` role, **When** they call teacher-accessible endpoints, **Then** access is granted (co-teacher shares teacher-level access).
5. **Given** a request with no authentication token, **When** a protected endpoint is called, **Then** the request is rejected regardless of the endpoint's role requirement.

---

### User Story 4 - Password Recovery (Priority: P4)

A user who has forgotten their password can request a recovery link sent to their registered email address. Following the link takes them through a secure, time-limited flow to set a new password. Once reset, their previous password is no longer valid.

**Why this priority**: Password recovery is a basic usability requirement for a production system. It can be delivered after core authentication is working without blocking any other milestone.

**Independent Test**: Can be tested by requesting a password reset for a known email, using the returned reset token to set a new password, and verifying that the new password works for login while the old one does not.

**Acceptance Scenarios**:

1. **Given** a registered email, **When** the user requests a password reset, **Then** a time-limited reset token is generated (valid for 1 hour) and the response acknowledges the request without revealing whether the email exists.
2. **Given** a valid reset token, **When** the user submits a new password, **Then** the password is updated and the reset token is consumed (cannot be reused).
3. **Given** an expired or already-used reset token, **When** a reset is attempted, **Then** the request is rejected.
4. **Given** a non-existent email, **When** the user requests a password reset, **Then** the system responds identically to a valid email (no account enumeration).

---

### Edge Cases

- What happens when the same user logs in from two devices simultaneously? Both sessions should remain valid independently.
- What happens when an admin deactivates a user account? Active tokens for that user must stop working before their natural expiry.
- What happens when a refresh token is stolen and used by an attacker before the legitimate user? Token rotation detects the reuse and can invalidate the entire session family.
- How does the system handle a login attempt after too many consecutive failures? The system imposes a short cooldown to slow brute-force attempts (rate limiting, not account lockout).
- What happens if a user tries to reset their password while already logged in? The reset flow should work regardless of current session state.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST authenticate users by verifying a submitted email and password against stored credentials.
- **FR-002**: System MUST issue a short-lived access token (15-minute validity) and a long-lived refresh token upon successful login.
- **FR-003**: System MUST allow users to exchange a valid refresh token for a new access token and a new refresh token, invalidating the old refresh token immediately (rotation).
- **FR-004**: System MUST allow users to explicitly invalidate their session (logout), after which the refresh token is permanently rejected.
- **FR-005**: System MUST protect every non-public endpoint by verifying the access token on each request.
- **FR-006**: System MUST enforce role-based access on protected endpoints, rejecting requests from users whose role is not permitted for that endpoint.
- **FR-007**: System MUST support four roles: `admin`, `docente`, `co_docente`, `estudiante`, with `admin` having access to all routes and `co_docente` sharing the same access level as `docente`.
- **FR-008**: System MUST store passwords as non-reversible hashes; plaintext passwords must never be persisted or logged.
- **FR-009**: System MUST support a password recovery flow: a user requests a reset link by email, which contains a time-limited single-use token to authorize setting a new password.
- **FR-010**: System MUST allow tracking and revocation of refresh tokens per user and per session, so that a single compromised session can be invalidated without affecting other active sessions for the same user.
- **FR-011**: System MUST return identical error responses for invalid email and invalid password during login to prevent account enumeration.
- **FR-012**: System MUST return identical responses for existing and non-existing emails during password reset requests.

### Key Entities

- **User**: represents a person with system access; has a unique email address, a hashed password, an assigned role, and an account status (active/inactive).
- **Role**: one of four values — `admin`, `docente`, `co_docente`, `estudiante` — assigned to each user at account creation and controlling endpoint access.
- **Session**: the authenticated context established at login; associated with a user, has an access token (short-lived) and a refresh token (long-lived), can be individually revoked.
- **Password Reset Request**: a time-limited, single-use authorization token sent to a user's registered email to allow setting a new password.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user with valid credentials can complete the login flow and receive a working access token in under 2 seconds under normal load.
- **SC-002**: An access token that has passed its 15-minute validity window is rejected by all protected endpoints with no additional configuration.
- **SC-003**: A user whose role does not permit access to a given endpoint is denied 100% of the time, regardless of how the request is constructed.
- **SC-004**: A logout action renders the corresponding refresh token invalid within the same request cycle; no subsequent refresh using that token succeeds.
- **SC-005**: A password reset token expires after 1 hour and cannot be used more than once; both conditions are enforced independently.
- **SC-006**: No endpoint in the authentication or authorization flows reveals whether a specific email address is registered in the system.
- **SC-007**: The system can handle at least 50 concurrent authentication requests without returning errors or significant latency degradation.

## Assumptions

- M1 (project foundation with Docker, PostgreSQL, and health check) is fully operational before M2 work begins.
- User accounts are created directly in the database (no self-registration endpoint is in scope for M2); account provisioning by admin is a M12 concern.
- Email delivery for password reset is out of scope for M2; the reset token will be returned in the API response directly (simulating delivery) and real email sending will be added in a later milestone.
- The `co_docente` role has identical endpoint access to `docente` for all routes in M2 scope.
- There is no account lockout policy (permanent lockout after N failures) — only temporary rate limiting to slow brute-force attempts; permanent lockout would create a denial-of-service vector.
- Token signing keys are provided via environment variables and are not rotated dynamically during M2.
- Password minimum length is 8 characters; no complexity rules are enforced beyond that for M2.
