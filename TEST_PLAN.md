# Plan de pruebas integral — NextLab

| Campo | Valor |
|---|---|
| Sistema | NextLab (`nextlab-api` + `nextlab-web`) |
| Versión bajo prueba | Estado actual del repositorio |
| Fecha | 18 de noviembre de 2026 |
| Propósito | Verificar que la plataforma gestione el ciclo académico y la integridad de las evaluaciones de forma correcta, segura, auditable y usable. |
| Fuentes analizadas | Código Go/Fiber, React/Vite, migraciones 000001–000017, `integrador-elder_Vfinal.md`, `nextlab-api/TESTING.md`, pruebas y script existentes. |

## 1. Sistema cubierto

NextLab es una plataforma universitaria con cuatro roles: **administrador**, **docente**, **co-docente** y **estudiante**. La solución es un monolito modular: cliente React/Vite; API y worker en Go/Fiber; PostgreSQL/pgvector, Redis, MinIO y un proveedor de IA/embeddings.

El plan cubre estos dominios reales del repositorio:

1. Autenticación, recuperación de acceso, OAuth y autorización por rol.
2. Administración académica: carreras, materias, perfiles, docentes y cursos.
3. Cursos, grupos, inscripciones, actividades y permisos de co-docencia.
4. Portal de estudiante y entregas de archivo o respuestas de examen.
5. Sesiones de evaluación, trazabilidad, WebSocket y buffer sin conexión.
6. Worker, validación de formato, detección heurística/IA, calificación e informes.
7. RAG curricular y docente, configuración y router de IA.
8. Investigación por etapas, repositorios Git y defensas orales.
9. Alertas, notificaciones, analíticas y exportaciones PDF.
10. Plantillas de formato, alcance interdisciplinario, webhooks y reentrenamiento ML.
11. Apelaciones y cadena de evidencia/auditoría.
12. Calidad del frontend, observabilidad, seguridad, rendimiento y recuperación.

Quedan fuera de la ejecución automatizada local los proveedores que exigen una cuenta o sistema externo real: Google OAuth, repositorio Git remoto y receptor de webhooks. No quedan fuera de la cobertura: se validarán mediante dobles de prueba y luego en una prueba de aceptación controlada con credenciales de QA.

## 2. Objetivos y riesgos

### Objetivos de calidad

- Ningún usuario puede consultar, alterar o descargar información fuera de sus permisos.
- Las actividades, entregas, notas, plazos y estados conservan sus reglas de negocio.
- Los eventos de una sesión son ordenados, íntegros y recuperables ante desconexión.
- El procesamiento asíncrono llega a un resultado observable o a un error recuperable, sin perder archivos ni trabajos.
- Las conclusiones de detección son trazables y no sustituyen indebidamente la decisión docente.
- Las apelaciones mantienen evidencia verificable y sus transiciones están autorizadas.
- La interfaz presenta correctamente carga, error, sesión expirada y estados vacíos en los cuatro roles.

### Riesgos que determinan la prioridad

| Riesgo | Impacto | Mitigación de prueba |
|---|---:|---|
| Acceso horizontal o entre roles a cursos, archivos, sesiones y apelaciones | Crítico | Matriz de autorización para cada endpoint y flujo UI. |
| Pérdida/manipulación de trazas o eventos duplicados tras reconexión | Crítico | Pruebas WS, lotes offline, orden, hash y reintentos. |
| Entrega o análisis asíncrono incompleto | Crítico | E2E con MinIO/worker, reintento e inspección de estados. |
| Regresiones de contrato entre APIs del frontend y rutas Go | Alto | Contratos automatizados contra API real; no sólo mocks de Axios. |
| Detección/IA no determinista o indisponible | Alto | Doble determinista, timeout, fallback, rate-limit y mensajes al usuario. |
| Divulgación de datos académicos en archivos, PDF, RAG o webhook | Alto | Pruebas de pertenencia, descarga, aislamiento y secretos. |
| Datos o estados inválidos por operaciones concurrentes | Alto | Pruebas de doble envío, concurrencia y restricciones de BD. |

