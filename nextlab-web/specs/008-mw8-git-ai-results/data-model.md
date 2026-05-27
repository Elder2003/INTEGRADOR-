# Data Model: MW8 — Git & AI Results

## Extended Interfaces (academic.ts)

### StudentActividadEntry (extended)
| Field | Type | Notes |
|-------|------|-------|
| git_enabled | boolean | Whether this activity requires git submission |
| ai_enabled | boolean | Whether AI analysis is configured for this activity |
| detection_enabled | boolean | Whether detection/similarity analysis is configured |

### Actividad (extended)
| Field | Type | Notes |
|-------|------|-------|
| git_enabled | boolean | Same flags for docente-facing activity object |
| ai_enabled | boolean | |
| detection_enabled | boolean | |

### SubmissionResponse (extended)
| Field | Type | Notes |
|-------|------|-------|
| repo_url | string \| null | Git repo URL; null for file submissions |

### SubmissionListItem (extended)
| Field | Type | Notes |
|-------|------|-------|
| repo_url | string \| null | Git repo URL; null for file submissions |

---

## New Interfaces

### GitSubmissionRequest
```ts
{ repo_url: string }
```

### GitAnalysisReport
| Field | Type | Notes |
|-------|------|-------|
| status | 'pending' \| 'done' \| 'error' | Analysis pipeline state |
| error_message | string \| null \| undefined | Present when status === 'error' |
| commit_count | number \| undefined | Present when status === 'done' |
| authors | string[] \| undefined | Distinct commit authors |
| date_from | string \| undefined | ISO date of earliest commit |
| date_to | string \| undefined | ISO date of latest commit |
| files | GitFileEntry[] \| undefined | File change breakdown |

### GitFileEntry
| Field | Type | Notes |
|-------|------|-------|
| path | string | File path relative to repo root |
| additions | number | Lines added |
| deletions | number | Lines removed |

### AIReport
| Field | Type | Notes |
|-------|------|-------|
| status | 'pending' \| 'done' \| 'error' | |
| error_message | string \| null \| undefined | |
| sections | AIReportSection[] \| undefined | Present when status === 'done' |

### AIReportSection
| Field | Type | Notes |
|-------|------|-------|
| title | string | Section heading (e.g., "Resumen", "Hallazgos") |
| content | string | Full text; may be multi-paragraph |

### DetectionReport
| Field | Type | Notes |
|-------|------|-------|
| status | 'pending' \| 'done' \| 'error' | |
| error_message | string \| null \| undefined | |
| similarity_score | number \| undefined | 0–100 percentage; present when done |
| flagged_segments | FlaggedSegment[] \| undefined | |

### FlaggedSegment
| Field | Type | Notes |
|-------|------|-------|
| source | string | Name or URL of matched source |
| match_percentage | number | 0–100 |
| content | string \| undefined | Matched text excerpt (optional) |

---

## State Transitions

### GitSubmission (student perspective)
```
no submission + window open → [submit] → confirmed
confirmed + window open + not locked → [edit] → editing → [update] → confirmed (analyses reset to pending)
no submission + deadline past → closed
```

### AnalysisReport (docente perspective)
```
pending → done | error
(no client transitions — backend drives state)
```

---

## Validation Rules

- `repo_url` in submit/update: non-empty AND contains `://` (client-side, FR-002)
- All three report types are read-only on the client (Constitution Principle V)
- Analysis report links shown only when corresponding flag is true on the activity (FR-013)
