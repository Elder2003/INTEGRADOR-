# Sistema de Auditoría Académica con IA
## Documento de Diseño y Arquitectura
**Universidad Privada Franz Tamayo — Unifranz**
**Versión 3.0 — 2026**

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

---

## 1. Visión General

### El Problema

La masificación de herramientas de inteligencia artificial generativa ha transformado el entorno educativo universitario. Los estudiantes de carreras tecnológicas y exactas recurren cada vez más a estas herramientas para obtener soluciones algorítmicas de forma inmediata, evitando el proceso cognitivo real que implica analizar, abstraer y resolver problemas. Los métodos de evaluación tradicionales no están diseñados para detectar este comportamiento porque califican el resultado final, no el proceso de construcción.

Esto genera un riesgo concreto: las calificaciones dejan de reflejar el verdadero nivel de razonamiento lógico del estudiante. A largo plazo, la institución puede estar certificando profesionales con deficiencias críticas en su capacidad analítica.

### Qué es este sistema

Una plataforma académica integral que cubre todo el ciclo de actividades académicas digitales: examina el proceso cognitivo real de los estudiantes mientras resuelven evaluaciones, analiza el contenido de tareas e informes entregados, valida el cumplimiento de parámetros institucionales de formato, y audita el proceso de desarrollo de proyectos. Con esa información, el sistema detecta patrones compatibles con el uso no declarado de IA y alerta al docente para que tome una decisión informada.

Adicionalmente, permite a los administradores institucionales describir problemáticas del mundo real y obtener una propuesta de qué materias de qué carreras pueden articularse para resolverla, con la capacidad de crear grupos de trabajo interdisciplinarios directamente desde esa propuesta.

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
- Recepción y resolución de apelaciones escaladas

### Docente

Crea y gestiona sus cursos, grupos y actividades. Sube el material de referencia que el sistema usará para generar feedback contextualizado. Recibe alertas y reportes cuando el sistema detecta comportamientos sospechosos o incumplimientos de formato. Revisa la evidencia, valida o descarta la alerta, y asigna la calificación final. Puede habilitar co-docencia en actividades específicas e invitar a otro docente como guía.

El docente controla los horarios de entrega de todas las actividades. Las entregas fuera de plazo quedan marcadas en la auditoría pero el sistema nunca bloquea — el docente decide si las acepta.

### Co-docente

Un docente habilitado puntualmente por el docente titular para participar en una actividad específica. Puede ver las alertas y calificar las entregas de esa actividad. No tiene acceso al resto del curso ni a otras evaluaciones.

### Estudiante

Resuelve las evaluaciones dentro del entorno del sistema. En actividades prácticas tiene acceso a feedback de la IA basado en el contenido subido por su docente. En modo examen resuelve sin asistencia y con auditoría activa. Sabe que existe un sistema de auditoría, pero no tiene acceso a los detalles del análisis sobre sus sesiones. Puede apelar calificaciones formalmente.

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

### 4.4 Detección de Patrones

Cuando el estudiante entrega o el tiempo finaliza, el sistema analiza la trazabilidad acumulada en dos capas:

**Heurísticas deterministas:** Reglas fijas con umbrales configurables. Detectan comportamientos objetivamente sospechosos: bloque grande de código en menos de dos segundos, velocidad de tipeo sostenida imposible para un humano, código perfecto sin ningún borrado, inactividad prolongada seguida de solución completa.

**Análisis por IA:** El sistema envía al LLM el resumen de flags heurísticos, el contenido entregado, el contexto del RAG curricular (qué se espera que sepa el estudiante) y el RAG docente (material declarado válido por el docente). El modelo produce un análisis justificado en lenguaje natural.

El resultado es una calificación sugerida con justificación. El sistema guarda tanto la calificación sugerida como la final asignada por el docente. Si difieren, queda registrado el motivo.

### 4.5 Validación de Formato de Informes

Módulo que analiza los archivos PDF y Word entregados como informes. Compara el documento contra la plantilla institucional vigente y genera un reporte de cumplimiento con observaciones específicas por parámetro.

**Para PDF:** Extrae texto con estructura, detecta imágenes y verifica pie de figura, usa heurísticas visuales para márgenes y fuente.

**Para Word (.docx):** Accede directamente a estilos de título, párrafos, márgenes, fuente, tamaño de letra y estructura de secciones.

El sistema siempre acepta la entrega. El reporte de formato llega al docente junto con el reporte de detección de IA.

### 4.6 Gestión de Proyectos

