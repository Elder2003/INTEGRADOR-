package academic

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Carreras ─────────────────────────────────────────────────────────────────

func GetCarreras(ctx context.Context, db *pgxpool.Pool) ([]Carrera, error) {
	rows, err := db.Query(ctx,
		`SELECT id, code, name, is_active, created_at, updated_at FROM carreras ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Carrera
	for rows.Next() {
		var c Carrera
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func CreateCarrera(ctx context.Context, db *pgxpool.Pool, code, name string) (*Carrera, error) {
	c := &Carrera{}
	err := db.QueryRow(ctx,
		`INSERT INTO carreras (code, name) VALUES ($1, $2)
		 RETURNING id, code, name, is_active, created_at, updated_at`,
		code, name).Scan(&c.ID, &c.Code, &c.Name, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func UpdateCarrera(ctx context.Context, db *pgxpool.Pool, id uuid.UUID, name *string, isActive *bool) (*Carrera, error) {
	c := &Carrera{}
	err := db.QueryRow(ctx,
		`UPDATE carreras SET
		   name      = COALESCE($2, name),
		   is_active = COALESCE($3, is_active),
		   updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, code, name, is_active, created_at, updated_at`,
		id, name, isActive).Scan(&c.ID, &c.Code, &c.Name, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func CarreraHasActiveMaterias(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (bool, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM materias WHERE carrera_id = $1 AND is_active = true`, id).Scan(&count)
	return count > 0, err
}

// ─── Materias ─────────────────────────────────────────────────────────────────

