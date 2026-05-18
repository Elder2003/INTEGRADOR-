# Backend — Milestones de Desarrollo
## Sistema de Auditoría Académica con IA — Unifranz
**Stack: Go + Fiber + PostgreSQL (pgvector) + Redis + MinIO + Ollama + Vercel AI Gateway**

---

## Resumen

| Milestone | Nombre | Entregable clave |
|---|---|---|
| M1 | Fundación | Proyecto corriendo en Docker, schema de DB, health check |
| M2 | Autenticación y roles | Login, JWT, middleware de roles |
| M3 | Gestión académica | CRUD de cursos, grupos y actividades |
| M4 | Almacenamiento de archivos | MinIO integrado, upload y descarga de archivos |
| M5 | Entorno de resolución | WebSocket de trazabilidad, sesiones de examen, Judge0 |
| M6 | Router de IA | Ollama + Vercel Gateway con tres modos configurables |
| M7 | Sistema RAG | Ingesta y retrieval de RAG curricular y RAG docente |
| M8 | Motor de detección | Heurísticas + análisis LLM, worker asíncrono completo |
| M9 | Validación de formato | Parser PDF y Word contra plantilla institucional |
| M10 | Gestión de proyectos | Etapas, análisis Git, defensa oral |
| M11 | Alertas y analítica | SSE al docente, panel de métricas |
| M12 | Módulos del administrador | Plantillas, RAG curricular, interdisciplinario, config IA |
| M13 | Apelaciones y auditoría final | Flujo de apelación, cadena de decisiones, inmutabilidad |

---

## M1 — Fundación del Proyecto

**Objetivo:** Proyecto Go ejecutable en Docker con base de datos lista y estructura de código definida.

### Tareas

- [ ] Inicializar módulo Go (`go mod init`)
- [ ] Estructura de carpetas por módulo (`/auth`, `/academic`, `/tracing`, `/rag`, `/detection`, `/ai`, `/storage`, `/projects`, `/alerts`, `/analytics`, `/admin`)
- [ ] Docker Compose con todos los servicios: Fiber API, Worker Go, PostgreSQL, Redis, MinIO
- [ ] Imagen Docker multi-stage para el binario Go (build + runtime)
- [ ] Schema inicial de PostgreSQL con extensión `pgvector`
- [ ] Variables de entorno centralizadas (`config.go` con env vars tipadas)
- [ ] Health check endpoint `GET /health` que valida conectividad con DB, Redis y MinIO
- [ ] Migraciones de base de datos con herramienta (`golang-migrate` o similar)
- [ ] Logger estructurado (JSON) con niveles info/warn/error

### Entregable

`docker compose up` levanta todos los servicios. `GET /health` responde 200 con estado de cada dependencia.

---

## M2 — Autenticación y Roles

**Objetivo:** Sistema de identidad completo con cuatro roles y tokens de corta duración.

### Tareas

- [ ] Tabla `users` con campos: id, email, password_hash, role, created_at
- [ ] Roles: `admin`, `docente`, `co_docente`, `estudiante`
- [ ] Endpoint `POST /auth/login` — valida credenciales, emite access token (15 min) + refresh token
- [ ] Endpoint `POST /auth/refresh` — rota el refresh token, emite nuevo access token
- [ ] Endpoint `POST /auth/logout` — invalida el refresh token en DB
- [ ] Middleware de autenticación JWT para Fiber
- [ ] Middleware de autorización por rol (guards por endpoint)
- [ ] Tabla `refresh_tokens` con revocación por usuario y por sesión
- [ ] Hash de contraseñas con bcrypt
- [ ] Endpoint `POST /auth/forgot-password` y `POST /auth/reset-password`

### Entregable

Login funcional. Endpoints protegidos rechazan tokens inválidos o expirados. Cada rol accede solo a sus rutas.

---

## M3 — Gestión Académica

**Objetivo:** CRUD completo de la estructura académica que el docente necesita para crear actividades.

### Tareas

- [ ] Tablas: `courses`, `groups`, `group_members`, `activities`, `activity_configs`
- [ ] `POST /courses` — crear curso (docente)
- [ ] `GET /courses` — listar cursos del docente autenticado
- [ ] `POST /courses/:id/groups` — crear grupo dentro de un curso
- [ ] `POST /groups/:id/members` — agregar estudiantes al grupo (por email o bulk CSV)
- [ ] `POST /activities` — crear actividad con tipo, modalidad, fecha límite y config de auditoría
- [ ] Tipos de actividad: `exam`, `task`, `report`, `practice`, `project`
- [ ] Validación de que actividades de tipo `practice` solo se crean en materias elegibles
- [ ] `GET /activities/:id` — detalle de actividad con su configuración
- [ ] `PATCH /activities/:id` — editar fecha límite y estado (draft / published / closed)
- [ ] `GET /students/activities` — actividades activas visibles para el estudiante autenticado
- [ ] Marcado automático de entregas tardías al comparar timestamp con fecha límite

