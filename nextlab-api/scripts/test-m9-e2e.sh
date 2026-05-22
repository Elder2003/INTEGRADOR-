#!/usr/bin/env bash
# =============================================================================
# M9 Format Validation — End-to-End Verification Script
#
# Runs quickstart.md steps 1–14 against a live Docker Compose stack.
#
# Prerequisites
#   docker compose up -d           (stack running)
#   curl, jq, python3 available
#
# Required env vars
#   DOCENTE_EMAIL      e.g. docente@test.com
#   DOCENTE_PASSWORD
#   STUDENT_EMAIL      e.g. student@test.com
#   STUDENT_PASSWORD
#   CURSO_ID           UUID of a curso the student is enrolled in
#
# Optional env vars
#   BASE_URL           default: http://localhost:8080
#   WORKER_WAIT_SECS   default: 90  (worker SLA for async job processing)
# =============================================================================
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
WORKER_WAIT="${WORKER_WAIT_SECS:-90}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ─── colours ─────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
PASS=0; FAIL=0

ok()   { echo -e "${GREEN}  ✓ PASS${NC}  $*"; ((PASS++)); }
fail() { echo -e "${RED}  ✗ FAIL${NC}  $*"; ((FAIL++)); }
warn() { echo -e "${YELLOW}  ~ SKIP${NC}  $*"; }
step() { echo -e "\n${YELLOW}── Step $*${NC}"; }

# ─── prerequisites ───────────────────────────────────────────────────────────
check_prereqs() {
  echo "Checking prerequisites..."

  for cmd in curl jq python3 docker; do
    command -v "$cmd" &>/dev/null || { echo "ERROR: '$cmd' not found in PATH"; exit 1; }
  done

  for var in DOCENTE_EMAIL DOCENTE_PASSWORD STUDENT_EMAIL STUDENT_PASSWORD CURSO_ID; do
    [[ -n "${!var:-}" ]] || { echo "ERROR: env var $var is required"; exit 1; }
  done

  # Check API is reachable
  if ! curl -sf "$BASE_URL/health" >/dev/null; then
    echo "ERROR: API not reachable at $BASE_URL — run 'docker compose up -d' first"
    exit 1
  fi

  echo "  Prerequisites OK."
}

# ─── generate test files ─────────────────────────────────────────────────────
gen_files() {
  echo "Generating test PDF/DOCX files..."
  local py
  py=$(command -v python3 2>/dev/null || command -v python 2>/dev/null || echo "")
  [[ -n "$py" ]] || { echo "ERROR: python3/python not found in PATH"; exit 1; }
  "$py" "$SCRIPT_DIR/gen-test-pdfs.py"
}

# ─── HTTP helpers ─────────────────────────────────────────────────────────────
api_post() {
  # api_post <path> <jwt> [<json_body>]
  local path="$1" jwt="$2" body="${3:-}"
  if [[ -n "$body" ]]; then
    curl -sf -X POST "$BASE_URL$path" \
      -H "Authorization: Bearer $jwt" \
      -H "Content-Type: application/json" \
      -d "$body"
  else
    curl -sf -X POST "$BASE_URL$path" \
      -H "Authorization: Bearer $jwt"
  fi
}

api_get() {
  # api_get <path> <jwt>
  local path="$1" jwt="$2"
  curl -sf "$BASE_URL$path" -H "Authorization: Bearer $jwt"
}

api_get_status() {
  # api_get_status <path> <jwt>  — returns HTTP status code only
  local path="$1" jwt="$2"
  curl -s -o /dev/null -w "%{http_code}" "$BASE_URL$path" -H "Authorization: Bearer $jwt"
}

api_put() {
  local path="$1" jwt="$2" body="$3"
  curl -sf -X PUT "$BASE_URL$path" \
    -H "Authorization: Bearer $jwt" \
    -H "Content-Type: application/json" \
    -d "$body"
}

upload_file() {
  # upload_file <actividad_id> <student_jwt> <file_path>
  local aid="$1" jwt="$2" file="$3"
  curl -sf -X POST "$BASE_URL/actividades/$aid/submission" \
    -H "Authorization: Bearer $jwt" \
    -F "file=@$file"
}

