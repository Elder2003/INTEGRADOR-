<!--
SYNC IMPACT REPORT
==================
Version change: 0.0.0 (uninitialized template) → 1.0.0
Type of bump: MINOR — initial population of all sections from template placeholders.
Rationale: First substantive version; no prior principles existed to conflict or remove.

Modified principles:
  [PRINCIPLE_1_NAME] → I. Immutable Evidence
  [PRINCIPLE_2_NAME] → II. Asynchronous Analysis Isolation
  [PRINCIPLE_3_NAME] → III. Human Authority Over AI
  [PRINCIPLE_4_NAME] → IV. Strict Role-Based Data Isolation
  [PRINCIPLE_5_NAME] → V. Dual RAG Independence
  [SECTION_2_NAME]   → VI. Non-Invasive Auditing (promoted to principle)
  [SECTION_3_NAME]   → VII. Submission Always Accepted (promoted to principle)

Added sections:
  - Technology & Architecture Constraints
  - Development Milestones & Workflow
  - Governance

Removed sections: none (template had no populated content).

Templates requiring updates:
  ✅ .specify/templates/plan-template.md
     Constitution Check section uses [Gates determined based on constitution file]
     — flexible reference, no change required; principle names can be cited directly.
  ✅ .specify/templates/spec-template.md
     No principle-specific language; requirements section is generic.
  ✅ .specify/templates/tasks-template.md
     No constitution-specific references; task categories are generic.
  ⚠ .specify/templates/constitution-template.md
     Source template not modified; retains original placeholders for future projects.

Deferred TODOs: none — all placeholders resolved.
-->

# Sistema de Auditoría Académica con IA — Constitution

**Universidad Privada Franz Tamayo — Unifranz | Proyecto Integrador, Ingeniería de Sistemas**

## Core Principles

### I. Immutable Evidence

Trace events, analysis records, and decision logs MUST be append-only once created.
No `UPDATE` or `DELETE` operations are permitted on `trace_events`,
`analysis_results`, `decision_log`, or `appeals` rows after insertion.

Every submission timestamp MUST be recorded at the moment of receipt and MUST NOT
be altered retroactively. Hash verification of trace-event sequences MUST be
exposed via the audit endpoint (`GET /appeals/:id/evidence`) so that integrity
can be confirmed without trusting application-layer claims alone.

**Rationale:** The immutability of evidence is the legal and academic foundation
that makes the formal appeal process (`RF-20`) institutionally defensible. A single
mutable record would invalidate the entire chain of custody.

---

### II. Asynchronous Analysis Isolation

The student-facing resolution environment MUST never be degraded or blocked by
post-submission analysis work. The following operations MUST be executed exclusively
by the async worker via Redis Streams — never inline in an HTTP request handler:

- Heuristic pattern detection
- LLM analysis (via the AI router)
- Format validation of PDF and Word documents
- Git history analysis
- Cross-stage coherence comparison

The worker SLA is **≤ 3 minutes** from submission to stored result (`RNF-08`).
If the LLM call fails, the heuristic result MUST be persisted independently —
partial results are preferable to a missing record.

**Rationale:** Decoupling capture from analysis is what allows the system to scale
and remain responsive for students during peak exam periods without sacrificing the
depth of the analysis pipeline.

---

### III. Human Authority Over AI

The system MUST NOT take autonomous decisions on grades, sanctions, or appeal
outcomes. All AI-generated output — scores, justifications, suggested grades,
format reports — is advisory and clearly labelled as such.

The docente MUST assign the final grade. If the final grade differs from the
system-suggested grade, the reason MUST be recorded in the same transaction
(`RF-18`). No enforcement mechanism (score thresholds, automatic penalties)
may trigger without an explicit human action.

**Rationale:** The system is an evidence organiser, not an adjudicator. Keeping
authority in the docente preserves institutional accountability and prevents
the audit system from becoming a punitive mechanism (`integrador-elder_v4.md §1`).

---

### IV. Strict Role-Based Data Isolation

Every data access operation MUST be scoped to the authenticated user's role:

- **Administrador:** global scope; exclusively manages curriculum, format templates,
  AI config, docente accounts, and the appeals escalation endpoint.
- **Docente:** MUST see only courses, groups, activities, submissions, alerts, and
  analytics belonging to their own account.
- **Co-docente:** MUST see only the single activity for which they hold an explicit
  co-teaching grant — no access to the rest of the course or other activities.
- **Estudiante:** MUST see only their own submissions, session history, grades, and
  feedback; MUST NOT see analysis reports or alert status on their own sessions.

RAG docente retrieval MUST apply `docente_id` and `curso_id` filters **before**
any vector similarity search (`RF-14`, §9).

**Rationale:** Data isolation is both a privacy requirement (`RNF-06`) and an
academic-integrity requirement — a student who can read their own detection report
can craft submissions to evade it.

---

### V. Dual RAG Independence

The curricular RAG (`rag_curricular`) and the docente RAG (`rag_docente`) MUST
remain physically separated (distinct tables) and MUST NOT be merged in a single
retrieval query.

- **Curricular RAG** — filtered by `carrera + semestre + codigo_materia` before
  similarity search; used for detection context and interdisciplinary proposals.
- **Docente RAG** — filtered by `docente_id + curso_id (+ activity_id)` before
  similarity search; used for practice feedback and post-activity analysis.

Rubrics (`is_rubric = TRUE`) MUST always be injected in full into the prompt.
They MUST NOT be fragmented, retrieved by similarity, or truncated regardless of
token budget.

