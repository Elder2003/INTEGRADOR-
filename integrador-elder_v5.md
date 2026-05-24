# Sistema de Auditoría Académica con IA
## Documento de Diseño y Arquitectura
**Universidad Privada Franz Tamayo — Unifranz**
**Versión 5.0 — 2026**

---

## Índice

1. [Visión General](#1-visión-general)
2. [Actores del Sistema](#2-actores-del-sistema)
3. [Tipos de Actividad](#3-tipos-de-actividad)
4. [Módulos Funcionales](#4-módulos-funcionales)
5. [Flujos Principales](#5-flujos-principales)
6. [Arquitectura del Sistema](#6-arquitectura-del-sistema)
7. [Inteligencia Artificial en el Sistema](#7-inteligencia-artificial-en-el-sistema)
8. [Sistema RAG Dual](#8-sistema-rag-dual)
9. [Seguridad y Privacidad](#9-seguridad-y-privacidad)
10. [Requerimientos](#10-requerimientos)
11. [Decisiones de Diseño Clave](#11-decisiones-de-diseño-clave)
12. [Vistas del Sistema](#12-vistas-del-sistema)
13. [Estado de Implementación](#13-estado-de-implementación)

---

## 1. Visión General

### El Problema

La masificación de herramientas de inteligencia artificial generativa ha transformado el entorno educativo universitario. Los estudiantes de carreras tecnológicas y exactas recurren cada vez más a estas herramientas para obtener soluciones algorítmicas de forma inmediata, evitando el proceso cognitivo real que implica analizar, abstraer y resolver problemas. Los métodos de evaluación tradicionales no están diseñados para detectar este comportamiento porque califican el resultado final, no el proceso de construcción.

Esto genera un riesgo concreto: las calificaciones dejan de reflejar el verdadero nivel de razonamiento lógico del estudiante. A largo plazo, la institución puede estar certificando profesionales con deficiencias críticas en su capacidad analítica.

### Qué es este sistema

Una plataforma académica integral que cubre todo el ciclo de actividades académicas digitales: examina el proceso cognitivo real de los estudiantes mientras resuelven evaluaciones, analiza el contenido de tareas e informes entregados, valida el cumplimiento de parámetros institucionales de formato, y audita el proceso de desarrollo de proyectos. Con esa información, el sistema detecta patrones compatibles con el uso no declarado de IA y alerta al docente para que tome una decisión informada.

Adicionalmente, permite a los administradores institucionales describir problemáticas del mundo real y obtener una propuesta de qué materias de qué carreras pueden articularse para resolverla, con la capacidad de crear grupos de trabajo interdisciplinarios directamente desde esa propuesta.

A partir de la versión 5.0, el sistema incorpora un mecanismo formal de apelación con cadena de evidencia inmutable y verificable mediante SHA-256, que permite a los estudiantes impugnar resultados del análisis de IA o decisiones de calificación con garantías institucionales auditables.

### Qué no es este sistema

No es un sistema de vigilancia. No usa cámaras, no accede al dispositivo del estudiante más allá del navegador, y no toma decisiones automáticas sobre calificaciones. El sistema genera evidencia y sugerencias; el docente siempre tiene la última palabra.

### Pregunta de Investigación

> *"¿En qué medida la implementación de un sistema de auditoría con capacidad de detección de patrones de uso no declarado de IA contribuye a garantizar que las actividades académicas digitales reflejen el nivel real de competencias de los estudiantes de Ingeniería en Sistemas en Unifranz?"*

---

## 2. Actores del Sistema

### Administrador Institucional

Gestiona la configuración global de la plataforma. Es el único rol con acceso a:
- Carga y gestión de mallas curriculares
- Configuración de plantillas de formato institucional por carrera
- Configuración del proveedor de IA
- Módulo de resolución de problemas reales e interdisciplinarios
- Creación y gestión de cuentas de docentes
- Marcado de materias como elegibles para práctica asistida
- Recepción y resolución final de apelaciones escaladas

### Docente

Crea y gestiona sus cursos, grupos y actividades. Sube el material de referencia que el sistema usará para generar feedback contextualizado. Recibe alertas y reportes cuando el sistema detecta comportamientos sospechosos o incumplimientos de formato. Revisa la evidencia, valida o descarta la alerta, y asigna la calificación final. Puede habilitar co-docencia en actividades específicas e invitar a otro docente como guía.

El docente controla los horarios de entrega de todas las actividades. Las entregas fuera de plazo quedan marcadas en la auditoría pero el sistema nunca bloquea — el docente decide si las acepta.

El docente es también el primer revisor de las apelaciones que los estudiantes presentan sobre actividades de su propiedad. Puede resolver o escalar al administrador.

### Co-docente

Un docente habilitado puntualmente por el docente titular para participar en una actividad específica. Puede ver las alertas y calificar las entregas de esa actividad. No tiene acceso al resto del curso ni a otras evaluaciones.

### Estudiante

Resuelve las evaluaciones dentro del entorno del sistema. En actividades prácticas tiene acceso a feedback de la IA basado en el contenido subido por su docente. En modo examen resuelve sin asistencia y con auditoría activa. Sabe que existe un sistema de auditoría, pero no tiene acceso a los detalles del análisis sobre sus sesiones. Puede apelar calificaciones o resultados de análisis de IA formalmente a través del módulo de apelaciones.

---

## 3. Tipos de Actividad

El sistema soporta cinco tipos de actividad con comportamientos distintos:

| Tipo | Auditoría tiempo real | Detección IA | Validación formato | Análisis Git | Coherencia entre etapas | Feedback estudiante | Calificación |
|---|---|---|---|---|---|---|---|
| Examen | ✅ Keystroke-level | ✅ Heurísticas + LLM | ❌ | ❌ | ❌ | ❌ | ✅ Docente |
| Tarea | ❌ | ✅ LLM | ❌ | ❌ | ❌ | ❌ | ✅ Docente |
| Informe | ❌ | ✅ LLM | ✅ Plantilla institucional | ❌ | ❌ | ❌ | ✅ Docente |
| Práctica | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ Reporte final | ❌ |
| Proyecto | ✅ Solo en defensa oral | ✅ Por etapa | ✅ En etapas de informe | ✅ Opcional | ✅ Entre etapas | ❌ | ✅ Docente |

### 3.1 Examen

Actividad con auditoría de trazabilidad cognitiva activa en tiempo real. El estudiante trabaja en el entorno del sistema sin asistencia de IA. Al entregar, el worker analiza la sesión con heurísticas y LLM. El docente recibe un reporte de alerta si el análisis supera el umbral configurado.

### 3.2 Tarea

El estudiante entrega un archivo o texto fuera del entorno de resolución. No hay trazabilidad de proceso. El sistema analiza el contenido entregado con el LLM y lo compara con el RAG docente para detectar patrones de uso no declarado de IA. El docente recibe un reporte de detección.

### 3.3 Informe

El estudiante sube un archivo PDF o Word (.docx). El sistema realiza dos análisis en paralelo:

**Validación de formato:** Contrasta el documento contra la plantilla institucional vigente (definida por el administrador, sin intervención del docente). Verifica:
- Estructura de portada y campos requeridos
- Formato de citación (APA, Vancouver u otro definido institucionalmente)
- Secciones mínimas requeridas (introducción, desarrollo, conclusión, bibliografía)
- Extensión dentro del rango establecido
- Pie de figura en imágenes
- Número mínimo de fuentes bibliográficas
- Formato de fuente, márgenes y espaciado (en .docx con mayor precisión)

**Detección de IA:** Análisis del contenido con el LLM comparado contra el RAG docente.

El sistema siempre acepta la entrega independientemente del resultado de la validación. El docente recibe un reporte unificado de formato y detección. El estudiante no recibe feedback en tiempo real — el docente ve el reporte después.

**Plantilla institucional:** Definida y gestionada exclusivamente por el administrador institucional. Puede tener variaciones por carrera. Los docentes no intervienen en este proceso.

```
Administrador institucional
    └── Plantilla base Unifranz
            └── Variación por carrera (opcional)
```

### 3.4 Actividad Práctica

Actividad de estudio con asistencia de IA, sin auditoría, sin calificación formal. El estudiante trabaja en el entorno del sistema con feedback del LLM basado en el material del docente (RAG docente). Al finalizar recibe un reporte con el feedback acumulado de la sesión.

**Elegibilidad:** Solo disponible en materias marcadas por el administrador como aptas para práctica asistida. La elegibilidad se define a nivel de materia, no de carrera ni de docente. El docente decide si crear o no una actividad práctica en materias elegibles.

```
Administrador marca materia como elegible (true/false)
    └── Si true → docente puede crear actividades prácticas
    └── Si false → solo exámenes, tareas, informes y proyectos
```

**Ejemplos:**
- Estructura de Datos → elegible
- Programación Orientada a Objetos → elegible
- Álgebra Lineal → no elegible
- Física → no elegible

El estudiante no tiene historial de intentos. El reporte es para el estudiante, no para el docente.

### 3.5 Proyecto

Actividad compleja con múltiples etapas, entrega de código y defensa oral opcional. Está diseñada para auditar el proceso de desarrollo de sistemas o aplicaciones, no solo el resultado final.

**Componentes del proyecto (configurables por el docente):**

**Etapas (obligatorio):**
El docente define hitos intermedios con fecha límite y tipo de entrega independiente. Cada etapa es analizada según su tipo (tarea o informe).

Ejemplo de configuración de etapas para un sistema contable:
```
Semana 2  → Diagrama de base de datos (Informe)
Semana 4  → Módulo de autenticación (Tarea)
Semana 6  → Módulo contable con pruebas (Tarea)
Semana 8  → Sistema completo + informe final (Informe)
```

El sistema realiza además un **análisis de coherencia entre etapas**: compara calidad y estilo del código o contenido a lo largo del tiempo para detectar saltos anómalos entre entregas.

**Repositorio Git (opcional por docente):**
El estudiante enlaza su repositorio al entregar. El sistema analiza:
- Distribución de commits en el tiempo
- Volumen de código por commit
- Coherencia del progreso
- Consistencia entre el historial y el entregable final

Produce un reporte de patrones para que el docente interprete, no un veredicto automático.

**Defensa oral (opcional por docente):**
Sesión programada dentro del sistema. Funciona como un examen especial con auditoría activa. El docente hace preguntas y el estudiante responde en el entorno. Las respuestas quedan registradas como trazabilidad. El sistema analiza la coherencia entre lo que el estudiante escribe en la defensa y el código entregado.

**Entrega por etapa:**
Cada etapa requiere dos elementos:
1. **Archivo comprimido (.zip):** Entregable formal e inmutable una vez subido. Si el estudiante sube una versión posterior queda marcada como entrega tardía con su timestamp. El docente decide cuál versión califica pero ambas quedan en el sistema.
2. **Enlace al repositorio Git:** Evidencia del proceso de desarrollo. Obligatorio si el docente activa la auditoría de repositorio.

**Reporte unificado por etapa:**
```
Docente recibe:
    ├── Reporte del entregable (.zip) — análisis de contenido + detección IA
    ├── Reporte del proceso (Git) — análisis de historial de commits
    ├── Análisis de coherencia entre el repo y el zip
    └── Análisis de coherencia con etapas anteriores
```

---

## 4. Módulos Funcionales

### 4.1 Gestión Académica

Permite a los docentes crear la estructura académica: cursos, grupos de estudiantes y actividades. Cada actividad tiene un tipo (examen, tarea, informe, práctica, proyecto), una modalidad de contenido (código, texto libre, selección múltiple, archivo), una fecha y hora límite, y una configuración que determina si la auditoría está activa.

La auditoría de trazabilidad solo se activa en exámenes y defensas orales de proyectos donde existe riesgo real de uso no autorizado de IA: programación, bases de datos, materias teóricas con desarrollo de argumentos. No se activa en selección múltiple simple ni en actividades de práctica.

**Jerarquía académica implementada:**
```
Carrera
  └── Materia (con código institucional y semestre)
        └── Curso (semestre_label, sección, docente_id)
              ├── Grupo (nombre, lista de estudiantes)
              └── Actividad (tipo, deadline, instrucciones)
```

### 4.2 Entorno de Resolución

El espacio donde el estudiante trabaja durante una evaluación. Tiene dos modos:

**Modo práctica:** El estudiante puede ejecutar su código, ver el resultado, y recibir retroalimentación de la IA basada exclusivamente en el material que el docente subió al sistema. No hay auditoría de detección activa. Disponible solo en actividades de tipo práctica en materias elegibles.

**Modo examen:** El estudiante puede ejecutar su código para verificar que funciona, pero no recibe ningún feedback ni sugerencia. La auditoría de trazabilidad está activa desde el momento en que abre la evaluación hasta que entrega.

El entorno soporta múltiples modalidades:
- Editor de código con ejecución sandboxed
- Editor de texto libre para pseudocódigo, lógica en lenguaje natural y desarrollo teórico
- Formulario de selección múltiple
- Carga de archivos (imágenes, PDF) para exámenes en papel

### 4.3 Auditoría de Trazabilidad Cognitiva

El núcleo diferencial del sistema. Mientras el estudiante trabaja en modo examen o defensa oral, el sistema registra en tiempo real todos los eventos del proceso: cada tecla presionada, cada paste, cada periodo de inactividad, cada borrado, y los tiempos entre todos estos eventos.

Este registro es inmutable. Una vez creado, no puede ser editado ni eliminado. Representa la evidencia objetiva del proceso y puede ser revisado formalmente en apelaciones.

La trazabilidad varía según la modalidad:
- **En código:** captura keystroke, paste, borrados, inactividades y ejecuciones
- **En texto libre:** captura los mismos eventos enfocados en la construcción del argumento
- **En selección múltiple:** captura el tiempo por pregunta y cambios de respuesta

**Tipos de evento implementados:** `keystroke`, `paste`, `delete`, `execute`, `blur` (cambio de pestaña), `focus` (retorno), `inactivity`.

Los eventos se transmiten en tiempo real por WebSocket. El servidor los valida, asigna un `event_hash` y los persiste de forma atómica. La sesión de examen tiene un ciclo de vida explícito: `open` → `in_progress` → `completed`.

### 4.4 Detección de Patrones

Cuando el estudiante entrega o el tiempo finaliza, el sistema analiza la trazabilidad acumulada en dos capas:

**Heurísticas deterministas (tres métricas con pesos configurables):**

| Métrica | Variable de entorno | Peso por defecto |
|---|---|---|
| Cambios de pestaña (`tab_switching`) | `HEURISTIC_TAB_WEIGHT` | 5 |
| Paste de contenido (`clipboard_paste`) | `HEURISTIC_CLIP_WEIGHT` | 3 |
| Periodos de inactividad seguidos de paste (`inactivity_gap`) | `HEURISTIC_GAP_WEIGHT` | 2 |

El score final se compara contra `HEURISTIC_HIGH_THRESHOLD` (por defecto 60). Si el score supera el umbral, se genera una alerta.

**Análisis por IA:** El sistema envía al LLM el resumen de flags heurísticos, el contenido entregado, el contexto del RAG curricular (qué se espera que sepa el estudiante) y el RAG docente (material declarado válido por el docente). El modelo produce un análisis justificado en lenguaje natural.

El resultado es una calificación sugerida con justificación. El sistema guarda tanto la calificación sugerida (`suggested_grade_range`) como la final asignada por el docente (`final_grade`). Si difieren más del umbral, se requiere motivo de anulación (`override_reason`).

### 4.5 Validación de Formato de Informes

Módulo que analiza los archivos PDF y Word entregados como informes. Compara el documento contra la plantilla institucional vigente y genera un reporte de cumplimiento con observaciones específicas por parámetro.

**Para PDF:** Extrae texto con estructura mediante pdfcpu, detecta imágenes y verifica pie de figura, usa heurísticas visuales para márgenes y fuente.

**Para Word (.docx):** Accede directamente a estilos de título, párrafos, márgenes, fuente, tamaño de letra y estructura de secciones.

El sistema siempre acepta la entrega. El reporte de formato llega al docente junto con el reporte de detección de IA.

### 4.6 Gestión de Proyectos

Permite al docente configurar proyectos con etapas, repositorio Git y defensa oral. Gestiona el ciclo de vida de cada etapa: publicación, recepción de entregas (.zip + enlace Git), análisis de cada componente y generación del reporte unificado por etapa.

Mantiene el historial de versiones de cada entrega (primera entrega vs. entregas tardías) y el análisis de coherencia acumulado entre etapas.

El análisis de repositorio Git se ejecuta de forma asíncrona mediante `worker_jobs` (Redis Streams). El worker clona el repositorio en un directorio temporal, analiza el historial de commits y libera el espacio al finalizar.

### 4.7 Gestión de Contenido Docente para el RAG

Los docentes pueden subir material de referencia para dos propósitos: generar feedback en modo práctica y enriquecer el análisis post-actividad con el contexto del curso.

Tipos de contenido soportados:
- **Archivos (PDFs, diapositivas, apuntes):** Fragmentación semántica por sección
- **Libros comerciales:** Solo se indexa estructura (tabla de contenidos), no el cuerpo del texto (protección de copyright)
- **Libros de acceso abierto:** Indexados completos con fragmentación semántica
- **Rúbricas:** Nunca pasan por retrieval — siempre se inyectan completas al prompt
- **Enlaces web:** Capturados y almacenados localmente en MinIO en el momento de la carga
- **Videos de YouTube:** Transcript extraído automáticamente si está disponible
- **Repositorios de código:** Indexados según ruta indicada por el docente

El sistema clasifica automáticamente el tipo de documento y aplica la estrategia correspondiente. El docente recibe un mensaje claro del resultado por cada recurso subido.

**Estrategia de chunking:** 512 tokens con solapamiento de 64 tokens. Modelo de embeddings: `nomic-embed-text` (dimensión 1536). Presupuesto de retrieval: 2048 tokens totales por consulta.

### 4.8 Alertas y Notificaciones

Cuando el análisis supera el umbral configurado, el sistema genera una alerta dirigida al docente. La alerta incluye los flags heurísticos activados, el análisis en lenguaje natural de la IA, y un enlace directo para revisar la sesión completa.

El docente recibe la alerta en tiempo real en su panel web mediante SSE (Server-Sent Events). Para acceder al stream, el docente genera primero un token de corta duración (`POST /alerts/stream-token`) que se usa como query parameter en la conexión SSE. Esto evita exponer el JWT en la URL.

**Tipos de alerta emitidos:**

| Tipo | Disparador |
|---|---|
| `submission_received` | Estudiante entrega en actividad del docente |
| `format_violation` | Validación de formato devuelve incumplimientos |
| `integrity_flag` | Score heurístico supera umbral |
| `git_analysis_done` | Worker completa análisis de repositorio |
| `late_submission` | Entrega posterior al deadline |

Las alertas están filtradas por materia: un docente solo recibe alertas de sus propios estudiantes, salvo en actividades con co-docencia habilitada.

El docente puede marcar la alerta como confirmada, falso positivo o inconclusa. Este feedback queda registrado y alimenta la mejora progresiva del sistema.

### 4.9 Panel de Analítica para Docentes

Vista agregada del comportamiento de un grupo a lo largo del tiempo. Disponible en dos niveles:

**Por actividad (`GET /actividades/:id/analytics`):**
- Total de entregas, entregas pendientes, entregas tardías
- Conteo de flags de integridad levantados
- Distribución de calificaciones (histograma)

**Por curso (`GET /cursos/:id/analytics`):**
- Métricas agregadas de todas las actividades del curso
- Tendencias por estudiante a lo largo del semestre

### 4.10 Resolución de Problemas Reales e Interdisciplinarios

Módulo exclusivo del administrador institucional con un flujo de cinco pasos:

**Paso 1 — Descripción del problema:**
El administrador describe en texto libre una problemática concreta del mundo real.

**Paso 2 — Propuesta del sistema:**
El sistema analiza la descripción contra las mallas curriculares cargadas y devuelve una propuesta estructurada: qué carreras y materias son relevantes, qué aporta cada una, y cómo se articularían en un programa interdisciplinario.

**Paso 3 — Configuración del programa:**
El administrador selecciona qué materias de la propuesta incluir en el programa.

**Paso 4 — Creación de grupos:**
Por cada materia seleccionada, el administrador asigna un docente encargado y configura el grupo de estudiantes.

**Paso 5 — Activación:**
El programa queda activo. Cada docente asignado lo ve en su panel como un curso especial, visualmente diferenciado de sus cursos normales, vinculado a la problemática institucional.

Todas las consultas quedan guardadas con su resultado para referencia futura.

### 4.11 Gestión Curricular

El administrador sube los planes globales de formación de cada materia en el formato estándar de Unifranz. El sistema los procesa, extrae la información relevante y la convierte en la base de conocimiento del RAG curricular.

El sistema arranca con Ingeniería de Sistemas y Derecho. Incorporar nuevas carreras no requiere cambios estructurales.

El administrador puede listar y eliminar documentos curriculares existentes a través del panel administrativo (`GET /admin/curriculum/documents`, `DELETE /admin/curriculum/documents/:id`). La eliminación quita tanto el registro en PostgreSQL como el archivo en MinIO.

### 4.12 Gestión de Módulos Administrativos

Gestión centralizada de recursos institucionales que no pertenecen a ningún curso específico, sino a la institución.

**Gestión de cuentas de docentes:**
El administrador pre-provisiona cuentas de docente con email institucional y nombre completo. El email debe tener el formato `doc.*@unifranz.edu.bo`. Al crear la cuenta, el sistema genera una contraseña temporal que se devuelve en la respuesta (solo visible una vez).

En entorno de desarrollo (`APP_ENV=development`), la creación acepta un campo `test_password` para facilitar pruebas sin necesidad de flujo OAuth. En producción, ese campo es ignorado.

Las cuentas pueden desactivarse (`DELETE /admin/docentes/:id` pone `is_active=false`) y reactivarse (`PUT /admin/docentes/:id` con `{"is_active": true}`). Un docente inactivo no puede iniciar sesión.

**Biblioteca de plantillas de formato:**
El administrador define plantillas de reglas de formato reutilizables (ej. "TFG Estándar: PDF, 10–50 páginas, APA"). Los docentes aplican una plantilla a sus actividades de tipo informe (`POST /actividades/:aid/format-template`), en lugar de configurar reglas manualmente. Una plantilla en uso no puede eliminarse (devuelve 409).

**Configuración de scopes interdisciplinarios:**
El administrador puede asociar materias específicas a alcances interdisciplinarios, permitiendo al sistema RAG curricular filtrar contenido relevante cuando se analiza un problema multidisciplinario.

### 4.13 Apelaciones y Cadena de Auditoría Inmutable

Sistema de apelaciones formales con cadena de evidencia verificable mediante hashing SHA-256. Implementado en M13.

**Tipos de apelación:**

| Tipo | Descripción |
|---|---|
| `analysis_dispute` | El estudiante disputa el análisis de IA (detección, heurísticas) |
| `grade_dispute` | El estudiante disputa la calificación final asignada por el docente |

**Ciclo de vida de una apelación:**

```
pending
  ├── [docente review, analysis_dispute] → resolved
  ├── [docente escalate, cualquier tipo] → escalated
  └── [docente review, grade_dispute sobre su propio curso] → escalated (auto)
                                                              └── [admin review] → admin_resolved
```

**Regla de auto-escalación:** Si el tipo es `grade_dispute` y el docente que intenta resolver es el mismo que asignó la calificación (propietario del curso), el sistema escala automáticamente. Esto implementa el principio de evitar conflictos de interés sin requerir que el docente lo declare manualmente.

**Restricción de duplicados:** Solo puede existir una apelación abierta (`status IN ('pending', 'escalated')`) por sesión de examen. Un segundo intento retorna 409 Conflict.

**Cadena de auditoría inmutable:**
Cada apelación genera automáticamente una cadena de eventos en la tabla `audit_events`. Los eventos se encadenan mediante SHA-256:

```
event_hash = SHA-256(occurred_at|event_type|sequence|payload_json|prev_hash)
```

Donde `prev_hash` para el primer evento es cadena vacía, y para los siguientes es el `event_hash` del evento anterior. Cada evento tiene un número de secuencia único por apelación.

**Eventos generados automáticamente por apelación:**

| Secuencia | Tipo | Disparador |
|---|---|---|
| 1 | `submission_received` | Al crear la apelación (contexto de la entrega) |
| 2 | `analysis_completed` | Al crear la apelación (contexto del análisis) |
| 3 | `appeal_filed` | Al crear la apelación |
| 4 | `decision_recorded` | Al insertar cada decisión (docente o admin) |

**Endpoint de evidencia (`GET /appeals/:id/evidence`):**
Devuelve la cadena completa con un campo `valid: true/false` que indica si todos los hashes son coherentes. Admin y docentes propietarios pueden acceder. Estudiantes: 403.

```json
{
  "appeal_id": "...",
  "valid": true,
  "events": [
    { "sequence": 1, "event_type": "submission_received", "prev_hash": null, "event_hash": "a3f8..." },
    { "sequence": 2, "event_type": "analysis_completed",  "prev_hash": "a3f8...", "event_hash": "d91c..." },
    ...
  ]
}
```

**Grado de log (`GET /sessions/:sessionId/grade-log`):**
El docente puede consultar la decisión de calificación registrada para una sesión: nota final, rango sugerido por la IA y motivo de anulación (si aplica). Solo accesible por el docente propietario o administrador.

### 4.14 Configuración del Proveedor de IA

El administrador puede cambiar el modo de IA activo desde un panel administrativo sin intervención técnica ni reinicio del sistema. El cambio aplica a partir del siguiente análisis. Los resultados anteriores quedan asociados al proveedor que los generó.

Tres modos disponibles:
- **Solo local:** Todas las solicitudes van a Ollama (GPU VPS on-premise)
- **Auto:** Ollama como primario; Vercel AI Gateway como fallback automático si Ollama no está disponible o el caso supera el umbral de complejidad configurado
- **Solo cloud:** Todas las solicitudes van a Vercel AI Gateway (OpenAI / Anthropic / Gemini)

La configuración activa se almacena en PostgreSQL y se cachea 5 segundos en Redis. El cambio aplica sin reiniciar ningún proceso.

---

## 5. Flujos Principales

### Flujo 1: Estudiante resuelve un examen

1. El estudiante accede a la evaluación publicada por su docente.
2. El sistema registra el inicio de la sesión y activa la captura de trazabilidad.
3. El estudiante trabaja. Cada evento se registra en tiempo real de forma transparente.
4. El estudiante ejecuta su código para verificar que funciona. No recibe feedback de la IA.
5. El estudiante entrega su solución.
6. El sistema cierra la sesión y envía la trazabilidad al worker de análisis asíncrono.
7. El worker analiza con heurísticas y LLM usando contexto del RAG curricular y RAG docente.
8. Si el score supera el umbral, se genera una alerta para el docente.
9. La entrega queda disponible para revisión manual con la calificación sugerida visible.

### Flujo 2: Estudiante entrega una tarea

1. El estudiante accede a la actividad y sube su archivo o escribe su respuesta.
2. El sistema registra el timestamp de la entrega y lo compara con la fecha límite configurada.
3. Si la entrega es tardía, queda marcada en la auditoría.
4. El worker analiza el contenido con el LLM comparado contra el RAG docente.
5. El docente recibe el reporte de detección.

### Flujo 3: Estudiante entrega un informe

1. El estudiante sube el archivo PDF o Word.
2. El sistema acepta la entrega y la encola para análisis.
3. El worker ejecuta en paralelo:
   - Validación de formato contra la plantilla institucional vigente
   - Detección de IA en el contenido
4. El docente recibe un reporte unificado con ambos análisis.

### Flujo 4: Estudiante trabaja en una actividad práctica

1. El estudiante accede a la actividad práctica publicada por su docente.
2. Resuelve en el entorno con asistencia activa del LLM basada en el RAG docente.
3. Puede ejecutar su código y recibir feedback iterativamente.
4. Al finalizar, el sistema genera un reporte de feedback acumulado de la sesión.
5. El estudiante recibe el reporte. No hay entrega formal ni calificación.

### Flujo 5: Estudiante entrega una etapa de proyecto

1. El estudiante sube el archivo .zip de la etapa.
2. El estudiante enlaza su repositorio Git (si el docente lo activó).
3. El sistema acepta ambos y los encola para análisis.
4. El worker ejecuta en paralelo:
   - Análisis del .zip (detección IA + validación de formato si es informe)
   - Análisis del historial Git (distribución de commits, volumen, coherencia)
   - Comparación entre el repo y el .zip entregado
   - Comparación de coherencia con etapas anteriores
5. El docente recibe el reporte unificado de la etapa.

### Flujo 6: Defensa oral de proyecto

1. El docente programa la sesión de defensa con fecha y hora.
2. El estudiante accede a la sesión. El sistema activa la auditoría de trazabilidad.
3. El docente hace preguntas. El estudiante responde en el entorno del sistema.
4. Las respuestas quedan registradas como trazabilidad.
5. El worker analiza la coherencia entre las respuestas de la defensa y el código entregado en el proyecto.
6. El docente recibe el reporte de la defensa.

### Flujo 7: Docente revisa una alerta

1. El docente recibe una notificación de una sesión sospechosa.
2. Accede al detalle: flags activados, análisis de la IA, resumen de trazabilidad.
3. Puede reproducir el timeline de la sesión.
4. Decide: confirmar, marcar como falso positivo, o dejar inconclusa.
5. Asigna la calificación final. Si difiere de la sugerida, el sistema registra ambos valores.

### Flujo 8: Administrador crea programa interdisciplinario

1. El administrador describe la problemática en texto libre.
2. El sistema devuelve una propuesta de materias por carrera.
3. El administrador selecciona las materias a incluir.
4. Por cada materia, asigna un docente encargado y configura el grupo.
5. El programa queda activo. Los docentes asignados lo ven en su panel como curso especial.

### Flujo 9: Apelación formal de una calificación o análisis de IA

1. El estudiante accede al módulo de apelaciones y presenta la impugnación (`POST /appeals`) indicando el tipo (`analysis_dispute` o `grade_dispute`) y el motivo detallado.
2. El sistema registra la apelación con estado `pending` y genera automáticamente los primeros tres eventos de la cadena de auditoría (submission_received, analysis_completed, appeal_filed) con sus hashes SHA-256 encadenados.
3. El docente propietario de la actividad recibe la apelación en su cola. Puede:
   - **Resolver directamente** (para `analysis_dispute`): rechazar o avalar la apelación con justificación → estado `resolved`.
   - **Auto-escalación automática** (para `grade_dispute` cuando el docente asignó la nota): el sistema escala sin requerir acción manual → estado `escalated`.
   - **Escalar manualmente**: el docente puede escalar cualquier apelación al administrador con notas → estado `escalated`.
4. El administrador revisa la apelación escalada y emite resolución final (`PUT /appeals/:id/admin-review`) → estado `admin_resolved`.
5. Cada decisión genera un nuevo evento en la cadena de auditoría (`decision_recorded`) encadenado al hash anterior.
6. El administrador puede verificar la integridad de toda la cadena en cualquier momento (`GET /appeals/:id/evidence`). El campo `valid: true` certifica que ningún evento fue alterado desde su creación.

---

## 6. Arquitectura del Sistema

### Visión general

El sistema se divide en dos procesos principales que comparten la misma base de código pero se despliegan de forma independiente:

**El servidor principal** atiende todas las peticiones HTTP y las conexiones WebSocket de los estudiantes durante los exámenes. Gestiona la autenticación, sirve los módulos de gestión académica, recibe los eventos de trazabilidad en tiempo real, y procesa la carga de contenido docente al RAG.

**El worker de análisis** es un proceso separado que consume una cola de trabajos (Redis Streams). Cuando una sesión termina o una entrega llega, el servidor encola el trabajo. El worker lo procesa de forma asíncrona: ejecuta las heurísticas, valida el formato si aplica, consulta ambas bases de conocimiento, llama al router de IA configurado, y guarda el resultado. Esto garantiza que el análisis nunca afecte el rendimiento del entorno de resolución.

### Stack Tecnológico

- **Backend:** Go 1.25 + Fiber v2 + PostgreSQL 16 (+ pgvector) + Redis 7
- **Frontend:** React + Vite
- **Almacenamiento de archivos:** MinIO (S3-compatible, self-hosted)
- **IA — Router multimodelo (Go):** Enruta solicitudes entre Ollama y Vercel AI Gateway según modo configurado, disponibilidad y complejidad del caso
- **IA — Primario on-premise:** Ollama con LLaMA 3 8B / Mistral 7B en VPS dedicado con GPU
- **IA — Fallback / cloud:** Vercel AI Gateway → OpenAI GPT-4o / GPT-4o mini, Anthropic Claude, Google Gemini
- **Base vectorial:** pgvector (extensión de PostgreSQL — RAG curricular y RAG docente en tablas separadas)
- **Comunicación tiempo real:** WebSocket (trazabilidad de estudiante), SSE (notificaciones al docente)
- **Ejecución sandboxed:** Judge0 (cloud o self-hosted)
- **DevOps:** Docker Compose, GitHub Actions
- **Gestión:** Git, especificaciones en `specs/NNN-nombre/`

### Separación de responsabilidades

Módulos internos con responsabilidad única:

| Paquete Go | Responsabilidad | Milestone |
|---|---|---|
| `internal/auth` | Identidad, roles, JWT, OAuth | M2 |
| `internal/academic` | Cursos, grupos, actividades | M3 |
| `internal/submission` | Ciclo de vida de entregas, MinIO | M4 |
| `internal/tracing` | Captura WebSocket, sesiones de examen | M5 |
| `internal/ai` | Router multimodelo Ollama/Vercel | M6 |
| `internal/rag` | Ingesta, vectorización y retrieval dual | M7 |
| `internal/detection` | Heurísticas, análisis LLM, grade decisions | M8 |
| `internal/format` | Validación PDF/DOCX contra plantilla | M9 |
| `internal/projects` | Etapas, análisis Git, worker jobs | M10 |
| `internal/alerts` | SSE stream, tokens corta duración | M11 |
| `internal/analytics` | Agregación de métricas por actividad/curso | M11 |
| `internal/admin` | Docentes, plantillas, curriculum, interdisciplinar | M12 |
| `internal/appeals` | Apelaciones, decisiones, cadena hash SHA-256 | M13 |
| `internal/health` | Healthcheck PostgreSQL, Redis, MinIO | M1 |
| `internal/config` | Carga de variables de entorno | M1 |

### Router de IA híbrido

El sistema no llama directamente a ningún proveedor de IA. El router Go decide qué backend usar en cada solicitud:

```
Router de IA
├── Modo "local"  → Ollama GPU VPS (siempre)
├── Modo "cloud"  → Vercel AI Gateway (siempre)
└── Modo "auto"   → Ollama primario
                    └── Fallback automático a Vercel AI Gateway si:
                        - Ollama no responde
                        - Score de complejidad supera umbral configurado
```

La configuración activa se almacena en PostgreSQL con cache de 5 segundos en Redis. El cambio de modo aplica sin reiniciar ningún proceso. Ambos clientes implementan la misma interfaz Go (`AIClient`), por lo que el router los intercambia de forma transparente.

### Comunicación en tiempo real

- **WebSocket:** Eventos de trazabilidad del estudiante al servidor durante exámenes y defensas orales. El middleware JWT para WebSocket se registra con `app.Use("/sessions/:sid/ws", wsJWTMiddleware)`.
- **SSE (Server-Sent Events):** Notificaciones unidireccionales al panel del docente. El stream en `/alerts/stream` se autentica con un token de corta duración generado en `POST /alerts/stream-token`, almacenado 30 segundos en Redis.

### Migraciones de base de datos

Las migraciones se gestionan con `golang-migrate` usando el driver `iofs` y archivos SQL embebidos en el binario (`embed.FS`). Al arrancar el servidor, se aplican automáticamente todas las migraciones pendientes.

```
migrations/
  000001_foundation.up.sql
  000002_auth.up.sql
  ...
  000013_appeals_final_audit.up.sql
```

Para los tests de integración, `TestMain` conecta a `DATABASE_URL_TEST`, aplica las mismas migraciones y trunca las tablas antes de cada suite.

### Despliegue

Dos VPS independientes en Hetzner:

**VPS 1 — Servicios de aplicación (Hetzner CX42 — €16.40/mes)**

| Recurso | Valor |
|---|---|
| vCPU | 8 |
| RAM | 16 GB |
| Disco | 160 GB SSD |
| Tráfico | 20 TB/mes |

Servicios: API (Fiber), Worker asíncrono, PostgreSQL + pgvector, Redis, MinIO

```yaml
# docker-compose.yml — VPS 1
services:
  api:       # Go + Fiber — cmd/api
  worker:    # Go worker asíncrono — cmd/worker
  postgres:  # pgvector/pgvector:pg16 — puerto 5433 (local) / 5432 (interno)
  redis:     # redis:7-alpine
  minio:     # minio/minio — puertos 9000 (API) y 9001 (consola)
```

**VPS 2 — Modelo de IA on-premise (Hetzner GEX44 — €184/mes)**

| Recurso | Valor |
|---|---|
| GPU | NVIDIA RTX 4000 Ada (20 GB VRAM) |
| RAM | 64 GB DDR5 |
| CPU | Intel Xeon w3-2423 (6 cores) |
| Disco | 2× 480 GB NVMe |

Servicio: Ollama únicamente. Accesible desde VPS 1 por red privada en `http://<ollama-vps>:11434`.

Modelos compatibles con 20 GB VRAM:
- LLaMA 3 8B Q4_K_M → ~5 GB VRAM
- Mistral 7B Q4_K_M → ~5 GB VRAM
- LLaMA 3 13B Q4_K_M → ~8 GB VRAM

**Vercel AI Gateway** — Fallback y overflow cloud. Sin servidor adicional; el router apunta al endpoint del Gateway cuando el modo lo requiere.

**Costo total mensual: ~€202/mes**

---

## 7. Inteligencia Artificial en el Sistema

### Dos capas de detección

**Heurísticas deterministas:** Reglas con umbrales numéricos. Instantáneas, sin costo de API. Detectan: más del 60% del código llegó por paste, velocidad de tipeo sostenida imposible para un humano, solución sin ningún borrado a pesar de su complejidad, inactividad prolongada seguida de entrega inmediata.

Las tres métricas (tab_switching, clipboard_paste, inactivity_gap) tienen pesos configurables por variable de entorno. El threshold para generar alerta también es configurable sin redeploy.

**Análisis por LLM:** El modelo recibe el resumen de flags heurísticos, el contenido entregado, el contexto del RAG curricular y el RAG docente. Produce análisis en lenguaje natural con razones específicas y una evaluación de consistencia con el nivel académico esperado.

### Router de IA intercambiable en caliente

El modo activo es una operación administrativa. Aplica inmediatamente sin reiniciar nada. Los análisis anteriores quedan asociados al backend que los procesó (Ollama o Vercel Gateway + modelo específico).

### Evolución progresiva

El sistema funciona desde el primer día con heurísticas y el LLM configurado. No requiere datos históricos previos. A medida que los docentes validan o rechazan alertas, el sistema acumula casos etiquetados que pueden usarse para entrenar un modelo de clasificación específico al final de cada semestre.

---

## 8. Sistema RAG Dual

Dos bases de conocimiento independientes en PostgreSQL + pgvector. **Nunca se mezclan en una misma consulta.**

### 8.1 RAG Curricular

**Propósito:** Contexto sobre qué se espera que sepa un estudiante de esa materia en ese semestre. Uso interno del sistema — no visible para estudiantes ni docentes directamente.

**Fuente:** Planes instruccionales subidos por el administrador en formato XLSX estándar de Unifranz.

**Filtrado:** Siempre se filtra por `carrera + semestre + codigo_materia` antes de buscar por similitud semántica con pgvector.

**Usos:**
- Motor de detección en exámenes
- Módulo de resolución de problemas reales (con proceso de retrieval completamente distinto)

### 8.2 RAG Docente

**Propósito:** Que el LLM responda basado exclusivamente en lo que el docente definió como contenido válido para su curso o evaluación.

**Usos:**
- Feedback en modo práctica
- Análisis post-actividad (detección de IA en tareas, informes y proyectos)
- Calificación sugerida en exámenes

**Jerarquía de retrieval (presupuesto de 2.048 tokens):**

| Prioridad | Tipo | Scope | Mecanismo |
|---|---|---|---|
| 1 | Rúbrica | Evaluación específica | Inyección directa, siempre |
| 2 | Material indexable | Evaluación específica | Retrieval por similitud (pgvector) |
| 3 | Material indexable | Curso completo | Retrieval por similitud (pgvector) |
| 4 | Referencias bibliográficas | Evaluación o curso | Inyección directa, siempre |
| 5 | Contexto curricular (RAG 1) | Materia + semestre | Siempre presente, piso mínimo |

---

## 9. Seguridad y Privacidad

### Principios generales

- **No invasividad:** solo se captura lo que ocurre dentro del navegador
- **Proporcionalidad:** la auditoría solo se activa donde existe riesgo real
- **Transparencia:** los estudiantes saben que existe un sistema de auditoría
- **Privacidad de datos de examen:** en modo "local" o "auto", el contenido de las sesiones de estudiantes se procesa exclusivamente en los servidores de Unifranz y nunca sale a proveedores externos

### Autenticación y acceso

Tokens de acceso de corta duración (15 minutos) combinados con tokens de refresco almacenados en el servidor. Cada rol tiene acceso estrictamente limitado a su función.

**Flujo de autenticación:**
- `POST /auth/login` → access token (15m) + refresh token (7d)
- `POST /auth/refresh` → rotación de refresh token (revoca el anterior)
- `POST /auth/logout` → revoca el refresh token activo
- `GET /auth/google` → inicio flujo OAuth con Google
- `GET /auth/google/callback` → completa OAuth, valida email institucional

**Rate limiting:** 5 intentos fallidos por IP+email en 60 segundos activan un bloqueo de cooldown. El contador usa Redis con TTL.

### Datos de trazabilidad

Registros inmutables una vez creados. El acceso está limitado al docente de la materia y, en actividades con co-docencia, al co-docente asignado.

### Contenido del RAG docente

Accesible únicamente en el scope declarado. Filtros de `docente_id` y `curso_id` antes de cualquier operación de retrieval. Contenido sanitizado antes de indexar. Archivos almacenados en MinIO con acceso controlado por el backend — nunca expuestos directamente al cliente.

### Cadena de evidencia de apelaciones

Los hashes SHA-256 de la cadena de audit_events garantizan que cualquier modificación posterior a un evento sea detectable. El campo `valid: true/false` del endpoint de evidencia puede verificarse de forma independiente por cualquier parte externa recomputando los hashes con la fórmula pública.

---

## 10. Requerimientos

### Requerimientos Funcionales

**RF-01 Gestión de Usuarios y Roles**
Cuatro roles: Administrador Institucional, Docente, Co-docente y Estudiante. Permisos estrictamente definidos por rol. Nombres de roles en base de datos: `admin`, `docente`, `co_docente`, `estudiante`.

**RF-02 Gestión de Cursos y Actividades**
Docentes crean cursos, grupos y actividades de cinco tipos: examen, tarea, informe, práctica y proyecto. Cada actividad tiene configuración de tipo, modalidad, fecha límite y estado de auditoría.

**RF-03 Entorno de Resolución Web**
Área de trabajo en el navegador. Soporta editor de código con ejecución, editor de texto libre, selección múltiple y carga de archivos. Modo práctica con feedback, modo examen sin feedback.

**RF-04 Captura de Trazabilidad Cognitiva**
Registro en tiempo real de keystroke, paste, borrados, inactividades, ejecuciones y tiempos entre eventos. Registro inmutable. Transmisión por WebSocket.

**RF-05 Motor de Ejecución Sandboxed**
Entorno aislado para ejecutar código. En práctica devuelve resultado y feedback. En examen solo el resultado.

**RF-06 Motor de Detección de Patrones**
Heurísticas deterministas (tab_switching, clipboard_paste, inactivity_gap) con pesos y threshold configurables por variable de entorno. Análisis LLM con contexto RAG dual. Produce score y justificación en lenguaje natural.

**RF-07 Validación de Formato de Informes**
Parseo de PDF (pdfcpu) y Word (.docx). Contraste contra plantilla institucional. Reporte de cumplimiento por parámetro. Siempre acepta la entrega.

**RF-08 Gestión de Proyectos por Etapas**
Configuración de etapas con tipo y fecha límite. Recepción de .zip + enlace Git por etapa. Análisis de historial de commits. Análisis de coherencia entre etapas. Defensa oral con auditoría activa.

**RF-09 Generación de Alertas y Reportes**
Notificaciones SSE al docente para cinco tipos de evento. Token de corta duración para autenticar el stream. Reporte unificado por tipo de actividad.

**RF-10 Panel de Analítica para Docentes**
Vista agregada de comportamiento por actividad y curso: submission_count, pending_count, late_count, flagged_count, grade_distribution.

**RF-11 Módulo de Resolución de Problemas Reales**
El administrador describe una problemática. El sistema propone materias articuladas. El administrador selecciona, asigna docentes y crea grupos. El programa queda activo como cursos especiales en el panel de cada docente asignado.

**RF-12 Gestión de Plantillas de Formato Institucional**
El administrador define y actualiza plantillas de reglas de formato reutilizables. Los docentes aplican plantillas a sus actividades. Una plantilla en uso no puede eliminarse.

**RF-13 Gestión Curricular (RAG Curricular)**
Carga, validación, procesamiento e indexación de planes de formación. Lista y eliminación de documentos existentes. Soporte para múltiples carreras sin cambios estructurales.

**RF-14 Gestión de Contenido Docente (RAG Docente)**
Docentes suben archivos y enlaces. El sistema clasifica, procesa y fragmenta automáticamente. Detección de copyright con manejo diferenciado. Archivos almacenados en MinIO.

**RF-15 Configuración de Proveedor de IA Multimodelo**
Router Go con tres modos: local (Ollama), auto (Ollama primario + Vercel Gateway fallback), cloud (Vercel Gateway). Cambio de modo desde panel administrativo sin reinicio. Cache Redis de 5 segundos.

**RF-16 Marcado de Materias para Práctica Asistida**
El administrador marca materias como elegibles o no elegibles para actividades prácticas con asistencia de IA.

**RF-17 Co-docencia**
El docente titular habilita a otro docente como guía de una actividad específica con acceso limitado.

**RF-18 Calificación con Auditoría**
El sistema sugiere calificación. El docente puede editarla. Ambos valores quedan registrados. Si difieren del rango sugerido, se requiere y registra el motivo de anulación (`override_reason`).

**RF-19 Registro de Feedback de Alertas**
Docentes marcan alertas como confirmadas, falsos positivos o inconclusas. Alimenta la mejora progresiva.

**RF-20 Apelación Formal con Cadena de Evidencia Inmutable**
Estudiantes presentan apelaciones de tipo `analysis_dispute` o `grade_dispute`. Flujo de revisión docente → escalación → resolución admin. Auto-escalación para grade_dispute con conflicto de interés. Cadena SHA-256 verificable externamente. Una apelación abierta por sesión (constraint UNIQUE).

**RF-21 Gestión de Cuentas de Docentes**
El administrador pre-provisiona cuentas de docente con email `doc.*@unifranz.edu.bo`. Activación/desactivación reversible. Campo `test_password` disponible en entorno de desarrollo.

### Requerimientos No Funcionales

**RNF-01 Procesamiento Asíncrono**
La captura y el análisis ocurren desacoplados. El estudiante nunca experimenta retraso en su entorno de resolución.

**RNF-02 Alta Disponibilidad**
99.9% de disponibilidad objetivo. Servidor principal y worker reiniciables de forma independiente.

**RNF-03 Escalabilidad por Carrera**
Nuevas carreras se incorporan cargando sus materias en gestión curricular, sin cambios en el código.

**RNF-04 Baja Fricción Cognitiva**
El entorno de resolución es minimalista. Sin complejidad técnica visible para el estudiante.

**RNF-05 No Invasividad**
Solo opera dentro del navegador. Sin cámaras, sin acceso al sistema de archivos, sin software adicional.

**RNF-06 Privacidad de Datos**
Datos de trazabilidad y RAG docente accesibles únicamente por roles con autorización explícita. En modo local, ningún dato de estudiante sale de los servidores de Unifranz.

**RNF-07 Transmisión Segura**
Toda la comunicación usa cifrado en tránsito (HTTPS/TLS).

**RNF-08 SLA del Worker de Análisis**
El análisis asíncrono se completa en máximo 3 minutos desde la entrega.

**RNF-09 Trazabilidad de Ingestas**
Cada carga de material queda registrada con autor, fecha, archivo, resultado y advertencias.

**RNF-10 Proveedor de IA Intercambiable**
Cambio de modo (local/auto/cloud) sin intervención técnica ni tiempo de inactividad.

**RNF-11 Registro Inmutable de Evidencia**
Eventos de trazabilidad, análisis y cadena de auditoría de apelaciones no pueden modificarse una vez creados. Los hashes SHA-256 hacen detectable cualquier alteración.

**RNF-12 Consistencia del Contenido Capturado**
Los recursos externos se almacenan localmente en MinIO en el momento de la carga. El sistema nunca depende de disponibilidad externa durante una evaluación activa.

**RNF-13 Horarios de Entrega Auditados**
Todas las entregas registran su timestamp. Las entregas tardías se marcan automáticamente. El sistema nunca bloquea una entrega.

---

## 11. Decisiones de Diseño Clave

### Un sistema, dos procesos

Monolito modular con un worker asíncrono separado. Reduce complejidad operacional sin sacrificar capacidad de escalar componentes críticos de forma independiente.

### Cinco tipos de actividad con lógica propia

Cada tipo (examen, tarea, informe, práctica, proyecto) tiene su propio flujo de análisis, sus propios reportes y sus propias reglas. No se intenta unificar lo que es conceptualmente distinto.

### Proyectos auditados en tres dimensiones

La auditoría de proyectos combina análisis por etapas, historial Git y defensa oral. Ninguna de las tres es infalible por sí sola, pero juntas hacen muy difícil engañar al sistema de forma consistente a lo largo de todo el proyecto.

### El .zip y el repo son evidencias distintas

El archivo comprimido es el entregable formal e inmutable. El repositorio Git es la evidencia del proceso. Son cosas distintas con propósitos distintos y el sistema los trata como tal.

### Validación de formato sin bloqueo

El sistema siempre acepta la entrega independientemente del resultado de la validación de formato. El reporte es para el docente, no una barrera para el estudiante. Esto evita pérdidas de entrega por problemas técnicos de parseo.

### Plantilla institucional fuera del alcance del docente

Los parámetros de formato son una decisión institucional, no académica. Los docentes reciben el resultado de la validación pero no configuran los criterios. Esto garantiza consistencia en toda la institución.

### Elegibilidad de práctica definida por materia

El administrador determina qué materias pueden tener actividades prácticas con asistencia de IA. El docente decide si crear o no esa actividad dentro de las materias elegibles.

### Programas interdisciplinarios como ciudadanos de primera clase

El módulo de resolución de problemas reales no termina en una propuesta de texto. Genera grupos activos con docentes asignados que aparecen en el panel de cada docente como cursos especiales.

### Dos RAGs independientes con propósitos distintos

El RAG curricular y el RAG docente nunca se mezclan en una misma consulta. El primero alimenta la detección con expectativas académicas del semestre. El segundo alimenta el feedback y la calificación sugerida con el contenido específico declarado por el docente. Ambos se implementan con pgvector en tablas separadas de PostgreSQL.

### La rúbrica nunca pasa por retrieval

Las rúbricas se inyectan siempre completas al prompt. Fragmentar una rúbrica produce calificaciones con criterios incompletos.

### El docente siempre tiene la última palabra

El sistema nunca toma decisiones automáticas sobre calificaciones o sanciones. Su rol es producir evidencia organizada y análisis justificados. La autoridad académica permanece en el docente.

### Cadena SHA-256 para evidencia de apelaciones

Los eventos de auditoría de una apelación se encadenan con SHA-256: cada evento incluye el hash del evento anterior en su propio hash. Esto hace que cualquier modificación retroactiva a cualquier evento sea detectable al recomputar la cadena. La fórmula es pública y puede verificarse independientemente:

```
hash = SHA-256(occurred_at|event_type|sequence|payload_json|prev_hash)
```

El primer evento usa cadena vacía como `prev_hash`. Los hashes se almacenan en formato hexadecimal en minúsculas.

### Auto-escalación de grade_dispute para evitar conflictos de interés

Cuando un docente intenta resolver una apelación de tipo `grade_dispute` sobre una sesión que él mismo calificó, el sistema escala automáticamente en lugar de resolver. Esto no requiere que el docente declare el conflicto manualmente — el sistema lo detecta comparando el `docente_id` del curso con el `caller_id` del revisor. Garantiza imparcialidad sin depender de autodeclaración.

### Una apelación abierta por sesión

La constraint `UNIQUE(session_id)` en la tabla `appeals` (junto con verificación `HasOpenAppeal` en el handler) impide que un estudiante abra múltiples impugnaciones simultáneas sobre la misma sesión. Esto obliga a cerrar el ciclo de cada apelación antes de iniciar otra.

### Nombres de roles en español

Los roles almacenados en base de datos usan la terminología española del contexto institucional: `estudiante` (no `student`), `docente` (no `teacher`), `co_docente`, `admin`. Esto elimina traducciones implícitas entre la capa de datos y la lógica de negocio.

### Mejora progresiva sin dependencia de datos históricos

El sistema funciona desde el primer día. A medida que se acumulan casos validados por docentes, puede incorporar un modelo de clasificación específico. Esta progresión es deliberada y controlada.

### Mecanismo de apelación como parte del diseño

El flujo de apelación formal no es un parche posterior. La trazabilidad inmutable combinada con un flujo de revisión escalable al administrador y una cadena de evidencia SHA-256 hace que el sistema sea institucionalmente sostenible y auditablemente defensible.

### Fiber sobre Gin y Echo

Se eligió Fiber como framework HTTP por su soporte nativo de WebSocket y SSE, ambos requerimientos de primer nivel del sistema. Fiber corre sobre fasthttp, lo que no representa una limitación para este stack ya que todos los clientes utilizados son compatibles.

**Nota importante sobre Fiber v2.52.0:** `app.Group("", handlers...)` con prefijo vacío registra los handlers como middleware USE global, afectando todas las rutas — incluyendo rutas públicas como `/auth/login`. Para evitar esto, los módulos con rutas en el nivel raíz registran las rutas directamente con `app.Get/Post(path, mw1, mw2, handler)` en lugar de usar un grupo con prefijo vacío.

### pgvector en lugar de Qdrant

El volumen de vectores proyectado para Unifranz no justifica una base vectorial dedicada. pgvector con índice HNSW cubre el caso de uso con menor complejidad operacional: un solo servicio de base de datos, transacciones unificadas entre datos relacionales y vectoriales, y filtrado nativo por SQL antes del retrieval semántico.

### MinIO para almacenamiento de archivos

Los archivos entregados por estudiantes y el material RAG docente no se almacenan en PostgreSQL. MinIO corre en el mismo VPS de aplicación como servicio Docker, expone una API S3-compatible, y permite migrar a un bucket cloud cambiando únicamente las credenciales de conexión sin tocar el código.

### Router de IA híbrido: Ollama primario + Vercel AI Gateway fallback

Unifranz prefiere procesar datos de estudiantes en sus propios servidores. Ollama en VPS dedicado con GPU es el backend primario. Vercel AI Gateway actúa como fallback automático cuando Ollama no está disponible, y como override cuando el administrador activa el modo cloud.

### Dos VPS independientes con roles separados

VPS 1 (CX42, €16.40/mes) corre todos los servicios de aplicación. VPS 2 (GEX44, €184/mes) corre exclusivamente Ollama con GPU. Esta separación permite reiniciar, actualizar o escalar el modelo de IA sin afectar la disponibilidad del servidor principal.

---

## 12. Vistas del Sistema

### Orden de desarrollo de vistas

**Bloque 1 — Autenticación** *(completado)*
- Login
- Recuperación de contraseña

**Bloque 2 — Panel del Administrador** *(especificado)*
- Dashboard general
- Gestión de plantillas de formato institucional (por institución y carrera)
- Gestión de mallas curriculares
- Marcado de materias como elegibles para práctica asistida
- Configuración de proveedor de IA (tres modos: local / auto / cloud)
- Gestión de docentes
- Módulo de resolución de problemas reales:
  - Formulario de descripción de problemática
  - Vista de propuesta del sistema (materias por carrera)
  - Selección de materias y asignación de docentes
  - Creación de grupos interdisciplinarios
  - Historial de consultas

**Bloque 3 — Panel del Docente** *(especificado)*
- Dashboard con resumen de cursos, actividades pendientes y alertas recientes
- Cursos normales vs. cursos de programas interdisciplinarios (diferenciados visualmente)
- Creación y gestión de actividades por tipo (examen, tarea, informe, práctica, proyecto)
- Configuración de proyectos (etapas, Git, defensa oral)
- Vista de alertas y reportes por actividad
- Calificación con evidencia (trazabilidad, análisis IA, reporte de formato)
- Panel de analítica por curso y estudiante
- Gestión de material del RAG docente
- Habilitación de co-docencia
- Cola de apelaciones pendientes por actividad
- Resolución de apelaciones con vista de cadena de evidencia

**Bloque 4 — Entorno del Estudiante** *(completado — React + Vite, paleta naranja #E85D24)*
- Dashboard personal con actividades activas y calificaciones
- Entorno de resolución — modo examen
- Entorno de resolución — modo práctica (con feedback de IA)
- Subida de tareas
- Subida de informes
- Subida de etapas de proyecto (.zip + enlace Git)
- Sesión de defensa oral
- Vista de calificaciones y feedback
- Formulario de apelación formal (analysis_dispute / grade_dispute)
- Estado de apelaciones presentadas

---

## 13. Estado de Implementación

### Milestones completados

| Milestone | Descripción | Estado |
|---|---|---|
| M1 | Fundación del proyecto (estructura Go, Docker, config, health) | ✅ Completo |
| M2 | Autenticación y roles (JWT, bcrypt, Google OAuth, refresh token) | ✅ Completo |
| M3 | Gestión académica (carreras, materias, cursos, grupos, actividades, co-docencia) | ✅ Completo |
| M4 | File storage (MinIO, submissions, gestión de archivos) | ✅ Completo |
| M5 | Entorno de resolución y trazabilidad (WebSocket, trace events, exam sessions) | ✅ Completo |
| M6 | AI Router (Ollama, Vercel AI Gateway, tres modos, hot-swap) | ✅ Completo |
| M7 | Dual RAG Pipeline (ingesta, chunking, vectorización, retrieval curricular + docente) | ✅ Completo |
| M8 | Motor de detección (heurísticas, LLM, grade decisions, score threshold) | ✅ Completo |
| M9 | Validación de formato (PDF/pdfcpu, DOCX, format reports) | ✅ Completo |
| M10 | Gestión de proyectos (etapas, análisis Git, worker jobs, defensa oral) | ✅ Completo |
| M11 | Alertas y analítica (SSE stream, tokens Redis, 5 tipos de alerta, métricas) | ✅ Completo |
| M12 | Módulos administrativos (docentes, plantillas, curriculum, interdisciplinario) | ✅ Completo |
| M13 | Apelaciones y auditoría final (hash chain SHA-256, evidencia verificable, tests integración) | ✅ Completo |

### Tests de integración

Suite completa con `//go:build integration` ejecutada contra base de datos real (sin mocks):

| Test | Descripción |
|---|---|
| `TestExamSessionFlow` | Sesión de examen → trace events → análisis heurístico |
| `TestSubmissionGradeFlow` | Entrega → grade decision → constraint UNIQUE → override_reason |
| `TestAppealFileAndReject` | Ciclo completo: appeal → decisión docente → verificación hash chain |
| `TestAppealEscalateAndAdminResolve` | grade_dispute → escalación → resolución admin |
| `TestEvidenceChainTampering` | Modificar un evento → chain válida=false |

Ejecución: `go test ./tests/integration/... -tags integration -v` con `DATABASE_URL_TEST` configurado.

### Correcciones identificadas durante validación del stack

**Bug 1 — Middleware global involuntario (Fiber v2.52.0)**
`app.Group("", handlers...)` con prefijo vacío llama internamente a `app.register(methodUse, "", ...)`, registrando los handlers como middleware USE global que intercepta todas las rutas, incluyendo `POST /auth/login`. Corregido en `internal/analytics/module.go` registrando las rutas directamente con middleware por ruta.

**Bug 2 — Scan de NULL en campos no-pointer**
`GradeDecisionResult.SuggestedGradeRange` y `OverrideReason` eran `string` pero las columnas en PostgreSQL son `TEXT` nullable. pgx v5 no puede escanear NULL en un `string` no-pointer. Corregido cambiando ambos campos a `*string` en `internal/detection/model.go`.

### Configuración Docker Compose de desarrollo

```yaml
postgres:
  image: pgvector/pgvector:pg16
  ports: ["5433:5432"]           # Puerto 5433 para evitar conflicto con Postgres nativo de Windows
  environment:
    POSTGRES_HOST_AUTH_METHOD: trust  # Requerido para conexiones desde red Docker

api:
  environment:
    APP_ENV: development
    JWT_SECRET: nextlab-dev-secret-key-change-in-production-env
    # En desarrollo, POST /admin/docentes acepta test_password para pruebas sin OAuth
```

El puerto `5433` (en lugar de `5432`) es necesario en entornos Windows donde Postgres puede estar instalado como dependencia de otro software y ocupar el puerto estándar.

---

*Documento preparado para el proyecto integrador de Elder — Universidad Privada Franz Tamayo, Unifranz, Cochabamba. Versión 5.0 — Mayo 2026.*
