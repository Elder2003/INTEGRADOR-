# Quickstart: M2 — Authentication and Roles

## Prerequisites

M1 must be running: `docker compose up -d` with all services healthy.

---

## 1. Apply M2 migrations

Migrations run automatically when the API starts. After rebuilding:

```bash
docker compose up --build -d
docker compose logs api | grep "migrations applied"
# → {"msg":"migrations applied","version":2,...}
```

---

## 2. Seed the admin user

Connect to PostgreSQL and insert a seed admin account:

```bash
docker compose exec postgres psql -U postgres -d nextlab
```

```sql
-- Generate a bcrypt hash for "admin123" first, then insert:
INSERT INTO users (email, password_hash, role)
VALUES (
    'admin@unifranz.edu.bo',
    '$2a$12$QKpjYMvQIXgcZZaPam8Mw.BVmKi2WRuGw/G0xdUAtugAmYwTdOmgm',
    'admin'
);
-- The hash above encodes the password "admin123" at cost 12.
-- In production: generate your own hash.
```

---

## 3. Login

```bash
curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@unifranz.edu.bo","password":"admin123"}' | jq .
```

Expected response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "550e8400-e29b-41d4-a716-446655440000"
}
```

Save both tokens:
```bash
ACCESS_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
REFRESH_TOKEN="550e8400-e29b-41d4-a716-446655440000"
```

---

## 4. Call a protected endpoint

```bash
curl -s http://localhost:8080/health \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
# Health is public; test a protected route once M3 adds one.
```

Try an unauthorized call (no token):
```bash
curl -s http://localhost:8080/auth/logout | jq .
# → {"error":"unauthorized","message":"Missing or invalid authorization token",...}
```

---

## 5. Refresh the access token

```bash
curl -s -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq .
# → new access_token + new refresh_token
```

---

## 6. Logout

```bash
curl -s -X POST http://localhost:8080/auth/logout \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq .
# → {"message":"Logged out successfully"}
```

Confirm the refresh token is now dead:
```bash
curl -s -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq .
# → {"error":"invalid_token",...}
```

---

## 7. Test role enforcement

Insert a student user and try a teacher-only endpoint:

```sql
INSERT INTO users (email, password_hash, role)
VALUES (
    'estudiante@unifranz.edu.bo',
    '$2a$12$QKpjYMvQIXgcZZaPam8Mw.BVmKi2WRuGw/G0xdUAtugAmYwTdOmgm',
    'estudiante'
);
```

```bash
# Login as student
STUDENT_TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"estudiante@unifranz.edu.bo","password":"admin123"}' \
  | jq -r '.access_token')

# Attempt admin-only route (available after M3+):
curl -s http://localhost:8080/admin/ai-config \
  -H "Authorization: Bearer $STUDENT_TOKEN" | jq .
# → {"error":"forbidden","message":"Insufficient permissions for this resource",...}
```

---

## 8. Test password recovery

```bash
curl -s -X POST http://localhost:8080/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@unifranz.edu.bo"}' | jq .
# → {"message":"...","reset_token":"prt_..."}

RESET_TOKEN="prt_..."

curl -s -X POST http://localhost:8080/auth/reset-password \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"$RESET_TOKEN\",\"new_password\":\"newSecret456\"}" | jq .
# → {"message":"Password updated successfully"}
```

---

## Validation Checklist (SC mapping)

| Success Criterion | Test |
|---|---|
| SC-001: Login < 2s | `time curl -X POST /auth/login ...` |
| SC-002: Expired token rejected | Wait 15 min or set `ACCESS_TOKEN_TTL=1s` in `.env` for testing |
| SC-003: Role guard 100% | Steps 7 above |
| SC-004: Logout invalidates refresh token | Steps 6 above |
| SC-005: Reset token single-use + 1h expiry | Step 8; submit same reset token twice |
| SC-006: No enumeration | Wrong email returns same 401 as wrong password |
| SC-007: 50 concurrent requests | `ab -n 50 -c 50 -p login.json -T application/json http://localhost:8080/auth/login` |
