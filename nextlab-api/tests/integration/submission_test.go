//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestSubmissionGradeFlow(t *testing.T) {
	ctx := context.Background()

	docente := seedUser(t, "docente")
	student := seedUser(t, "estudiante")
	actividadID := seedActividad(t, docente)
	sessionID := seedExamSession(t, student, actividadID)

	// Insert a submission row
	var submissionID interface{}
	err := db.QueryRow(ctx, `
		INSERT INTO submissions (student_id, actividad_id, storage_key, original_filename, file_size_bytes, mime_type, submitted_at, is_late)
		VALUES ($1, $2, 'test/key.pdf', 'test.pdf', 1024, 'application/pdf', $3, false)
		RETURNING id`,
		student, actividadID, time.Now(),
	).Scan(&submissionID)
	if err != nil {
		t.Fatalf("insert submission: %v", err)
	}

	// Seed a heuristic analysis result so grade_decisions foreign key works
	seedAnalysis(t, sessionID, 50, "75-85")

	// Fetch the analysis_result id
	var analysisID interface{}
	err = db.QueryRow(ctx,
		`SELECT id FROM analysis_results WHERE session_id=$1 AND analysis_type='heuristic'`, sessionID,
	).Scan(&analysisID)
	if err != nil {
		t.Fatalf("fetch analysis_result: %v", err)
	}

	// Insert a grade decision (final_grade=80, within range, no override needed)
	_, err = db.Exec(ctx, `
		INSERT INTO grade_decisions (session_id, analysis_result_id, final_grade, suggested_grade_range, decided_by)
		VALUES ($1, $2, 80, '75-85', $3)`,
		sessionID, analysisID, docente,
	)
	if err != nil {
		t.Fatalf("insert grade_decision: %v", err)
	}

	// Assert grade_decisions row exists
	var count int
	err = db.QueryRow(ctx, `SELECT COUNT(*) FROM grade_decisions WHERE session_id=$1`, sessionID).Scan(&count)
	if err != nil {
		t.Fatalf("query grade_decisions: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 grade_decision row, got %d", count)
	}

	// Attempt to insert a second grade_decision — expect UNIQUE violation
	_, err = db.Exec(ctx, `
		INSERT INTO grade_decisions (session_id, analysis_result_id, final_grade, suggested_grade_range, decided_by)
		VALUES ($1, $2, 85, '75-85', $3)`,
		sessionID, analysisID, docente,
	)
	if err == nil {
		t.Error("expected unique violation on second grade_decision insert, got nil")
	} else {
		var pgErr *pgconn.PgError
		if !isPgError(err, &pgErr) || pgErr.Code != "23505" {
			t.Errorf("expected unique_violation (23505), got %v", err)
		}
	}

	// Test override_reason: use a separate student + session to avoid unique constraint
	student2 := seedUser(t, "estudiante")
	sessionID2 := seedExamSession(t, student2, actividadID)
	seedAnalysis(t, sessionID2, 30, "75-85")

	var analysisID2 interface{}
	db.QueryRow(ctx, `SELECT id FROM analysis_results WHERE session_id=$1 AND analysis_type='heuristic'`, sessionID2).Scan(&analysisID2) //nolint:errcheck

	_, err = db.Exec(ctx, `
		INSERT INTO grade_decisions (session_id, analysis_result_id, final_grade, suggested_grade_range, override_reason, decided_by)
		VALUES ($1, $2, 95, '75-85', 'demonstrated exceptional understanding', $3)`,
		sessionID2, analysisID2, docente,
	)
	if err != nil {
		t.Fatalf("insert grade_decision with override_reason: %v", err)
	}

	var overrideReason *string
	err = db.QueryRow(ctx, `SELECT override_reason FROM grade_decisions WHERE session_id=$1`, sessionID2).Scan(&overrideReason)
	if err != nil {
		t.Fatalf("query override_reason: %v", err)
	}
	if overrideReason == nil || *overrideReason != "demonstrated exceptional understanding" {
		t.Errorf("expected override_reason 'demonstrated exceptional understanding', got %v", overrideReason)
	}
}

func isPgError(err error, target **pgconn.PgError) bool {
	e, ok := err.(*pgconn.PgError)
	if ok {
		*target = e
	}
	return ok
}