## 3. Estrategia y niveles

| Nivel | Qué valida | Herramientas/forma |
|---|---|---|
| Unitario backend | Reglas puras: validación de formato, heurística, hashes, cambios de actividad, estados y validadores. | `go test ./...`; tablas de casos y mocks de repositorio/HTTP. |
| Unitario frontend | Componentes, hooks de autenticación, interceptor de refresh, formularios y buffer offline. | Añadir Vitest + React Testing Library + MSW. |
| Integración | API + PostgreSQL/Redis/MinIO y repositorios reales; migraciones, restricciones y autorización. | Pruebas Go con etiqueta `integration`, base aislada. |
| Contrato | URL, método, parámetros, cuerpos, estados HTTP y esquema entre cliente y API. | OpenAPI versionado o colección de contratos ejecutada contra la API. |
| Sistema/E2E | Recorridos de usuario atravesando navegador, API, worker y servicios. | Playwright; `scripts/test-system.sh` para la ruta de backend aislada. |
| No funcional | Seguridad, rendimiento, accesibilidad, compatibilidad y resiliencia. | OWASP ZAP/escáner autenticado, k6, axe-core y pruebas de fallos. |

Se seguirá una pirámide de pruebas: reglas y contratos en cada cambio; integración en cada pull request; smoke E2E en cada despliegue QA; regresión completa, seguridad y carga antes de liberar.

## 4. Entorno, datos y evidencias

### Entorno QA aislado

- Docker Compose con PostgreSQL 16 + pgvector, Redis 7, MinIO, API, worker y `test-ai` determinista.
- Usar `nextlab-api/docker-compose.test.yml`, que publica la API en `http://localhost:18080` y crea recursos propios `nextlab-system-test`.
- El navegador apuntará a esa API mediante `VITE_API_URL`; no se usarán servicios, buckets, usuarios ni claves de producción.
- Go 1.26.1 (la versión declarada por el módulo) y pnpm deben estar disponibles en el ejecutor. El entorno de elaboración de este plan no contiene el binario `go`; por tanto no se han ejecutado las pruebas Go aquí.

### Datos mínimos de QA

| Alias | Rol | Uso |
|---|---|---|
| `admin.qa` | admin | Catálogos, docentes, plantillas, IA, ML, auditoría final. |
| `docente.a.qa` / `docente.b.qa` | docente | Propietario y usuario no propietario para autorización horizontal. |
| `codocente.ok.qa` / `codocente.no.qa` | co_docente | Con y sin grant sobre la actividad. |
| `estudiante.a.qa` / `estudiante.b.qa` | estudiante | Inscrito/no inscrito; dueño/no dueño de entrega y sesión. |
| Archivos | — | PDF válido (1 y 3 páginas), PDF corrupto, DOCX, tipo no permitido, tamaño límite y sobrelímite. |
| IA/Git/webhook | — | Doble determinista, repo de QA pequeño y receptor HTTP de captura. |

Cada prueba debe guardar: ID, versión/commit, entorno, datos, pasos, resultado esperado/obtenido, estado, evidencia (captura, respuesta, log o consulta) y defecto vinculado. Los datos se reinicializan entre suites; nunca se ejecutan pruebas destructivas contra desarrollo o producción.

## 5. Catálogo funcional de casos

La prioridad P0 bloquea una liberación; P1 debe completarse antes de producción; P2 es regresión programada. Cada fila representa un caso parametrizable: se ejecutarán tanto casos válidos como límites y negativos cuando se indique.

### 5.1 Autenticación, sesión y control de acceso

