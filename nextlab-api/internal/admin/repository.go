package admin

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ── Docente methods ───────────────────────────────────────────────────────────

func (r *Repository) InsertDocente(ctx context.Context, email, fullName, passwordHash string) (*DocenteAccount, error) {
	var d DocenteAccount
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, full_name, password_hash, role)
		VALUES ($1, $2, $3, 'docente')
		RETURNING id, email, full_name, is_active, created_at, updated_at`,
		email, fullName, passwordHash,
	).Scan(&d.ID, &d.Email, &d.FullName, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) ListDocentes(ctx context.Context, activeOnly *bool, limit, offset int) ([]DocenteAccount, int, error) {
	base := `FROM users WHERE role = 'docente'`
	args := []any{}
	idx := 1

	if activeOnly != nil {
		base += ` AND is_active = $` + itoa(idx)
		args = append(args, *activeOnly)
		idx++
	}

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) "+base, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append(args, limit, offset)
	rows, err := r.db.Query(ctx,
		`SELECT id, email, full_name, is_active, created_at, updated_at `+base+
			` ORDER BY created_at DESC LIMIT $`+itoa(idx)+` OFFSET $`+itoa(idx+1),
		queryArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var docentes []DocenteAccount
	for rows.Next() {
		var d DocenteAccount
		if err := rows.Scan(&d.ID, &d.Email, &d.FullName, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, err
		}
		docentes = append(docentes, d)
	}
	if docentes == nil {
		docentes = []DocenteAccount{}
	}
	return docentes, total, rows.Err()
}

func (r *Repository) GetDocente(ctx context.Context, id uuid.UUID) (*DocenteAccount, error) {
	var d DocenteAccount
	err := r.db.QueryRow(ctx, `
		SELECT id, email, full_name, is_active, created_at, updated_at
		FROM users WHERE id = $1 AND role = 'docente'`, id,
	).Scan(&d.ID, &d.Email, &d.FullName, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) UpdateDocente(ctx context.Context, id uuid.UUID, fullName *string, isActive *bool) (*DocenteAccount, error) {
	var d DocenteAccount
	err := r.db.QueryRow(ctx, `
		UPDATE users
		SET full_name  = COALESCE($2, full_name),
		    is_active  = COALESCE($3, is_active),
		    updated_at = now()
		WHERE id = $1 AND role = 'docente'
		RETURNING id, email, full_name, is_active, created_at, updated_at`,
		id, fullName, isActive,
	).Scan(&d.ID, &d.Email, &d.FullName, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ── Format template methods ───────────────────────────────────────────────────

func (r *Repository) InsertFormatTemplate(ctx context.Context, name, description string, rules json.RawMessage, createdBy uuid.UUID) (*FormatTemplate, error) {
	var ft FormatTemplate
	var rulesRaw []byte
	err := r.db.QueryRow(ctx, `
		INSERT INTO format_templates (name, description, rules, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, description, rules, is_active, created_by, created_at, updated_at`,
		name, description, rules, createdBy,
	).Scan(&ft.ID, &ft.Name, &ft.Description, &rulesRaw, &ft.IsActive, &ft.CreatedBy, &ft.CreatedAt, &ft.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rulesRaw, &ft.Rules); err != nil {
		return nil, err
	}
	return &ft, nil
}

func (r *Repository) ListFormatTemplates(ctx context.Context, activeOnly bool) ([]FormatTemplate, error) {
	query := `SELECT id, name, description, rules, is_active, created_by, created_at, updated_at FROM format_templates`
	if activeOnly {
		query += ` WHERE is_active = TRUE`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []FormatTemplate
	for rows.Next() {
		var ft FormatTemplate
		var rulesRaw []byte
		if err := rows.Scan(&ft.ID, &ft.Name, &ft.Description, &rulesRaw, &ft.IsActive, &ft.CreatedBy, &ft.CreatedAt, &ft.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rulesRaw, &ft.Rules); err != nil {
			return nil, err
		}
		templates = append(templates, ft)
	}
	if templates == nil {
		templates = []FormatTemplate{}
	}
	return templates, rows.Err()
}

func (r *Repository) GetFormatTemplate(ctx context.Context, id uuid.UUID) (*FormatTemplate, error) {
	var ft FormatTemplate
	var rulesRaw []byte
	err := r.db.QueryRow(ctx, `
		SELECT id, name, description, rules, is_active, created_by, created_at, updated_at
		FROM format_templates WHERE id = $1`, id,
	).Scan(&ft.ID, &ft.Name, &ft.Description, &rulesRaw, &ft.IsActive, &ft.CreatedBy, &ft.CreatedAt, &ft.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rulesRaw, &ft.Rules); err != nil {
		return nil, err
	}
	return &ft, nil
}

func (r *Repository) UpdateFormatTemplate(ctx context.Context, id uuid.UUID, req UpdateFormatTemplateRequest) (*FormatTemplate, error) {
	var rulesArg *json.RawMessage
	if req.Rules != nil {
		b, err := json.Marshal(req.Rules)
		if err != nil {
			return nil, err
		}
		raw := json.RawMessage(b)
		rulesArg = &raw
	}

	var ft FormatTemplate
	var rulesRaw []byte
	err := r.db.QueryRow(ctx, `
		UPDATE format_templates
		SET name        = COALESCE($2, name),
		    description = COALESCE($3, description),
		    rules       = COALESCE($4, rules),
		    is_active   = COALESCE($5, is_active),
		    updated_at  = now()
		WHERE id = $1
		RETURNING id, name, description, rules, is_active, created_by, created_at, updated_at`,
		id, req.Name, req.Description, rulesArg, req.IsActive,
	).Scan(&ft.ID, &ft.Name, &ft.Description, &rulesRaw, &ft.IsActive, &ft.CreatedBy, &ft.CreatedAt, &ft.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rulesRaw, &ft.Rules); err != nil {
		return nil, err
	}
	return &ft, nil
}

func (r *Repository) IsTemplateInUse(ctx context.Context, id uuid.UUID) (bool, error) {
	var inUse bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM actividad_format_template WHERE format_template_id = $1)`, id,
	).Scan(&inUse)
	return inUse, err
}