Permite al docente configurar proyectos con etapas, repositorio Git y defensa oral. Gestiona el ciclo de vida de cada etapa: publicación, recepción de entregas (.zip + enlace Git), análisis de cada componente y generación del reporte unificado por etapa.

Mantiene el historial de versiones de cada entrega (primera entrega vs. entregas tardías) y el análisis de coherencia acumulado entre etapas.

### 4.7 Gestión de Contenido Docente para el RAG

Los docentes pueden subir material de referencia para dos propósitos: generar feedback en modo práctica y enriquecer el análisis post-actividad con el contexto del curso.

Tipos de contenido soportados:
- **Archivos (PDFs, diapositivas, apuntes):** Fragmentación semántica por sección
- **Libros comerciales:** Solo se indexa estructura (tabla de contenidos), no el cuerpo del texto (protección de copyright)
- **Libros de acceso abierto:** Indexados completos con fragmentación semántica
- **Rúbricas:** Nunca pasan por retrieval — siempre se inyectan completas al prompt
- **Enlaces web:** Capturados y almacenados localmente en el momento de la carga
- **Videos de YouTube:** Transcript extraído automáticamente si está disponible
- **Repositorios de código:** Indexados según ruta indicada por el docente

El sistema clasifica automáticamente el tipo de documento y aplica la estrategia correspondiente. El docente recibe un mensaje claro del resultado por cada recurso subido.

### 4.8 Alertas y Notificaciones

Cuando el análisis supera el umbral configurado, el sistema genera una alerta dirigida al docente. La alerta incluye los flags heurísticos activados, el análisis en lenguaje natural de la IA, y un enlace directo para revisar la sesión completa.

El docente recibe la alerta en tiempo real en su panel web. Las alertas están filtradas por materia: un docente solo recibe alertas de sus propios estudiantes, salvo en actividades con co-docencia habilitada.

El docente puede marcar la alerta como confirmada, falso positivo o inconclusa. Este feedback queda registrado y alimenta la mejora progresiva del sistema.

### 4.9 Panel de Analítica para Docentes

Vista agregada del comportamiento de un grupo a lo largo del tiempo. El docente puede ver tendencias por estudiante, por actividad y por materia. No es un sistema punitivo — está diseñado para identificar dónde reforzar la enseñanza.

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

### 4.12 Configuración del Proveedor de IA

El administrador puede cambiar el modelo de IA activo desde un panel administrativo sin intervención técnica ni reinicio del sistema. El cambio aplica a partir del siguiente análisis. Los resultados anteriores quedan asociados al proveedor que los generó.

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

### Flujo 9: Apelación formal de una calificación

1. El estudiante solicita formalmente la revisión de su calificación.
2. El docente puede escalar la revisión al administrador institucional.
3. El administrador accede a la sesión completa: trazabilidad inmutable, análisis de la IA, decisiones del docente con sus justificaciones.
4. El administrador emite una resolución final registrada en el sistema.
5. Toda la cadena queda auditada: quién revisó, qué decidió, cuándo.

---

## 6. Arquitectura del Sistema

### Visión general

El sistema se divide en dos procesos principales que comparten la misma base de código pero se despliegan de forma independiente:

**El servidor principal** atiende todas las peticiones HTTP y las conexiones WebSocket de los estudiantes durante los exámenes. Gestiona la autenticación, sirve los módulos de gestión académica, recibe los eventos de trazabilidad en tiempo real, y procesa la carga de contenido docente al RAG.

**El worker de análisis** es un proceso separado que consume una cola de trabajos (Redis Streams). Cuando una sesión termina o una entrega llega, el servidor encola el trabajo. El worker lo procesa de forma asíncrona: ejecuta las heurísticas, valida el formato si aplica, consulta ambas bases de conocimiento, llama al modelo de IA configurado, y guarda el resultado. Esto garantiza que el análisis nunca afecte el rendimiento del entorno de resolución.

### Stack Tecnológico

- **Backend:** Go (Gin / Fiber) + PostgreSQL + Redis
- **Frontend:** React + Vite
- **IA — Capa de abstracción multimodelo:** Enruta solicitudes según costo, disponibilidad y complejidad del caso
- **IA — Modelos en la nube:** OpenAI API (GPT-4o), Anthropic (Claude), Gemini (Google)
- **IA — Modelo local (fallback):** Ollama con LLaMA 3 / Mistral — sin costo de API
- **Base vectorial:** Qdrant (RAG curricular y RAG docente en namespaces separados)
- **Comunicación tiempo real:** WebSocket para trazabilidad, SSE para notificaciones al docente
- **Ejecución sandboxed:** Judge0 (cloud o self-hosted)
- **DevOps:** Docker, GitHub Actions
- **Gestión:** Git, Jira / Trello

