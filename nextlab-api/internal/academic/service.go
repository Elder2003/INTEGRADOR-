package academic

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrActivityHasSubmissions = errors.New("activity_has_submissions")
	ErrActivityFieldLocked    = errors.New("field_locked")
	ErrGrantNotFound          = errors.New("grant_not_found")
)

// GuardActivityMutation blocks deletion and field changes once submissions exist.
func GuardActivityMutation(a *Actividad, req *UpdateActividadRequest, isDelete bool) error {
	if !a.HasSubmissions {
		return nil
	}
	if isDelete {
		return ErrActivityHasSubmissions
	}
	if req != nil && (req.Title != nil || req.Tipo != nil) {
		return ErrActivityFieldLocked
	}
	return nil
}

// EnrollmentSync creates or revokes inscripciones to match the student's current carrera+semester.
func EnrollmentSync(ctx context.Context, db *pgxpool.Pool, userID, carreraID uuid.UUID, semester int) error {
	targetIDs, err := GetActiveCursosByMateriaSemester(ctx, db, carreraID, semester)
	if err != nil {
		return err
	}

	current, err := GetActiveInscripcionesByUser(ctx, db, userID)
	if err != nil {
		return err
	}

	enrolledSet := make(map[uuid.UUID]bool, len(current))
	for _, i := range current {
		enrolledSet[i.CursoID] = true
	}

	targetSet := make(map[uuid.UUID]bool, len(targetIDs))
	for _, id := range targetIDs {
		targetSet[id] = true
	}

	for _, id := range targetIDs {
		if !enrolledSet[id] {
			if err := CreateInscripcion(ctx, db, userID, id, carreraID, semester); err != nil {
				return err
			}
		}
	}

	for _, i := range current {
		if !targetSet[i.CursoID] {
			if err := RevokeInscripcion(ctx, db, userID, i.CursoID, "semester_advance"); err != nil {
				return err
			}
		}
	}

	slog.InfoContext(ctx, "enrollment synced",
		"event", "enrollment_synced",
		"user_id", userID,
		"carrera_id", carreraID,
		"semester", semester,
	)
	return nil
}

// ValidateCodocenteAccess returns ErrGrantNotFound if the co_docente has no active grant.
func ValidateCodocenteAccess(ctx context.Context, db *pgxpool.Pool, actividadID, codocenteID uuid.UUID) error {
	grant, err := GetActiveGrantByActividadAndCodocente(ctx, db, actividadID, codocenteID)
	if err != nil {
		return err
	}
	if grant == nil {
		return ErrGrantNotFound
	}
	return nil
}
