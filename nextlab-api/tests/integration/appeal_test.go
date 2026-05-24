//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Elder2003/nextlab-api/internal/appeals"
	"github.com/google/uuid"
)

func TestAppealFileAndReject(t *testing.T) {
	ctx := context.Background()

	docente := seedUser(t, "docente")
	student := seedUser(t, "estudiante")
	actividadID := seedActividad(t, docente)
	sessionID := seedExamSession(t, student, actividadID)
	seedAnalysis(t, sessionID, 60, "65-75")

	repo := appeals.NewRepository(db)

	// File an analysis_dispute appeal
	appeal, err := repo.InsertAppeal(ctx, appeals.FileAppealRequest{
		SessionID: sessionID,
		Type:      "analysis_dispute",
		Reason:    "The AI flagged copy-paste but this is original work",
	}, student)
	if err != nil {
		t.Fatalf("InsertAppeal: %v", err)
	}
	if appeal.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", appeal.Status)
	}

	// Assert 3 audit events were created
	var auditCount int
	err = db.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE appeal_id=$1`, appeal.ID).Scan(&auditCount)
	if err != nil {
		t.Fatalf("count audit_events: %v", err)
	}
	if auditCount != 3 {
		t.Errorf("expected 3 audit_events, got %d", auditCount)
	}

	// Assert event types in order
	rows, err := db.Query(ctx, `SELECT event_type FROM audit_events WHERE appeal_id=$1 ORDER BY sequence ASC`, appeal.ID)
	if err != nil {
		t.Fatalf("query audit_events: %v", err)
	}
	defer rows.Close()
	expected := []string{"submission_received", "analysis_completed", "appeal_filed"}
	var i int
	for rows.Next() {
		var et string
		rows.Scan(&et) //nolint:errcheck
		if i < len(expected) && et != expected[i] {
			t.Errorf("event[%d]: expected %q, got %q", i, expected[i], et)
		}
		i++
	}

	// Insert a decision (docente rejects)
	decision, err := repo.InsertDecision(ctx, appeal.ID, docente, "docente", "rejected", "Heuristic evidence confirmed pattern")
	if err != nil {
		t.Fatalf("InsertDecision: %v", err)
	}
	if decision.Decision != "rejected" {
		t.Errorf("expected decision 'rejected', got %q", decision.Decision)
	}

	// Update status to resolved
	if err := repo.UpdateAppealStatus(ctx, appeal.ID, "resolved"); err != nil {
		t.Fatalf("UpdateAppealStatus: %v", err)
	}

	updated, err := repo.GetAppeal(ctx, appeal.ID)
	if err != nil || updated == nil {
		t.Fatalf("GetAppeal: %v", err)
	}
	if updated.Status != "resolved" {
		t.Errorf("expected status 'resolved', got %q", updated.Status)
	}

	// Assert 4th audit event is decision_recorded
	var newCount int
	db.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE appeal_id=$1`, appeal.ID).Scan(&newCount) //nolint:errcheck
	if newCount != 4 {
		t.Errorf("expected 4 audit_events after decision, got %d", newCount)
	}

	var lastType string
	db.QueryRow(ctx, `SELECT event_type FROM audit_events WHERE appeal_id=$1 ORDER BY sequence DESC LIMIT 1`, appeal.ID).Scan(&lastType) //nolint:errcheck
	if lastType != "decision_recorded" {
		t.Errorf("expected last event 'decision_recorded', got %q", lastType)
	}

	// Verify the chain
	events, err := repo.GetAuditEvents(ctx, appeal.ID)
	if err != nil {
		t.Fatalf("GetAuditEvents: %v", err)
	}
	if !verifyChainHelper(events) {
		t.Error("evidence chain is invalid after reject decision")
	}
}

