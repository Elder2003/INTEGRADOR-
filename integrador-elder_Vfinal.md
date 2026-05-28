# NextLab — Documentación Final del Proyecto Integrador

**Versión:** 1.0.0  
**Autor:** Elder Hamer  
**Fecha:** 2026-05-27  
**Repositorios:** `nextlab-api` (backend Go) · `nextlab-web` (frontend React)

---

## Tabla de Contenidos

1. [Visión General del Sistema](#1-visión-general-del-sistema)
2. [Arquitectura](#2-arquitectura)
3. [Stack Tecnológico](#3-stack-tecnológico)
4. [Infraestructura y Despliegue](#4-infraestructura-y-despliegue)
5. [Base de Datos — Esquema Completo](#5-base-de-datos--esquema-completo)
6. [Backend — Módulos y API](#6-backend--módulos-y-api)
7. [Frontend — Estructura y Páginas](#7-frontend--estructura-y-páginas)
8. [Autenticación y Roles](#8-autenticación-y-roles)
9. [Sistema RAG e Inteligencia Artificial](#9-sistema-rag-e-inteligencia-artificial)
10. [Motor de Detección de Integridad](#10-motor-de-detección-de-integridad)
11. [Sistema de Apelaciones](#11-sistema-de-apelaciones)
12. [Guía de Ejecución](#12-guía-de-ejecución)
13. [Variables de Entorno](#13-variables-de-entorno)
14. [Usuarios de Prueba](#14-usuarios-de-prueba)

---

## 1. Visión General del Sistema

NextLab es una plataforma educativa universitaria diseñada para la gestión integral de cursos, actividades evaluativas y seguimiento de integridad académica. El sistema contempla cuatro roles de usuario: administrador, docente, co-docente y estudiante.

### 1.1. Objetivos Principales

- Centralizar la gestión de carreras, materias, cursos y actividades académicas.
- Proveer trazabilidad en tiempo real del comportamiento del estudiante durante exámenes y defensas orales.
- Detectar posibles irregularidades académicas mediante análisis heurístico y modelos de lenguaje (LLM).
- Permitir la subida, revisión y calificación de entregas (archivos, proyectos Git, investigaciones por etapas).
- Gestionar apelaciones con flujo de revisión docente → escalación admin.
- Alimentar una base de conocimiento curricular consultable mediante RAG (Retrieval-Augmented Generation) para asistencia pedagógica con IA.

### 1.2. Milestones de Desarrollo

| MW | Nombre | Descripción |
|----|--------|-------------|
| MW1 | Inicialización | Estructura base, Docker Compose, migraciones |
| MW2 | Gestión Académica | Cursos, materias, actividades, inscripciones |
| MW3 | Panel Admin | Carreras, materias, perfiles de estudiantes, docentes |
| MW4 | Portal Estudiante | Dashboard, actividades, entregas de archivos |
| MW5 | Trazabilidad | Sesiones de examen, eventos de trace, WebSocket en vivo |
| MW6 | Router IA | Enrutamiento local/cloud para modelos LLM |
| MW7 | RAG Dual | Pipeline de embeddings curricular y docente |
| MW8 | Motor de Detección | Análisis heurístico + LLM sobre trazas de examen |
| MW9 | Validación de Formato | Extracción y validación de documentos entregados |
| MW10 | Gestión de Proyectos | Investigaciones por etapas, análisis Git, defensas orales |
| MW11 | Alertas y Analytics | Sistema de alertas para docentes, analíticas de actividad |
| MW12 | Módulos Admin | Templates de formato, actividades interdisciplinarias |
| MW13 | Apelaciones y Auditoría | Flujo completo de apelaciones con log inmutable |

---

## 2. Arquitectura

### 2.1. Diagrama de Alto Nivel

```
┌─────────────────────────────────────────────────────────────────┐
│                        Cliente (Browser)                        │
│              React 19 + Vite 8 + Tailwind CSS v4                │
│                     http://localhost:5173                        │
└───────────────────────────┬─────────────────────────────────────┘
                            │ HTTP + WebSocket
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Go API (Fiber v2)                             │
│                     http://localhost:8080                        │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐   │
│  │   auth   │ │ academic │ │submission│ │    tracing/WS    │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐   │
│  │detection │ │   rag    │ │projects  │ │  appeals/audit   │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                        │
│  │  admin   │ │  alerts  │ │analytics │                        │
│  └──────────┘ └──────────┘ └──────────┘                        │
└───┬──────────────┬──────────────┬────────────────┬─────────────┘
    │              │              │                │
    ▼              ▼              ▼                ▼
┌────────┐  ┌──────────┐  ┌───────────┐  ┌─────────────┐
│Postgres│  │  Redis   │  │   MinIO   │  │   Ollama    │
│pgvector│  │(sessions)│  │(archivos) │  │(LLM+embed)  │
│ :5433  │  │  :6379   │  │  :9000    │  │   :11434    │
└────────┘  └──────────┘  └───────────┘  └─────────────┘
```

### 2.2. Principios de Diseño

- **Append-only para datos críticos**: las tablas `trace_events`, `audit_events`, `analysis_results` y `grade_decisions` nunca reciben UPDATE ni DELETE. Garantizan trazabilidad forense.
- **Sin lógica de negocio en el frontend**: el cliente sólo muestra lo que devuelve la API.
- **Token de acceso en memoria**: el JWT de acceso nunca se persiste en `localStorage`. El refresh token se almacena en Redis.
- **Separación de pipelines RAG**: los documentos curriculares (admin) y los del docente viven en tablas físicamente separadas para evitar contaminación cruzada.
- **CORS por función lambda**: la API permite cualquier origen `localhost` para simplificar el desarrollo sin exponer credenciales.

---

## 3. Stack Tecnológico

### 3.1. Backend (`nextlab-api`)

| Componente | Tecnología | Versión |
|------------|-----------|---------|
| Lenguaje | Go | 1.25 |
| Framework HTTP | Fiber v2 | 2.52.0 |
| ORM / DB Driver | pgx v5 | 5.6.0 |
| Migraciones | golang-migrate | 4.17.0 |
| Autenticación | golang-jwt/jwt v5 | 5.2.1 |
| Caché / Sesiones | go-redis v9 | 9.5.1 |
| Almacenamiento objetos | minio-go v7 | 7.0.67 |
| Extracción PDF | pdfcpu | 0.12.1 |
| Análisis Git | go-git v5 | 5.19.1 |
| OAuth2 Google | golang.org/x/oauth2 | 0.36.0 |
| WebSocket | gofiber/websocket v2 | 2.2.1 |
| IDs únicos | google/uuid | 1.6.0 |
| Hashing contraseñas | golang.org/x/crypto (bcrypt) | 0.51.0 |

### 3.2. Frontend (`nextlab-web`)

| Componente | Tecnología | Versión |
|------------|-----------|---------|
| Lenguaje | TypeScript | 5.9.3 (strict) |
| Framework UI | React | 19.2.4 |
| Build tool | Vite | 8.0.1 |
| Estilos | Tailwind CSS | 4.2.2 |
| Routing | React Router DOM | 7.13.1 |
| HTTP Client | Axios | 1.16.1 |
| Iconos | Lucide React | 1.16.0 |

### 3.3. Infraestructura

| Servicio | Imagen Docker | Puerto host |
|----------|--------------|-------------|
| PostgreSQL + pgvector | pgvector/pgvector:pg16 | 5433 |
| Redis | redis:7-alpine | 6379 |
| MinIO | minio/minio:latest | 9000 / 9001 |
| Ollama (LLM + embeddings) | ollama/ollama:latest | 11434 |
| API Go | build local | 8080 |
| Worker Go | build local | — |

---

## 4. Infraestructura y Despliegue

### 4.1. Docker Compose

El archivo `nextlab-api/docker-compose.yml` orquesta todos los servicios con `restart: unless-stopped` para que Docker los reinicie automáticamente ante cualquier fallo o reinicio del sistema.

```yaml
# Servicios con restart automático:
# postgres, redis, minio, ollama, api, worker
```

La política `restart: unless-stopped` garantiza que:
- El sistema se recupera automáticamente tras un crash.
- Al reiniciar el equipo, los servicios vuelven a levantarse solos.
- No requiere intervención manual para restaurar el estado.

### 4.2. Build de la API

El `docker/Dockerfile` usa construcción multi-stage:

1. **Stage builder** — `golang:1.25-alpine`: compila los binarios `api` y `worker`.
2. **Stage runtime** — `alpine:3.19`: imagen mínima (~10 MB) con sólo los binarios compilados y certificados CA.

### 4.3. Migraciones

Las migraciones SQL están embebidas en el binario usando `//go:embed` (`migrations/embed.go`). Al iniciar, la API ejecuta automáticamente todas las migraciones pendientes con `golang-migrate`. No se requiere paso manual de migración.

### 4.4. Comandos de Operación

```bash
# Iniciar todo (primera vez o tras cambios en el código Go)
cd nextlab-api
docker compose up --build -d

# Iniciar sin recompilar (cuando no hay cambios Go)
docker compose up -d

# Ver logs en vivo
docker compose logs -f api

# Detener todo
docker compose down

# Iniciar frontend (en otra terminal)
cd nextlab-web
npm run dev
```

---

## 5. Base de Datos — Esquema Completo

### 5.1. Extensión de base de datos

```sql
-- Migración 000001
CREATE EXTENSION IF NOT EXISTS vector;
```

La extensión `pgvector` habilita el tipo de dato `VECTOR(n)` y los índices `ivfflat` para búsqueda por similitud coseno, base del pipeline RAG.

### 5.2. Autenticación (000002)

- **`users`** — tabla central de usuarios con roles: `admin`, `docente`, `co_docente`, `estudiante`. Contiene `password_hash` (bcrypt costo 12) y soporte opcional para `google_sub` (OAuth2).
- **`refresh_tokens`** — tokens de refresco con rotación y revocación. Campo `replaced_by` permite trazabilidad de la cadena de rotación.
- **`password_reset_tokens`** — tokens de un solo uso para recuperación de contraseña.

### 5.3. Estructura Académica (000003)

- **`carreras`** — catálogo de carreras universitarias (código, nombre).
- **`materias`** — materias por carrera con semestre (1–10).
- **`student_profiles`** — perfil pre-provisionado por admin: email → carrera + semestre + código de estudiante.
- **`cursos`** — instancia de una materia dictada por un docente en un semestre/sección. Constraint único: `(materia_id, docente_id, semester_label, section)`.
- **`grupos`** — grupos de trabajo dentro de un curso.
- **`actividades`** — evaluaciones del curso. Tipos: `practica`, `tarea`, `informe`, `examen`, `defensa_oral`, `investigacion`, `git_analysis`. El campo `trace_enabled` se calcula automáticamente para `examen` y `defensa_oral`.
- **`inscripciones`** — membresías de estudiantes en cursos (generadas automáticamente al hacer login).
- **`grupo_asignaciones`** — asignación de estudiantes a grupos por el docente.
- **`codocente_grants`** — permisos de co-docencia por actividad, con revocación.

### 5.4. Almacenamiento de Archivos (000004)

- **`submissions`** — entregas de estudiantes (un registro activo por estudiante por actividad). Almacena la clave MinIO, nombre original, tamaño, MIME type y si fue tardía.
- **`actividad_config_changes`** — log inmutable de cambios de configuración en actividades (append-only).

### 5.5. Trazabilidad de Exámenes (000005)

- **`exam_sessions`** — sesión de examen por estudiante por actividad. Estados: `active`, `completed`, `deadline_closed`, `abandoned`, `timed_out`.
- **`trace_events`** — log append-only de eventos de comportamiento: `tab_blur`, `tab_focus`, `window_blur`, `window_focus`, `clipboard_copy`, `clipboard_paste`, `heartbeat`, `session_open`, `session_close`. Cada evento tiene un hash de cadena para detección de manipulación.
- **`worker_jobs`** — cola de trabajos asíncronos para procesamiento post-sesión.

### 5.6. Router de IA (000006)

- **`router_configs`** — historial de cambios de modo del router IA: `local`, `auto`, `cloud`. Append-only.
- **`ai_request_logs`** — metadatos de cada petición IA (sin contenido de prompts). Registra backend usado, latencia, fallback y estado.

### 5.7. Pipeline RAG Dual (000007)

- **`rag_curricular`** — documentos del currículo institucional (gestionados por admin). Indexados por `(carrera, semestre, codigo_materia)`.
- **`rag_curricular_chunks`** — fragmentos con embedding `VECTOR(1536)`. Índice `ivfflat` para búsqueda ANN.
- **`rag_docente`** — documentos subidos por docentes (ligados a su curso).
- **`rag_docente_chunks`** — fragmentos del pipeline docente (físicamente separados del curricular).

### 5.8. Motor de Detección (000008)

- **`analysis_results`** — resultados de análisis heurístico y LLM por sesión. Máximo dos filas por sesión (una por tipo). Append-only.
- **`grade_decisions`** — decisión de calificación final del docente, con referencia al análisis y razón de override si aplica.

### 5.9. Validación de Formato (000009)

- **`format_reports`** — reporte de validación estructural del archivo entregado: páginas, palabras, caracteres, metadata, cumplimiento de reglas configuradas en `actividades.format_rules` (JSONB).

### 5.10. Gestión de Proyectos (000010)

- **`investigacion_stages`** — etapas de una actividad tipo `investigacion` con sus propios deadlines.
- **`stage_submissions`** — entrega por etapa por estudiante, con calificación individual.
- **`git_submissions`** — URL de repositorio Git enviada por el estudiante para análisis.
- **`git_analyses`** — resultado del análisis asíncrono de Git: commits, autores, distribución temporal, flags de alerta, advisory LLM.
- **`git_analysis_jobs`** — cola de trabajos para análisis Git asíncrono.
- **`defense_appointments`** — turnos programados para defensa oral por actividad/estudiante.

### 5.11. Alertas (000011)

- **`alerts`** — notificaciones para docentes: ligadas a actividad y/o estudiante, con payload JSONB libre. Incluye flag `advisory` para sugerencias no urgentes.

### 5.12. Módulos Admin (000012)

- **`format_templates`** — plantillas de reglas de formato reutilizables (creadas por admin).
- **`actividad_format_template`** — aplicación de una plantilla a una actividad específica.
- **`actividad_interdisciplinary_scopes`** — scopes curriculares asociados a una actividad interdisciplinaria `(carrera, semestre, codigo_materia)`.

### 5.13. Apelaciones y Auditoría (000013)

- **`appeals`** — apelaciones ligadas a una `exam_session`. Tipos: `analysis_dispute`, `grade_dispute`. Estados: `pending`, `escalated`, `resolved`, `admin_resolved`. Una sola apelación por sesión.
- **`appeal_decisions`** — decisiones de revisión (docente o admin): `upheld`, `rejected`, `overridden`.
- **`audit_events`** — log hash-chained inmutable de todos los eventos del ciclo de vida de cada apelación.

---

## 6. Backend — Módulos y API

### 6.1. Estructura de Paquetes

```
nextlab-api/
├── cmd/
│   ├── api/main.go          # Punto de entrada HTTP
│   └── worker/main.go       # Worker de procesamiento asíncrono
├── internal/
│   ├── academic/            # Cursos, materias, actividades, inscripciones
│   ├── admin/               # Gestión admin (docentes, templates, curriculum)
│   ├── ai/                  # Router LLM local/cloud
│   ├── alerts/              # Alertas y notificaciones
│   ├── analytics/           # Analíticas de actividad y curso
│   ├── appeals/             # Apelaciones y auditoría
│   ├── auth/                # JWT, refresh tokens, Google OAuth
│   ├── config/              # Carga de variables de entorno
│   ├── detection/           # Motor heurístico + LLM de detección
│   ├── format/              # Validación de formato de documentos
│   ├── health/              # Endpoint /health
│   ├── projects/            # Investigaciones, Git analysis, defensas
│   ├── rag/                 # Pipeline RAG dual (ingest + retrieve + solve)
│   ├── submission/          # Entregas de estudiantes
│   ├── storage/             # Abstracción MinIO
│   └── tracing/             # Sesiones de examen, trace events, WebSocket
├── migrations/              # 13 migraciones SQL embebidas
└── docker/Dockerfile        # Build multi-stage
```

### 6.2. Módulo Auth

**Endpoints:**

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/auth/login` | Login con email/password. Devuelve `access_token` + `refresh_token`. |
| POST | `/auth/refresh` | Rota el refresh token. Devuelve nuevo par de tokens. |
| POST | `/auth/logout` | Revoca el refresh token activo. |
| GET | `/auth/google` | Inicia flujo OAuth2 Google. |
| GET | `/auth/google/callback` | Callback OAuth2. Crea usuario estudiante si no existe. |

**Seguridad:**
- Contraseñas hasheadas con bcrypt costo 12.
- Access token JWT con TTL de 15 minutos.
- Refresh token rotado en cada uso, almacenado como hash SHA-256 en PostgreSQL y en Redis para revocación rápida.
- Refresh token TTL: 168 horas (7 días).

### 6.3. Módulo Academic

**Endpoints docente:**

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/docente/cursos` | Lista cursos del docente autenticado. |
| POST | `/docente/cursos` | Crea un nuevo curso. |
| GET | `/docente/cursos/:id` | Detalle de un curso. |
| PUT | `/docente/cursos/:id` | Actualiza curso. |
| DELETE | `/docente/cursos/:id` | Desactiva curso. |
| GET | `/docente/cursos/:id/actividades` | Lista actividades. |
| POST | `/docente/cursos/:id/actividades` | Crea actividad. |
| PUT | `/docente/cursos/:id/actividades/:aid` | Actualiza actividad. |
| DELETE | `/docente/cursos/:id/actividades/:aid` | Elimina actividad. |
| POST | `/docente/cursos/:id/actividades/:aid/codocentes` | Otorga acceso co-docente. |
| DELETE | `/docente/cursos/:id/actividades/:aid/codocentes/:uid` | Revoca acceso co-docente. |

**Endpoints estudiante:**

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/student/cursos` | Lista cursos del estudiante (inscripciones activas). |
| GET | `/student/cursos/:id/actividades` | Lista actividades del curso. |
| GET | `/student/cursos/:id/actividades/:aid` | Detalle de actividad con estado de entrega. |

**Endpoints admin:**

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET/POST | `/admin/carreras` | CRUD de carreras. |
| GET/PUT/DELETE | `/admin/carreras/:id` | Operaciones sobre carrera. |
| GET/POST | `/admin/materias` | CRUD de materias. |
| GET/PUT/DELETE | `/admin/materias/:id` | Operaciones sobre materia. |
| GET/POST | `/admin/student-profiles` | CRUD de perfiles de estudiantes. |
| GET/PUT/DELETE | `/admin/student-profiles/:id` | Operaciones sobre perfil. |

### 6.4. Módulo Submission

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/student/actividades/:aid/submit` | Sube archivo de entrega (multipart). |
| GET | `/student/actividades/:aid/submission` | Obtiene entrega activa del estudiante. |
| GET | `/docente/cursos/:cid/actividades/:aid/submissions` | Lista todas las entregas. |
| GET | `/docente/cursos/:cid/actividades/:aid/submissions/:sid` | Detalle de entrega. |
| GET | `/docente/cursos/:cid/actividades/:aid/submissions/:sid/download` | Descarga archivo. |
| POST | `/docente/cursos/:cid/actividades/:aid/submissions/:sid/grade` | Califica entrega. |

### 6.5. Módulo Tracing

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/tracing/sessions` | Abre sesión de examen. |
| POST | `/tracing/sessions/:sid/events` | Envía lote de eventos de trace. |
| POST | `/tracing/sessions/:sid/heartbeat` | Heartbeat periódico (cada 30s). |
| POST | `/tracing/sessions/:sid/close` | Cierra la sesión. |
| GET | `/docente/cursos/:cid/actividades/:aid/sessions` | Lista sesiones. |
| GET | `/docente/cursos/:cid/actividades/:aid/sessions/:sid/trace` | Traza completa de una sesión. |
| WS | `/ws/sessions/:sid/live` | Feed WebSocket en tiempo real para el docente. |

### 6.6. Módulo RAG

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/rag/curricular/ingest` | Sube documento curricular (multipart: `file`, `carrera`, `semestre`, `codigo_materia`, `is_rubric`). |
| POST | `/rag/docente/ingest` | Sube documento del docente (multipart: `file`, `curso_id`, `is_rubric`). |
| GET | `/rag/curricular/retrieve` | Recupera chunks similares a un query. |
| GET | `/rag/docente/retrieve` | Recupera chunks del pipeline docente. |
| POST | `/rag/curricular/solve` | Analiza una problemática con IA y devuelve materias relevantes + sugerencia. |

**Flujo de ingest:**
1. Extracción de texto del archivo (PDF → pdfcpu, DOCX → XML descomprimido, TXT → directo).
2. Tokenización y chunking (~512 tokens, overlap 64).
3. Solicitud de embeddings a Ollama (`nomic-embed-text`, dimensión 1536).
4. Almacenamiento del archivo original en MinIO.
5. Inserción de chunks en estado `pending`.
6. Swap atómico `pending → active` en transacción (reemplaza chunks anteriores si es re-ingest).

**Flujo de solve (Resolver Problemáticas):**
1. Genera embedding del texto del problema.
2. Consulta `QuerySimilarAll` — búsqueda por coseno sobre todos los chunks sin filtro de materia.
3. Agrupa por materia, mantiene el chunk más relevante por materia.
4. Filtra las que tienen similitud < 0.2.
5. Toma el top-5 y construye un prompt en español para el LLM.
6. Devuelve `SolveProblemResponse` con `suggestion` y array de `materias` con `relevance_score`.

### 6.7. Módulo Detection

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/docente/cursos/:cid/actividades/:aid/submissions/:subid/analyze` | Ejecuta análisis heurístico + LLM sobre la sesión asociada. |
| GET | `/docente/cursos/:cid/actividades/:aid/submissions/:subid/detection-report` | Obtiene el reporte de detección. |

**Análisis heurístico:** Suma ponderada de eventos sospechosos:
- Cambio de pestaña: peso 5
- Copia al portapapeles: peso 3
- Brecha de heartbeat (>90s): peso 2

El score resultante se clasifica: `< 60` = normal, `≥ 60` = flag alto.

**Análisis LLM:** Genera un advisory en lenguaje natural basado en el contexto de la sesión y los flags del análisis heurístico. Usa el router IA configurado.

### 6.8. Módulo Projects

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/docente/cursos/:cid/actividades/:aid/stages` | Crea etapa de investigación. |
| GET | `/docente/cursos/:cid/actividades/:aid/stages` | Lista etapas. |
| POST | `/student/actividades/:aid/stages/:stid/submit` | Entrega archivo a etapa. |
| POST | `/student/actividades/:aid/git` | Registra URL de repositorio Git. |
| POST | `/docente/cursos/:cid/actividades/:aid/git/:gid/analyze` | Dispara análisis Git asíncrono. |
| GET | `/docente/cursos/:cid/actividades/:aid/submissions/:subid/git-analysis` | Resultado del análisis Git. |
| GET/POST | `/docente/cursos/:cid/actividades/:aid/defensa-slots` | Gestiona turnos de defensa oral. |

### 6.9. Módulo Admin

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET/POST | `/admin/docentes` | CRUD de cuentas docente. |
| GET/PUT/DELETE | `/admin/docentes/:id` | Operaciones sobre docente. |
| GET/POST | `/admin/format-templates` | CRUD de plantillas de formato. |
| GET/PUT/DELETE | `/admin/format-templates/:id` | Operaciones sobre plantilla. |
| POST | `/actividades/:aid/format-template` | Aplica plantilla a actividad. |
| GET | `/admin/curriculum/documents` | Lista documentos curriculares indexados. |
| DELETE | `/admin/curriculum/documents/:id` | Elimina documento e índice asociado de MinIO. |
| POST | `/admin/actividades/:aid/interdisciplinary` | Define scopes interdisciplinarios. |
| DELETE | `/admin/actividades/:aid/interdisciplinary` | Elimina scopes. |

### 6.10. Módulo Appeals

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/appeals` | Crea apelación (estudiante). Body: `{ session_id, type, reason }`. |
| GET | `/appeals?session_id=:id` | Consulta apelación por sesión. |
| PUT | `/appeals/:id/review` | Docente revisa: `{ decision, justification }`. |
| PUT | `/appeals/:id/escalate` | Escala apelación a admin. |
| PUT | `/appeals/:id/admin-review` | Admin resuelve: `{ decision, justification }`. |
| GET | `/actividades/:aid/appeals` | Lista apelaciones de una actividad (docente/admin). |

**Flujo de estados:**
```
pending → (docente revisa) → resolved | escalated
escalated → (admin revisa) → admin_resolved
```

**Auto-escalación:** Si el tipo es `grade_dispute`, la apelación se escala automáticamente al admin sin pasar por el docente.

### 6.11. Módulo Alerts

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/docente/alertas` | Lista alertas del docente autenticado. |
| PATCH | `/docente/alertas/:id/read` | Marca alerta como leída. |

### 6.12. Módulo Analytics

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/docente/cursos/:cid/actividades/:aid/analytics` | Analytics de actividad: distribución de notas, tasa de entrega, detecciones. |
| GET | `/docente/cursos/:cid/analytics` | Analytics de curso: progreso por estudiante, resumen de actividades. |

---

## 7. Frontend — Estructura y Páginas

### 7.1. Estructura de Directorios

```
nextlab-web/src/
├── api/              # Funciones de llamada a la API (una por dominio)
│   ├── academic.ts   # Cursos, actividades, inscripciones
│   ├── admin.ts      # Endpoints de admin
│   ├── analytics.ts  # Analytics
│   ├── appeals.ts    # Apelaciones
│   ├── auth.ts       # Login, refresh, logout
│   ├── gitAnalysis.ts# Análisis Git
│   ├── grading.ts    # Calificaciones
│   ├── rag.ts        # Ingest, listado de documentos, solve
│   ├── student.ts    # Endpoints estudiante
│   ├── submissions.ts# Entregas
│   └── tracing.ts    # Sesiones, trace events
├── components/
│   ├── common/       # Layout, Sidebar, Topbar, LoadingSpinner, ErrorMessage, ProtectedRoute
│   ├── docente/      # CourseCard, GradePanel, SubmissionTable, TraceEventTable, etc.
│   └── student/      # ActivityCard, FileUpload, GradeDisplay, SessionPanel, TimerDisplay, etc.
├── context/
│   └── AuthContext.tsx  # Proveedor de autenticación global
├── hooks/
│   └── useTracingSession.ts  # Hook WebSocket para sesiones de examen en vivo
├── lib/
│   ├── api.ts        # Instancia Axios configurada (baseURL, interceptores JWT)
│   └── auth.ts       # Helpers de token en memoria
├── pages/
│   ├── admin/        # 6 páginas
│   ├── auth/         # LoginPage
│   ├── codocente/    # 5 páginas
│   ├── docente/      # 16 páginas
│   └── student/      # 3 páginas
├── router/index.tsx  # Rutas con ProtectedRoute por rol
└── types/
    ├── academic.ts   # Tipos del dominio académico
    └── index.ts      # ApiError, Role, User
```

### 7.2. Convenciones del Frontend

- **Todas las llamadas HTTP** pasan por `src/lib/api.ts` (instancia Axios). Sin `fetch()` nativo.
- **Manejo de errores**: cada archivo `src/api/*.ts` tiene una función local `toApiError` que convierte errores Axios en `ApiError(message, code)`.
- **Exports nombrados** en todos los componentes y páginas. Sin exports default.
- **Sin librerías de componentes externas**. Todo en Tailwind CSS v4.
- **Paleta de colores**:

| Variable | Valor | Uso |
|----------|-------|-----|
| Fondo principal | `#111111` | `<body>` |
| Superficie | `#1a1a1a` | Cards, paneles |
| Borde | `#2e2e2e` | Separadores |
| Acento | `#E85D24` | Naranja NextLab |
| Texto principal | `#f0f0f0` | Títulos, valores |
| Texto secundario | `#888888` | Labels, descripciones |

### 7.3. Páginas por Rol

#### 7.3.1. Autenticación

| Página | Ruta | Descripción |
|--------|------|-------------|
| `LoginPage` | `/login` | Formulario email/password + panel de acceso rápido con usuarios de prueba por rol. |

#### 7.3.2. Administrador

| Página | Ruta | Descripción |
|--------|------|-------------|
| `DashboardPage` | `/admin/dashboard` | Panel con 5 tarjetas de navegación. |
| `CarrerasPage` | `/admin/carreras` | CRUD de carreras. |
| `MateriasPage` | `/admin/materias` | CRUD de materias (filtradas por carrera). |
| `StudentProfilesPage` | `/admin/student-profiles` | CRUD de perfiles de estudiantes. |
| `ProblematicaPage` | `/admin/problematica` | Interfaz de consulta RAG: textarea → sugerencia IA + materias relevantes con barra de relevancia. |
| `CurriculumPage` | `/admin/curriculum` | Subida de documentos curriculares (PDF/DOCX/TXT) con metadatos + listado + eliminación. |

#### 7.3.3. Docente

| Página | Ruta | Descripción |
|--------|------|-------------|
| `DashboardPage` | `/docente/dashboard` | Lista de cursos activos del docente. |
| `CreateCursoPage` | `/docente/cursos/new` | Formulario de creación de curso. |
| `EditCursoPage` | `/docente/cursos/:id/edit` | Edición de curso. |
| `CursoDetailPage` | `/docente/cursos/:id` | Detalle de curso con lista de actividades. |
| `CreateActividadPage` | `/docente/cursos/:id/actividades/new` | Formulario de creación de actividad con selector de tipo. |
| `EditActividadPage` | `/docente/cursos/:id/actividades/:aid/edit` | Edición de actividad. |
| `ActividadDetailPage` | `/docente/cursos/:id/actividades/:aid` | Detalle de actividad. |
| `SubmissionsPage` | `/docente/cursos/:cid/actividades/:aid/submissions` | Lista de entregas con badge de apelaciones pendientes. |
| `SubmissionDetailPage` | `/docente/cursos/:cid/actividades/:aid/submissions/:sid` | Detalle de entrega + panel de calificación + gestión de apelación. |
| `TraceViewerPage` | `/docente/cursos/:cid/actividades/:aid/submissions/:subid/trace` | Visualizador de eventos de trace. |
| `LiveSessionsPage` | `/docente/cursos/:cid/actividades/:aid/sessions` | Lista de sesiones activas/completadas. |
| `LiveSessionFeedPage` | `/docente/cursos/:cid/actividades/:aid/sessions/:sid/live` | Feed en tiempo real vía WebSocket. |
| `DefensaSchedulingPage` | `/docente/cursos/:cid/actividades/:aid/defensa-slots` | Gestión de turnos de defensa oral. |
| `GitAnalysisPage` | `/docente/cursos/:cid/actividades/:aid/submissions/:subid/git-analysis` | Resultado de análisis Git. |
| `AIReportPage` | `/docente/cursos/:cid/actividades/:aid/submissions/:subid/ai-report` | Reporte de análisis LLM. |
| `DetectionReportPage` | `/docente/cursos/:cid/actividades/:aid/submissions/:subid/detection-report` | Reporte heurístico de integridad. |
| `AlertsPage` | `/docente/alertas` | Centro de alertas del docente. |
| `ActivityAnalyticsPage` | `/docente/cursos/:cid/actividades/:aid/analytics` | Analytics de actividad. |
| `CourseAnalyticsPage` | `/docente/cursos/:cid/analytics` | Analytics de curso. |

#### 7.3.4. Co-Docente

| Página | Ruta | Descripción |
|--------|------|-------------|
| `DashboardPage` | `/codocente/dashboard` | Lista de actividades con acceso co-docente. |
| `SubmissionsPage` | `/codocente/cursos/:cid/actividades/:aid/submissions` | Lista de entregas (vista co-docente). |
| `LiveSessionsPage` | `/codocente/cursos/:cid/actividades/:aid/sessions` | Sesiones en vivo (sólo lectura). |
| `LiveSessionFeedPage` | `/codocente/cursos/:cid/actividades/:aid/sessions/:sid/live` | Feed en tiempo real (sólo lectura). |
| `TraceViewerPage` | `/codocente/cursos/:cid/actividades/:aid/submissions/:subid/trace` | Visualizador de traza. |

#### 7.3.5. Estudiante

| Página | Ruta | Descripción |
|--------|------|-------------|
| `DashboardPage` | `/student/dashboard` | Lista de cursos inscritos. |
| `ActividadesPage` | `/student/cursos/:id/actividades` | Lista de actividades del curso. |
| `ActividadDetailPage` | `/student/cursos/:id/actividades/:aid` | Detalle de actividad: subida de archivo, estado de entrega, calificación, formulario de apelación. |

### 7.4. Routing y Protección de Rutas

El router (`src/router/index.tsx`) usa `ProtectedRoute` que verifica el rol del usuario autenticado. Cada grupo de rutas está envuelto en su propio `ProtectedRoute` con `allowedRoles`.

```tsx
// Ejemplo de protección por rol
{ element: <ProtectedRoute allowedRoles={['admin']} />,
  children: [{ element: <Layout />, children: [...rutas admin] }] }
```

La redirección raíz (`/`) lleva al dashboard del rol correspondiente, o a `/login` si no hay sesión.

---

## 8. Autenticación y Roles

### 8.1. Flujo de Login

```
1. POST /auth/login { email, password }
2. API verifica bcrypt
3. Genera access_token (JWT, 15m) + refresh_token (UUID, 168h)
4. Almacena hash del refresh_token en PostgreSQL y en Redis
5. Frontend guarda access_token en memoria (variable JS)
6. Frontend usa access_token en header Authorization: Bearer <token>
7. Al expirar, POST /auth/refresh con el refresh_token rota el par
```

### 8.2. Roles y Permisos

| Rol | Acceso |
|-----|--------|
| `admin` | Panel administrativo completo: carreras, materias, docentes, templates, currículo, problemáticas |
| `docente` | Sus propios cursos y actividades, calificaciones, trazas, análisis, defensas, analytics |
| `co_docente` | Actividades donde fue otorgado acceso, trazas y sesiones en modo lectura |
| `estudiante` | Sus cursos inscritos, entregas, sesiones de examen, apelaciones |

### 8.3. Middleware

El middleware `auth.AuthMiddleware(cfg)` valida el JWT en el header `Authorization`. El middleware `auth.RoleGuard` verifica que el rol del token esté en la lista de roles permitidos para cada endpoint.

---

## 9. Sistema RAG e Inteligencia Artificial

### 9.1. Router IA

El módulo `internal/ai` implementa un router de tres modos:
- **`local`**: envía todas las peticiones a Ollama (localhost:11434).
- **`cloud`**: envía al endpoint cloud configurado (ej. Vercel AI Gateway).
- **`auto`**: intenta local primero; si falla o supera el timeout, hace fallback a cloud.

Todos los cambios de modo quedan registrados en `router_configs` (append-only).

### 9.2. Pipeline de Embeddings

- Modelo por defecto: `nomic-embed-text` (Ollama).
- Dimensión: 1536.
- Chunking: 512 tokens con overlap de 64 tokens.
- Índice: `ivfflat` con operador `vector_cosine_ops` (10 listas).

### 9.3. Funcionalidades IA por Módulo

| Módulo | Función IA |
|--------|-----------|
| RAG Curricular | Indexa documentos curriculares y los hace consultables por similitud semántica. |
| RAG Docente | Indexa documentos del docente para contexto de actividades. |
| Solve Problemática | Dado un problema, encuentra materias relevantes y genera una sugerencia pedagógica. |
| Detection Engine | Genera advisory en lenguaje natural sobre posibles irregularidades en el examen. |
| Git Analysis | Genera advisory sobre los patrones del repositorio Git del estudiante. |

---

## 10. Motor de Detección de Integridad

### 10.1. Análisis Heurístico

Procesa los eventos de la sesión de examen y calcula un score ponderado:

| Evento | Peso |
|--------|------|
| Cambio de pestaña (`tab_blur`) | 5 |
| Copia al portapapeles (`clipboard_copy`) | 3 |
| Brecha de heartbeat > 90 segundos | 2 |

**Umbral:** Score ≥ 60 → sesión marcada como sospechosa.

Los parámetros son configurables por variables de entorno (`HEURISTIC_TAB_WEIGHT`, etc.) para que cada institución pueda calibrar los falsos positivos.

### 10.2. Análisis LLM

Usando los resultados heurísticos como contexto, el LLM genera:
- Texto advisory descriptivo en español.
- Rango de calificación sugerido (e.g., "60–75").

El docente puede aceptar el rango sugerido o sobreescribirlo con una razón documentada.

### 10.3. Integridad de la Traza

Cada evento de trace tiene un `event_hash` calculado como hash de la cadena `(prev_hash || event_type || sequence_num || received_at)`. Cualquier modificación de un evento rompe la cadena y puede detectarse.

---

## 11. Sistema de Apelaciones

### 11.1. Flujo Completo

```
Estudiante
  └─ POST /appeals { session_id, type: 'grade_dispute', reason }
       │
       ├─ Si type == 'grade_dispute'
       │    └─ Status: escalated (va directo a admin)
       │
       └─ Si type == 'analysis_dispute'
            └─ Status: pending (espera revisión del docente)
                 │
                 └─ Docente PUT /appeals/:id/review { decision, justification }
                      ├─ Si decision == 'upheld' → Status: resolved
                      └─ Si decision == 'rejected'
                           └─ Estudiante puede solicitar escalación
                                └─ Admin PUT /appeals/:id/admin-review
                                     └─ Status: admin_resolved
```

### 11.2. Auditoría Inmutable

Cada cambio de estado en una apelación genera un `audit_event` con:
- `sequence`: número de orden dentro de la apelación.
- `event_hash`: hash encadenado con el evento anterior (`prev_hash`).
- `payload`: datos del evento (quién, qué decisión, cuándo).

Este log no puede modificarse y sirve como evidencia forense del proceso.

---

## 12. Guía de Ejecución

### 12.1. Requisitos Previos

- Docker Desktop instalado y corriendo.
- Node.js 20+ (para el frontend).
- Go 1.25+ (sólo si se desarrolla nativamente sin Docker).

### 12.2. Primera Ejecución

```bash
# 1. Clonar o ubicarse en el directorio del proyecto
cd INTEGRADOR

# 2. Levantar toda la infraestructura + API (compila y arranca)
cd nextlab-api
docker compose up --build -d

# 3. Verificar que la API esté sana
curl http://localhost:8080/health
# Esperado: {"status":"ok","dependencies":{"cache":...,"database":...,"storage":...}}

# 4. (Opcional) Correr el seed de usuarios de prueba
docker exec -i nextlab-api-postgres-1 psql -U postgres -d nextlab < scripts/seed-test-users.sql

# 5. Levantar el frontend
cd ../nextlab-web
npm install
npm run dev
# Abre http://localhost:5173 (o 5174 si 5173 está ocupado)
```

### 12.3. Ejecución Cotidiana

```bash
# Todo ya está corriendo gracias a restart: unless-stopped
# Solo es necesario iniciar el frontend:
cd nextlab-web && npm run dev
```

### 12.4. Aplicar Cambios en el Código Go

```bash
cd nextlab-api
docker compose up --build -d
# Docker recompila la imagen y reinicia sólo el contenedor api/worker
```

### 12.5. Ver Logs

```bash
# API en tiempo real
docker compose logs -f api

# Todos los servicios
docker compose logs -f
```

---

## 13. Variables de Entorno

El archivo `nextlab-api/.env` configura la ejecución nativa. Para Docker, las variables están definidas directamente en `docker-compose.yml`.

| Variable | Descripción | Valor por defecto (dev) |
|----------|-------------|------------------------|
| `DATABASE_URL` | Cadena de conexión PostgreSQL | `postgres://postgres:postgres@localhost:5433/nextlab?sslmode=disable` |
| `REDIS_ADDR` | Host:puerto de Redis | `localhost:6379` |
| `MINIO_ENDPOINT` | Endpoint de MinIO | `localhost:9000` |
| `MINIO_ACCESS_KEY` | Access key MinIO | `minioadmin` |
| `MINIO_SECRET_KEY` | Secret key MinIO | `minioadmin` |
| `MINIO_BUCKET` | Nombre del bucket | `nextlab` |
| `MINIO_USE_SSL` | Usar SSL con MinIO | `false` |
| `JWT_SECRET` | Clave de firma JWT (≥32 chars) | `nextlab-dev-secret-key-change-in-production-env` |
| `ACCESS_TOKEN_TTL` | Duración del access token | `15m` |
| `REFRESH_TOKEN_TTL` | Duración del refresh token | `168h` |
| `API_PORT` | Puerto del servidor HTTP | `8080` |
| `LOG_LEVEL` | Nivel de log (`debug`,`info`,`warn`,`error`) | `info` |
| `APP_ENV` | Entorno (`development`, `production`) | `development` |
| `GOOGLE_CLIENT_ID` | Client ID OAuth2 Google | placeholder dev |
| `GOOGLE_CLIENT_SECRET` | Client secret OAuth2 Google | placeholder dev |
| `GOOGLE_REDIRECT_URL` | Callback URL OAuth2 | `http://localhost:8080/auth/google/callback` |
| `AI_PRIMARY_URL` | URL del backend LLM primario | `http://localhost:11434` |
| `AI_PRIMARY_MODEL` | Modelo LLM primario | `llama3` |
| `AI_PRIMARY_TIMEOUT_SECS` | Timeout LLM (segundos) | `30` |
| `AI_CLOUD_URL` | URL gateway cloud IA (opcional) | vacío |
| `AI_CLOUD_API_KEY` | API key cloud IA (opcional) | vacío |
| `AI_EMBED_URL` | URL del servicio de embeddings | `http://localhost:11434` |
| `AI_EMBED_MODEL` | Modelo de embeddings | `nomic-embed-text` |
| `AI_EMBED_DIMENSION` | Dimensión del vector | `1536` |
| `RAG_CHUNK_TOKENS` | Tokens por chunk | `512` |
| `RAG_CHUNK_OVERLAP` | Tokens de overlap entre chunks | `64` |
| `RAG_TOKEN_BUDGET` | Presupuesto de tokens para contexto RAG | `2048` |
| `HEURISTIC_TAB_WEIGHT` | Peso de eventos tab_blur | `5` |
| `HEURISTIC_CLIP_WEIGHT` | Peso de eventos clipboard_copy | `3` |
| `HEURISTIC_GAP_WEIGHT` | Peso de brechas de heartbeat | `2` |
| `HEURISTIC_HIGH_THRESHOLD` | Umbral de score alto | `60` |

---

## 14. Usuarios de Prueba

Los siguientes usuarios están disponibles en el entorno de desarrollo (seeded en `scripts/seed-test-users.sql`). Todos usan la contraseña `test1234`.

| Rol | Email | Contraseña |
|-----|-------|------------|
| Administrador | `admin@nextlab.dev` | `test1234` |
| Docente | `docente@nextlab.dev` | `test1234` |
| Co-docente | `codocente@nextlab.dev` | `test1234` |
| Estudiante | `estudiante@nextlab.dev` | `test1234` |

El hash de contraseña almacenado es bcrypt con costo 12:
```
$2a$12$pacRbXnEKqulywfzXgSABujKGGBiW/0EJJsNBQM096oSLRrw9YxW6
```

El seed también crea los datos académicos mínimos para que el estudiante tenga algo con qué interactuar:
- Carrera: `INF` — Ingeniería de Sistemas Informáticos
- Materia: `INF-301` — Seminario Integrador
- Curso: semestre `2024-II`, sección `A`
- Inscripción del usuario estudiante al curso
- Actividad de tipo `examen` con fecha límite futura

El acceso rápido a estos usuarios está disponible en el panel de la página de login (botones "Admin", "Docente", "Co-docente", "Estudiante") que completan y envían el formulario automáticamente.

---

*Documento generado automáticamente como parte del Proyecto Integrador — Universidad Unifranz — 2026.*