| ID | Pri. | Caso y resultado esperado |
|---|---:|---|
| AUTH-01 | P0 | Login válido para cada rol entrega access/refresh token y redirige a su dashboard. |
| AUTH-02 | P0 | Credenciales inválidas, usuario inactivo y campos vacíos se rechazan sin filtrar información sensible. |
| AUTH-03 | P0 | Refresh rota el token; el refresh previo deja de ser utilizable; logout lo revoca y limpia el cliente. |
| AUTH-04 | P0 | Access token vencido causa un único refresh y reintenta la petición original; refresh inválido lleva a login. |
| AUTH-05 | P1 | Recuperación/restablecimiento: token de un uso, expiración, contraseña inválida y contraseña nueva válida. |
| AUTH-06 | P1 | Inicio Google: estado, callback, usuario existente/nuevo y cancelación; sólo contra cliente OAuth de QA. |
| AUTH-07 | P0 | Rutas protegidas y URL directa: anónimo va a login; usuario autenticado va a su inicio; ruta inexistente muestra 404. |
| AUTH-08 | P0 | Para cada endpoint, token ausente/malformado/vencido devuelve 401 y rol no autorizado 403. |
| AUTH-09 | P0 | Un docente/co-docente/estudiante ajeno no puede acceder por UUID a curso, actividad, archivo, sesión, reporte, calificación o apelación de otro. |

### 5.2 Administración y estructura académica

| ID | Pri. | Caso y resultado esperado |
|---|---:|---|
| ADM-01 | P0 | Admin crea, lista, edita y elimina carreras; valida código/nombre obligatorios y duplicados. |
| ADM-02 | P0 | Admin gestiona materias por carrera y semestre; no permite referencias inexistentes ni borrado con dependencias sin respuesta controlada. |
| ADM-03 | P0 | Admin crea, edita y desactiva perfiles de estudiante; perfil inactivo no habilita acceso/inscripción. |
| ADM-04 | P1 | Admin crea, lista, actualiza y elimina docentes; se validan credenciales iniciales y restricciones de borrado. |
| ACAD-01 | P0 | Docente crea/consulta/edita/elimina sólo sus cursos; se valida unicidad materia-docente-semestre-sección. |
| ACAD-02 | P0 | Grupo: CRUD, asignación y retiro de alumno; no duplicados, no alumno ajeno, no grupo de otro curso. |
| ACAD-03 | P0 | Inscripción automática/concurrente no duplica `(student_id, curso_id)` y presenta cursos correctos al estudiante. |
| ACAD-04 | P0 | Actividad: tipos permitidos, fecha, instrucciones, reglas de formato y `trace_enabled` según tipo. |
| ACAD-05 | P0 | Tras una entrega, bloquear cambio de título/tipo y borrado; permitir sólo los campos definidos como editables y registrar el cambio. |
| ACAD-06 | P0 | Grant/revocación permite al co-docente autorizado sólo las operaciones previstas; revocado o ajeno no accede. |

### 5.3 Entregas y calificación

| ID | Pri. | Caso y resultado esperado |
|---|---:|---|
| SUB-01 | P0 | Estudiante inscrito sube archivo permitido antes del plazo: crea una entrega, objeto MinIO y trabajo asíncrono. |
| SUB-02 | P0 | Rechazar archivo sin multipart, vacío, corrupto, extensión/MIME no permitido o que supera el límite; no deja objeto huérfano. |
| SUB-03 | P0 | Sustituir y eliminar una entrega actualiza objeto/metadatos correctamente; estudiante ajeno y actividad cerrada son rechazados. |
| SUB-04 | P0 | Fecha vencida marca `is_late` conforme a regla; se prueba zona horaria, instante exacto y cambio de plazo. |
| SUB-05 | P0 | Respuesta de examen se guarda y sólo es visible a dueño y personal autorizado. |
| SUB-06 | P0 | Descarga de estudiante/docente exige pertenencia y devuelve contenido, tipo y nombre seguros. |
| SUB-07 | P0 | Docente lista entregas de su actividad, detalle y cambios de configuración; filtros/paginación si existen no omiten ni duplican filas. |
| SUB-08 | P0 | Calificación: rango permitido, feedback, decisión única, override justificado y auditoría; doble envío no crea decisiones duplicadas. |

