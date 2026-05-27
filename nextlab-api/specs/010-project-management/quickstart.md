# Quickstart: M10 — Project Management

Integration test scenarios for the three independent user stories. Each can be run against a live stack independently.

**Prerequisites**: Running stack via `docker compose up`; `BASE_URL=http://localhost:8080`; docente and student JWTs obtained via `/auth/login`.

---

## Story 1 — Investigacion Activity (multi-stage)

### Step 1: Create `investigacion` actividad

```bash
ACTIVIDAD=$(curl -sf -X POST "$BASE_URL/api/v1/cursos/$CURSO_ID/actividades" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Investigación Semestral",
    "tipo": "investigacion",
    "deadline": "2026-08-01T23:59:00Z",
    "instructions": "Investigar el tema asignado en etapas."
  }')
ACTIVIDAD_ID=$(echo "$ACTIVIDAD" | jq -r '.id')
```

### Step 2: Add three stages

```bash
S1=$(curl -sf -X POST "$BASE_URL/api/v1/actividades/$ACTIVIDAD_ID/stages" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
  -H "Content-Type: application/json" \
  -d '{"name":"Propuesta de tema","display_order":1,"deadline":"2026-06-15T23:59:00Z"}')
S1_ID=$(echo "$S1" | jq -r '.id')

S2=$(curl -sf -X POST "$BASE_URL/api/v1/actividades/$ACTIVIDAD_ID/stages" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
  -H "Content-Type: application/json" \
  -d '{"name":"Marco teórico","display_order":2,"deadline":"2026-07-01T23:59:00Z"}')
S2_ID=$(echo "$S2" | jq -r '.id')

S3=$(curl -sf -X POST "$BASE_URL/api/v1/actividades/$ACTIVIDAD_ID/stages" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
  -H "Content-Type: application/json" \
  -d '{"name":"Entrega final","display_order":3,"deadline":"2026-07-10T23:59:00Z"}')
S3_ID=$(echo "$S3" | jq -r '.id')
```

### Step 3: Student submits to stage 1 (on time)

```bash
curl -sf -X POST "$BASE_URL/api/v1/actividades/$ACTIVIDAD_ID/stages/$S1_ID/submit" \
  -H "Authorization: Bearer $STUDENT_JWT" \
  -F "file=@propuesta.pdf"
# Expect: is_late=false
```

### Step 4: Student submits to stage 3 (after deadline — still accepted)

```bash
curl -sf -X POST "$BASE_URL/api/v1/actividades/$ACTIVIDAD_ID/stages/$S3_ID/submit" \
  -H "Authorization: Bearer $STUDENT_JWT" \
  -F "file=@entrega_final.pdf"
# Expect: is_late=true (submission accepted, Principle VII)
```

### Step 5: Docente views stage-progress matrix

```bash
curl -sf "$BASE_URL/api/v1/actividades/$ACTIVIDAD_ID/stage-progress" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
| jq '.students[0].stages'
# Expect: stage1=submitted, stage2=pending, stage3=submitted_late
```

### Step 6: Docente grades stage 1

```bash
curl -sf -X PUT "$BASE_URL/api/v1/actividades/$ACTIVIDAD_ID/stages/$S1_ID/submissions/$STUDENT_ID/grade" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
  -H "Content-Type: application/json" \
  -d '{"grade":85,"grade_note":"Propuesta clara y bien delimitada."}'
# Expect 200 with grade=85
```

### Step 7: Verify delete blocked when submission exists

```bash
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
  "$BASE_URL/api/v1/actividades/$ACTIVIDAD_ID/stages/$S1_ID" \
  -H "Authorization: Bearer $DOCENTE_JWT")
# Expect: 409
```

---

## Story 2 — Git Analysis Activity

### Step 1: Create `git_analysis` actividad

```bash
GIT_ACT=$(curl -sf -X POST "$BASE_URL/api/v1/cursos/$CURSO_ID/actividades" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Proyecto Final Git",
    "tipo": "git_analysis",
    "deadline": "2026-06-01T23:59:00Z",
    "instructions": "Entregar URL del repositorio de su proyecto."
  }')
GIT_ACT_ID=$(echo "$GIT_ACT" | jq -r '.id')
```

### Step 2: Student submits a Git URL

```bash
curl -sf -X POST "$BASE_URL/api/v1/actividades/$GIT_ACT_ID/git-submit" \
  -H "Authorization: Bearer $STUDENT_JWT" \
  -H "Content-Type: application/json" \
  -d '{"repo_url":"https://github.com/octocat/Hello-World"}'
# Expect 202, analysis_status=pending
```

### Step 3: Wait for worker (up to 3 minutes), then check report

```bash
for i in $(seq 1 18); do
  STATUS=$(curl -sf "$BASE_URL/api/v1/actividades/$GIT_ACT_ID/git-report/$STUDENT_ID" \
    -H "Authorization: Bearer $DOCENTE_JWT" | jq -r '.analysis.status')
  [ "$STATUS" = "success" ] && break
  sleep 10
done
# Verify: commit_count > 0, authors array present, heuristic flags and llm_advisory present
```

### Step 4: Verify student cannot access report (FR-013)

```bash
HTTP=$(curl -s -o /dev/null -w "%{http_code}" \
  "$BASE_URL/api/v1/actividades/$GIT_ACT_ID/git-report/$STUDENT_ID" \
  -H "Authorization: Bearer $STUDENT_JWT")
# Expect: 403
```

### Step 5: Docente assigns grade

```bash
curl -sf -X POST "$BASE_URL/api/v1/actividades/$GIT_ACT_ID/git-grade/$STUDENT_ID" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
  -H "Content-Type: application/json" \
  -d '{"grade":72}'
# Expect 200 with grade=72
```

---

## Story 3 — Oral Defense Scheduling

### Step 1: Create `defensa_oral` actividad

```bash
DEF_ACT=$(curl -sf -X POST "$BASE_URL/api/v1/cursos/$CURSO_ID/actividades" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Defensa Oral Final",
    "tipo": "defensa_oral",
    "deadline": "2026-07-15T23:59:00Z"
  }')
DEF_ACT_ID=$(echo "$DEF_ACT" | jq -r '.id')
```

### Step 2: Docente schedules student's defense

```bash
curl -sf -X PUT "$BASE_URL/api/v1/actividades/$DEF_ACT_ID/defensa/$STUDENT_ID/schedule" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
  -H "Content-Type: application/json" \
  -d '{"scheduled_at":"2026-07-10T10:00:00Z"}'
# Expect 200 with scheduled_at set
```

### Step 3: Student views their appointment

```bash
curl -sf "$BASE_URL/api/v1/actividades/$DEF_ACT_ID/defensa/$STUDENT_ID" \
  -H "Authorization: Bearer $STUDENT_JWT" \
| jq '.scheduled_at'
# Expect: "2026-07-10T10:00:00Z"
```

### Step 4: Student without appointment sees pending

```bash
# Different student with no appointment
curl -sf "$BASE_URL/api/v1/actividades/$DEF_ACT_ID/defensa/$OTHER_STUDENT_ID" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
| jq '.scheduled_at'
# Expect: null
```

### Step 5: Docente records grade after defense

```bash
curl -sf -X PUT "$BASE_URL/api/v1/actividades/$DEF_ACT_ID/defensa/$STUDENT_ID/grade" \
  -H "Authorization: Bearer $DOCENTE_JWT" \
  -H "Content-Type: application/json" \
  -d '{"grade":88}'
# Expect 200 with grade=88 and graded_at set
```
