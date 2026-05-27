# Research: Admin Modules (M12)

## Decision 1: Docente Name Storage
- **Decision**: Add `full_name TEXT NOT NULL DEFAULT ''` to the existing `users` table via a migration.
- **Rationale**: The spec requires admin to create docente accounts with a full name. Adding a column to users is simpler than a separate profile table. An empty string default allows the migration to run on existing rows without breaking anything.
- **Alternatives considered**: Separate `docente_profiles` table — overly complex for this feature.

## Decision 2: Format Template Package Location
- **Decision**: New `internal/admin` package covering US1–US4 of M12.
- **Rationale**: All four user stories are admin-only and share no logic with existing packages. A dedicated `admin` package keeps admin logic isolated (Principle IV). The academic package already has admin routes for carreras/materias; M12 admin routes will live alongside those in the same `/admin` prefix but via the new package to avoid bloating `academic`.
- **Alternatives considered**: Extending `internal/academic` — would bloat that package further.

## Decision 3: Format Template Rules Schema
- **Decision**: Reuse the `format.FormatRulesConfig` Go type for the rules JSONB payload in `format_templates`. The docente "apply" endpoint copies the template's rules JSON into `actividades.format_rules`.
- **Rationale**: No new schema needed; the existing format validator already handles `FormatRulesConfig`.
- **Alternatives considered**: Separate schema — unnecessary duplication.

## Decision 4: Curriculum Document Deletion Strategy
- **Decision**: Hard delete from `rag_curricular` (ON DELETE CASCADE removes chunks automatically). Also delete the MinIO file.
- **Rationale**: `rag_curricular` rows are not referenced as immutable evidence (Principle I only protects trace events and analysis results). Stale curriculum documents actively degrade RAG quality.
- **Alternatives considered**: Soft delete — would require filtering in all retrieval queries; added complexity with no benefit.

## Decision 5: Interdisciplinary RAG Union Strategy
- **Decision**: In `rag/service_retrieve.go`, when `QueryCurricular` is called, first look up `actividad_interdisciplinary_scopes` for the actividad_id. If extra scopes exist, run one `QuerySimilar` per scope, concatenate results, deduplicate by `chunk_id`, and return the union.
- **Rationale**: Keeps Principle V intact (no merged query; still physically separate tables). Each per-scope query uses the same cosine similarity search. Deduplication prevents the same rubric text appearing twice.
- **Alternatives considered**: Single JOIN query across scopes — would need dynamic SQL and risks mixing scope filters.

## Decision 6: Temporary Password Generation
- **Decision**: Generate a 12-character random password (alphanumeric), hash via bcrypt, return the plain-text password only in the 201 response (not stored). Admin must relay it to the new docente out-of-band.
- **Rationale**: Simple and secure. The plain password appears only in the creation response; subsequent reads never return it.
- **Alternatives considered**: Email reset flow — requires email service not in scope for M12.