### 5.4 Sesión, trazabilidad y tiempo real

| ID | Pri. | Caso y resultado esperado |
|---|---:|---|
| TRC-01 | P0 | Sólo estudiante inscrito abre sesión de actividad trazable; se crea una sesión activa con hora de inicio. |
| TRC-02 | P0 | No se abren sesiones duplicadas; cierre normal, plazo vencido, abandono y timeout terminan en el estado correcto. |
| TRC-03 | P0 | WebSocket autentica, acepta eventos válidos, mantiene ping/pong y rechaza conexión/sesión ajena. |
| TRC-04 | P0 | Eventos de foco, pestaña, portapapeles, teclado y heartbeat se persisten en orden; contenido se limita/sanitiza según contrato. |
| TRC-05 | P0 | Lote offline preserva orden, soporta reintento idempotente, respeta máximo de buffer y no pierde eventos al recuperar conexión. |
| TRC-06 | P0 | Verificación de hash detecta evento alterado, eliminando o reordenado; una cadena válida se verifica correctamente. |
| TRC-07 | P0 | Docente dueño y co-docente con grant ven sesiones/eventos; otros docentes no. WS de observación no permite escribir. |
| TRC-08 | P1 | Feed en vivo reconecta con máximo de tres intentos/backoff y muestra error recuperable; no duplica listeners/eventos. |
| TRC-09 | P1 | Consulta de trazas paginada y exportada mantiene orden/total; sesión grande no agota memoria ni bloquea la UI. |

### 5.5 Worker, formato, detección e informes

| ID | Pri. | Caso y resultado esperado |
|---|---:|---|
| DET-01 | P0 | Worker consume una entrega una sola vez y genera reporte de formato/heurística; su reintento no duplica resultados. |
| DET-02 | P0 | Reglas de formato: páginas mín./máx., palabras, tipo y ausencia de reglas producen cada estado esperado. |
| DET-03 | P0 | PDF/DOCX válido, PDF de varias páginas, PDF inválido y documento no soportado generan resultado o error explícito y observable. |
| DET-04 | P0 | Heurística suma pesos de pestaña, portapapeles y huecos de heartbeat; límites/umbrales y secuencias temporales dan score y flags correctos. |
| DET-05 | P0 | Detección IA usa contexto permitido, timeout/fallback determinista y persiste tipo, score, categorías y recomendación sin exponer prompt sensible. |
| DET-06 | P0 | Reporte de detección y su PDF sólo están disponibles al personal autorizado; datos de sesión, formato y decisión son coherentes. |
| DET-07 | P1 | Falla de MinIO, IA o worker no deja trabajo bloqueado silenciosamente: se registra error, alerta y posibilidad de reintento. |
| DET-08 | P1 | Baseline/ML: sin etiquetas suficientes, el reentrenamiento se rechaza; con conjunto válido publica versión reproducible y conserva el modelo previo ante fallo. |

### 5.6 IA y RAG dual

| ID | Pri. | Caso y resultado esperado |
|---|---:|---|
| AI-01 | P0 | Router `/ai/complete` aplica autorización, modo local/auto/cloud, timeout, fallback y registra sólo metadatos permitidos. |
| AI-02 | P0 | Límite de tasa devuelve respuesta controlada con `retry_after`; contador se separa entre usuarios y expira correctamente. |
| AI-03 | P0 | Sólo admin cambia configuración IA; historial es append-only y refleja actor, fecha y cambio. |
| RAG-01 | P0 | Ingesta curricular valida archivo/metadatos, extrae, fragmenta, obtiene embedding y permite recuperar fragmentos relevantes. |
| RAG-02 | P0 | Ingesta/consulta docente queda aislada por docente/curso: otro docente o estudiante no recupera sus documentos. |
| RAG-03 | P0 | Dimensión de embedding inconsistente impide inicio/ingesta con diagnóstico claro; dimensión correcta permite consulta. |
| RAG-04 | P1 | Pregunta sin evidencia, documento vacío/corrupto, duplicado y eliminación curricular tienen respuesta segura y no dejan vectores/objetos huérfanos. |