### Entregable

Docente puede crear un curso, agregar estudiantes y publicar una actividad. Estudiante ve sus actividades activas.

---

## M4 — Almacenamiento de Archivos

**Objetivo:** MinIO integrado como backend de almacenamiento para todos los archivos del sistema.

### Tareas

- [ ] Cliente MinIO en Go con configuración por env vars (`MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`)
- [ ] Buckets separados por tipo: `submissions`, `rag-content`, `captured-urls`, `reports`
- [ ] Middleware de upload multipart con límite de tamaño configurable
- [ ] `POST /submissions/:activityId` — subida de archivo de tarea o informe (.pdf, .docx, .zip)
- [ ] Registro de entrega en DB con: student_id, activity_id, bucket_path, timestamp, is_late
- [ ] Versionado de entregas: si el estudiante sube otra versión, la anterior queda marcada como `superseded`
- [ ] `GET /submissions/:id/download` — descarga de archivo (solo docente o admin de esa actividad)
- [ ] `POST /rag/content` — subida de material docente (PDF, diapositivas, etc.) a MinIO
- [ ] Captura y almacenamiento local de URLs externas al momento de la carga (no depender de disponibilidad futura)
- [ ] Extracción de transcript de YouTube al subir un enlace de video

### Entregable

Estudiante sube un archivo. Se almacena en MinIO. Docente puede descargarlo. Entregas tardías quedan marcadas.

---

## M5 — Entorno de Resolución y Trazabilidad

**Objetivo:** WebSocket funcional para captura de trazabilidad cognitiva en tiempo real y ejecución sandboxed de código.

### Tareas

**Sesiones de examen**
- [ ] Tabla `exam_sessions` con: student_id, activity_id, started_at, submitted_at, status
- [ ] `POST /sessions/start` — inicia sesión, devuelve session_id y token WebSocket de corta duración
- [ ] `POST /sessions/:id/submit` — cierra la sesión, congela la trazabilidad, encola trabajo en Redis Streams

**WebSocket de trazabilidad**
- [ ] Endpoint WebSocket `WS /sessions/:id/trace`
- [ ] Eventos aceptados: `keystroke`, `paste`, `delete`, `inactivity`, `execution`, `focus_change`
- [ ] Cada evento se persiste inmediatamente en tabla `trace_events` (append-only, sin UPDATE ni DELETE)
- [ ] Validación de session_id y autenticación en el handshake WebSocket
- [ ] Reconexión transparente: el cliente puede reconectarse sin perder eventos previos

**Ejecución sandboxed**
- [ ] Cliente Judge0 en Go (HTTP)
- [ ] `POST /sessions/:id/run` — envía código a Judge0, devuelve stdout/stderr/exit_code
- [ ] En modo práctica: devuelve resultado completo
- [ ] En modo examen: devuelve solo si compiló/ejecutó correctamente, sin stdout

**Worker — cola Redis Streams**
- [ ] Proceso Go separado que consume el stream `analysis_jobs`
- [ ] Al recibir un job: lee la sesión, la trazabilidad y el tipo de actividad
- [ ] Estructura base del worker lista para recibir los módulos de análisis de M8

### Entregable

Estudiante abre una sesión, escribe código, los eventos llegan al servidor por WebSocket y quedan en DB. Al entregar, el job aparece en Redis Streams y el worker lo consume.

---

## M6 — Router de IA

**Objetivo:** Capa de abstracción que enruta solicitudes entre Ollama y Vercel AI Gateway con tres modos configurables desde el panel del administrador.

### Tareas

- [ ] Interfaz Go `AIClient` con método `Complete(ctx, req) (AIResponse, error)`
- [ ] `OllamaClient` — HTTP al endpoint `/v1/chat/completions` del VPS 2 (OpenAI-compatible)
- [ ] `VercelGatewayClient` — HTTP a Vercel AI Gateway con API key por env var
- [ ] Tabla `ai_config` con columna `mode` (`local` | `auto` | `cloud`) y `complexity_threshold`
- [ ] `AIRouter` — lee config de DB con cache de 5 segundos, decide qué cliente usar
- [ ] Modo `local`: siempre Ollama
- [ ] Modo `cloud`: siempre Vercel Gateway
- [ ] Modo `auto`: Ollama primario; si falla o complexity > threshold → Vercel Gateway
- [ ] `PATCH /admin/ai-config` — actualiza modo sin reinicio (solo admin)
- [ ] `GET /admin/ai-config` — estado actual del router
- [ ] Logging de qué backend procesó cada solicitud (para auditoría de análisis)

