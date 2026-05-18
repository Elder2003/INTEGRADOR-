# Feature Specification: M1 — Fundación del Proyecto

**Feature Branch**: `001-m1-foundation`

**Created**: 2026-05-17

**Status**: Draft

**Input**: Milestone M1 from `integrador-backend-milestones.md` — establish a running,
containerised Go project with a healthy database, structured logging, and a
modular codebase ready for all subsequent milestones.

---

## Clarifications

### Session 2026-05-17

- Q: Should database migrations run automatically at API startup or via an explicit separate command? → A: Automatically at API container startup; a standalone rollback command also exists for explicit use.
- Q: Beyond timestamp and message, which fields are required in every log entry? → A: `level`, `service`, and `correlation_id`.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Developer Starts the Full Stack with One Command (Priority: P1)

A developer clones the repository, copies the example environment file, and runs
a single orchestration command. Within 60 seconds all services are running and
accepting requests. The developer can immediately confirm everything is healthy
by calling the health-check endpoint.

**Why this priority**: Without a reproducible local environment nothing else can
be built or tested. This is the absolute prerequisite for every subsequent milestone.

**Independent Test**: Running the orchestration command from a clean clone and
receiving a 200 response from the health-check endpoint with all dependencies
reported as connected — no other feature needs to exist.

**Acceptance Scenarios**:

1. **Given** a developer has Docker and Docker Compose installed on their machine,
   **When** they run the project orchestration command from the repository root,
   **Then** all five services (API, background worker, database, cache, object
   storage) are running and healthy within 60 seconds.

2. **Given** all services are running,
   **When** the developer calls `GET /health`,
   **Then** the endpoint responds with HTTP 200 and a JSON body that reports the
   connectivity status of the database, cache, and object storage independently.

3. **Given** one dependency (e.g., cache) is stopped,
   **When** the developer calls `GET /health`,
   **Then** the endpoint responds with HTTP 503 and the JSON body identifies the
   failing dependency by name.

---

### User Story 2 — Developer Adds a New Domain Module Without Touching Shared Code (Priority: P2)

A developer assigned to a new business domain (e.g., notifications) can create
their module's folder, add their source files, and register their routes — all
without modifying bootstrap files, the main entry point, or any sibling module.

**Why this priority**: The 13-milestone roadmap depends on parallel feature work.
A foundation that requires modifying shared files to add a module becomes a
merge-conflict bottleneck from Milestone 2 onward.

**Independent Test**: Creating a new empty module directory and a stub handler
file, then verifying the project still builds and the health check still passes —
without touching any existing file outside the new directory.

**Acceptance Scenarios**:

1. **Given** the project structure is established,
   **When** a developer adds a new module directory with a stub source file,
   **Then** the project builds successfully without modifying any existing file.

2. **Given** a module is added,
   **When** the developer registers a new route in the module's own file,
   **Then** that route is reachable without changes to central routing or bootstrap
   code.

---

### User Story 3 — Operator Applies and Rolls Back Database Schema Migrations (Priority: P3)

An operator or CI pipeline can advance or revert the database schema version
using a dedicated migration command. Each migration is versioned, idempotent
on re-run, and independently reversible.

**Why this priority**: Reliable schema management is required from Milestone 2
(when the first tables are created). Without it, any schema change risks
inconsistent database state across environments.

**Independent Test**: Running the migration command twice on an empty database
produces the same final schema on both runs. Running the rollback command returns
the schema to the previous version without data corruption.

**Acceptance Scenarios**:

1. **Given** an empty PostgreSQL database with the pgvector extension available,
   **When** the migration command is executed,
   **Then** all defined schema objects are created and the migration version table
   records the applied version.

2. **Given** migrations have been applied,
   **When** the migration command is run a second time,
   **Then** no error occurs and the schema is unchanged (idempotent).

3. **Given** a migration has been applied,
   **When** the rollback command is executed for that version,
   **Then** the schema reverts to the previous version without corrupting existing
   data.

---

### Edge Cases

- What happens when the database container starts but pgvector extension is not
  yet installed? The health check MUST report the database as unhealthy and the
  migration command MUST fail with a descriptive error.