### Separación de responsabilidades

Módulos internos con responsabilidad única:
- **Autenticación:** gestión de identidad, roles y sesiones
- **Gestión académica:** cursos, grupos, actividades y su configuración
- **Sesiones de resolución:** ciclo de vida de cada intento
- **Trazabilidad:** captura, validación y almacenamiento inmutable de eventos
- **Validación de formato:** parseo y análisis de PDF y Word contra plantilla institucional
- **Gestión de proyectos:** etapas, entregables, análisis Git y defensas
- **RAG Curricular:** ingesta, vectorización y búsqueda de mallas curriculares
- **RAG Docente:** ingesta, clasificación, fragmentación y búsqueda de contenido docente
- **Detección:** heurísticas y orquestación del análisis con IA
- **Alertas:** generación, enrutamiento y estado de notificaciones
- **Analítica:** agregación de datos para dashboards de docentes
- **Resolución de problemas:** módulo de consulta interdisciplinaria
- **Proveedores de IA:** capa de abstracción multimodelo

### Capa de abstracción multimodelo

El sistema no llama directamente a ningún proveedor de IA. Existe una capa de abstracción que decide qué modelo usar en cada solicitud según:

- **Complejidad del caso:** Alto riesgo → modelo en la nube de mayor calidad
- **Costo:** Si el presupuesto mensual de API está alto → Ollama local
- **Disponibilidad:** Si un proveedor falla → conmuta automáticamente al siguiente
- **Configuración del administrador:** El proveedor preferido se configura desde el panel

### Comunicación en tiempo real

- **WebSocket:** Eventos de trazabilidad del estudiante al servidor durante exámenes
- **SSE (Server-Sent Events):** Notificaciones unidireccionales al panel del docente
- **Push móvil:** Notificaciones mediante servicios nativos de cada plataforma

---

## 7. Inteligencia Artificial en el Sistema

### Dos capas de detección

**Heurísticas deterministas:** Reglas con umbrales numéricos. Instantáneas, sin costo de API. Detectan: más del 60% del código llegó por paste, velocidad de tipeo sostenida imposible para un humano, solución sin ningún borrado a pesar de su complejidad, inactividad prolongada seguida de entrega inmediata.

**Análisis por LLM:** El modelo recibe el resumen de flags heurísticos, el contenido entregado, el contexto del RAG curricular y el RAG docente. Produce análisis en lenguaje natural con razones específicas y una evaluación de consistencia con el nivel académico esperado.

### Proveedor de IA intercambiable

El cambio de proveedor es una operación administrativa. Aplica inmediatamente sin reiniciar nada. Los análisis anteriores quedan asociados al proveedor que los generó.

### Evolución progresiva

El sistema funciona desde el primer día con heurísticas y el LLM configurado. No requiere datos históricos previos. A medida que los docentes validan o rechazan alertas, el sistema acumula casos etiquetados que pueden usarse para entrenar un modelo de clasificación específico al final de cada semestre.

---

## 8. Sistema RAG Dual

Dos bases de conocimiento independientes. **Nunca se mezclan en una misma consulta.**

### 8.1 RAG Curricular

**Propósito:** Contexto sobre qué se espera que sepa un estudiante de esa materia en ese semestre. Uso interno del sistema — no visible para estudiantes ni docentes directamente.

**Fuente:** Planes instruccionales subidos por el administrador en formato XLSX estándar de Unifranz.

**Filtrado:** Siempre se filtra por `carrera + semestre + codigo_materia` antes de buscar por similitud semántica.

**Usos:**
- Motor de detección en exámenes
- Módulo de resolución de problemas reales (con proceso de retrieval completamente distinto)

### 8.2 RAG Docente

**Propósito:** Que el LLM responda basado exclusivamente en lo que el docente definió como contenido válido para su curso o evaluación.

**Usos:**
- Feedback en modo práctica
- Análisis post-actividad (detección de IA en tareas, informes y proyectos)
- Calificación sugerida en exámenes

**Jerarquía de retrieval (presupuesto de 3.000 tokens):**

| Prioridad | Tipo | Scope | Mecanismo |
|---|---|---|---|
| 1 | Rúbrica | Evaluación específica | Inyección directa, siempre |
| 2 | Material indexable | Evaluación específica | Retrieval por similitud |
| 3 | Material indexable | Curso completo | Retrieval por similitud |
| 4 | Referencias bibliográficas | Evaluación o curso | Inyección directa, siempre |
| 5 | Contexto curricular (RAG 1) | Materia + semestre | Siempre presente, piso mínimo |