### Entregable

Admin cambia el modo desde el panel. La siguiente solicitud de IA usa el backend correcto. Si Ollama está caído en modo `auto`, el sistema conmuta automáticamente a Vercel.

---

## M7 — Sistema RAG Dual

**Objetivo:** Ingesta, vectorización y retrieval de los dos RAGs independientes usando pgvector.

### Tablas

```sql
CREATE TABLE rag_curricular (
    id UUID PRIMARY KEY,
    carrera TEXT, semestre INT, codigo_materia TEXT,
    content TEXT, embedding vector(1536)
);

CREATE TABLE rag_docente (
    id UUID PRIMARY KEY,
    docente_id UUID, curso_id UUID, activity_id UUID,
    content_type TEXT, content TEXT, embedding vector(1536),
    is_rubric BOOLEAN DEFAULT FALSE
);
```

### Tareas

**RAG Curricular**
- [ ] Parser de XLSX en formato estándar Unifranz
- [ ] Extracción de: código de materia, semestre, carrera, objetivos, contenidos mínimos
- [ ] Vectorización y almacenamiento en `rag_curricular`
- [ ] `POST /admin/curriculum` — carga de plan instruccional (admin)
- [ ] Retrieval filtrado por `carrera + semestre + codigo_materia` antes de similitud semántica

**RAG Docente**
- [ ] Pipeline de clasificación de contenido subido por el docente:
  - PDF / diapositivas → fragmentación semántica por sección
  - Libro comercial → solo tabla de contenidos (copyright)
  - Libro de acceso abierto → indexado completo
  - Rúbrica → almacenada sin fragmentar (inyección directa siempre)
  - URL capturada → texto extraído y fragmentado
  - Transcript de YouTube → fragmentado por segmentos
  - Repositorio de código → indexado por ruta indicada
- [ ] Vectorización con el modelo de embeddings configurado
- [ ] Retrieval por prioridad (rúbrica → material de evaluación → material de curso → RAG curricular)
- [ ] Filtros: `docente_id`, `curso_id`, `activity_id` antes de cualquier operación de retrieval

### Entregable

Docente sube un PDF. El sistema lo fragmenta y vectoriza. Una consulta de retrieval devuelve los fragmentos más relevantes filtrados por curso.

---

## M8 — Motor de Detección

**Objetivo:** Worker asíncrono completo que analiza sesiones de examen y entregas de tareas/informes con heurísticas y LLM.

### Tareas

**Heurísticas deterministas**
- [ ] Detector de paste masivo: >N caracteres en <2 segundos
- [ ] Detector de velocidad de tipeo imposible: WPM sostenido > umbral configurable
- [ ] Detector de solución sin borrados: código entregado sin ningún evento `delete`
- [ ] Detector de inactividad seguida de entrega inmediata: silencio >X minutos + submit
- [ ] Cada heurística produce un flag con nombre, valor y umbral activado
- [ ] Umbrales configurables por actividad (no hardcodeados)

**Análisis LLM**
- [ ] Construcción del prompt: flags heurísticos + contenido entregado + RAG curricular + RAG docente
- [ ] Llamada al router de IA (M6)
- [ ] Parseo de respuesta: score numérico + justificación en lenguaje natural
- [ ] Almacenamiento del resultado: `analysis_results` con score, justificación, backend_used, model_used

**Worker**
- [ ] El worker de Redis Streams (M5) llama al módulo de detección al procesar cada job
- [ ] Jobs por tipo: `exam_analysis`, `task_analysis`, `report_analysis`
- [ ] Manejo de errores: si el LLM falla, el resultado heurístico se guarda igualmente
- [ ] SLA: análisis completo en máximo 3 minutos desde la entrega

**Calificación sugerida**
- [ ] El análisis produce una calificación sugerida basada en el score
- [ ] Endpoint `GET /activities/:id/submissions/:submissionId/analysis` — devuelve análisis al docente
- [ ] Endpoint `PATCH /activities/:id/submissions/:submissionId/grade` — docente asigna calificación final; si difiere de la sugerida, registra motivo