func TestAppealEscalateAndAdminResolve(t *testing.T) {
	ctx := context.Background()

	admin := seedUser(t, "admin")
	docente := seedUser(t, "docente")
	student := seedUser(t, "estudiante")
	actividadID := seedActividad(t, docente)
	sessionID := seedExamSession(t, student, actividadID)
	seedAnalysis(t, sessionID, 45, "60-70")

	repo := appeals.NewRepository(db)

	appeal, err := repo.InsertAppeal(ctx, appeals.FileAppealRequest{
		SessionID: sessionID,
		Type:      "grade_dispute",
		Reason:    "Grade seems unfair",
	}, student)
	if err != nil {
		t.Fatalf("InsertAppeal: %v", err)
	}

	if err := repo.UpdateAppealStatus(ctx, appeal.ID, "escalated"); err != nil {
		t.Fatalf("UpdateAppealStatus escalated: %v", err)
	}

	decision, err := repo.InsertDecision(ctx, appeal.ID, admin, "admin", "upheld", "Evidence shows original work; grade adjusted to 85")
	if err != nil {
		t.Fatalf("InsertDecision admin: %v", err)
	}
	if decision.Role != "admin" {
		t.Errorf("expected role 'admin', got %q", decision.Role)
	}

	if err := repo.UpdateAppealStatus(ctx, appeal.ID, "admin_resolved"); err != nil {
		t.Fatalf("UpdateAppealStatus admin_resolved: %v", err)
	}

	updated, _ := repo.GetAppeal(ctx, appeal.ID)
	if updated.Status != "admin_resolved" {
		t.Errorf("expected 'admin_resolved', got %q", updated.Status)
	}

	// Should have 4 events: 3 initial + 1 decision
	events, err := repo.GetAuditEvents(ctx, appeal.ID)
	if err != nil {
		t.Fatalf("GetAuditEvents: %v", err)
	}
	if len(events) != 4 {
		t.Errorf("expected 4 events, got %d", len(events))
	}
	if !verifyChainHelper(events) {
		t.Error("evidence chain invalid after admin_resolved")
	}
}

func TestEvidenceChainTampering(t *testing.T) {
	ctx := context.Background()

	docente := seedUser(t, "docente")
	student := seedUser(t, "estudiante")
	actividadID := seedActividad(t, docente)
	sessionID := seedExamSession(t, student, actividadID)
	seedAnalysis(t, sessionID, 70, "65-75")

	repo := appeals.NewRepository(db)

	appeal, err := repo.InsertAppeal(ctx, appeals.FileAppealRequest{
		SessionID: sessionID,
		Type:      "analysis_dispute",
		Reason:    "Testing tampering detection",
	}, student)
	if err != nil {
		t.Fatalf("InsertAppeal: %v", err)
	}

	// Verify clean chain
	events, _ := repo.GetAuditEvents(ctx, appeal.ID)
	if !verifyChainHelper(events) {
		t.Fatal("chain should be valid before tampering")
	}

	// Tamper: overwrite the payload of event at sequence=2
	_, err = db.Exec(ctx,
		`UPDATE audit_events SET payload='{}' WHERE appeal_id=$1 AND sequence=2`,
		appeal.ID,
	)
	if err != nil {
		t.Fatalf("tamper audit_event: %v", err)
	}

	// Reload events and verify chain is now broken
	events, _ = repo.GetAuditEvents(ctx, appeal.ID)
	if verifyChainHelper(events) {
		t.Error("expected chain to be invalid after tampering, but got valid=true")
	}
}

// verifyChainHelper re-implements verifyChain using the exported ComputeHash.
func verifyChainHelper(events []appeals.AuditEvent) bool {
	for _, e := range events {
		payloadBytes, err := json.Marshal(e.Payload)
		if err != nil {
			return false
		}
		prevHash := ""
		if e.PrevHash != nil {
			prevHash = *e.PrevHash
		}
		expected := appeals.ComputeHash(
			e.OccurredAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
			e.EventType,
			e.Sequence,
			string(payloadBytes),
			prevHash,
		)
		if expected != e.EventHash {
			return false
		}
	}
	return true
}

var _ = uuid.Nil // suppress unused import