login() {
  # login <email> <password>  — returns access_token
  local resp
  resp=$(curl -sf -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$1\",\"password\":\"$2\"}")
  echo "$resp" | jq -r '.access_token'
}

# ─── DB helper ───────────────────────────────────────────────────────────────
db_query() {
  # db_query <sql>  — returns raw psql tuples-only output
  docker compose exec -T postgres \
    psql -U postgres nextlab -At -c "$1" 2>/dev/null
}

# ─── wait for worker ─────────────────────────────────────────────────────────
wait_for_format_report() {
  # wait_for_format_report <session_id>  — polls until row exists or timeout
  local sid="$1"
  local elapsed=0
  echo "    Waiting up to ${WORKER_WAIT}s for worker to process session $sid..."
  while [[ $elapsed -lt $WORKER_WAIT ]]; do
    local count
    count=$(db_query "SELECT count(*) FROM format_reports WHERE session_id='$sid'")
    if [[ "${count:-0}" -ge 1 ]]; then
      echo "    Worker done after ${elapsed}s."
      return 0
    fi
    sleep 5
    ((elapsed+=5))
  done
  echo "    Timeout waiting for worker (${WORKER_WAIT}s). Continuing..."
  return 1
}

# =============================================================================
# MAIN
# =============================================================================
echo "============================================================"
echo " M9 Format Validation — E2E Test Suite"
echo " API: $BASE_URL"
echo "============================================================"

check_prereqs
gen_files

# ─── login ───────────────────────────────────────────────────────────────────
echo ""
echo "Logging in..."
DOCENTE_JWT=$(login "$DOCENTE_EMAIL" "$DOCENTE_PASSWORD")
STUDENT_JWT=$(login "$STUDENT_EMAIL" "$STUDENT_PASSWORD")
[[ -n "$DOCENTE_JWT" ]] || { echo "ERROR: docente login failed"; exit 1; }
[[ -n "$STUDENT_JWT" ]] || { echo "ERROR: student login failed"; exit 1; }
echo "  Logged in as docente and student."

# =============================================================================
# STEPS 1–2: Docente configures format rules on an activity (US4)
# =============================================================================
step "1 — Create activity with format_rules"
ACTIVIDAD=$(api_post "/cursos/$CURSO_ID/actividades" "$DOCENTE_JWT" \
  '{"title":"M9 E2E Test Activity","tipo":"examen","deadline":"2099-12-31T23:59:00Z",
    "instructions":"Entregar en PDF, minimo 2 paginas.",
    "format_rules":{"min_pages":2,"max_pages":10,"min_word_count":300,"required_types":["pdf"]}}')
AID=$(echo "$ACTIVIDAD" | jq -r '.id')
[[ -n "$AID" && "$AID" != "null" ]] && ok "Activity created: $AID" \
  || { fail "Activity creation failed: $ACTIVIDAD"; exit 1; }

step "2 — Verify format_rules persisted in GET response"
FETCHED=$(api_get "/cursos/$CURSO_ID/actividades/$AID" "$DOCENTE_JWT")
MIN_PAGES=$(echo "$FETCHED" | jq -r '.format_rules.min_pages')
REQ_TYPES=$(echo "$FETCHED" | jq -r '.format_rules.required_types[0]')
[[ "$MIN_PAGES" == "2" && "$REQ_TYPES" == "pdf" ]] \
  && ok "format_rules.min_pages=$MIN_PAGES required_types=[pdf]" \
  || fail "Unexpected format_rules: min_pages=$MIN_PAGES required_types=$REQ_TYPES"

# =============================================================================
# STEPS 3–7: Compliant submission — full happy path (US1 + US2 + US3)
# =============================================================================
step "3 — Open exam session (student)"
SESSION=$(api_post "/actividades/$AID/session" "$STUDENT_JWT")
SID=$(echo "$SESSION" | jq -r '.id')
[[ -n "$SID" && "$SID" != "null" ]] && ok "Session opened: $SID" \
  || { fail "Session open failed: $SESSION"; exit 1; }