func (r *Repository) DeleteFormatTemplate(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM format_templates WHERE id = $1`, id)
	return err
}

// ── Apply template methods ────────────────────────────────────────────────────

func (r *Repository) ApplyTemplate(ctx context.Context, actividadID, templateID, appliedBy uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO actividad_format_template (actividad_id, format_template_id, applied_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (actividad_id) DO UPDATE
		SET format_template_id = EXCLUDED.format_template_id,
		    applied_at         = now(),
		    applied_by         = EXCLUDED.applied_by`,
		actividadID, templateID, appliedBy,
	)
	return err
}

func (r *Repository) GetAppliedTemplate(ctx context.Context, actividadID uuid.UUID) (*AppliedTemplate, error) {
	var at AppliedTemplate
	var rulesRaw []byte
	err := r.db.QueryRow(ctx, `
		SELECT aft.format_template_id, ft.name, ft.rules, aft.applied_at
		FROM actividad_format_template aft
		JOIN format_templates ft ON ft.id = aft.format_template_id
		WHERE aft.actividad_id = $1`, actividadID,
	).Scan(&at.FormatTemplateID, &at.Name, &rulesRaw, &at.AppliedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rulesRaw, &at.Rules); err != nil {
		return nil, err
	}
	return &at, nil
}

func (r *Repository) UpdateActividadFormatRules(ctx context.Context, actividadID uuid.UUID, rules json.RawMessage) error {
	_, err := r.db.Exec(ctx,
		`UPDATE actividades SET format_rules = $2 WHERE id = $1`,
		actividadID, rules,
	)
	return err
}

func (r *Repository) VerifyDocenteOwnsActividad(ctx context.Context, actividadID, docenteID uuid.UUID) (bool, error) {
	var owns bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM actividades a
			JOIN cursos c ON c.id = a.curso_id
			WHERE a.id = $1 AND c.docente_id = $2
		)`, actividadID, docenteID,
	).Scan(&owns)
	return owns, err
}

// ── Interdisciplinary scope methods ──────────────────────────────────────────

func (r *Repository) SetScopes(ctx context.Context, actividadID uuid.UUID, scopes []InterdisciplinaryScopeInput) ([]InterdisciplinaryScope, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM actividad_interdisciplinary_scopes WHERE actividad_id = $1`, actividadID,
	); err != nil {
		return nil, err
	}

	var result []InterdisciplinaryScope
	for _, s := range scopes {
		var sc InterdisciplinaryScope
		if err := tx.QueryRow(ctx, `
			INSERT INTO actividad_interdisciplinary_scopes (actividad_id, carrera, semestre, codigo_materia)
			VALUES ($1, $2, $3, $4)
			RETURNING id, actividad_id, carrera, semestre, codigo_materia`,
			actividadID, s.Carrera, s.Semestre, s.CodigoMateria,
		).Scan(&sc.ID, &sc.ActividadID, &sc.Carrera, &sc.Semestre, &sc.CodigoMateria); err != nil {
			return nil, err
		}
		result = append(result, sc)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if result == nil {
		result = []InterdisciplinaryScope{}
	}
	return result, nil
}

func (r *Repository) GetScopes(ctx context.Context, actividadID uuid.UUID) ([]InterdisciplinaryScope, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, actividad_id, carrera, semestre, codigo_materia
		FROM actividad_interdisciplinary_scopes
		WHERE actividad_id = $1
		ORDER BY carrera, semestre, codigo_materia`, actividadID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scopes []InterdisciplinaryScope
	for rows.Next() {
		var sc InterdisciplinaryScope
		if err := rows.Scan(&sc.ID, &sc.ActividadID, &sc.Carrera, &sc.Semestre, &sc.CodigoMateria); err != nil {
			return nil, err
		}
		scopes = append(scopes, sc)
	}
	if scopes == nil {
		scopes = []InterdisciplinaryScope{}
	}
	return scopes, rows.Err()
}

func (r *Repository) DeleteScopes(ctx context.Context, actividadID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM actividad_interdisciplinary_scopes WHERE actividad_id = $1`, actividadID,
	)
	return err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func itoa(n int) string {
	return strconv.Itoa(n)
}