---

## 9. Seguridad y Privacidad

### Principios generales

- **No invasividad:** solo se captura lo que ocurre dentro del navegador
- **Proporcionalidad:** la auditoría solo se activa donde existe riesgo real
- **Transparencia:** los estudiantes saben que existe un sistema de auditoría

### Autenticación y acceso

Tokens de acceso de corta duración (15 minutos) combinados con tokens de refresco almacenados en el servidor. Cada rol tiene acceso estrictamente limitado a su función.

### Datos de trazabilidad

Registros inmutables una vez creados. El acceso está limitado al docente de la materia y, en actividades con co-docencia, al co-docente asignado.

### Contenido del RAG docente

Accesible únicamente en el scope declarado. Filtros de `docente_id` y `curso_id` antes de cualquier operación de retrieval. Contenido sanitizado antes de indexar.

---

## 10. Requerimientos

### Requerimientos Funcionales

**RF-01 Gestión de Usuarios y Roles**
Cuatro roles: Administrador Institucional, Docente, Co-docente y Estudiante. Permisos estrictamente definidos por rol.

**RF-02 Gestión de Cursos y Actividades**
Docentes crean cursos, grupos y actividades de cinco tipos: examen, tarea, informe, práctica y proyecto. Cada actividad tiene configuración de tipo, modalidad, fecha límite y estado de auditoría.

**RF-03 Entorno de Resolución Web**
Área de trabajo en el navegador. Soporta editor de código con ejecución, editor de texto libre, selección múltiple y carga de archivos. Modo práctica con feedback, modo examen sin feedback.

**RF-04 Captura de Trazabilidad Cognitiva**
Registro en tiempo real de keystroke, paste, borrados, inactividades, ejecuciones y tiempos entre eventos. Registro inmutable.

**RF-05 Motor de Ejecución Sandboxed**
Entorno aislado para ejecutar código. En práctica devuelve resultado y feedback. En examen solo el resultado.

**RF-06 Motor de Detección de Patrones**
Heurísticas deterministas + LLM con contexto RAG dual. Produce score y justificación en lenguaje natural.

**RF-07 Validación de Formato de Informes**
Parseo de PDF y Word (.docx). Contraste contra plantilla institucional. Reporte de cumplimiento por parámetro. Siempre acepta la entrega.

**RF-08 Gestión de Proyectos por Etapas**
Configuración de etapas con tipo y fecha límite. Recepción de .zip + enlace Git por etapa. Análisis de historial de commits. Análisis de coherencia entre etapas. Defensa oral con auditoría activa.

**RF-09 Generación de Alertas y Reportes**
Notificaciones automáticas al docente cuando el análisis supera el umbral. Reporte unificado por tipo de actividad. Acceso directo a la sesión para revisión.

**RF-10 Panel de Analítica para Docentes**
Vista agregada de comportamiento por estudiante, actividad y materia a lo largo del tiempo.

**RF-11 Módulo de Resolución de Problemas Reales**
El administrador describe una problemática. El sistema propone materias articuladas. El administrador selecciona, asigna docentes y crea grupos. El programa queda activo como cursos especiales en el panel de cada docente asignado.

**RF-12 Gestión de Plantillas de Formato Institucional**
El administrador define y actualiza parámetros de formato por institución y carrera. Los docentes no intervienen en este proceso.

**RF-13 Gestión Curricular (RAG Curricular)**
Carga, validación, procesamiento e indexación de planes de formación. Soporte para múltiples carreras sin cambios estructurales.

**RF-14 Gestión de Contenido Docente (RAG Docente)**
Docentes suben archivos y enlaces. El sistema clasifica, procesa y fragmenta automáticamente. Detección de copyright con manejo diferenciado.

**RF-15 Configuración de Proveedor de IA Multimodelo**
Capa de abstracción que enruta solicitudes según costo, disponibilidad y complejidad. Cambio de proveedor desde panel administrativo sin reinicio.

**RF-16 Marcado de Materias para Práctica Asistida**
El administrador marca materias como elegibles o no elegibles para actividades prácticas con asistencia de IA.

**RF-17 Co-docencia**
El docente titular habilita a otro docente como guía de una actividad específica con acceso limitado.

**RF-18 Calificación con Auditoría**
El sistema sugiere calificación. El docente puede editarla. Ambos valores quedan registrados. Si difieren, se registra el motivo.

**RF-19 Registro de Feedback de Alertas**
Docentes marcan alertas como confirmadas, falsos positivos o inconclusas. Alimenta la mejora progresiva.