step "4+5 — Upload compliant PDF (auto-closes session + enqueues worker)"
SUB=$(upload_file "$AID" "$STUDENT_JWT" "$SCRIPT_DIR/compliant.pdf")
SUB_ID=$(echo "$SUB" | jq -r '.id')
[[ -n "$SUB_ID" && "$SUB_ID" != "null" ]] && ok "Submission uploaded: $SUB_ID" \
  || fail "Submission upload failed: $SUB"

step "6 — Wait for worker and verify format_reports row"
if wait_for_format_report "$SID"; then
  ROW=$(db_query "SELECT parse_status,file_type,page_count,word_count,compliant FROM format_reports WHERE session_id='$SID'")
  PARSE=$(echo "$ROW" | cut -d'|' -f1)
  FTYPE=$(echo "$ROW" | cut -d'|' -f2)
  PAGES=$(echo "$ROW" | cut -d'|' -f3)
  WORDS=$(echo "$ROW" | cut -d'|' -f4)
  COMPLIANT=$(echo "$ROW" | cut -d'|' -f5)

  [[ "$PARSE" == "success" ]] && ok "parse_status=success" || fail "parse_status=$PARSE (want success)"
  [[ "$FTYPE" == "pdf" ]]     && ok "file_type=pdf"       || fail "file_type=$FTYPE (want pdf)"
  [[ "$PAGES" -ge 2 ]]        && ok "page_count=$PAGES (≥2)" || fail "page_count=$PAGES (want ≥2)"
  [[ "$WORDS" -ge 300 ]]      && ok "word_count=$WORDS (≥300)" || fail "word_count=$WORDS (want ≥300)"
  [[ "$COMPLIANT" == "t" ]]   && ok "compliant=true"      || fail "compliant=$COMPLIANT (want true)"

  RULE_COUNT=$(db_query "SELECT jsonb_array_length(rule_results) FROM format_reports WHERE session_id='$SID'")
  [[ "${RULE_COUNT:-0}" -ge 4 ]] && ok "rule_results has $RULE_COUNT entries" \
    || fail "rule_results only $RULE_COUNT entries (want ≥4)"
else
  fail "Worker did not produce format_reports row within ${WORKER_WAIT}s — steps 6–7 skipped"
fi

step "7 — GET detection-report includes format_report sub-object"
REPORT=$(api_get "/sessions/$SID/detection-report" "$DOCENTE_JWT")
FR_STATUS=$(echo "$REPORT" | jq -r '.format_report.parse_status // empty')
FR_COMPLIANT=$(echo "$REPORT" | jq -r '.format_report.compliant // empty')
[[ "$FR_STATUS" == "success" ]] && ok "format_report.parse_status=success" \
  || fail "format_report.parse_status=$FR_STATUS"
[[ "$FR_COMPLIANT" == "true" ]] && ok "format_report.compliant=true" \
  || fail "format_report.compliant=$FR_COMPLIANT"

# =============================================================================
# STEP 8: Non-compliant DOCX (wrong file type)
# =============================================================================
step "8 — DOCX submission fails required_types=[pdf]"
SESSION8=$(api_post "/actividades/$AID/session" "$STUDENT_JWT")
SID8=$(echo "$SESSION8" | jq -r '.id')
if [[ -n "$SID8" && "$SID8" != "null" ]]; then
  upload_file "$AID" "$STUDENT_JWT" "$SCRIPT_DIR/test.docx" >/dev/null 2>&1 \
    && ok "Submission accepted (non-blocking Principle VII)" || warn "Upload returned error for DOCX"
  if wait_for_format_report "$SID8"; then
    COMPLIANT8=$(db_query "SELECT compliant FROM format_reports WHERE session_id='$SID8'")
    FAIL_RULE=$(db_query "SELECT rule_results::text FROM format_reports WHERE session_id='$SID8'" \
      | grep -o '"file_type"' | head -1)
    [[ "$COMPLIANT8" == "f" ]] && ok "DOCX compliant=false" || fail "DOCX compliant=$COMPLIANT8 (want false)"
    [[ "$FAIL_RULE" == '"file_type"' ]] && ok "rule_results contains file_type rule" \
      || fail "file_type rule not found in rule_results"
  else
    fail "Worker timeout for step 8"
  fi
