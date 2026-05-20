# Research: M3 — Academic Management

## Decision 1: Google OAuth Library

**Decision**: `golang.org/x/oauth2` for the OAuth 2.0 flow + `google.golang.org/api/idtoken` for ID token verification.

**Rationale**: Both are maintained by Google, widely used in production Go services, and support the full OpenID Connect flow. `idtoken.Validate` handles signature verification, audience check, and expiry — no manual JWT parsing needed for Google tokens. The existing M2 JWT library (`golang-jwt/jwt/v5`) is separate and continues handling the internal access/refresh tokens we issue.

**Alternatives considered**:
- `markbates/goth`: Higher-level abstraction, but adds an unnecessary dependency layer for a single provider; also not idiomatic with Fiber.
- Parsing the Google ID token manually with `golang-jwt/jwt/v5`: Would require fetching Google's public keys and managing key rotation — handled automatically by `idtoken.Validate`.

---

## Decision 2: Student Identity and Enrollment Source

**Decision**: Admin-provisioned `student_profiles` table (email → carrera_id + semester). Google OAuth verifies the student's identity (proves they own the `@unifranz.edu.bo` email); enrollment data comes entirely from the admin table.

**Rationale**: Google Workspace custom attributes (the alternative) require the Unifranz IT administrator to configure a custom schema and populate attributes for every student — a dependency the team cannot control or guarantee. The admin-table approach gives the project team full ownership of the enrollment data and works without IT coordination. It also allows semester corrections without waiting on Google Workspace propagation.

**Alternatives considered**:
- **Google Workspace Directory API + custom schema**: Read `carrera_code` and `semester` from custom attributes on the Google user object. Rejected because it requires IT admin setup and introduces an external dependency at runtime (API call to Google Admin SDK on every login).
- **Email prefix parsing** (e.g., extract student ID from `20221234@unifranz.edu.bo`): Rejected because Unifranz email format is not confirmed to encode a parseable student ID; brittle if format changes.

---

## Decision 3: Enrollment Trigger Timing

**Decision**: Sync inscripciones on **every** student Google login, not just on first login.

**Rationale**: Students advance one semester per period. If sync only happens on first login, a student who logged in during semester 3 and advances to semester 4 would still be enrolled in semester 3 cursos indefinitely. Re-syncing on every login keeps enrollment current automatically: new cursos for the new semester are added; stale cursos from the previous semester are revoked with `revocation_reason = 'semester_advance'`.

**Performance note**: The sync query is O(n) where n = number of active cursos for the student's carrera+semester. At Unifranz scale (~5–8 cursos per semester), this adds < 50ms to the login response. Acceptable.

**Alternatives considered**:
- First-login only: Simpler, but requires a manual admin trigger or a cron job to update enrollment when a student advances — operational overhead.
- Nightly cron job: Correct but requires a background worker that doesn't exist until M5. Deferred.

---

## Decision 4: Activity Submission Lock Guard

**Decision**: `has_submissions BOOLEAN NOT NULL DEFAULT false` column on `actividades`. M3 initializes it to `false`. M4 (file storage) sets it to `true` when the first submission is stored. M3's service layer checks this flag before allowing title/type edits or deletion.

**Rationale**: M3 precedes M4 — there is no submissions table yet. Rather than leave FR-003's lock behavior unimplemented, a boolean flag creates a clean contract between milestones: M3 defines and respects the flag; M4 activates it. This avoids a cross-milestone schema dependency and is easy to test (admin can manually flip the flag in integration tests).

**Alternatives considered**:
- FK `COUNT(submissions WHERE actividad_id = ...)` check at mutation time: Correct long-term, but requires the submissions table to exist, which is M4+. Cannot be implemented in M3.
- Defer the lock behavior entirely to M4: Acceptable, but FR-003 is in the M3 spec. The flag approach satisfies the spec without blocking M4.

---

## Decision 5: `trace_enabled` Column Implementation

**Decision**: PostgreSQL stored generated column: `trace_enabled BOOLEAN GENERATED ALWAYS AS (tipo IN ('examen', 'defensa_oral')) STORED`.

**Rationale**: The flag is 100% derived from `tipo` — there is no scenario where a docente should manually override it (Constitution Principle VI: trace activation is proportional to risk; it applies to exams and oral defenses only). A generated column enforces this at the DB level — no application code can accidentally set `trace_enabled = true` for a `practica`. It also means the flag is always correct even if the activity type is somehow updated (though type is locked after submissions exist).

**Alternatives considered**:
- Application-layer derivation (compute in Go, never store): Would require every query to compute `tipo IN ('examen', 'defensa_oral')` — fragile if the list ever changes.
- Regular column with a CHECK trigger: More verbose; the generated column approach is simpler and enforces the constraint atomically.

---

## Decision 6: Student Group Uniqueness Enforcement

**Decision**: `grupo_asignaciones` includes a denormalized `curso_id` column with a `UNIQUE(user_id, curso_id)` constraint. This enforces one-group-per-student-per-course at the DB level.

**Rationale**: The logical constraint is "a student belongs to at most one grupo per curso." Without denormalizing `curso_id` into `grupo_asignaciones`, this constraint would require a deferred trigger or application-layer enforcement. Denormalizing the `curso_id` (which is deterministically derivable from `grupo_id` via the `grupos` table) makes the constraint expressible as a simple UNIQUE index — the DB enforces it atomically on every INSERT or conflict.

**Alternatives considered**:
- Service-layer check only: Susceptible to race conditions under concurrent requests. Rejected.
- Exclusion constraint via JOIN: Not supported as a declarative constraint in PostgreSQL without an extension.

---

## Decision 7: OAuth CSRF State Token

**Decision**: Generate a 32-byte random state value, store it in Redis with a 60-second TTL (`SET oauth_state:{value} "1" EX 60`), validate on callback (GET + DEL), then proceed. Stateless PKCE is not used because the callback is server-side (no native client involved).

**Rationale**: The existing Redis client from M1 is already available. A 60-second TTL covers the time a user spends on the Google consent screen. Validating and immediately deleting the key prevents state replay attacks.

**Alternatives considered**:
- Signed state cookie: Viable but adds complexity; Redis is already available.
- No state validation: Exposes the callback to CSRF — rejected on security grounds.