### Entregable

Estudiante entrega examen. En menos de 3 minutos, el docente puede ver el reporte de flags heurísticos, el análisis en lenguaje natural y la calificación sugerida.

---

## M9 — Validación de Formato de Informes

**Objetivo:** Módulo que contrasta PDFs y documentos Word contra la plantilla institucional y genera un reporte de cumplimiento.

### Tareas

**Plantilla institucional**
- [ ] Tabla `format_templates` con parámetros configurables por carrera: secciones requeridas, rango de páginas, formato de cita, mínimo de fuentes, formato de fuente y márgenes
- [ ] `POST /admin/format-templates` — crear o actualizar plantilla (admin)

**Parser PDF**
- [ ] Extracción de texto con estructura de secciones (biblioteca `pdfcpu` o similar en Go)
- [ ] Detección de imágenes y verificación de pie de figura
- [ ] Heurísticas de márgenes y fuente desde la metadata del PDF
- [ ] Detección de secciones: portada, introducción, desarrollo, conclusión, bibliografía

**Parser Word (.docx)**
- [ ] Extracción de estilos de título, párrafos, márgenes y tamaño de fuente desde el XML del .docx
- [ ] Conteo de fuentes bibliográficas con detección de formato APA/Vancouver
- [ ] Verificación de estructura de secciones

**Reporte de cumplimiento**
- [ ] Por cada parámetro de la plantilla: estado (`ok`, `warning`, `fail`) + observación específica
- [ ] El reporte se almacena en `format_reports` y se adjunta al análisis unificado del informe
- [ ] La entrega siempre se acepta independientemente del resultado

**Integración con el worker**
- [ ] Job `report_analysis` ejecuta en paralelo: validación de formato + detección de IA (M8)
- [ ] El docente recibe un reporte unificado con ambos resultados

### Entregable

Estudiante sube un informe PDF o Word. El docente recibe un reporte que indica qué parámetros institucionales cumple y cuáles no, junto con el análisis de IA del contenido.

---

## M10 — Gestión de Proyectos

**Objetivo:** Flujo completo de proyectos con etapas, análisis Git y defensa oral.

### Tareas

**Configuración de proyectos**
- [ ] Tablas: `projects`, `project_stages`, `stage_submissions`, `git_analyses`
- [ ] `POST /projects` — crear proyecto con etapas, activar/desactivar análisis Git y defensa oral
- [ ] Cada etapa tiene: nombre, fecha límite, tipo (`task` | `report`)

**Entregas por etapa**
- [ ] `POST /stages/:id/submit` — sube .zip a MinIO + enlace Git opcional
- [ ] Versionado: segunda entrega queda como `late_submission` con timestamp; ambas versiones persisten
- [ ] Job `stage_analysis` en Redis Streams al recibir la entrega

**Análisis Git**
- [ ] Clonar el repositorio del enlace proporcionado (shallow clone)
- [ ] Extraer: número de commits por período, volumen de líneas por commit, autores, distribución temporal
- [ ] Detectar patrones anómalos: commit masivo único, commits sin historial previo, volumen inconsistente
- [ ] Reporte de historial Git adjunto al reporte de etapa

**Análisis de coherencia entre etapas**
- [ ] Al procesar una etapa, comparar el contenido y calidad con las etapas anteriores del mismo proyecto
- [ ] Detectar saltos anómalos en complejidad o estilo entre entregas consecutivas

**Defensa oral**
- [ ] `POST /projects/:id/defense` — programa sesión de defensa con fecha y hora
- [ ] La sesión de defensa funciona como un examen especial: activa WebSocket de trazabilidad
- [ ] Al finalizar, el worker analiza coherencia entre respuestas de la defensa y el código del .zip

**Reporte unificado de etapa**
- [ ] Combina: análisis del .zip + reporte Git + coherencia con etapas anteriores
- [ ] En etapas de tipo `report`: incluye también validación de formato (M9)

### Entregable

Docente configura un proyecto de 4 etapas. Estudiante sube el .zip y enlace Git de cada etapa. El docente recibe el reporte unificado con análisis de contenido, historial Git y coherencia con etapas previas.

---

## M11 — Alertas, Notificaciones y Analítica

**Objetivo:** Sistema de alertas en tiempo real al docente y panel de analítica agregada.

### Tareas

**Alertas**
- [ ] Tabla `alerts` con: analysis_id, docente_id, status (`pending` | `confirmed` | `false_positive` | `inconclusive`), created_at
- [ ] Al finalizar un análisis con score > umbral, crear alerta y notificar al docente
- [ ] `PATCH /alerts/:id` — docente marca estado de la alerta
- [ ] Registro del motivo cuando la calificación final difiere de la sugerida