**Rationale:** Mixing the two RAGs in a single query would contaminate
course-specific feedback with institutional curriculum averages, producing
generic and unreliable analysis (`integrador-elder_v4.md §8`).

---

### VI. Non-Invasive Auditing

Auditing is bounded exclusively to events that occur within the browser during
an active exam session or oral defense. The system MUST NOT:

- Access the student's camera or microphone.
- Access the student's local filesystem.
- Require installation of additional software.
- Activate trace capture outside an explicitly opened exam session.

Students MUST be informed that an audit system exists before they begin any
evaluated activity. Trace activation is proportional to risk: it applies only to
`exam` activities and project oral defenses — never to `practice`, `tarea`, or
`informe` submissions (`RF-04`, `RNF-05`).

**Rationale:** Non-invasiveness is both an ethical commitment and a practical
one. Requiring browser plugins or camera access would create access inequality
and undermine student trust, making the system unsustainable institutionally.

---

### VII. Submission Always Accepted

Every student submission MUST be accepted regardless of:

- Format validation outcome (PDF/Word structure violations)
- Lateness relative to the configured deadline
- Analysis score or heuristic flags triggered

Late submissions are marked automatically with their timestamp; the docente
decides whether to accept or penalise them. Format reports and detection reports
are delivered to the docente — they are evidence, not gates.

Versioned submissions (second upload of the same stage) MUST persist alongside
the first — both versions remain in MinIO and in the database; neither is deleted
(`RF-07`, `integrador-elder_v4.md §11`).

**Rationale:** Blocking a submission on format grounds risks permanent loss of
academic work due to parser edge cases. The system must never be the reason a
student fails to submit.

---

## Technology & Architecture Constraints

The following stack decisions are locked for this project and MUST NOT be swapped
without a constitution amendment:

| Layer | Technology | Notes |
|---|---|---|
| API framework | Go + Fiber | WebSocket + SSE native support required |
| Database | PostgreSQL 16 + pgvector | Unified relational + vector store |
| Cache / queue | Redis 7 (Streams) | Async worker job queue |
| File storage | MinIO (S3-compatible) | Self-hosted on VPS 1 |
| AI — primary | Ollama (LLaMA 3 / Mistral) | VPS 2 GPU, OpenAI-compatible endpoint |
| AI — fallback | Vercel AI Gateway | Cloud overflow; no per-token fixed cost |
| Code sandbox | Judge0 | HTTP client; cloud or self-hosted |
| Frontend | React + Vite | Separate repository |
| Deployment | Docker Compose + GitHub Actions | Two-VPS topology |

**AI Router modes** (configurable at runtime without restart):
`local` (Ollama only) | `auto` (Ollama → Vercel fallback) | `cloud` (Vercel only)

Mode changes take effect within 5 seconds via the in-process config cache.
The router logs which backend processed each request for audit traceability.

External resources referenced by docentes (URLs, YouTube videos) MUST be
captured and stored in MinIO at upload time. The system MUST NOT depend on
external availability during an active evaluation (`RNF-12`).

---

## Development Milestones & Workflow

Development is organised into 13 sequential milestones. Each milestone has a
single deliverable that MUST be verified before the next begins:

| Milestone | Deliverable | Principles Exercised |
|---|---|---|
| M1 — Foundation | Docker Compose up; `GET /health` → 200 | — |
| M2 — Authentication | Login, JWT, role middleware | IV |
| M3 — Academic Management | CRUD courses, groups, activities | IV |
| M4 — File Storage | MinIO upload/download; late marking | VII |
| M5 — Resolution & Tracing | WebSocket trace events; Redis Streams worker base | I, II |
| M6 — AI Router | Three-mode router; hot swap | III |
| M7 — Dual RAG | Ingest + retrieval; filters applied | V |
| M8 — Detection Engine | Heuristics + LLM; SLA ≤ 3 min | II, III |
| M9 — Format Validation | PDF/Word parser; compliance report | VII |
| M10 — Project Management | Stages, Git analysis, oral defense | I, II |
| M11 — Alerts & Analytics | SSE to docente; analytics endpoints | III, IV |
| M12 — Admin Modules | Curriculum, templates, interdisciplinary | IV |
| M13 — Appeals & Final Audit | Immutable chain; end-to-end tests | I, III |

Each milestone deliverable MUST be demonstrated via its defined entregable before
merging to `main`. Integration tests covering the critical flows (exam session,
report submission, appeal) MUST be present by M13 (`integrador-backend-milestones.md §M13`).

---

## Governance

**Authority:** This constitution supersedes all other practice documents, README
conventions, and verbal agreements for the nextlab-api project.

**Amendment procedure:**
1. Open a PR against `main` that modifies `.specify/memory/constitution.md`.
2. State the version bump type (MAJOR / MINOR / PATCH) and rationale in the PR
   description.
3. Update the Sync Impact Report comment at the top of this file.
4. At least one reviewer with knowledge of the affected milestone MUST approve.
5. All dependent templates (`.specify/templates/`) MUST be reviewed for alignment
   before merge.

**Versioning policy:**
- **MAJOR** — backward-incompatible removal or redefinition of a principle.
- **MINOR** — new principle or section added; material expansion of guidance.
- **PATCH** — clarification, wording, typo fix; no semantic change.

**Compliance review:** Constitution gates MUST be checked at the "Constitution
Check" step of every implementation plan (`plan-template.md`). Any plan phase
that would violate a principle MUST be redesigned before proceeding.

**Version**: 1.0.0 | **Ratified**: 2026-05-17 | **Last Amended**: 2026-05-17
