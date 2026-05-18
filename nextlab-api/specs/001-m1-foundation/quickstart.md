# Quickstart: M1 — Fundación del Proyecto

**Time to first green health check: ~5 minutes**

---

## Prerequisites

- Docker Desktop (or Docker Engine + Compose v2) installed and running
- Git

---

## Steps

### 1. Clone and configure

```bash
git clone <repo-url>
cd nextlab-api
cp .env.example .env
```

The `.env.example` file contains safe placeholder values for local development.
No edits are required to get the stack running for the first time.

---

### 2. Start all services

```bash
docker compose up --build
```

This command:
- Builds the `api` and `worker` binaries inside the Docker builder stage
- Starts PostgreSQL, Redis, and MinIO
- Waits for all three to pass their health checks
- Starts the API (which runs pending migrations before accepting requests)
- Starts the worker

Expected output (last few lines):
```
nextlab-api-api-1     | {"time":"...","level":"INFO","msg":"migrations applied","service":"api","correlation_id":"system","count":1}
nextlab-api-api-1     | {"time":"...","level":"INFO","msg":"server started","service":"api","correlation_id":"system","port":"8080"}
nextlab-api-worker-1  | {"time":"...","level":"INFO","msg":"worker started","service":"worker","correlation_id":"system"}
```

---

### 3. Verify health

```bash
curl -s http://localhost:8080/health | jq .
```

Expected response:
```json
{
  "status": "ok",
  "dependencies": {
    "database": { "status": "ok", "latency_ms": 2 },
    "cache":    { "status": "ok", "latency_ms": 1 },
    "storage":  { "status": "ok", "latency_ms": 4 }
  }
}
```

HTTP status must be `200`.

---

### 4. Roll back migrations (optional)

```bash
docker compose run --rm migrate down 1
```

This reverts the last migration (drops the pgvector extension). Re-apply with:

```bash
docker compose run --rm migrate up
```

---

## Environment Variables Reference

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | `postgres://...` | Full PostgreSQL connection string |
| `REDIS_ADDR` | yes | `redis:6379` | Redis address |
| `MINIO_ENDPOINT` | yes | `minio:9000` | MinIO endpoint |
| `MINIO_ACCESS_KEY` | yes | `minioadmin` | MinIO access key |
| `MINIO_SECRET_KEY` | yes | `minioadmin` | MinIO secret key |
| `API_PORT` | no | `8080` | Port the API listens on |
| `LOG_LEVEL` | no | `info` | Minimum log severity |
| `APP_ENV` | no | `development` | `development` or `production` |

---

## Stopping the stack

```bash
docker compose down          # stop and remove containers (data volumes preserved)
docker compose down -v       # stop and remove containers AND volumes (fresh start)
```

---

## Validating acceptance criteria

| Criterion | Command | Expected |
|---|---|---|
| SC-001: All services ready in 60s | `time docker compose up --build` (images cached) | `real` < 60s |
| SC-002: Health check < 500ms | `curl -w "\n%{time_total}s\n" http://localhost:8080/health` | < 0.5s |
| SC-005: All logs are valid JSON | `docker compose logs api \| jq .` | No parse errors |
| SC-006: Migration rollback clean | `docker compose run --rm migrate down 1 && psql ... -c '\dx'` | No vector extension listed |
