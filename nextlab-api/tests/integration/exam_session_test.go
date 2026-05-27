//go:build integration

package integration_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestExamSessionFlow(t *testing.T) {
	ctx := context.Background()

	admin := seedUser(t, "admin")
	_ = admin
	docente := seedUser(t, "docente")
	student := seedUser(t, "estudiante")
	actividadID := seedActividad(t, docente)
	sessionID := seedExamSession(t, student, actividadID)

	// Insert 3 trace events with a simple hash chain
	prevHash := ""
	for i := 1; i <= 3; i++ {
		canonical := fmt.Sprintf("%s|heartbeat|%d|{}|%s",
			time.Now().UTC().Format(time.RFC3339Nano), i, prevHash)
		sum := sha256.Sum256([]byte(canonical))
		hash := fmt.Sprintf("%x", sum)

		_, err := db.Exec(ctx, `
			INSERT INTO trace_events (session_id, sequence_num, event_type, event_hash)
			VALUES ($1, $2, 'heartbeat', $3)`,
			sessionID, i, hash,
		)
		if err != nil {
			t.Fatalf("insert trace_event %d: %v", i, err)
		}
		prevHash = hash
	}

	// Update session status to closed
	_, err := db.Exec(ctx, `UPDATE exam_sessions SET status='completed', closed_at=now() WHERE id=$1`, sessionID)
	if err != nil {
		t.Fatalf("update exam_session: %v", err)
	}

	seedAnalysis(t, sessionID, 42, "70-80")

	// Assert analysis_results has a heuristic row
	var count int
	err = db.QueryRow(ctx,
		`SELECT COUNT(*) FROM analysis_results WHERE session_id=$1 AND analysis_type='heuristic'`,
		sessionID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query analysis_results: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 heuristic row, got %d", count)
	}

	// Assert trace events exist
	var traceCount int
	err = db.QueryRow(ctx, `SELECT COUNT(*) FROM trace_events WHERE session_id=$1`, sessionID).Scan(&traceCount)
	if err != nil {
		t.Fatalf("query trace_events: %v", err)
	}
	if traceCount != 3 {
		t.Errorf("expected 3 trace_events, got %d", traceCount)
	}

	// Assert session is completed
	var status string
	err = db.QueryRow(ctx, `SELECT status FROM exam_sessions WHERE id=$1`, sessionID).Scan(&status)
	if err != nil {
		t.Fatalf("query exam_session status: %v", err)
	}
	if status != "completed" {
		t.Errorf("expected status 'completed', got %q", status)
	}

	_ = uuid.Nil // suppress unused import
}