### 5.7 Proyectos, Git y defensa oral

| ID | Pri. | Caso y resultado esperado |
|---|---:|---|
| PRJ-01 | P0 | Docente crea, edita, ordena y elimina etapas de investigación; estudiante consulta sólo las publicadas. |
| PRJ-02 | P0 | Estudiante carga avance en etapa abierta; se bloquea etapa inexistente/cerrada, doble entrega y actividad ajena. |
| PRJ-03 | P0 | Progreso por etapas y calificación muestran estado/nota coherentes a estudiante y docente. |
| GIT-01 | P0 | Entrega Git válida de repositorio QA se analiza y genera reporte; actualización respeta regla de entrega/plazo. |
| GIT-02 | P0 | Rechazar URL malformada, host no permitido, red privada/SSRF, repositorio gigante, timeout y contenido/symlink malicioso. |
| DEF-01 | P0 | Docente agenda/reagenda defensa de alumno inscrito, con fecha/lugar válidos; estudiante consulta sólo su turno. |
| DEF-02 | P0 | Calificación de defensa valida rango, autorización y unicidad; cambios posteriores quedan auditados. |

### 5.8 Alertas, analíticas, administración avanzada y apelaciones

| ID | Pri. | Caso y resultado esperado |
|---|---:|---|
| OBS-01 | P1 | Alerta por riesgo/fallo llega al docente dueño por lista/SSE; token, reconexión y `Last-Event-ID` no filtran alertas ajenas. |
| OBS-02 | P1 | Notificación de estudiante se lista, marca como leída y se transmite por stream; no se duplica con reconexión. |
| OBS-03 | P0 | Métricas por actividad/curso/admin usan sólo datos autorizados y los totales, percentiles y filtros coinciden con datos fuente. |
| OBS-04 | P1 | Reporte PDF de actividad/detección/auditoría se genera con datos correctos y manejo de fallo de generación. |
| ADM-05 | P1 | Plantillas de formato: CRUD admin, aplicación por docente autorizado y reglas efectivas en la actividad. |
| ADM-06 | P1 | Ámbitos interdisciplinarios: admin configura/elimina y sólo actores permitidos los consultan. |
| ADM-07 | P1 | Webhook: CRUD/admin, prueba, firma/secretos, reintento/fallo y payload sin información indebida. |
| APL-01 | P0 | Estudiante dueño crea apelación válida; duplicado/estado no apelable/sesión ajena se rechazan. |
| APL-02 | P0 | Docente autorizado revisa y resuelve; transiciones inválidas, decisor ajeno y justificación vacía se rechazan. |
| APL-03 | P0 | Escalamiento y decisión admin respetan flujo y notifican a los destinatarios correctos. |
| APL-04 | P0 | Evidencia/auditoría tiene secuencia y hash encadenado válidos; modificación simulada se detecta. |
| APL-05 | P0 | Consulta/lista/PDF de apelación aplica pertenencia y no muestra evidencia de otros estudiantes. |

## 6. Recorridos E2E obligatorios

