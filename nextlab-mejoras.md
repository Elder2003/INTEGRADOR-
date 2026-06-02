# NextLab — Mejoras Propuestas

**Proyecto:** NextLab (nextlab-api + nextlab-web)  
**Autor del análisis:** Consultor Técnico  
**Fecha:** 2026-05-30  
**Basado en:** Documentación Final v1.0.0 (Elder Hamer)

---

## Tabla de Contenidos

1. [Problemas Críticos de Seguridad](#1-problemas-críticos-de-seguridad)
2. [Problemas de Diseño Moderados](#2-problemas-de-diseño-moderados)
3. [Deuda Técnica](#3-deuda-técnica)
4. [Mejoras al Motor de Detección de Integridad](#4-mejoras-al-motor-de-detección-de-integridad)
5. [Mejoras al Pipeline RAG](#5-mejoras-al-pipeline-rag)
6. [Mejoras de Infraestructura y Observabilidad](#6-mejoras-de-infraestructura-y-observabilidad)
7. [Nuevas Funcionalidades Propuestas](#7-nuevas-funcionalidades-propuestas)
8. [Resumen de Prioridades](#8-resumen-de-prioridades)

---

## 1. Problemas Críticos de Seguridad

### 1.1. Tokens JWT en memoria — falsa protección contra XSS

**Problema:** El access token JWT se guarda en una variable JavaScript en memoria en lugar de `localStorage`. Esto no elimina el riesgo XSS: cualquier script inyectado en la misma pestaña puede leer la variable durante la ventana de vida del token (15 minutos).

**Impacto:** Alto. Un ataque XSS exitoso puede exfiltrar el access token activo.

**Solución propuesta:**
- Mover ambos tokens (access + refresh) a cookies `HttpOnly; Secure; SameSite=Strict`.
- El access token en cookie HttpOnly es inaccesible desde JavaScript, eliminando el vector XSS.
- Ajustar el middleware Go para leer el token desde la cookie además del header `Authorization`.

```go
// Middleware Go: leer desde cookie si no hay header
token := c.Cookies("access_token")
if token == "" {
    token = strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
}
```

---

### 1.2. Sandbox insuficiente para análisis Git

**Problema:** El módulo de análisis Git clona repositorios externos con `go-git` sin validación de URL, límite de tamaño, timeout de proceso, ni sandbox de ejecución.

**Impacto:** Crítico. Vectores de ataque incluyen:
- URLs con SSRF: `git://127.0.0.1/`, `git://169.254.169.254/` (metadata AWS)
- Repos de 10GB que agotan el disco del worker
- Repos con symlinks maliciosos que traversan el filesystem

**Solución propuesta:**

```go
// Validación de URL antes de clonar
var allowedHosts = []string{"github.com", "gitlab.com", "bitbucket.org"}

func validateGitURL(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil {
        return err
    }
    for _, h := range allowedHosts {
        if u.Hostname() == h {
            return nil
        }
    }
    return fmt.Errorf("host no permitido: %s", u.Hostname())
}
```

- Agregar variable de entorno `GIT_MAX_REPO_SIZE_MB` (default: 100).
- Implementar timeout duro en el worker: `context.WithTimeout(ctx, 5*time.Minute)`.
- Clonar en directorio temporal con `--depth=1` para limitar la historia.

---

### 1.3. Archivos de entrega sin cifrado en reposo

**Problema:** Los archivos subidos por estudiantes (exámenes, proyectos, investigaciones) se almacenan en MinIO sin cifrado SSE (Server-Side Encryption).

**Impacto:** Medio-Alto. En caso de acceso no autorizado al bucket, todos los archivos académicos quedan expuestos en texto claro.

**Solución propuesta:**
- Habilitar SSE-S3 en MinIO a nivel de bucket:

```bash
mc encrypt set sse-s3 myminio/nextlab
```

- O bien, habilitar SSE-C pasando claves por request desde la API:

```go
opts := minio.PutObjectOptions{
    ServerSideEncryption: encrypt.NewSSE(),
}
```

- Documentar la política de rotación de claves en el README operativo.

---

### 1.4. Autorización granular en WebSocket no verificada

**Problema:** El endpoint `WS /ws/sessions/:sid/live` valida el JWT pero no verifica que el `docente_id` del token tenga acceso al `curso_id` asociado a esa `session_id` específica.

**Impacto:** Medio. Un docente podría conectarse al feed en vivo de la sesión de otro docente si conoce el `session_id`.

**Solución propuesta:**

```go
// Al upgrade del WebSocket, verificar pertenencia
func LiveSessionHandler(c *websocket.Conn) {
    sessionID := c.Params("sid")
    docenteID := getDocenteFromToken(c)

    ok, err := db.VerifyDocenteOwnsSession(ctx, docenteID, sessionID)
    if !ok || err != nil {
        c.WriteMessage(websocket.CloseMessage, []byte("unauthorized"))
        return
    }
    // ... resto del handler
}
```

---

## 2. Problemas de Diseño Moderados

### 2.1. Race condition en auto-inscripción al hacer login

**Problema:** Las inscripciones de estudiantes se crean automáticamente al hacer login. Si ocurren dos requests de login simultáneos (reconexión rápida, doble click), pueden generarse inscripciones duplicadas.

**Impacto:** Medio. Puede causar duplicados en la tabla `inscripciones` y comportamiento inconsistente en el dashboard del estudiante.

**Solución propuesta:**

Verificar que la migración 000003 incluya el constraint único y usar `INSERT ... ON CONFLICT DO NOTHING`:

```sql
-- Migración: constraint único
ALTER TABLE inscripciones ADD CONSTRAINT uq_inscripcion 
UNIQUE (student_id, curso_id);
```

```go
// Go: upsert seguro
_, err = db.Exec(ctx, `
    INSERT INTO inscripciones (student_id, curso_id, enrolled_at)
    VALUES ($1, $2, NOW())
    ON CONFLICT (student_id, curso_id) DO NOTHING
`, studentID, cursoID)
```

---

### 2.2. Sistema de apelaciones — una sola por sesión

**Problema:** El schema permite una única apelación por `exam_session`. Si el docente rechaza una `analysis_dispute` y el estudiante necesita apelar la calificación final, no puede crear una segunda apelación.

**Impacto:** Medio. Limita el flujo de apelaciones al caso más simple y no cubre el proceso universitario real.

**Solución propuesta:**

Modificar el constraint de `appeals` para permitir múltiples apelaciones por sesión, diferenciadas por tipo y estado:

```sql
-- Reemplazar constraint único simple por uno compuesto
ALTER TABLE appeals DROP CONSTRAINT uq_appeal_session;
ALTER TABLE appeals ADD CONSTRAINT uq_appeal_session_type 
UNIQUE (session_id, type) WHERE status NOT IN ('resolved', 'admin_resolved');
```

Esto permite que exista una `analysis_dispute` activa y luego una `grade_dispute` activa de forma secuencial.

---

### 2.3. `actividad_config_changes` — log sin read path

**Problema:** La tabla `actividad_config_changes` registra cambios de configuración en actividades (append-only, bien diseñado) pero no existe ningún endpoint que lo exponga. Es un asset forense write-only.

**Impacto:** Bajo-Medio. Se pierde valor de auditoría: el docente no puede ver qué configuraciones tuvo una actividad en el pasado.

**Solución propuesta:**

Agregar endpoint de historial:

```
GET /docente/cursos/:cid/actividades/:aid/config-history
```

Devuelve el listado cronológico de cambios con `changed_by`, `changed_at`, y `diff` de los campos modificados.

---

### 2.4. Dimensión de embeddings inconsistente

**Problema:** La variable de entorno `AI_EMBED_DIMENSION` está documentada como `1536`, pero `nomic-embed-text` (el modelo local de Ollama configurado) genera vectores de **768 dimensiones**. El tipo `VECTOR(1536)` en pgvector rechazará silenciosamente los inserts o fallará con un error de dimensión.

**Impacto:** Crítico para el modo local. El pipeline RAG local está roto si la dimensión no coincide.

**Solución propuesta:**

Opción A — Usar `nomic-embed-text` con la dimensión correcta:
```env
AI_EMBED_MODEL=nomic-embed-text
AI_EMBED_DIMENSION=768
```

Y actualizar la migración 000007:
```sql
ALTER TABLE rag_curricular_chunks ALTER COLUMN embedding TYPE VECTOR(768);
ALTER TABLE rag_docente_chunks ALTER COLUMN embedding TYPE VECTOR(768);
```

Opción B — Si se prefiere 1536 para compatibilidad con OpenAI `text-embedding-ada-002` en cloud:
```env
AI_EMBED_MODEL=mxbai-embed-large  # Ollama, 1024 dims
# o usar cloud directamente
```

**Recomendación:** Agregar validación al inicio de la API que verifique la dimensión real del modelo antes de arrancar:

```go
func validateEmbedDimension(cfg *Config) error {
    testVec, err := embedText(ctx, "test")
    if len(testVec) != cfg.EmbedDimension {
        return fmt.Errorf("dimensión esperada %d, modelo devuelve %d", 
            cfg.EmbedDimension, len(testVec))
    }
    return nil
}
```

---

## 3. Deuda Técnica

### 3.1. Sin paginación en endpoints de listas críticas

**Problema:** Los endpoints `GET` de listas (`/submissions`, `/sessions`, `/trace`, `/appeals`) no implementan paginación. Una sesión de examen de 2 horas con heartbeats cada 30s y eventos de comportamiento puede generar fácilmente 300-1000 registros de `trace_events`.

**Impacto:** Alto en producción. Sin paginación, el endpoint `/trace` carga todos los eventos en memoria y los serializa en una sola respuesta, causando OOM en el API pod bajo carga.

**Solución propuesta — paginación cursor-based:**

```
GET /docente/cursos/:cid/actividades/:aid/sessions/:sid/trace
    ?cursor=<uuid>&limit=100
```

```go
type PaginatedResponse[T any] struct {
    Data       []T     `json:"data"`
    NextCursor *string `json:"next_cursor,omitempty"`
    HasMore    bool    `json:"has_more"`
}
```

Query SQL con cursor:
```sql
SELECT * FROM trace_events
WHERE session_id = $1 AND id > $2  -- cursor por UUID ordenado
ORDER BY sequence_num ASC
LIMIT $3;
```

---

### 3.2. Sin rate limiting en endpoints de IA

**Problema:** Los endpoints de RAG (`/rag/curricular/solve`, `/rag/docente/ingest`) y detección (`/analyze`) llaman directamente al LLM sin rate limiting por usuario o por curso. Un único usuario puede saturar Ollama con requests concurrentes.

**Impacto:** Medio. En modo local con Ollama, el LLM es single-threaded; múltiples requests concurrentes se encolan o fallan con timeout.

**Solución propuesta:**

```go
// Rate limiter por usuario con go-redis
func AIRateLimitMiddleware(rdb *redis.Client, limit int, window time.Duration) fiber.Handler {
    return func(c *fiber.Ctx) error {
        key := fmt.Sprintf("rl:ai:%s", getUserID(c))
        count, _ := rdb.Incr(ctx, key).Result()
        if count == 1 {
            rdb.Expire(ctx, key, window)
        }
        if int(count) > limit {
            return c.Status(429).JSON(fiber.Map{"error": "rate limit exceeded"})
        }
        return c.Next()
    }
}
```

Variables de entorno sugeridas:
```env
AI_RATE_LIMIT_PER_USER=10   # requests
AI_RATE_LIMIT_WINDOW=60s    # por ventana
```

---

### 3.3. Worker sin dead letter queue

**Problema:** Las tablas `worker_jobs` y `git_analysis_jobs` implementan una cola de trabajos asíncronos, pero no hay mención de manejo de trabajos fallidos reiterados (dead letter queue). Un trabajo que falla repetidamente por un bug o un repo corrupto puede bloquear slots del worker indefinidamente.

**Impacto:** Medio. En producción, jobs zombies pueden acumular y degradar el throughput del worker.

**Solución propuesta:**

```sql
-- Agregar columnas a worker_jobs
ALTER TABLE worker_jobs ADD COLUMN retry_count INT DEFAULT 0;
ALTER TABLE worker_jobs ADD COLUMN max_retries INT DEFAULT 3;
ALTER TABLE worker_jobs ADD COLUMN last_error TEXT;
ALTER TABLE worker_jobs ADD COLUMN dead_at TIMESTAMPTZ;
```

```go
// Lógica de retry en el worker
if job.RetryCount >= job.MaxRetries {
    db.MarkJobDead(ctx, job.ID, err.Error())
    alertDocente(ctx, job) // notificar que el análisis falló
    return
}
db.IncrementRetry(ctx, job.ID, err.Error())
```

---

## 4. Mejoras al Motor de Detección de Integridad

### 4.1. Análisis temporal de eventos — más allá del conteo

**Problema actual:** El heurístico suma pesos ponderados de eventos sin considerar su distribución temporal. Un score de 55 puntos (11 tab_blurs) en 10 minutos es muy diferente al mismo score en 3 horas.

**Mejora propuesta:**

Introducir métricas temporales en el análisis heurístico:

```go
type TemporalMetrics struct {
    EventsPerMinute    float64
    BurstWindows       int     // ventanas de 60s con >N eventos
    MaxEventsInWindow  int
    SuspiciousSequences int    // tab_blur inmediatamente seguido de clipboard_paste
}
```

Incorporar la duración del examen como denominador:

```go
// Score normalizado por duración
normalizedScore := rawScore / (sessionDurationMinutes / 60.0)
```

Detectar secuencias sospechosas (no solo eventos aislados):

```
tab_blur → (< 5s) → clipboard_paste  →  peso adicional x2
session_open → (< 30s) → session_close → flag de abandono temprano
```

---

### 4.2. Baseline de comportamiento por estudiante

**Problema actual:** El sistema evalúa cada sesión en aislamiento. No hay contexto histórico del estudiante: un estudiante que siempre cambia de pestaña (tic nervioso, lector compulsivo de enunciados) recibirá flags continuos sin que sea indicativo de deshonestidad.

**Mejora propuesta:**

Nueva tabla para acumular métricas históricas por estudiante:

```sql
CREATE TABLE student_behavior_baseline (
    student_id UUID REFERENCES users(id),
    avg_tab_blurs_per_hour NUMERIC(8,2),
    avg_clipboard_events_per_hour NUMERIC(8,2),
    total_sessions_analyzed INT DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (student_id)
);
```

El análisis heurístico incluye el z-score respecto al baseline:

```
z_score = (current_rate - baseline_mean) / baseline_stddev
flag_alto = z_score > 2.0  (2 desviaciones estándar sobre el promedio propio)
```

---

### 4.3. Modelo ML supervisado para detección

**Problema actual:** El sistema LLM genera un advisory en lenguaje natural, pero no hay un modelo entrenado específicamente en los patrones de comportamiento de la plataforma.

**Mejora propuesta a mediano plazo:**

Una vez acumuladas suficientes sesiones etiquetadas por docentes (a través de `grade_decisions`), entrenar un clasificador binario:

- **Features:** `tab_blurs_rate`, `clipboard_rate`, `heartbeat_gaps`, `session_duration`, `burst_windows`, `z_score_vs_baseline`, `time_of_day`, `activity_type`
- **Label:** `was_academic_fraud` (derivado de `grade_decisions.override_reason`)
- **Modelo:** Gradient Boosting (sklearn/XGBoost) exportable a ONNX para inferencia en Go

Endpoint de reentrenamiento:
```
POST /admin/detection/retrain
```

---

## 5. Mejoras al Pipeline RAG

### 5.1. Búsqueda híbrida BM25 + embeddings

**Problema actual:** La búsqueda RAG usa únicamente similitud coseno sobre embeddings. Para queries cortos o con términos técnicos específicos (códigos de materia, nombres propios), la búsqueda léxica BM25 supera a la semántica.

**Mejora propuesta:**

Habilitar `pg_search` o `tsvector` en PostgreSQL para búsqueda full-text:

```sql
-- Agregar columna tsvector a chunks
ALTER TABLE rag_curricular_chunks ADD COLUMN search_vector TSVECTOR;

-- Trigger para mantenerla actualizada
CREATE TRIGGER update_search_vector
BEFORE INSERT OR UPDATE ON rag_curricular_chunks
FOR EACH ROW EXECUTE FUNCTION
tsvector_update_trigger(search_vector, 'pg_catalog.spanish', content);
```

Fusión de resultados con Reciprocal Rank Fusion (RRF):

```go
// RRF score = Σ 1/(k + rank_i) para cada resultado en cada ranker
func reciprocalRankFusion(semanticRanks, bm25Ranks []string, k int) []string {
    scores := map[string]float64{}
    for i, id := range semanticRanks {
        scores[id] += 1.0 / float64(k+i+1)
    }
    for i, id := range bm25Ranks {
        scores[id] += 1.0 / float64(k+i+1)
    }
    // ordenar por score descendente
    return sortByScore(scores)
}
```

---

### 5.2. Re-ranking con cross-encoder

**Problema actual:** El top-5 de chunks se selecciona únicamente por similitud coseno del embedding, sin validar relevancia semántica profunda.

**Mejora propuesta:**

Agregar un paso de re-ranking con un modelo cross-encoder liviano (por ejemplo, `ms-marco-MiniLM-L-6-v2` via Ollama o un endpoint dedicado):

```go
type RerankRequest struct {
    Query     string   `json:"query"`
    Documents []string `json:"documents"`
}

// Llamar al cross-encoder después del retrieval inicial
reranked, err := aiClient.Rerank(ctx, RerankRequest{
    Query:     userQuery,
    Documents: initialTop20,
})
// Usar top-5 del reranked para el prompt final
```

---

### 5.3. Versionado de documentos curriculares

**Problema actual:** El re-ingest de un documento curricular hace swap atómico de chunks, pero no conserva versiones anteriores accesibles. Un admin que sobreescribe el sílabo de una materia no puede volver a la versión anterior.

**Mejora propuesta:**

```sql
-- Agregar versión a rag_curricular
ALTER TABLE rag_curricular ADD COLUMN version INT DEFAULT 1;
ALTER TABLE rag_curricular ADD COLUMN superseded_by INT REFERENCES rag_curricular(id);
ALTER TABLE rag_curricular ADD COLUMN is_current BOOLEAN DEFAULT TRUE;
```

Endpoint de historial:
```
GET /admin/curriculum/documents/:id/versions
```

---

## 6. Mejoras de Infraestructura y Observabilidad

### 6.1. Observabilidad con OpenTelemetry

**Problema actual:** No hay trazas distribuidas entre los módulos Go. Es imposible saber cuánto tiempo toma cada paso del pipeline (recepción del request → embeddings → búsqueda vectorial → generación LLM → respuesta).

**Mejora propuesta:**

```go
// Instrumentación con otel en main.go
tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
)
otel.SetTracerProvider(tp)

// En cada handler
ctx, span := otel.Tracer("nextlab-api").Start(ctx, "rag.solve")
defer span.End()
span.SetAttributes(attribute.String("query", req.Text))
```

Métricas clave a instrumentar:
- Latencia P50/P95/P99 por endpoint
- Latencia de llamadas LLM (local vs cloud)
- Hit rate de búsqueda RAG (% queries con similitud > umbral)
- Tasa de fallback local → cloud del router IA

Stack sugerido: OpenTelemetry → Jaeger (traces) + Prometheus (metrics) + Grafana (dashboards). Todos orquestables en el mismo `docker-compose.yml`.

---

### 6.2. Health check extendido

**Problema actual:** El endpoint `/health` verifica database, cache y storage, pero no verifica que Ollama esté respondiendo ni que los modelos requeridos estén cargados.

**Mejora propuesta:**

```go
// Agregar verificación de Ollama al health check
func checkOllama(cfg *Config) HealthStatus {
    resp, err := http.Post(cfg.AIPrimaryURL+"/api/generate", "application/json",
        strings.NewReader(`{"model":"`+cfg.AIPrimaryModel+`","prompt":"ping","stream":false}`))
    if err != nil || resp.StatusCode != 200 {
        return HealthStatus{OK: false, Error: err.Error()}
    }
    return HealthStatus{OK: true}
}
```

Respuesta extendida:
```json
{
  "status": "ok",
  "dependencies": {
    "database": {"ok": true, "latency_ms": 2},
    "cache": {"ok": true, "latency_ms": 1},
    "storage": {"ok": true},
    "llm_local": {"ok": true, "model": "llama3", "latency_ms": 450},
    "embed_model": {"ok": true, "model": "nomic-embed-text", "dim": 768}
  }
}
```

---

### 6.3. Backup automatizado de PostgreSQL

**Problema actual:** No hay mención de estrategia de backup para PostgreSQL. Los datos académicos (calificaciones, apelaciones, trazas) son críticos y no se pueden recuperar si el volumen Docker se corrompe.

**Mejora propuesta:**

Agregar un servicio `backup` al `docker-compose.yml`:

```yaml
backup:
  image: prodrigestivill/postgres-backup-local
  restart: unless-stopped
  environment:
    - POSTGRES_HOST=postgres
    - POSTGRES_DB=nextlab
    - POSTGRES_USER=postgres
    - POSTGRES_PASSWORD=postgres
    - SCHEDULE=@daily
    - BACKUP_KEEP_DAYS=7
    - BACKUP_KEEP_WEEKS=4
  volumes:
    - ./backups:/backups
```

---

## 7. Nuevas Funcionalidades Propuestas

### 7.1. Notificaciones push para estudiantes

**Descripción:** Los estudiantes actualmente no reciben notificaciones cuando son calificados, cuando una apelación cambia de estado, o cuando se abre una sesión de examen.

**Implementación sugerida:**

- Server-Sent Events (SSE) para notificaciones en tiempo real en el frontend del estudiante.
- O bien, notificaciones por email usando un servicio SMTP (Resend, SendGrid) para eventos fuera de la sesión activa.

Nuevos endpoints:
```
GET  /student/notifications          # SSE stream
POST /student/notifications/:id/read # Marcar como leída
```

Nueva tabla:
```sql
CREATE TABLE student_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID REFERENCES users(id),
    type TEXT, -- 'grade_posted', 'appeal_updated', 'exam_starting'
    payload JSONB,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

### 7.2. Analytics comparativos anonimizados (privacy-preserving)

**Descripción:** El docente puede ver los analytics de su curso pero no puede comparar a un estudiante con el resto de la clase sin ver datos de los demás individualmente.

**Implementación sugerida:**

Nuevos endpoints de analytics que devuelven percentiles de clase sin PII:

```
GET /docente/cursos/:cid/actividades/:aid/analytics/percentiles
```

Respuesta:
```json
{
  "p25_score": 58,
  "p50_score": 72,
  "p75_score": 85,
  "student_score": 79,
  "student_percentile": 68,
  "class_size": 32
}
```

El `student_id` nunca sale del backend; solo se devuelve la posición relativa anonimizada.

---

### 7.3. Exportación de reportes en PDF

**Descripción:** Docentes y administradores necesitan imprimir o archivar reportes de calificaciones, análisis de detección y resolución de apelaciones para sus expedientes institucionales.

**Endpoints propuestos:**
```
GET /docente/cursos/:cid/actividades/:aid/report.pdf   # Reporte de actividad
GET /docente/cursos/:cid/actividades/:aid/submissions/:sid/detection-report.pdf
GET /appeals/:id/audit-report.pdf                       # Log de apelación para archivo
```

**Implementación:** Usar `chromedp` en Go para renderizar HTML → PDF, o bien una librería como `go-pdf`. El reporte de apelación debe incluir el log de `audit_events` completo con los hashes para validación forense.

---

### 7.4. Panel de comparación entre cursos para el administrador

**Descripción:** El admin actualmente gestiona carreras y materias pero no tiene visibilidad de métricas académicas agregadas: qué cursos tienen más irregularidades detectadas, qué materias tienen tasas de entrega bajas, etc.

**Endpoints propuestos:**
```
GET /admin/analytics/overview         # Métricas globales de la plataforma
GET /admin/analytics/integrity        # Resumen de detecciones por carrera/materia
GET /admin/analytics/submissions      # Tasas de entrega y calificación promedio
```

---

### 7.5. Modo offline para sesiones de examen

**Descripción:** Si el estudiante pierde conectividad durante un examen, los eventos de trace se pierden y la sesión puede quedar en estado `abandoned` por falta de heartbeats.

**Mejora propuesta:**

Implementar un buffer local en el frontend con `IndexedDB`:

```typescript
// En useTracingSession.ts
const offlineBuffer: TraceEvent[] = [];

function sendEvent(event: TraceEvent) {
  if (!navigator.onLine) {
    offlineBuffer.push(event);
    return;
  }
  flushBuffer();
  apiSendEvent(event);
}

window.addEventListener('online', flushBuffer);

async function flushBuffer() {
  if (offlineBuffer.length === 0) return;
  await apiSendEvents(offlineBuffer); // endpoint de batch
  offlineBuffer.length = 0;
}
```

El servidor debe aceptar eventos fuera de orden y recalcular la cadena de hashes al recibirlos en batch.

---

### 7.6. Integración con sistemas institucionales (LMS/SIS)

**Descripción:** Las universidades suelen tener sistemas de información académica (Banner, Ellucian, SAP) o LMS (Moodle, Canvas). NextLab actualmente es un silo.

**Mejora a largo plazo:**

- Exponer una API de exportación de calificaciones en formato LTI 1.3 o CSV estándar.
- Importar estudiantes y materias desde un endpoint CSV/JSON del SIS institucional.
- Webhook configurable para notificar al SIS cuando se publica una calificación final.

```
POST /admin/integrations/webhook
{
  "event": "grade.published",
  "url": "https://sis.universidad.edu/api/grades/webhook",
  "secret": "hmac-secret"
}
```

---

## 8. Resumen de Prioridades

| Prioridad | Ítem | Impacto | Esfuerzo |
|-----------|------|---------|----------|
| 🔴 Inmediata | Paginación en `trace_events` | Alto | Bajo |
| 🔴 Inmediata | Validación y sandbox de URLs Git | Alto | Bajo |
| 🔴 Inmediata | Corregir dimensión de embeddings | Alto | Bajo |
| 🟠 Corto plazo | HttpOnly cookies para JWT | Alto | Medio |
| 🟠 Corto plazo | Autorización granular en WebSocket | Medio | Bajo |
| 🟠 Corto plazo | Rate limiting en endpoints IA | Medio | Bajo |
| 🟠 Corto plazo | Dead letter queue en workers | Medio | Bajo |
| 🟡 Medio plazo | Cifrado SSE en MinIO | Alto | Bajo |
| 🟡 Medio plazo | Race condition en inscripciones | Medio | Bajo |
| 🟡 Medio plazo | Análisis temporal del heurístico | Alto | Medio |
| 🟡 Medio plazo | Búsqueda híbrida RAG (BM25 + embeddings) | Alto | Alto |
| 🟡 Medio plazo | Observabilidad con OpenTelemetry | Medio | Alto |
| 🟢 Largo plazo | Notificaciones push para estudiantes | Medio | Medio |
| 🟢 Largo plazo | Baseline de comportamiento por estudiante | Alto | Alto |
| 🟢 Largo plazo | Modelo ML supervisado para detección | Alto | Muy Alto |
| 🟢 Largo plazo | Exportación de reportes PDF | Medio | Medio |
| 🟢 Largo plazo | Integración LMS/SIS institucional | Medio | Muy Alto |

---

*Documento generado como parte del análisis técnico de NextLab — 2026-05-30*
