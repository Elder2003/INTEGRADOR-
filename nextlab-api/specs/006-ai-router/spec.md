# Feature Specification: M6 — AI Router

**Feature Branch**: `006-ai-router`

**Created**: 2026-05-20

**Status**: Draft

**Input**: User description: "go with milestone 6"

## Clarifications

### Session 2026-05-20

- Q: When an administrator changes the routing mode, should that change be recorded as an append-only audit entry, or is updating a single config row sufficient? → A: Append-only history — each mode change inserts a new record; current mode is derived from the latest entry; full change history is preserved.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Route an AI Request Through the Configured Backend (Priority: P1)

A downstream component (detection engine, format validator, RAG pipeline) sends a prompt to the AI router and receives a completion. The router directs the request to the backend currently configured by the active mode — local, auto, or cloud — and returns the response to the caller. The caller does not need to know which backend was used.

**Why this priority**: Without a working router that can accept and forward a request, no AI-powered feature in the system can function. This is the core capability every subsequent milestone depends on.

**Independent Test**: Can be fully tested by sending a prompt to the router endpoint and confirming a valid completion is returned; the log entry confirms which backend processed the request.

**Acceptance Scenarios**:

1. **Given** the router is in `local` mode, **When** a prompt is submitted, **Then** the response comes from the primary backend and the log records `backend=local`.
2. **Given** the router is in `cloud` mode, **When** a prompt is submitted, **Then** the response comes from the cloud backend and the log records `backend=cloud`.
3. **Given** the router is in `auto` mode and the primary backend is healthy, **When** a prompt is submitted, **Then** the response comes from the primary backend.
4. **Given** an invalid or empty prompt, **When** it is submitted to the router, **Then** the router returns a structured error without forwarding the request to any backend.

---

### User Story 2 — Administrator Changes the Routing Mode at Runtime (Priority: P2)

An administrator updates the active routing mode via an admin-only management endpoint. The change does not require a service restart. Within 5 seconds, all new AI requests use the new mode. Requests in-flight at the moment of the change complete on the backend they originally targeted.

**Why this priority**: Hot-swap is the key operational feature of the router. Without it, changing backend strategy requires downtime, which defeats the purpose of the three-mode design.

**Independent Test**: Can be fully tested by changing the mode via the management endpoint, then sending a prompt within 5 seconds, and confirming the log shows the new backend and that a new RouterConfig history entry was created.

**Acceptance Scenarios**:

1. **Given** the router is in `local` mode, **When** an administrator sets the mode to `cloud`, **Then** requests issued ≥ 5 seconds later use the cloud backend, and a new RouterConfig history entry records the change with the administrator's identity and timestamp.
2. **Given** the router is in `auto` mode, **When** an administrator sets the mode to `local`, **Then** subsequent requests no longer fall back to cloud even if the primary backend is unavailable.
3. **Given** a non-admin user, **When** they attempt to change the routing mode, **Then** the system rejects the request with an authorization error and no RouterConfig history entry is created.
4. **Given** an invalid mode value (e.g., `turbo`), **When** an administrator submits it, **Then** the system rejects the request, the current mode is unchanged, and no RouterConfig history entry is created.

---

### User Story 3 — Automatic Fallback in Auto Mode (Priority: P3)

When the router is in `auto` mode and the primary backend (Ollama) fails or times out, the router transparently retries the request against the cloud fallback (Vercel AI Gateway) without the caller having to handle the error. The caller receives a successful response; the log records that the fallback was used.

**Why this priority**: Auto mode is the production-safe configuration. It provides resilience without manual intervention, and is required by the constitution's worker SLA (≤ 3 minutes total).

**Independent Test**: Can be fully tested by setting the primary backend to an unreachable address while in `auto` mode, submitting a prompt, and confirming the response arrives from the cloud backend within the SLA.

**Acceptance Scenarios**:

1. **Given** the router is in `auto` mode and the primary backend is unreachable, **When** a prompt is submitted, **Then** the router retries via the cloud backend and returns a valid completion.
2. **Given** the router is in `auto` mode and the primary backend times out, **When** a prompt is submitted, **Then** the fallback is triggered and the log records `backend=cloud, fallback_reason=primary_timeout`.
3. **Given** both backends are unavailable in `auto` mode, **When** a prompt is submitted, **Then** the router returns a structured error identifying that all backends failed.
4. **Given** the router is in `local` mode (not `auto`), **When** the primary backend fails, **Then** the router returns an error immediately without attempting the cloud backend.

---

### User Story 4 — Every AI Request Is Logged for Audit Traceability (Priority: P4)

Every prompt sent through the router produces a structured log entry that records which backend processed the request, the routing mode active at the time, whether the request succeeded or failed, and any fallback events. This log feeds the constitution's audit-traceability requirement.

**Why this priority**: Audit traceability is a constitution requirement (Principle III). However, logging does not block the other user stories — it can be verified independently once the router is operational.

**Independent Test**: Can be fully tested by sending several requests under different modes, then querying the log store and confirming each entry contains the required fields.