| ID | Pri. | Recorrido de aceptación |
|---|---:|---|
| E2E-01 | P0 | Admin configura carrera, materia y perfil; docente crea curso/actividad; estudiante inscrito la ve en el portal. |
| E2E-02 | P0 | Docente crea examen con formato; estudiante abre sesión, pierde/redescubre conexión, entrega PDF; worker analiza; docente ve traza, reporte, alerta, califica y estudiante ve resultado. |
| E2E-03 | P0 | Estudiante apela una decisión; docente la revisa/escalona; admin resuelve; estudiante recibe notificación y auditoría/PDF conserva la cadena válida. |
| E2E-04 | P1 | Docente crea investigación por etapas, estudiante entrega avance, docente califica y el progreso se refleja en ambos portales. |
| E2E-05 | P1 | Actividad Git: estudiante registra repo de QA, análisis termina o falla de manera controlada y docente consulta los reportes. |
| E2E-06 | P1 | Actividad de defensa: docente agenda, estudiante consulta turno, docente califica y se registra la decisión. |
| E2E-07 | P1 | Admin ingesta currículo y docente ingesta documento; cada consulta RAG retorna sólo conocimiento del ámbito autorizado. |

## 7. Pruebas no funcionales

### Seguridad (P0/P1)

- Ejecutar OWASP ASVS básico y ZAP autenticado: autenticación, control de acceso, inyección SQL/NoSQL, XSS, CSRF según el modelo de tokens, CORS, cabeceras, rate limiting y enumeración de recursos.
- Revisar que JWT no viaje en URL. En particular, probar que el token de WebSocket no quede en logs, historial, referer ni mensajes de error.
- Fuzz de entradas HTTP, JSON, UUID, fechas, `multipart`, PDF/DOCX y URLs Git; límite de tamaño y tiempo para prevenir agotamiento de CPU, memoria o disco.
- Verificar credenciales por entorno, secreto JWT de longitud adecuada, cifrado TLS en QA/producción y cifrado/política de acceso de objetos MinIO.
- Ejecutar análisis de dependencias y secretos en cada release; no publicar tokens, prompts, texto de portapapeles ni PII innecesaria en logs, alertas o webhooks.

### Rendimiento y resiliencia (P1)

| Escenario | Criterio inicial de aceptación |
|---|---|
| API CRUD autenticada | p95 menor a 500 ms con carga acordada de QA y error menor a 1 %. |
| Apertura de sesión / batch de trazas | p95 menor a 1 s; sin eventos duplicados o perdidos. |
| WebSocket en vivo | Conexión y primer evento menor a 2 s en red QA; reconexión conserva integridad. |
| Carga/descarga de archivo | Progreso/error comprensible; objeto y metadatos consistentes ante corte de red. |
| Worker | 100 % de trabajos de un lote de QA alcanzan terminado o fallido explícito antes del SLA acordado. |
| Recuperación | Reiniciar API/worker/Redis/MinIO durante una prueba no corrompe BD ni evidencia; trabajos se recuperan o quedan reintentables. |

Las cifras se ajustarán antes de producción con volumen esperado (usuarios concurrentes, tamaño máximo de archivos, eventos por examen y SLA). Usar k6 para HTTP/WS y una prueba de soak de varias horas para sesiones/worker.

### Frontend: accesibilidad y compatibilidad (P1)

- Navegación completa con teclado, foco visible, modal con foco atrapado, labels, contraste y mensajes de error anunciables.
- axe-core sin infracciones críticas en login, dashboards y formularios de creación/entrega/apelación.
- Chrome/Edge/Firefox actuales; Safari actual si es navegador soportado; anchos móvil, tableta y escritorio.
- Estados de carga, vacío, error de red, 401/403/404/429/500 y recarga de página conservan un comportamiento comprensible.

## 8. Automatización, ejecución y regresión

### Comandos base

```bash
# Backend: unitarias y compilación de todos los paquetes
cd nextlab-api
go test ./...

# Integración (requiere DATABASE_URL_TEST de una BD desechable)
go test -tags=integration ./tests/integration/...

# Prueba aislada API + worker + servicios; usa una IA doble local
./scripts/test-system.sh

# Cliente: controles estáticos y build de producción
cd ../nextlab-web
pnpm lint && pnpm build
```

