#!/usr/bin/env bash
# Runs a disposable, end-to-end NextLab validation stack.
# It only deletes Docker resources belonging to the nextlab-system-test project.
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE=(docker compose -p nextlab-system-test -f "$ROOT_DIR/docker-compose.test.yml")
BASE_URL="${BASE_URL:-http://localhost:18080}"
KEEP_TEST_STACK="${KEEP_TEST_STACK:-0}"
PASS=0
FAIL=0

cleanup() {
  local code=$?
  if [[ "$KEEP_TEST_STACK" != "1" ]]; then
    "${COMPOSE[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
  fi
  exit "$code"
}
trap cleanup EXIT

for command in docker curl jq; do
  command -v "$command" >/dev/null || { echo "Falta el requisito: $command"; exit 1; }
done

pass() { printf '  ✓ %s\n' "$1"; PASS=$((PASS + 1)); }
fail() { printf '  ✗ %s\n' "$1" >&2; FAIL=$((FAIL + 1)); }

request() {
  local method=$1 path=$2 token=${3:-} data=${4:-}
  local args=(-sS -X "$method" -H 'Accept: application/json')
  [[ -n "$token" ]] && args+=(-H "Authorization: Bearer $token")
  [[ -n "$data" ]] && args+=(-H 'Content-Type: application/json' --data "$data")
  curl "${args[@]}" -w $'\n%{http_code}' "$BASE_URL$path"
}

status_of() { tail -n 1 <<<"$1"; }
body_of() { sed '$d' <<<"$1"; }
expect() {
  local label=$1 expected=$2 result=$3 actual
  actual=$(status_of "$result")
  if [[ "$actual" == "$expected" ]]; then pass "$label ($actual)"; else fail "$label: esperado $expected, recibido $actual: $(body_of "$result")"; fi
}
expect_2xx() {
  local label=$1 result=$2 actual
  actual=$(status_of "$result")
  if [[ "$actual" =~ ^2 ]]; then pass "$label ($actual)"; else fail "$label: esperado 2xx, recibido $actual: $(body_of "$result")"; fi
}
token_for() {
  local email=$1
  request POST /auth/login '' "{\"email\":\"$email\",\"password\":\"test1234\"}" | body_of | jq -r '.access_token'
}

echo '== NextLab: prueba integral aislada =='
"${COMPOSE[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
"${COMPOSE[@]}" up --build --detach

echo 'Esperando a que la API esté lista...'
for _ in $(seq 1 90); do
  if curl -fsS "$BASE_URL/health" >/dev/null 2>&1; then break; fi
  sleep 2
done
curl -fsS "$BASE_URL/health" >/dev/null || { echo 'La API no inició correctamente.'; "${COMPOSE[@]}" logs api; exit 1; }

"${COMPOSE[@]}" exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d nextlab_test < "$ROOT_DIR/scripts/seed-test-users.sql"

ADMIN_TOKEN=$(token_for admin@nextlab.dev)
DOCENTE_TOKEN=$(token_for docente@nextlab.dev)
STUDENT_TOKEN=$(token_for estudiante@nextlab.dev)
CODocente_TOKEN=$(token_for codocente@nextlab.dev)
[[ "$ADMIN_TOKEN" != null && "$DOCENTE_TOKEN" != null && "$STUDENT_TOKEN" != null && "$CODocente_TOKEN" != null ]] || { echo 'No fue posible iniciar sesión con las cuentas de prueba.'; exit 1; }

HEALTH=$(request GET /health)
expect_2xx 'Salud: base de datos, caché, almacenamiento e IA simulada' "$HEALTH"
expect 'Un estudiante no puede administrar carreras' 403 "$(request GET /admin/carreras "$STUDENT_TOKEN")"
expect_2xx 'Autenticación /auth/me (administrador)' "$(request GET /auth/me "$ADMIN_TOKEN")"
expect_2xx 'Autenticación /auth/me (docente)' "$(request GET /auth/me "$DOCENTE_TOKEN")"
expect_2xx 'Autenticación /auth/me (estudiante)' "$(request GET /auth/me "$STUDENT_TOKEN")"