**RF-20 Apelación Formal**
Flujo escalable al administrador con registro completo de la cadena de decisiones y acceso a trazabilidad inmutable.

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
Datos de trazabilidad y RAG docente accesibles únicamente por roles con autorización explícita.

**RNF-07 Transmisión Segura**
Toda la comunicación usa cifrado en tránsito (HTTPS/TLS).

**RNF-08 SLA del Worker de Análisis**
El análisis asíncrono se completa en máximo 3 minutos desde la entrega.

**RNF-09 Trazabilidad de Ingestas**
Cada carga de material queda registrada con autor, fecha, archivo, resultado y advertencias.

**RNF-10 Proveedor de IA Intercambiable**
Cambio de proveedor sin intervención técnica ni tiempo de inactividad.

**RNF-11 Registro Inmutable de Evidencia**
Eventos de trazabilidad y análisis no pueden modificarse una vez creados.

**RNF-12 Consistencia del Contenido Capturado**
Los recursos externos se almacenan localmente en el momento de la carga. El sistema nunca depende de disponibilidad externa durante una evaluación activa.

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

El administrador determina qué materias pueden tener actividades prácticas con asistencia de IA. El docente decide si crear o no esa actividad dentro de las materias elegibles. Esto evita que materias donde la asistencia de IA no tiene sentido (álgebra, física) tengan ese tipo de actividad disponible.

### Programas interdisciplinarios como ciudadanos de primera clase

El módulo de resolución de problemas reales no termina en una propuesta de texto. Genera grupos activos con docentes asignados que aparecen en el panel de cada docente como cursos especiales. Cierra el ciclo entre el análisis curricular y la acción institucional.

### Dos RAGs independientes con propósitos distintos

El RAG curricular y el RAG docente nunca se mezclan en una misma consulta. El primero alimenta la detección con expectativas académicas del semestre. El segundo alimenta el feedback y la calificación sugerida con el contenido específico declarado por el docente.

### La rúbrica nunca pasa por retrieval

Las rúbricas se inyectan siempre completas al prompt. Fragmentar una rúbrica produce calificaciones con criterios incompletos.

### El docente siempre tiene la última palabra

El sistema nunca toma decisiones automáticas sobre calificaciones o sanciones. Su rol es producir evidencia organizada y análisis justificados. La autoridad académica permanece en el docente.

### Mejora progresiva sin dependencia de datos históricos

El sistema funciona desde el primer día. A medida que se acumulan casos validados por docentes, puede incorporar un modelo de clasificación específico. Esta progresión es deliberada y controlada.

### Mecanismo de apelación como parte del diseño

El flujo de apelación formal no es un parche posterior. La trazabilidad inmutable combinada con un flujo de revisión escalable al administrador hace que el sistema sea institucionalmente sostenible.

---

## 12. Vistas del Sistema

### Orden de desarrollo de vistas

**Bloque 1 — Autenticación**
- Login
- Recuperación de contraseña

**Bloque 2 — Panel del Administrador**
- Dashboard general
- Gestión de plantillas de formato institucional (por institución y carrera)
- Gestión de mallas curriculares
- Marcado de materias como elegibles para práctica asistida
- Configuración de proveedor de IA (multimodelo)
- Gestión de docentes
- Módulo de resolución de problemas reales:
  - Formulario de descripción de problemática
  - Vista de propuesta del sistema (materias por carrera)
  - Selección de materias y asignación de docentes
  - Creación de grupos interdisciplinarios
  - Historial de consultas

**Bloque 3 — Panel del Docente**
- Dashboard con resumen de cursos, actividades pendientes y alertas recientes
- Cursos normales vs. cursos de programas interdisciplinarios (diferenciados visualmente)
- Creación y gestión de actividades por tipo (examen, tarea, informe, práctica, proyecto)
- Configuración de proyectos (etapas, Git, defensa oral)
- Vista de alertas y reportes por actividad
- Calificación con evidencia (trazabilidad, análisis IA, reporte de formato)
- Panel de analítica por curso y estudiante
- Gestión de material del RAG docente
- Habilitación de co-docencia

**Bloque 4 — Entorno del Estudiante**
- Dashboard personal con actividades activas y calificaciones
- Entorno de resolución — modo examen
- Entorno de resolución — modo práctica (con feedback de IA)
- Subida de tareas
- Subida de informes
- Subida de etapas de proyecto (.zip + enlace Git)
- Sesión de defensa oral
- Vista de calificaciones y feedback
- Formulario de apelación formal

---

*Documento preparado para el proyecto integrador de Elder — Universidad Privada Franz Tamayo, Unifranz, Cochabamba. Versión 3.0 — Mayo 2026.*