func GetMaterias(ctx context.Context, db *pgxpool.Pool, carreraID *uuid.UUID, semester *int) ([]Materia, error) {
	rows, err := db.Query(ctx,
		`SELECT id, carrera_id, code, name, semester, is_active, created_at, updated_at
		 FROM materias
		 WHERE ($1::uuid IS NULL OR carrera_id = $1)
		   AND ($2::int  IS NULL OR semester   = $2)
		 ORDER BY semester, name`,
		carreraID, semester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Materia
	for rows.Next() {
		var m Materia
		if err := rows.Scan(&m.ID, &m.CarreraID, &m.Code, &m.Name, &m.Semester, &m.IsActive, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func CreateMateria(ctx context.Context, db *pgxpool.Pool, carreraID uuid.UUID, code, name string, semester int) (*Materia, error) {
	m := &Materia{}
	err := db.QueryRow(ctx,
		`INSERT INTO materias (carrera_id, code, name, semester) VALUES ($1, $2, $3, $4)
		 RETURNING id, carrera_id, code, name, semester, is_active, created_at, updated_at`,
		carreraID, code, name, semester).
		Scan(&m.ID, &m.CarreraID, &m.Code, &m.Name, &m.Semester, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func UpdateMateria(ctx context.Context, db *pgxpool.Pool, id uuid.UUID, name *string, semester *int, isActive *bool) (*Materia, error) {
	m := &Materia{}
	err := db.QueryRow(ctx,
		`UPDATE materias SET
		   name      = COALESCE($2, name),
		   semester  = COALESCE($3, semester),
		   is_active = COALESCE($4, is_active),
		   updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, carrera_id, code, name, semester, is_active, created_at, updated_at`,
		id, name, semester, isActive).
		Scan(&m.ID, &m.CarreraID, &m.Code, &m.Name, &m.Semester, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func MateriaHasActiveCursos(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (bool, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM cursos WHERE materia_id = $1 AND is_active = true`, id).Scan(&count)
	return count > 0, err
}

// GetActiveMateriasByCarreraSemester returns active materias for enrollment sync.
func GetActiveMateriasByCarreraSemester(ctx context.Context, db *pgxpool.Pool, carreraID uuid.UUID, semester int) ([]Materia, error) {
	rows, err := db.Query(ctx,
		`SELECT id, carrera_id, code, name, semester, is_active, created_at, updated_at
		 FROM materias WHERE carrera_id = $1 AND semester = $2 AND is_active = true`,
		carreraID, semester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Materia
	for rows.Next() {
		var m Materia
		if err := rows.Scan(&m.ID, &m.CarreraID, &m.Code, &m.Name, &m.Semester, &m.IsActive, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ─── Student Profiles ────────────────────────────────────────────────────────

func GetStudentProfileByEmail(ctx context.Context, db *pgxpool.Pool, email string) (*StudentProfile, error) {
	p := &StudentProfile{}
	err := db.QueryRow(ctx,
		`SELECT id, email, carrera_id, semester, student_code, is_active, created_at, updated_at
		 FROM student_profiles WHERE email = $1`, email).
		Scan(&p.ID, &p.Email, &p.CarreraID, &p.Semester, &p.StudentCode, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func CreateStudentProfile(ctx context.Context, db *pgxpool.Pool, email string, carreraID uuid.UUID, semester int, studentCode *string) (*StudentProfile, error) {
	p := &StudentProfile{}
	err := db.QueryRow(ctx,
		`INSERT INTO student_profiles (email, carrera_id, semester, student_code)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, email, carrera_id, semester, student_code, is_active, created_at, updated_at`,
		email, carreraID, semester, studentCode).
		Scan(&p.ID, &p.Email, &p.CarreraID, &p.Semester, &p.StudentCode, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func UpdateStudentProfile(ctx context.Context, db *pgxpool.Pool, id uuid.UUID, req UpdateStudentProfileRequest) (*StudentProfile, error) {
	p := &StudentProfile{}
	err := db.QueryRow(ctx,
		`UPDATE student_profiles SET
		   carrera_id   = COALESCE($2, carrera_id),
		   semester     = COALESCE($3, semester),
		   student_code = COALESCE($4, student_code),
		   is_active    = COALESCE($5, is_active),
		   updated_at   = NOW()
		 WHERE id = $1
		 RETURNING id, email, carrera_id, semester, student_code, is_active, created_at, updated_at`,
		id, req.CarreraID, req.Semester, req.StudentCode, req.IsActive).
		Scan(&p.ID, &p.Email, &p.CarreraID, &p.Semester, &p.StudentCode, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

func ListStudentProfiles(ctx context.Context, db *pgxpool.Pool, carreraID *uuid.UUID, semester *int) ([]StudentProfile, error) {
	rows, err := db.Query(ctx,
		`SELECT id, email, carrera_id, semester, student_code, is_active, created_at, updated_at
		 FROM student_profiles
		 WHERE ($1::uuid IS NULL OR carrera_id = $1)
		   AND ($2::int  IS NULL OR semester   = $2)
		 ORDER BY email`,
		carreraID, semester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []StudentProfile
	for rows.Next() {
		var p StudentProfile
		if err := rows.Scan(&p.ID, &p.Email, &p.CarreraID, &p.Semester, &p.StudentCode, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// ─── Cursos ───────────────────────────────────────────────────────────────────

func GetCursosByDocente(ctx context.Context, db *pgxpool.Pool, docenteID uuid.UUID) ([]Curso, error) {
	rows, err := db.Query(ctx,
		`SELECT c.id, c.materia_id, m.name, c.docente_id, u.email,
		        c.semester_label, c.section, c.is_active, c.created_at, c.updated_at
		 FROM cursos c
		 JOIN materias m ON m.id = c.materia_id
		 JOIN users   u ON u.id = c.docente_id
		 WHERE c.docente_id = $1 AND c.is_active = true
		 ORDER BY c.created_at DESC`, docenteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Curso
	for rows.Next() {
		var c Curso
		if err := rows.Scan(&c.ID, &c.MateriaID, &c.MateriaName, &c.DocenteID, &c.DocenteEmail,
			&c.SemesterLabel, &c.Section, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func GetCursoByIDAndDocente(ctx context.Context, db *pgxpool.Pool, id, docenteID uuid.UUID) (*Curso, error) {
	c := &Curso{}
	err := db.QueryRow(ctx,
		`SELECT c.id, c.materia_id, m.name, c.docente_id, u.email,
		        c.semester_label, c.section, c.is_active, c.created_at, c.updated_at
		 FROM cursos c
		 JOIN materias m ON m.id = c.materia_id
		 JOIN users   u ON u.id = c.docente_id
		 WHERE c.id = $1 AND c.docente_id = $2`, id, docenteID).
		Scan(&c.ID, &c.MateriaID, &c.MateriaName, &c.DocenteID, &c.DocenteEmail,
			&c.SemesterLabel, &c.Section, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func CreateCurso(ctx context.Context, db *pgxpool.Pool, materiaID, docenteID uuid.UUID, semesterLabel, section string) (*Curso, error) {
	c := &Curso{}
	err := db.QueryRow(ctx,
		`INSERT INTO cursos (materia_id, docente_id, semester_label, section)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, materia_id, docente_id, semester_label, section, is_active, created_at, updated_at`,
		materiaID, docenteID, semesterLabel, section).
		Scan(&c.ID, &c.MateriaID, &c.DocenteID, &c.SemesterLabel, &c.Section, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func UpdateCurso(ctx context.Context, db *pgxpool.Pool, id, docenteID uuid.UUID, semesterLabel, section *string) (*Curso, error) {
	c := &Curso{}
	err := db.QueryRow(ctx,
		`UPDATE cursos SET
		   semester_label = COALESCE($3, semester_label),
		   section        = COALESCE($4, section),
		   updated_at     = NOW()
		 WHERE id = $1 AND docente_id = $2
		 RETURNING id, materia_id, docente_id, semester_label, section, is_active, created_at, updated_at`,
		id, docenteID, semesterLabel, section).
		Scan(&c.ID, &c.MateriaID, &c.DocenteID, &c.SemesterLabel, &c.Section, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func DeactivateCurso(ctx context.Context, db *pgxpool.Pool, id, docenteID uuid.UUID) error {
	tag, err := db.Exec(ctx,
		`UPDATE cursos SET is_active = false, updated_at = NOW()
		 WHERE id = $1 AND docente_id = $2`, id, docenteID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func CursoHasSubmissions(ctx context.Context, db *pgxpool.Pool, cursoID uuid.UUID) (bool, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM actividades WHERE curso_id = $1 AND has_submissions = true`, cursoID).Scan(&count)
	return count > 0, err
}

func GetCursoRoster(ctx context.Context, db *pgxpool.Pool, cursoID uuid.UUID) ([]RosterEntry, error) {
	rows, err := db.Query(ctx,
		`SELECT u.id, u.email, i.created_at,
		        ga.grupo_id, g.name
		 FROM inscripciones i
		 JOIN users u ON u.id = i.user_id
		 LEFT JOIN grupo_asignaciones ga ON ga.user_id = i.user_id AND ga.curso_id = i.curso_id
		 LEFT JOIN grupos g ON g.id = ga.grupo_id
		 WHERE i.curso_id = $1 AND i.is_active = true
		 ORDER BY u.email`, cursoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []RosterEntry
	for rows.Next() {
		var e RosterEntry
		var grupoID *uuid.UUID
		var grupoName *string
		if err := rows.Scan(&e.UserID, &e.Email, &e.EnrolledAt, &grupoID, &grupoName); err != nil {
			return nil, err
		}
		if grupoID != nil {
			e.Grupo = &GrupoRef{ID: *grupoID, Name: *grupoName}
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// MateriaIsActive verifies a materia exists and is active before curso creation.
func MateriaIsActive(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (bool, error) {
	var active bool
	err := db.QueryRow(ctx, `SELECT is_active FROM materias WHERE id = $1`, id).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return active, err
}

// ─── Grupos ───────────────────────────────────────────────────────────────────

func GetGruposByCurso(ctx context.Context, db *pgxpool.Pool, cursoID uuid.UUID) ([]Grupo, error) {
	rows, err := db.Query(ctx,
		`SELECT id, curso_id, name, capacity, created_at, updated_at
		 FROM grupos WHERE curso_id = $1 ORDER BY name`, cursoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Grupo
	for rows.Next() {
		var g Grupo
		if err := rows.Scan(&g.ID, &g.CursoID, &g.Name, &g.Capacity, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

func GetGrupoByIDAndCurso(ctx context.Context, db *pgxpool.Pool, id, cursoID uuid.UUID) (*Grupo, error) {
	g := &Grupo{}
	err := db.QueryRow(ctx,
		`SELECT id, curso_id, name, capacity, created_at, updated_at
		 FROM grupos WHERE id = $1 AND curso_id = $2`, id, cursoID).
		Scan(&g.ID, &g.CursoID, &g.Name, &g.Capacity, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return g, err
}

func CreateGrupo(ctx context.Context, db *pgxpool.Pool, cursoID uuid.UUID, name string, capacity *int) (*Grupo, error) {
	g := &Grupo{}
	err := db.QueryRow(ctx,
		`INSERT INTO grupos (curso_id, name, capacity) VALUES ($1, $2, $3)
		 RETURNING id, curso_id, name, capacity, created_at, updated_at`,
		cursoID, name, capacity).
		Scan(&g.ID, &g.CursoID, &g.Name, &g.Capacity, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func UpdateGrupo(ctx context.Context, db *pgxpool.Pool, id, cursoID uuid.UUID, name *string, capacity *int) (*Grupo, error) {
	g := &Grupo{}
	err := db.QueryRow(ctx,
		`UPDATE grupos SET
		   name     = COALESCE($3, name),
		   capacity = COALESCE($4, capacity),
		   updated_at = NOW()
		 WHERE id = $1 AND curso_id = $2
		 RETURNING id, curso_id, name, capacity, created_at, updated_at`,
		id, cursoID, name, capacity).
		Scan(&g.ID, &g.CursoID, &g.Name, &g.Capacity, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return g, err
}

func DeleteGrupo(ctx context.Context, db *pgxpool.Pool, id, cursoID uuid.UUID) error {
	tag, err := db.Exec(ctx, `DELETE FROM grupos WHERE id = $1 AND curso_id = $2`, id, cursoID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func StudentEnrolledInCurso(ctx context.Context, db *pgxpool.Pool, userID, cursoID uuid.UUID) (bool, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM inscripciones WHERE user_id = $1 AND curso_id = $2 AND is_active = true`,
		userID, cursoID).Scan(&count)
	return count > 0, err
}

func AssignStudentToGrupo(ctx context.Context, db *pgxpool.Pool, userID, grupoID, cursoID, assignedBy uuid.UUID) (*GrupoAsignacion, error) {
	a := &GrupoAsignacion{}
	err := db.QueryRow(ctx,
		`INSERT INTO grupo_asignaciones (user_id, grupo_id, curso_id, assigned_by)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, grupo_id, curso_id, assigned_by, assigned_at`,
		userID, grupoID, cursoID, assignedBy).
		Scan(&a.ID, &a.UserID, &a.GrupoID, &a.CursoID, &a.AssignedBy, &a.AssignedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func RemoveStudentFromGrupo(ctx context.Context, db *pgxpool.Pool, userID, grupoID uuid.UUID) error {
	_, err := db.Exec(ctx,
		`DELETE FROM grupo_asignaciones WHERE user_id = $1 AND grupo_id = $2`, userID, grupoID)
	return err
}

// ─── Actividades ─────────────────────────────────────────────────────────────

func GetActividadesByCurso(ctx context.Context, db *pgxpool.Pool, cursoID uuid.UUID) ([]Actividad, error) {
	rows, err := db.Query(ctx,
		`SELECT id, curso_id, title, tipo::text, deadline, instructions, trace_enabled, has_submissions, late_submission_allowed, created_at, updated_at
		 FROM actividades WHERE curso_id = $1 ORDER BY deadline`, cursoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Actividad
	for rows.Next() {
		var a Actividad
		if err := rows.Scan(&a.ID, &a.CursoID, &a.Title, &a.Tipo, &a.Deadline, &a.Instructions,
			&a.TraceEnabled, &a.HasSubmissions, &a.LateSubmissionAllowed, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func GetActividadByIDAndCurso(ctx context.Context, db *pgxpool.Pool, id, cursoID uuid.UUID) (*Actividad, error) {
	a := &Actividad{}
	err := db.QueryRow(ctx,
		`SELECT id, curso_id, title, tipo::text, deadline, instructions, trace_enabled, has_submissions, late_submission_allowed, created_at, updated_at
		 FROM actividades WHERE id = $1 AND curso_id = $2`, id, cursoID).
		Scan(&a.ID, &a.CursoID, &a.Title, &a.Tipo, &a.Deadline, &a.Instructions,
			&a.TraceEnabled, &a.HasSubmissions, &a.LateSubmissionAllowed, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func GetActividadByID(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*Actividad, error) {
	a := &Actividad{}
	err := db.QueryRow(ctx,
		`SELECT id, curso_id, title, tipo::text, deadline, instructions, trace_enabled, has_submissions, late_submission_allowed, created_at, updated_at
		 FROM actividades WHERE id = $1`, id).
		Scan(&a.ID, &a.CursoID, &a.Title, &a.Tipo, &a.Deadline, &a.Instructions,
			&a.TraceEnabled, &a.HasSubmissions, &a.LateSubmissionAllowed, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func CreateActividad(ctx context.Context, db *pgxpool.Pool, cursoID uuid.UUID, title, tipo string, deadline time.Time, instructions string) (*Actividad, error) {
	a := &Actividad{}
	err := db.QueryRow(ctx,
		`INSERT INTO actividades (curso_id, title, tipo, deadline, instructions)
		 VALUES ($1, $2, $3::actividad_tipo, $4, $5)
		 RETURNING id, curso_id, title, tipo::text, deadline, instructions, trace_enabled, has_submissions, late_submission_allowed, created_at, updated_at`,
		cursoID, title, tipo, deadline, instructions).
		Scan(&a.ID, &a.CursoID, &a.Title, &a.Tipo, &a.Deadline, &a.Instructions,
			&a.TraceEnabled, &a.HasSubmissions, &a.LateSubmissionAllowed, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func UpdateActividad(ctx context.Context, db *pgxpool.Pool, id, cursoID uuid.UUID, deadline *time.Time, instructions *string) (*Actividad, error) {
	a := &Actividad{}
	err := db.QueryRow(ctx,
		`UPDATE actividades SET
		   deadline     = COALESCE($3, deadline),
		   instructions = COALESCE($4, instructions),
		   updated_at   = NOW()
		 WHERE id = $1 AND curso_id = $2
		 RETURNING id, curso_id, title, tipo::text, deadline, instructions, trace_enabled, has_submissions, late_submission_allowed, created_at, updated_at`,
		id, cursoID, deadline, instructions).
		Scan(&a.ID, &a.CursoID, &a.Title, &a.Tipo, &a.Deadline, &a.Instructions,
			&a.TraceEnabled, &a.HasSubmissions, &a.LateSubmissionAllowed, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func UpdateActividadFull(ctx context.Context, db *pgxpool.Pool, id, cursoID uuid.UUID, title, tipo string, deadline time.Time, instructions string) (*Actividad, error) {
	a := &Actividad{}
	err := db.QueryRow(ctx,
		`UPDATE actividades SET
		   title        = $3,
		   tipo         = $4::actividad_tipo,
		   deadline     = $5,
		   instructions = $6,
		   updated_at   = NOW()
		 WHERE id = $1 AND curso_id = $2
		 RETURNING id, curso_id, title, tipo::text, deadline, instructions, trace_enabled, has_submissions, late_submission_allowed, created_at, updated_at`,
		id, cursoID, title, tipo, deadline, instructions).
		Scan(&a.ID, &a.CursoID, &a.Title, &a.Tipo, &a.Deadline, &a.Instructions,
			&a.TraceEnabled, &a.HasSubmissions, &a.LateSubmissionAllowed, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// UpdateLateSubmissionFlag atomically updates late_submission_allowed and inserts an audit log entry.
func UpdateLateSubmissionFlag(ctx context.Context, db *pgxpool.Pool, actividadID, docenteID uuid.UUID, newVal bool) (*Actividad, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	a := &Actividad{}
	err = tx.QueryRow(ctx,
		`UPDATE actividades SET late_submission_allowed = $1, updated_at = NOW()
		 WHERE id = $2
		   AND curso_id IN (SELECT id FROM cursos WHERE docente_id = $3)
		 RETURNING id, curso_id, title, tipo::text, deadline, instructions, trace_enabled, has_submissions, late_submission_allowed, created_at, updated_at`,
		newVal, actividadID, docenteID).
		Scan(&a.ID, &a.CursoID, &a.Title, &a.Tipo, &a.Deadline, &a.Instructions,
			&a.TraceEnabled, &a.HasSubmissions, &a.LateSubmissionAllowed, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	prev := "false"
	if !newVal {
		prev = "true"
	}
	next := "true"
	if !newVal {
		next = "false"
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO actividad_config_changes
		   (actividad_id, changed_by, field_name, previous_value, new_value)
		 VALUES ($1, $2, 'late_submission_allowed', $3, $4)`,
		actividadID, docenteID, prev, next,
	)
	if err != nil {
		return nil, err
	}

	return a, tx.Commit(ctx)
}

func DeleteActividad(ctx context.Context, db *pgxpool.Pool, id, cursoID uuid.UUID) error {
	tag, err := db.Exec(ctx, `DELETE FROM actividades WHERE id = $1 AND curso_id = $2`, id, cursoID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// GetActividadesByCursoForStudent returns activities without has_submissions (not visible to students).
func GetActividadesByCursoForStudent(ctx context.Context, db *pgxpool.Pool, cursoID uuid.UUID) ([]StudentActividadEntry, error) {
	rows, err := db.Query(ctx,
		`SELECT id, title, tipo::text, deadline, instructions, trace_enabled, created_at, updated_at
		 FROM actividades WHERE curso_id = $1 ORDER BY deadline`, cursoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []StudentActividadEntry
	for rows.Next() {
		var a StudentActividadEntry
		if err := rows.Scan(&a.ID, &a.Title, &a.Tipo, &a.Deadline, &a.Instructions,
			&a.TraceEnabled, &a.PostedAt, &a.LastEditedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// ─── Inscripciones ────────────────────────────────────────────────────────────

func GetActiveInscripcionesByUser(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID) ([]Inscripcion, error) {
	rows, err := db.Query(ctx,
		`SELECT id, user_id, curso_id, carrera_id, semester, is_active, created_at, revoked_at, revocation_reason
		 FROM inscripciones WHERE user_id = $1 AND is_active = true`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Inscripcion
	for rows.Next() {
		var i Inscripcion
		if err := rows.Scan(&i.ID, &i.UserID, &i.CursoID, &i.CarreraID, &i.Semester,
			&i.IsActive, &i.CreatedAt, &i.RevokedAt, &i.RevocationReason); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, rows.Err()
}

func CreateInscripcion(ctx context.Context, db *pgxpool.Pool, userID, cursoID, carreraID uuid.UUID, semester int) error {
	_, err := db.Exec(ctx,
		`INSERT INTO inscripciones (user_id, curso_id, carrera_id, semester)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, curso_id) DO UPDATE
		   SET is_active = true, revoked_at = NULL, revocation_reason = NULL`,
		userID, cursoID, carreraID, semester)
	return err
}

func RevokeInscripcion(ctx context.Context, db *pgxpool.Pool, userID, cursoID uuid.UUID, reason string) error {
	_, err := db.Exec(ctx,
		`UPDATE inscripciones SET is_active = false, revoked_at = NOW(), revocation_reason = $3
		 WHERE user_id = $1 AND curso_id = $2 AND is_active = true`,
		userID, cursoID, reason)
	return err
}

// GetCursosByStudent returns courses the student is enrolled in with group assignment if any.
func GetCursosByStudent(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID) ([]StudentCursoEntry, error) {
	rows, err := db.Query(ctx,
		`SELECT c.id, m.name, u.email, c.semester_label, c.section,
		        ga.grupo_id, g.name
		 FROM inscripciones i
		 JOIN cursos   c ON c.id = i.curso_id
		 JOIN materias m ON m.id = c.materia_id
		 JOIN users    u ON u.id = c.docente_id
		 LEFT JOIN grupo_asignaciones ga ON ga.user_id = i.user_id AND ga.curso_id = i.curso_id
		 LEFT JOIN grupos g ON g.id = ga.grupo_id
		 WHERE i.user_id = $1 AND i.is_active = true AND c.is_active = true
		 ORDER BY m.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []StudentCursoEntry
	for rows.Next() {
		var e StudentCursoEntry
		var grupoID *uuid.UUID
		var grupoName *string
		if err := rows.Scan(&e.ID, &e.MateriaName, &e.DocenteEmail, &e.SemesterLabel, &e.Section,
			&grupoID, &grupoName); err != nil {
			return nil, err
		}
		if grupoID != nil {
			e.Grupo = &GrupoRef{ID: *grupoID, Name: *grupoName}
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func StudentEnrolledActiveCurso(ctx context.Context, db *pgxpool.Pool, userID, cursoID uuid.UUID) (bool, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM inscripciones i
		 JOIN cursos c ON c.id = i.curso_id
		 WHERE i.user_id = $1 AND i.curso_id = $2 AND i.is_active = true AND c.is_active = true`,
		userID, cursoID).Scan(&count)
	return count > 0, err
}

// ─── Codocente Grants ─────────────────────────────────────────────────────────

func CreateGrant(ctx context.Context, db *pgxpool.Pool, actividadID, codocenteID, grantedBy uuid.UUID) (*CodocenteGrant, error) {
	g := &CodocenteGrant{}
	err := db.QueryRow(ctx,
		`INSERT INTO codocente_grants (actividad_id, codocente_id, granted_by)
		 VALUES ($1, $2, $3)
		 RETURNING id, actividad_id, codocente_id, granted_by, granted_at, revoked_at, revoked_by, is_active`,
		actividadID, codocenteID, grantedBy).
		Scan(&g.ID, &g.ActividadID, &g.CodocenteID, &g.GrantedBy, &g.GrantedAt, &g.RevokedAt, &g.RevokedBy, &g.IsActive)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func GetActiveGrantByActividadAndCodocente(ctx context.Context, db *pgxpool.Pool, actividadID, codocenteID uuid.UUID) (*CodocenteGrant, error) {
	g := &CodocenteGrant{}
	err := db.QueryRow(ctx,
		`SELECT id, actividad_id, codocente_id, granted_by, granted_at, revoked_at, revoked_by, is_active
		 FROM codocente_grants
		 WHERE actividad_id = $1 AND codocente_id = $2 AND is_active = true`,
		actividadID, codocenteID).
		Scan(&g.ID, &g.ActividadID, &g.CodocenteID, &g.GrantedBy, &g.GrantedAt, &g.RevokedAt, &g.RevokedBy, &g.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return g, err
}

func RevokeGrant(ctx context.Context, db *pgxpool.Pool, actividadID, codocenteID, revokedBy uuid.UUID) error {
	tag, err := db.Exec(ctx,
		`UPDATE codocente_grants SET is_active = false, revoked_at = NOW(), revoked_by = $3
		 WHERE actividad_id = $1 AND codocente_id = $2 AND is_active = true`,
		actividadID, codocenteID, revokedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ActividadOwnedByDocente checks if the actividad's curso belongs to the given docente.
func ActividadOwnedByDocente(ctx context.Context, db *pgxpool.Pool, actividadID, docenteID uuid.UUID) (bool, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM actividades a
		 JOIN cursos c ON c.id = a.curso_id
		 WHERE a.id = $1 AND c.docente_id = $2`, actividadID, docenteID).Scan(&count)
	return count > 0, err
}

// UserHasRole checks that a user exists and has the specified role.
func UserHasRole(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID, role string) (bool, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE id = $1 AND role = $2 AND is_active = true`,
		userID, role).Scan(&count)
	return count > 0, err
}

// GetActiveCursosByMateriaSemester returns cursos matching a student's enrollment criteria.
func GetActiveCursosByMateriaSemester(ctx context.Context, db *pgxpool.Pool, carreraID uuid.UUID, semester int) ([]uuid.UUID, error) {
	rows, err := db.Query(ctx,
		`SELECT c.id FROM cursos c
		 JOIN materias m ON m.id = c.materia_id
		 WHERE m.carrera_id = $1 AND m.semester = $2 AND m.is_active = true AND c.is_active = true`,
		carreraID, semester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