expect_2xx 'Administración académica: carreras' "$(request GET /admin/carreras "$ADMIN_TOKEN")"
expect_2xx 'Administración académica: materias' "$(request GET /admin/materias "$ADMIN_TOKEN")"
expect_2xx 'Administración académica: perfiles de estudiante' "$(request GET /admin/student-profiles "$ADMIN_TOKEN")"
expect_2xx 'Panel del estudiante: cursos inscritos' "$(request GET /student/me/cursos "$STUDENT_TOKEN")"
expect_2xx 'Panel docente: cursos' "$(request GET /cursos "$DOCENTE_TOKEN")"

COURSE_ID='44440000-0000-0000-0000-000000000001'
NEW_ACTIVITY=$(request POST "/cursos/$COURSE_ID/actividades" "$DOCENTE_TOKEN" '{"title":"Prueba integral de examen","tipo":"examen","deadline":"2099-12-31T23:59:00Z","instructions":"Archivo PDF de prueba.","format_rules":{"min_pages":1,"required_types":["pdf"]}}')
expect 201 'Docente crea actividad con reglas de formato' "$NEW_ACTIVITY"
ACTIVITY_ID=$(body_of "$NEW_ACTIVITY" | jq -r '.id')

SESSION=$(request POST "/actividades/$ACTIVITY_ID/session" "$STUDENT_TOKEN")
expect 201 'Estudiante abre sesión trazable' "$SESSION"
SESSION_ID=$(body_of "$SESSION" | jq -r '.id')
NOW_MS=$(( $(date +%s) * 1000 ))
expect 202 'Trazabilidad: lote offline de eventos' "$(request POST "/student/sessions/$SESSION_ID/trace/batch" "$STUDENT_TOKEN" "{\"events\":[{\"type\":\"focus\",\"client_timestamp\":$NOW_MS,\"metadata\":{\"source\":\"system-test\"}}]}")"

UPLOAD=$(curl -sS -X POST "$BASE_URL/actividades/$ACTIVITY_ID/submission" -H "Authorization: Bearer $STUDENT_TOKEN" -F "file=@$ROOT_DIR/scripts/single-page.pdf" -w $'\n%{http_code}')
expect 201 'Entrega de archivo a MinIO y encolado para el worker' "$UPLOAD"
SUBMISSION_ID=$(body_of "$UPLOAD" | jq -r '.id')
expect_2xx 'Consulta de entrega del estudiante' "$(request GET "/actividades/$ACTIVITY_ID/submission" "$STUDENT_TOKEN")"

REPORT=''
for _ in $(seq 1 45); do
  REPORT=$(request GET "/sessions/$SESSION_ID/detection-report" "$DOCENTE_TOKEN")
  [[ "$(status_of "$REPORT")" == 200 ]] && break
  sleep 2
done
expect 200 'Worker: formato, heurística y detección asistida' "$REPORT"
expect_2xx 'Docente consulta eventos de la sesión' "$(request GET "/sessions/$SESSION_ID/events" "$DOCENTE_TOKEN")"
expect_2xx 'Docente consulta analítica de actividad' "$(request GET "/actividades/$ACTIVITY_ID/analytics" "$DOCENTE_TOKEN")"
expect_2xx 'Docente consulta analítica de curso' "$(request GET "/cursos/$COURSE_ID/analytics" "$DOCENTE_TOKEN")"

expect_2xx 'Router IA con proveedor determinista' "$(request POST /ai/complete "$STUDENT_TOKEN" '{"messages":[{"role":"user","content":"Prueba"}]}')"
expect_2xx 'Consulta de configuración IA' "$(request GET /ai/config "$DOCENTE_TOKEN")"
expect_2xx 'Alertas del docente' "$(request GET /alerts "$DOCENTE_TOKEN")"
expect_2xx 'Notificaciones del estudiante' "$(request GET /student/notifications "$STUDENT_TOKEN")"