**Acceptance Scenarios**:

1. **Given** a successful request in `local` mode, **When** the log is inspected, **Then** the entry includes: `request_id`, `mode`, `backend_used`, `latency_ms`, `status=success`, `timestamp`.
2. **Given** a fallback event in `auto` mode, **When** the log is inspected, **Then** the entry additionally includes `fallback_reason` and records `backend_used=cloud`.
3. **Given** a failed request (both backends unavailable), **When** the log is inspected, **Then** the entry records `status=failed` and the reason for each backend's failure.
4. **Given** a high-volume scenario (multiple concurrent requests), **When** the logs are inspected, **Then** every request has exactly one log entry — no duplicates, no omissions.

---

### Edge Cases

- What happens if both backends are unavailable simultaneously in `auto` mode?
- How does the router handle a mode change that arrives while a request is already being forwarded to a backend?
- What if the cloud backend returns a rate-limit error (HTTP 429)?
- What if the primary backend is reachable but returns a non-200 status that does not represent a timeout?
- What if two administrators change the mode concurrently?
- How does the router behave during application startup before the first mode configuration is loaded?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST support three routing modes: `local` (primary backend only), `auto` (primary with cloud fallback), and `cloud` (cloud backend only).
- **FR-002**: The active routing mode MUST be configurable at runtime by an administrator without restarting the service.
- **FR-003**: A mode change MUST take effect for new requests within 5 seconds of the configuration update.
- **FR-004**: In `auto` mode, if the primary backend returns an error or does not respond within a configurable timeout, the system MUST automatically retry the request on the cloud backend.
- **FR-005**: In `local` mode, the system MUST NOT forward requests to the cloud backend under any circumstance, including primary backend failure.
- **FR-006**: In `cloud` mode, the system MUST NOT forward requests to the primary backend under any circumstance.
- **FR-007**: The router MUST expose a management endpoint to read the current routing mode (administrator role required for write; readable by any authenticated internal caller).
- **FR-008**: Every AI request MUST produce a log entry containing: request identifier, routing mode, backend used, response latency, success/failure status, and fallback reason (when applicable).
- **FR-009**: The router MUST accept prompt/completion requests in a format compatible with the primary backend's API contract.
- **FR-010**: When all available backends fail, the router MUST return a structured error to the caller indicating that no backend was able to process the request.
- **FR-011**: The router MUST NOT expose or log the content of prompts or completions — only metadata (mode, backend, latency, status) is retained in the audit log.
- **FR-012**: Every routing mode change MUST be recorded as an append-only history entry storing the new mode, the previous mode, the administrator's identity, and the timestamp. The current mode is always derived from the latest entry.

### Key Entities

- **RouterConfig**: An append-only history of routing mode changes. Each record stores: the mode set (`local` / `auto` / `cloud`), the previous mode, the administrator who made the change, and the timestamp. The active mode is always the most recent entry. No record is ever updated or deleted.
- **AIRequestLog**: An immutable record of a single routing event: request identifier, mode active at dispatch time, backend(s) attempted, latency per backend, final outcome, and fallback reason if triggered.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Routing overhead (time added by the router beyond backend inference) is under 50 milliseconds for 99% of requests.
- **SC-002**: A routing mode change takes effect for all new requests within 5 seconds of the administrator's update, verified by log timestamps.
- **SC-003**: In `auto` mode with the primary backend unavailable, the fallback response is delivered to the caller within 10 seconds total (combined primary timeout + cloud inference).
- **SC-004**: 100% of requests processed by the router produce a corresponding log entry with all required fields present.
- **SC-005**: When one backend is unavailable, the router continues to serve requests via the available backend with zero manual intervention required.
- **SC-006**: Concurrent mode changes by multiple administrators result in exactly one mode being active — no undefined or split state.
- **SC-007**: Every mode change produces exactly one RouterConfig history entry; the full change history is queryable at any time without data loss.

## Assumptions

- The primary backend (Ollama) exposes an OpenAI-compatible completions endpoint; the router uses this contract for all primary-backend calls.
- The cloud fallback (Vercel AI Gateway) also exposes an OpenAI-compatible endpoint; no additional translation layer is needed between the two backends.
- The routing mode is cached in-process (refreshed every ≤ 5 seconds from the database) so the DB is the source of truth and the cache ensures fast dispatch.
- Prompt content and completion content are NOT stored in the audit log — only request metadata — to comply with data-minimisation principles.
- The primary backend timeout (before triggering fallback in `auto` mode) defaults to 30 seconds and is configurable per deployment.
- Only users with the `admin` role may change the routing mode; other roles may read the current mode.
- The router is an internal component — it is not directly exposed to students or docentes; it is called by other backend services (detection engine, RAG pipeline, worker).
- Both backend connection details (URLs, API keys) are provided via environment configuration and are not changeable via the management endpoint.
- The initial routing mode on first startup (when no RouterConfig history exists) defaults to `local` and is recorded as the first history entry automatically.
