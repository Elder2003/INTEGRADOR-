# nextlab-web — Agent Context

For additional context about the project plan, constitution, and implementation
details, read the current milestone plan at:

<!-- SPECKIT START -->
`specs/005-mw5-file-submissions/plan.md`
<!-- SPECKIT END -->

## Key files

- Constitution: `.specify/memory/constitution.md`
- Overall plan: `nextlab-web.md` (repo root)
- Active feature: `.specify/feature.json`

## Critical rules (from constitution)

- ALL HTTP calls go through `src/lib/api.ts` — no raw `fetch()`
- Access token in memory only — never in localStorage
- No business logic on the frontend — display what the API returns
- No external UI component library — Tailwind CSS v4 only
- Run React Doctor after every milestone