# Gestión de proyectos: esta parte protege explícitamente las rutas que antes
# usaban el rol inexistente "student" en lugar de "estudiante".
STUDENT_ID='11110000-0000-0000-0000-000000000004'
PROJECT_ACTIVITY=$(request POST "/cursos/$COURSE_ID/actividades" "$DOCENTE_TOKEN" '{"title":"Proyecto de investigación de prueba","tipo":"investigacion","deadline":"2099-12-31T23:59:00Z","instructions":"Entregar avance."}')
expect 201 'Docente crea actividad de investigación' "$PROJECT_ACTIVITY"
PROJECT_ACTIVITY_ID=$(body_of "$PROJECT_ACTIVITY" | jq -r '.id')
STAGE=$(request POST "/actividades/$PROJECT_ACTIVITY_ID/stages" "$DOCENTE_TOKEN" '{"name":"Anteproyecto","display_order":1,"deadline":"2099-12-31T23:59:00Z","description":"Primera entrega"}')
expect 201 'Docente configura etapa de investigación' "$STAGE"
STAGE_ID=$(body_of "$STAGE" | jq -r '.id')
STAGE_UPLOAD=$(curl -sS -X POST "$BASE_URL/actividades/$PROJECT_ACTIVITY_ID/stages/$STAGE_ID/submit" -H "Authorization: Bearer $STUDENT_TOKEN" -F "file=@$ROOT_DIR/scripts/single-page.pdf" -w $'\n%{http_code}')
expect 201 'Estudiante entrega avance de investigación' "$STAGE_UPLOAD"
expect_2xx 'Docente consulta progreso por etapas' "$(request GET "/actividades/$PROJECT_ACTIVITY_ID/stage-progress" "$DOCENTE_TOKEN")"

DEFENSE_ACTIVITY=$(request POST "/cursos/$COURSE_ID/actividades" "$DOCENTE_TOKEN" '{"title":"Defensa oral de prueba","tipo":"defensa_oral","deadline":"2099-12-31T23:59:00Z","instructions":"Defensa programada."}')
expect 201 'Docente crea actividad de defensa oral' "$DEFENSE_ACTIVITY"
DEFENSE_ACTIVITY_ID=$(body_of "$DEFENSE_ACTIVITY" | jq -r '.id')
expect_2xx 'Docente agenda defensa' "$(request PUT "/actividades/$DEFENSE_ACTIVITY_ID/defensa/$STUDENT_ID/schedule" "$DOCENTE_TOKEN" '{"scheduled_at":"2099-12-30T10:00:00Z"}')"
expect_2xx 'Estudiante consulta su defensa' "$(request GET "/actividades/$DEFENSE_ACTIVITY_ID/defensa/$STUDENT_ID" "$STUDENT_TOKEN")"

TEMPLATE=$(request POST /admin/format-templates "$ADMIN_TOKEN" '{"name":"Plantilla prueba integral","description":"Creada por test-system","rules":{"min_pages":1,"required_types":["pdf"]}}')
expect 201 'Administrador crea plantilla de formato' "$TEMPLATE"
TEMPLATE_ID=$(body_of "$TEMPLATE" | jq -r '.id')
expect_2xx 'Docente aplica plantilla a la actividad' "$(request POST "/actividades/$ACTIVITY_ID/format-template" "$DOCENTE_TOKEN" "{\"template_id\":\"$TEMPLATE_ID\"}")"
expect_2xx 'Administrador consulta analítica global' "$(request GET /admin/analytics/overview "$ADMIN_TOKEN")"
expect 422 'ML: protege reentrenamiento cuando no hay datos etiquetados suficientes' "$(request POST /admin/detection/retrain "$ADMIN_TOKEN")"

APPEAL=$(request POST /appeals "$STUDENT_TOKEN" "{\"session_id\":\"$SESSION_ID\",\"type\":\"analysis_dispute\",\"reason\":\"Prueba integral de apelación\"}")
expect 201 'Estudiante registra apelación con cadena de evidencia' "$APPEAL"
APPEAL_ID=$(body_of "$APPEAL" | jq -r '.id')
expect_2xx 'Consulta de evidencia de apelación' "$(request GET "/appeals/$APPEAL_ID/evidence" "$STUDENT_TOKEN")"
expect_2xx 'Administrador lista apelaciones' "$(request GET /appeals "$ADMIN_TOKEN")"

printf '\nResultado: %d correctas, %d fallidas.\n' "$PASS" "$FAIL"
if (( FAIL > 0 )); then
  "${COMPOSE[@]}" logs --no-color api worker
  exit 1
fi
echo 'La pila de prueba se eliminará al salir. Use KEEP_TEST_STACK=1 para inspeccionarla.'
