# Pruebas del sistema

Desde `nextlab-api`, ejecute:

```bash
./scripts/test-system.sh
```

El comando crea el proyecto Docker `nextlab-system-test`, con volúmenes propios, en el puerto `18080`. No usa la base de datos, los volúmenes ni una clave de IA de desarrollo. Al finalizar elimina exclusivamente esos recursos. Para conservarlos y revisarlos use `KEEP_TEST_STACK=1 ./scripts/test-system.sh`; después puede eliminarlos con:

```bash
docker compose -p nextlab-system-test -f docker-compose.test.yml down --volumes
```

La prueba verifica el recorrido entre componentes: autenticación y autorización por rol, gestión académica, sesión y trazabilidad, carga en MinIO, proceso asíncrono de formato/detección, analíticas, alertas, notificaciones, router de IA, investigación por etapas, defensa oral, plantillas administrativas, control de precondiciones de reentrenamiento ML y apelaciones con evidencia. La IA es un doble local, por lo que se valida el contrato HTTP y el flujo de la aplicación sin coste ni resultados no deterministas.

También ejecute las comprobaciones estáticas del cliente:

```bash
cd ../nextlab-web
pnpm lint && pnpm build
```

Las integraciones que por definición requieren un tercero real (OAuth de Google, un repositorio Git remoto y un webhook externo) quedan aisladas del banco automático. Sus rutas se validan con los contratos de cada módulo; deben probarse con una cuenta, repositorio y receptor de webhook específicamente destinados a QA, nunca con credenciales de producción.