**SSE — notificaciones en tiempo real**
- [ ] Endpoint `GET /notifications/stream` — SSE que mantiene conexión abierta con el panel del docente
- [ ] Al crearse una alerta, el servidor la emite por SSE al docente correspondiente
- [ ] Filtrado por docente: cada docente solo recibe alertas de sus propios estudiantes (excepto co-docencia)

**Analítica**
- [ ] `GET /analytics/courses/:id` — métricas del curso: distribución de scores, actividades con más alertas, evolución temporal
- [ ] `GET /analytics/students/:id` — historial de análisis de un estudiante en un curso
- [ ] `GET /analytics/activities/:id` — métricas por actividad: score promedio, flags más frecuentes

### Entregable

Docente abre su panel y recibe alertas en tiempo real por SSE cuando un análisis supera el umbral. Puede ver tendencias de su grupo en el panel de analítica.

---

## M12 — Módulos del Administrador

**Objetivo:** Funcionalidades exclusivas del administrador: plantillas institucionales, gestión curricular, programas interdisciplinarios y configuración de IA.

### Tareas

**Gestión de docentes**
- [ ] `POST /admin/users` — crear cuenta de docente con rol y sede
- [ ] `PATCH /admin/users/:id` — activar/desactivar cuenta
- [ ] `GET /admin/users` — listar docentes con filtros

**Elegibilidad de materias para práctica asistida**
- [ ] Campo `practice_eligible` en tabla de materias
- [ ] `PATCH /admin/subjects/:id/eligibility` — marcar materia como elegible o no

**Módulo de resolución de problemas reales**
- [ ] `POST /admin/interdisciplinary/analyze` — envía descripción al LLM con RAG curricular, recibe propuesta de materias por carrera
- [ ] `POST /admin/interdisciplinary/programs` — crea programa con las materias seleccionadas, asigna docentes y grupos
- [ ] Los cursos de programas interdisciplinarios aparecen diferenciados en el panel del docente asignado
- [ ] `GET /admin/interdisciplinary/history` — historial de consultas y programas creados

**Configuración del proveedor de IA**
- [ ] Panel con toggle de tres modos: local / auto / cloud (ya implementado en M6, aquí se expone al frontend admin)
- [ ] `GET /admin/ai-config/status` — estado actual del router y del VPS de Ollama (ping)

### Entregable

Admin crea un programa interdisciplinario completo desde la descripción hasta la activación de grupos. Puede cambiar el modo de IA y verificar el estado de Ollama desde el panel.

---

## M13 — Apelaciones y Auditoría Final

**Objetivo:** Flujo formal de apelaciones con acceso a trazabilidad inmutable y cadena de decisiones completa.

### Tareas

**Flujo de apelación**
- [ ] Tabla `appeals` con: submission_id, student_id, reason, status, created_at
- [ ] `POST /appeals` — estudiante inicia apelación con justificación
- [ ] `PATCH /appeals/:id/escalate` — docente escala al administrador
- [ ] `PATCH /appeals/:id/resolve` — administrador emite resolución final registrada

**Acceso a trazabilidad para auditoría**
- [ ] `GET /appeals/:id/evidence` — admin accede a: trazabilidad completa (inmutable), análisis de IA, decisiones del docente con timestamps y justificaciones
- [ ] Verificación de inmutabilidad: endpoint que confirma que los eventos de trazabilidad no han sido modificados desde su creación (hash de registros)

**Registro de cadena de decisiones**
- [ ] Tabla `decision_log` con: appeal_id, actor_id, actor_role, action, justification, timestamp
- [ ] Cada acción en el flujo de apelación queda registrada con su actor y justificación

**Auditoría de ingestas**
- [ ] `GET /admin/audit/ingestions` — historial de cargas de material RAG con autor, fecha, archivo, resultado y advertencias

**Revisión final del sistema**
- [ ] Tests de integración end-to-end para los flujos críticos: examen completo, entrega de informe, apelación
- [ ] Verificación de SLA del worker (< 3 minutos por análisis)
- [ ] Revisión de permisos: confirmar que ningún rol accede a datos fuera de su scope

### Entregable

Estudiante apela una calificación. El administrador accede a toda la evidencia inmutable y emite una resolución. La cadena completa queda registrada y auditable.

---

*Backend milestones — Proyecto Integrador Elder, Unifranz, Mayo 2026.*