La suite `test-system.sh` ya cubre un smoke valioso de autenticación/roles, gestión académica, trazabilidad, carga a MinIO, worker, formato/detección, IA, alertas, analíticas, investigación, defensa, plantillas, ML y apelaciones. Debe mantenerse como smoke y ampliarse con aserciones de contenido, no sólo códigos 2xx.

Para completar el plan se deben agregar:

- Vitest, React Testing Library y MSW para los componentes, `AuthContext`, interceptor de refresh y hooks de trazabilidad/offline.
- Playwright para los recorridos E2E de la sección 6, con almacenamiento de videos/traces/capturas en CI.
- Especificación OpenAPI o colección versionada generada desde rutas Go y pruebas de contrato que recorran todos los módulos API del cliente.
- Reportes JUnit/HTML, cobertura Go/TS y un tablero de defectos enlazado a los ID anteriores.

### Puertas de calidad

| Momento | Obligatorio |
|---|---|
| Pull request | Formateo/lint/build; unitarias afectadas; contrato afectado; sin hallazgos críticos. |
| Merge a QA | Unitarias completas, integración, smoke de `test-system.sh` y E2E P0. |
| Candidato a producción | Regresión P0/P1 aprobada, E2E completos, ZAP/dependencias, accesibilidad y rendimiento/resiliencia acordados. |
| Después de despliegue | Health check, smoke por rol, métricas/logs/alertas sin anomalías y plan de rollback probado. |

## 9. Criterios de entrada, salida y severidad

**Entrada:** requisitos y rutas congelados para el ciclo; entorno aislado sano; migraciones aplicadas; usuarios/datos QA creados; dobles de IA activos; build desplegable; defectos bloqueantes conocidos visibles.

**Salida:** 100 % de P0 ejecutados y aprobados; 100 % de P1 ejecutados salvo excepción aceptada; cero defectos críticos o altos abiertos; no hay regresión en autenticación, autorización, entrega, trazabilidad, auditoría ni integridad de datos; evidencias y reporte final publicados.

| Severidad | Definición | Decisión |
|---|---|---|
| Crítica | Exposición de datos, escalamiento de privilegios, corrupción/pérdida de evidencia o caída del flujo académico central. | Bloquea liberación. |
| Alta | Flujo P0 inaccesible, resultados/notas incorrectos o procesamiento asíncrono no confiable. | Bloquea liberación. |
| Media | Función importante tiene alternativa temporal o resultado visual/operativo incorrecto limitado. | Corregir antes de la siguiente versión acordada. |
| Baja | Cosmético, texto o caso extremo sin impacto funcional/seguridad. | Planificar. |

## 10. Estado de cobertura de partida y acciones inmediatas

| Área | Evidencia encontrada | Lectura para este plan |
|---|---|---|
| Backend unitario | Casos para reglas de formato, heurística, guardia de actividad y servicio de auth/academic. | Base útil, pero faltan módulos y casos de error/concurrencia. |
| Integración Go | Pruebas de base de datos para entregas/calificación, sesión y apelaciones. | No ejercitan el API HTTP completo ni el worker. |
| Sistema | `nextlab-api/scripts/test-system.sh` y stack Docker aislado. | Excelente smoke inicial; convertir resultados en aserciones de negocio y ejecutar en CI. |
| Frontend | `pnpm lint && pnpm build` ejecutados correctamente al elaborar el plan; Vite avisa un bundle JS de 502 kB sin comprimir. | No hay pruebas unitarias/componentes/E2E propias del cliente; añadirlas es prioridad alta. |
| Contrato | Cliente contiene llamadas API y backend registra rutas en módulos independientes. | Crear contrato automatizado antes de ampliar UI; debe detectar método/ruta/payload divergentes. |

Este documento es el plan de ejecución. Los resultados de cada corrida se deben registrar en una matriz separada (`TEST_EXECUTION.md` o herramienta de QA) sin alterar los casos ni borrar la evidencia histórica.
