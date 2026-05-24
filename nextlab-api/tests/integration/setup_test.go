//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	embeddedmigrations "github.com/Elder2003/nextlab-api/migrations"
)

var db *pgxpool.Pool

func TestMain(m *testing.M) {
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		fmt.Println("DATABASE_URL_TEST not set — skipping integration tests")
		os.Exit(0)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Printf("cannot connect to test DB: %v\n", err)
		os.Exit(1)
	}
	if err := pool.Ping(ctx); err != nil {
		fmt.Printf("test DB ping failed: %v\n", err)
		os.Exit(1)
	}
	db = pool

	// Run migrations so all tables exist
	d, err := iofs.New(embeddedmigrations.FS, ".")
	if err != nil {
		fmt.Printf("load migrations: %v\n", err)
		os.Exit(1)
	}
	mg, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		fmt.Printf("init migrate: %v\n", err)
		os.Exit(1)
	}
	if err := mg.Up(); err != nil && err != migrate.ErrNoChange {
		fmt.Printf("run migrations: %v\n", err)
		os.Exit(1)
	}

	truncate(ctx)

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func truncate(ctx context.Context) {
	tables := []string{
		"audit_events",
		"appeal_decisions",
		"appeals",
		"grade_decisions",
		"analysis_results",
		"trace_events",
		"worker_jobs",
		"exam_sessions",
		"submissions",
		"format_reports",
		"actividad_format_template",
		"actividad_interdisciplinary_scopes",
		"actividades",
		"grupos",
		"inscripciones",
		"grupo_asignaciones",
		"cursos",
		"users",
	}
	for _, t := range tables {
		db.Exec(ctx, fmt.Sprintf("TRUNCATE %s CASCADE", t)) //nolint:errcheck
	}
}

// seedUser inserts a minimal user row and returns its ID.
func seedUser(t *testing.T, role string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRow(context.Background(), `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, 'x', $2)
		RETURNING id`,
		fmt.Sprintf("%s-%s@test.local", role, uuid.New().String()[:8]), role,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedUser(%s): %v", role, err)
	}
	return id
}

// seedActividad inserts a carrera + materia + curso + actividad chain and returns the actividad ID.
func seedActividad(t *testing.T, docenteID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	var carreraID uuid.UUID
	db.QueryRow(ctx, `INSERT INTO carreras (code, name) VALUES ($1, 'Test Career') RETURNING id`,
		"CAR-"+uuid.New().String()[:6],
	).Scan(&carreraID) //nolint:errcheck

	var materiaID uuid.UUID
	db.QueryRow(ctx, `INSERT INTO materias (carrera_id, code, name, semester) VALUES ($1, $2, 'Test Materia', 1) RETURNING id`,
		carreraID, "MAT-"+uuid.New().String()[:6],
	).Scan(&materiaID) //nolint:errcheck

	var cursoID uuid.UUID
	db.QueryRow(ctx, `
		INSERT INTO cursos (materia_id, docente_id, semester_label, section)
		VALUES ($1, $2, '2026-1', 'A') RETURNING id`,
		materiaID, docenteID,
	).Scan(&cursoID) //nolint:errcheck

	var actividadID uuid.UUID
	err := db.QueryRow(ctx, `
		INSERT INTO actividades (curso_id, title, tipo, deadline, instructions)
		VALUES ($1, 'Test Actividad', 'examen', $2, 'instructions') RETURNING id`,
		cursoID, time.Now().Add(24*time.Hour),
	).Scan(&actividadID)
	if err != nil {
		t.Fatalf("seedActividad: %v", err)
	}
	return actividadID
}

// seedExamSession inserts a closed exam_session row and returns its ID.
func seedExamSession(t *testing.T, studentID, actividadID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRow(context.Background(), `
		INSERT INTO exam_sessions (student_id, actividad_id, status, closed_at)
		VALUES ($1, $2, 'completed', now())
		RETURNING id`,
		studentID, actividadID,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedExamSession: %v", err)
	}
	return id
}

// seedAnalysis inserts a heuristic analysis_results row.
func seedAnalysis(t *testing.T, sessionID uuid.UUID, score int, suggestedRange string) {
	t.Helper()
	_, err := db.Exec(context.Background(), `
		INSERT INTO analysis_results (session_id, analysis_type, score, flagged_categories, event_count, suggested_grade_range)
		VALUES ($1, 'heuristic', $2, '{}', 0, $3)`,
		sessionID, score, suggestedRange,
	)
	if err != nil {
		t.Fatalf("seedAnalysis: %v", err)
	}
}
