# Quickstart: Admin Modules (M12)

Assumes a running stack. Set `ADMIN_TOKEN`, `DOCENTE_TOKEN`, `ACTIVIDAD_ID`.

## Story 1 — Docente Account Management

```bash
# 1. Create docente
curl -X POST http://localhost:3000/admin/docentes \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"newdoc@unifranz.edu.bo","full_name":"Ana Flores"}' \
  | jq '{id:.id, temp_pwd:.temporary_password}'

# 2. List docentes
curl http://localhost:3000/admin/docentes \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.total'

# 3. Update name
curl -X PUT http://localhost:3000/admin/docentes/$DOCENTE_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Ana María Flores"}' | jq '.full_name'

# 4. Soft-deactivate
curl -X DELETE http://localhost:3000/admin/docentes/$DOCENTE_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN"
# Expect 204

# 5. Verify non-admin gets 403
curl -X POST http://localhost:3000/admin/docentes \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -d '{"email":"x@x.com","full_name":"X"}' | jq '.error'
# Expect "forbidden"
```

## Story 2 — Format Template Library

```bash
# 1. Create template (admin)
TEMPLATE_ID=$(curl -s -X POST http://localhost:3000/admin/format-templates \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"TFG Standard","description":"Standard TFG format","rules":{"min_pages":10,"max_pages":80,"required_types":["pdf"]}}' \
  | jq -r '.id')

# 2. List templates
curl http://localhost:3000/admin/format-templates \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.total'

# 3. Docente applies template to actividad
curl -X POST http://localhost:3000/actividades/$ACTIVIDAD_ID/format-template \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"template_id\":\"$TEMPLATE_ID\"}" | jq '.name'
# Expect "TFG Standard"

# 4. Get applied template
curl http://localhost:3000/actividades/$ACTIVIDAD_ID/format-template \
  -H "Authorization: Bearer $DOCENTE_TOKEN" | jq '.rules'

# 5. Try to delete in-use template — expect 409
curl -X DELETE http://localhost:3000/admin/format-templates/$TEMPLATE_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.error'
```

## Story 3 — Curriculum Document Management

```bash
# 1. List curriculum docs (admin)
curl "http://localhost:3000/admin/curriculum/documents?carrera=ISI" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.total'

# 2. Delete a document
curl -X DELETE http://localhost:3000/admin/curriculum/documents/$DOC_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN"
# Expect 204

# 3. Verify it's gone
curl "http://localhost:3000/admin/curriculum/documents" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.total'
```

## Story 4 — Interdisciplinary Activity Configuration

```bash
# 1. Admin sets interdisciplinary scopes
curl -X POST http://localhost:3000/actividades/$ACTIVIDAD_ID/interdisciplinary \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scopes":[{"carrera":"Medicina","semestre":4,"codigo_materia":"BIO401"}]}' \
  | jq '.scopes | length'
# Expect 1

# 2. Get scopes
curl http://localhost:3000/actividades/$ACTIVIDAD_ID/interdisciplinary \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.scopes[0].carrera'

# 3. Delete scopes
curl -X DELETE http://localhost:3000/actividades/$ACTIVIDAD_ID/interdisciplinary \
  -H "Authorization: Bearer $ADMIN_TOKEN"
# Expect 204

# 4. Non-admin POST → 403
curl -X POST http://localhost:3000/actividades/$ACTIVIDAD_ID/interdisciplinary \
  -H "Authorization: Bearer $DOCENTE_TOKEN" \
  -d '{"scopes":[]}' | jq '.error'
```
