# Integración continua con Jenkins

El repositorio incluye un [Jenkinsfile](Jenkinsfile) que registra automáticamente en Jenkins:

- resultados unitarios Go en formato JUnit;
- el resultado de la prueba integral aislada como una prueba JUnit;
- el log completo de cada suite dentro de `reports/`;
- el build distribuible del frontend (`nextlab-web/dist/`).

No contiene despliegue ni secretos de producción. El objetivo de este pipeline es validar y registrar calidad antes de un despliegue.

## 1. Preparar el agente de Jenkins

Asigne al nodo que ejecutará el trabajo la etiqueta **`nextlab-ci`**. El usuario del agente debe poder ejecutar Docker sin `sudo` y contar con:

| Requisito | Motivo |
|---|---|
| Docker Engine y Docker Compose v2 | `test-system.sh` levanta PostgreSQL, Redis, MinIO, API, worker y el doble de IA. |
| Go 1.26.1 o compatible | Ejecuta y compila `nextlab-api`. |
| Node.js LTS con Corepack | Instala la versión bloqueada de pnpm y compila el cliente. |
| `curl` y `jq` | Son requisitos explícitos de `test-system.sh`. |
| Acceso a internet o caché interna de módulos | Descarga módulos Go/pnpm y el conversor JUnit durante la primera ejecución. |

También instale en Jenkins los plugins **Pipeline**, **Git** y **JUnit**. El pipeline usa comandos Docker del agente, por lo que no necesita montar el socket Docker dentro de un contenedor de Jenkins.

El puerto host `18080` queda reservado temporalmente por la prueba integral. La directiva `disableConcurrentBuilds()` evita choques entre ejecuciones del mismo job; no ejecute otros jobs NextLab que usen ese puerto en el mismo agente al mismo tiempo.

## 2. Crear el job

La opción recomendada es un **Multibranch Pipeline**:

1. En Jenkins, cree *New Item → Multibranch Pipeline*.
2. Agregue la fuente GitHub/Git y el repositorio `https://github.com/Elder2003/INTEGRADOR-.git`.
3. Configure la credencial de Jenkins sólo si el repositorio es privado. Debe tener lectura, no permisos de administración.
4. En *Build Configuration*, deje `Jenkinsfile` como script path.
5. Guarde y ejecute *Scan Multibranch Pipeline Now*.

Para un único branch, cree *Pipeline → Pipeline script from SCM*, seleccione Git, use la misma URL y `Jenkinsfile` como script path.

## 3. Activar la ejecución automática

En GitHub, agregue un webhook hacia:

```text
https://<tu-jenkins>/github-webhook/
```

Seleccione eventos de **push** y **pull request**. Si se usa GitHub Branch Source, el escaneo de ramas/PR se actualiza con ese webhook. Proteja Jenkins con HTTPS y autenticación; no exponga un endpoint de administración público.

Como alternativa en una red sin webhooks, configure un escaneo periódico del multibranch. Es menos inmediato y consume más recursos.

## 4. Qué ocurre en cada build

1. Comprueba herramientas del agente.
2. Instala dependencias bloqueadas del frontend y ejecuta `pnpm lint` y `pnpm build`.
3. Ejecuta `go test -v ./...` y transforma la salida a JUnit para la sección **Tests** de Jenkins.
4. Ejecuta `nextlab-api/scripts/test-system.sh`. La prueba crea una pila Docker propia, usa una IA determinista local y limpia sus recursos al terminar. Jenkins registra ese recorrido como `nextlab-system / end-to-end`.
5. Publica reportes, logs y `dist/` como artefactos. Incluso si una suite falla, su log se conserva.

Los artefactos se retienen para 15 builds y los builds para 30, según el `Jenkinsfile`; cambie esos valores si la política de la organización exige otra retención.

## 5. Resultados y diagnóstico

En el build de Jenkins consulte:

- **Tests**: tendencias y detalle de las pruebas Go y la prueba integral.
- **Artifacts**: `reports/go-unit.log`, `reports/go-unit.xml`, `reports/system-test.log`, `reports/system-test.xml` y el frontend compilado.
- **Console Output**: fallo de herramientas, Docker o comandos del pipeline.

Un fallo de `test-system.sh` deja el log de API/worker dentro de `system-test.log`, porque el script imprime esos diagnósticos antes de terminar.

## 6. Credenciales y seguridad

- No añada `JWT_SECRET`, credenciales MinIO reales, tokens de IA ni una base de producción: la prueba integral ya usa valores de prueba y volúmenes aislados.
- Si posteriormente se añade una fase de despliegue, cree un job/stage separado que lea secretos desde **Jenkins Credentials** y limite las credenciales al entorno correspondiente.
- Para OAuth, Git remoto y webhooks reales, use cuentas, repositorios y receptores exclusivos de QA; nunca credenciales personales o productivas.
- Mantenga actualizado el agente Docker/Go/Node y restrinja quién puede modificar `Jenkinsfile`, ya que un pipeline tiene acceso al agente que lo ejecuta.

## 7. Siguiente ampliación recomendada

El pipeline registra las pruebas existentes. Para cubrir el plan completo se recomienda añadir Vitest + React Testing Library para frontend y Playwright para los flujos E2E descritos en [TEST_PLAN.md](TEST_PLAN.md); sus resultados JUnit pueden publicarse en el mismo directorio `reports/` sin cambiar la configuración `post`.