- What happens if the automatic migration fails at API startup? The API MUST
  refuse to start, log the failing migration name and error at error severity,
  and exit with a non-zero code so the container orchestrator marks it as
  unhealthy.
- What happens if an environment variable required for a dependency is missing at
  startup? The service MUST refuse to start and log the missing variable name at
  error severity before exiting.
- What happens when the object-storage service is unavailable at startup? The API
  MUST start regardless (object storage is not required for the health check to
  partially succeed), but the health-check endpoint MUST report it as degraded.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST start all five services — REST API, background
  worker, relational database, in-memory cache, and object storage — with a single
  orchestration command.

- **FR-002**: All services MUST be containerised so that the local environment
  reproduces the production topology without manual installation of runtime
  dependencies.

- **FR-003**: The REST API MUST expose a `GET /health` endpoint that checks
  connectivity to the database, cache, and object storage, and returns a JSON
  document reporting the status of each dependency individually.

- **FR-004**: The codebase MUST be organised in domain modules; each module owns
  its own source files, handlers, and business logic with no forced coupling to
  sibling modules.

- **FR-005**: Database schema changes MUST be managed through versioned,
  reversible migrations. The API container MUST automatically apply all pending
  migrations before accepting requests at startup. A standalone command MUST also
  exist for rollback and explicit CI/CD use.

- **FR-006**: All runtime configuration — service addresses, credentials, feature
  flags — MUST be sourced from environment variables; no values MUST be hardcoded
  in application source.

- **FR-007**: All application log output MUST be structured JSON with at minimum
  `info`, `warn`, and `error` severity levels. Every entry MUST include:
  `timestamp`, `message`, `level`, `service` (identifies API vs. worker), and
  `correlation_id` (links a request across both processes).

- **FR-008**: The container image for the API and worker MUST use a multi-stage
  build so that the production image contains only the compiled binary and its
  runtime dependencies, not the build toolchain.

- **FR-009**: The database MUST have the vector-search extension enabled from
  the initial migration so that subsequent milestones can create vector-typed
  columns without schema changes to the foundation layer.


### Key Entities

- **Service configuration**: The complete set of environment variables that control
  connectivity addresses, credentials, and behavioural parameters for each service.
  Acts as the single source of truth for all environment-specific settings.

- **Health report**: A structured document returned by the health-check endpoint
  listing each dependency, its reachability status (`ok` / `degraded` / `down`),
  and an optional latency measurement. HTTP status reflects aggregate health
  (200 all-ok, 503 any-down).

- **Database migration**: A versioned, named, reversible schema change. Migrations
  are applied in ascending version order and may be rolled back one version at a
  time.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All five services are ready to accept requests within **60 seconds**
  of the orchestration command on a machine that already has container images
  cached.

- **SC-002**: `GET /health` responds in under **500 ms** and the response body
  identifies each dependency by name with its individual status.

- **SC-003**: A developer with access to the repository and a machine with
  container tooling installed can have the full stack running from a clean checkout
  in **under 10 minutes** by following the project README.

- **SC-004**: Adding a new domain module (directory + stub handler) and
  registering one route requires changes to **zero existing files** outside the
  new module directory.

- **SC-005**: Every log line emitted during a local test run is valid JSON; zero
  plain-text log lines are present.

- **SC-006**: Applying migrations to a clean database and then rolling them back
  returns the database to its pre-migration state with **no orphaned schema
  objects** remaining.

---

## Assumptions

- Docker and Docker Compose (v2+) are available on all developer machines; no
  other local runtimes need to be installed to run the project.
- The primary deployment target is a Linux/AMD64 server (Hetzner CX42); the
  container image is built for that architecture.
- Local development credentials (database password, MinIO access key, etc.) are
  non-sensitive placeholder values defined in the repository's example environment
  file; production secrets are injected separately.
- The API and background worker are two separate Go binaries compiled from the
  same codebase and the same commit. They share the same database, cache, and
  object storage — the worker does not manage any infrastructure independently.
- The initial database migration establishes the extension and any shared
  infrastructure (e.g., a `schema_migrations` version table) but does not create
  domain-specific tables — those are added in later milestones.