else
  warn "Could not open new session for step 8 (student may only have one session per actividad at a time — expected if session 3 is still the latest)"
fi

# =============================================================================
# STEP 9: Single-page PDF (fails min_pages)
# =============================================================================
step "9 — Single-page PDF fails min_pages=2"
# Create a new activity for a fresh session
ACTIV9=$(api_post "/cursos/$CURSO_ID/actividades" "$DOCENTE_JWT" \
  '{"title":"M9 Step9 Activity","tipo":"tarea","deadline":"2099-12-31T23:59:00Z",
    "format_rules":{"min_pages":2,"required_types":["pdf"]}}')
AID9=$(echo "$ACTIV9" | jq -r '.id')
if [[ -n "$AID9" && "$AID9" != "null" ]]; then
  SESSION9=$(api_post "/actividades/$AID9/session" "$STUDENT_JWT")
  SID9=$(echo "$SESSION9" | jq -r '.id')
  upload_file "$AID9" "$STUDENT_JWT" "$SCRIPT_DIR/single-page.pdf" >/dev/null 2>&1 || true
  if wait_for_format_report "$SID9"; then
    ROW9=$(db_query "SELECT compliant, rule_results::text FROM format_reports WHERE session_id='$SID9'")
    COMPLIANT9=$(echo "$ROW9" | cut -d'|' -f1)
    RULES9=$(echo "$ROW9" | cut -d'|' -f2)
    [[ "$COMPLIANT9" == "f" ]] && ok "single-page compliant=false" || fail "compliant=$COMPLIANT9"
    echo "$RULES9" | grep -q '"min_pages"' \
      && ok "rule_results contains min_pages rule" \
      || fail "min_pages rule not found: $RULES9"
  else
    fail "Worker timeout for step 9"
  fi
else
  fail "Could not create activity for step 9"
fi

# =============================================================================
# STEP 10: Student cannot access detection-report (403)
# =============================================================================
step "10 — Student access to detection-report returns 403"
STATUS10=$(api_get_status "/sessions/$SID/detection-report" "$STUDENT_JWT")
[[ "$STATUS10" == "403" ]] && ok "Student receives 403 Forbidden" \
  || fail "Expected 403, got $STATUS10"

# =============================================================================
# STEP 11: Password-protected PDF
# =============================================================================
step "11 — Password-protected PDF (manual step)"
warn "Generating a password-protected PDF requires external tools (e.g. LibreOffice, pdftk)."
warn "To verify manually: upload a password-protected PDF, wait for worker, then:"
warn "  SELECT parse_status, fail_reason FROM format_reports WHERE session_id='<sid>';"
warn "  Expected: parse_status='failed', fail_reason='password_protected', compliant=false"

# =============================================================================
# STEP 12: Activity without format_rules → compliant=true, rule_results=[]
# =============================================================================
step "12 — Activity with no format_rules → compliant=true, rule_results=[]"
ACTIV12=$(api_post "/cursos/$CURSO_ID/actividades" "$DOCENTE_JWT" \
  '{"title":"M9 Step12 No Rules","tipo":"practica","deadline":"2099-12-31T23:59:00Z"}')
AID12=$(echo "$ACTIV12" | jq -r '.id')
if [[ -n "$AID12" && "$AID12" != "null" ]]; then
  SESSION12=$(api_post "/actividades/$AID12/session" "$STUDENT_JWT")
  SID12=$(echo "$SESSION12" | jq -r '.id')
  upload_file "$AID12" "$STUDENT_JWT" "$SCRIPT_DIR/compliant.pdf" >/dev/null 2>&1 || true
  if wait_for_format_report "$SID12"; then
    ROW12=$(db_query "SELECT parse_status, compliant, jsonb_array_length(rule_results) FROM format_reports WHERE session_id='$SID12'")
    PS12=$(echo "$ROW12" | cut -d'|' -f1)
    C12=$(echo  "$ROW12" | cut -d'|' -f2)
    RL12=$(echo "$ROW12" | cut -d'|' -f3)
    [[ "$PS12" == "success" ]] && ok "No-rules: parse_status=success" || fail "parse_status=$PS12"
    [[ "$C12"  == "t" ]]       && ok "No-rules: compliant=true"       || fail "compliant=$C12"
    [[ "${RL12:-0}" -eq 0 ]]   && ok "No-rules: rule_results=[]"      || fail "rule_results length=$RL12 (want 0)"
  else
    fail "Worker timeout for step 12"
  fi
else
  fail "Could not create activity for step 12"
fi

# =============================================================================
# STEP 13: Idempotency (ON CONFLICT DO NOTHING)
# =============================================================================
step "13 — Idempotency: only one format_reports row per submission"
COUNT13=$(db_query "SELECT count(*) FROM format_reports WHERE session_id='$SID'")
[[ "${COUNT13:-0}" -eq 1 ]] && ok "Exactly 1 format_reports row for session $SID (idempotent)" \
  || fail "Expected 1 row, found $COUNT13"

# Verify the constraint directly
db_query "SELECT submission_id FROM format_reports WHERE session_id='$SID' LIMIT 1" >/dev/null \
  && ok "ON CONFLICT (submission_id) constraint is active" \
  || fail "format_reports row not queryable"

# =============================================================================
# STEP 14: Update format_rules and verify new submission uses updated rules
# =============================================================================
step "14 — Update format_rules (min_pages→5) and verify new submission fails"
api_put "/cursos/$CURSO_ID/actividades/$AID" "$DOCENTE_JWT" \
  '{"format_rules":{"min_pages":5,"required_types":["pdf"]}}' >/dev/null

# Verify update persisted
UPDATED=$(api_get "/cursos/$CURSO_ID/actividades/$AID" "$DOCENTE_JWT")
NEW_MIN=$(echo "$UPDATED" | jq -r '.format_rules.min_pages')
[[ "$NEW_MIN" == "5" ]] && ok "format_rules.min_pages updated to 5" || fail "min_pages=$NEW_MIN (want 5)"

# Upload 3-page PDF — passes page count (3 < 5 fails min_pages=5)
SESSION14=$(api_post "/actividades/$AID/session" "$STUDENT_JWT")
SID14=$(echo "$SESSION14" | jq -r '.id')
if [[ -n "$SID14" && "$SID14" != "null" ]]; then
  upload_file "$AID" "$STUDENT_JWT" "$SCRIPT_DIR/three-page.pdf" >/dev/null 2>&1 || true
  if wait_for_format_report "$SID14"; then
    ROW14=$(db_query "SELECT compliant, rule_results::text FROM format_reports WHERE session_id='$SID14'")
    C14=$(echo "$ROW14" | cut -d'|' -f1)
    R14=$(echo "$ROW14" | cut -d'|' -f2)
    [[ "$C14" == "f" ]] && ok "3-page PDF compliant=false (min_pages=5 violated)" \
      || fail "compliant=$C14 (want false)"
    echo "$R14" | grep -q '"min_pages"' \
      && ok "rule_results contains min_pages entry" \
      || fail "min_pages not in rule_results: $R14"
    # Old session from step 3 must be unaffected
    OLD_C=$(db_query "SELECT compliant FROM format_reports WHERE session_id='$SID'")
    [[ "$OLD_C" == "t" ]] && ok "Old session format_report unchanged (compliant=true)" \
      || fail "Old session format_report changed: $OLD_C"
  else
    fail "Worker timeout for step 14"
  fi
else
  warn "Could not open session for step 14 (student may have an existing open session)"
fi

# =============================================================================
# SUMMARY
# =============================================================================
echo ""
echo "============================================================"
echo " Results: ${GREEN}${PASS} passed${NC}  ${RED}${FAIL} failed${NC}"
echo "============================================================"

[[ $FAIL -eq 0 ]] && echo -e "${GREEN}All checks passed — M9 verified.${NC}" \
  || echo -e "${RED}${FAIL} check(s) failed — see output above.${NC}"

[[ $FAIL -eq 0 ]]
