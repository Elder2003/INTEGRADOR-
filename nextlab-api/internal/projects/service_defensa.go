package projects

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/google/uuid"
)

type DefensaService struct {
	repo *Repository
}

func newDefensaService(repo *Repository) *DefensaService {
	return &DefensaService{repo: repo}
}

// ScheduleDefense creates or replaces a student's defense appointment.
func (s *DefensaService) ScheduleDefense(ctx context.Context, docenteID, actividadID, studentID uuid.UUID, scheduledAt time.Time) (*DefenseAppointment, error) {
	if err := s.requireDocenteOwner(ctx, docenteID, actividadID); err != nil {
		return nil, err
	}
	_, cursoID, _, err := s.repo.GetActividadTipo(ctx, actividadID)
	if err != nil {
		return nil, err
	}
	enrolled, err := s.repo.StudentEnrolledInCurso(ctx, studentID, cursoID)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, ErrNotFound
	}
	return s.repo.UpsertDefenseAppointment(ctx, actividadID, studentID, scheduledAt)
}

// GetDefenseAppointment returns the appointment. Students can only see their own.
func (s *DefensaService) GetDefenseAppointment(ctx context.Context, callerID uuid.UUID, callerRole string, actividadID, studentID uuid.UUID) (*DefenseAppointment, error) {
	if callerRole == "estudiante" && callerID != studentID {
		return nil, ErrForbidden
	}
	return s.repo.GetDefenseAppointment(ctx, actividadID, studentID)
}

// GradeDefense records the final grade for an oral defense.
func (s *DefensaService) GradeDefense(ctx context.Context, docenteID, actividadID, studentID uuid.UUID, grade float64) (*DefenseAppointment, error) {
	if err := s.requireDocenteOwner(ctx, docenteID, actividadID); err != nil {
		return nil, err
	}
	if grade < 0 || grade > 100 {
		return nil, fmt.Errorf("grade must be 0–100")
	}
	return s.repo.GradeDefenseAppointment(ctx, actividadID, studentID, docenteID, grade)
}

func (s *DefensaService) requireDocenteOwner(ctx context.Context, docenteID, actividadID uuid.UUID) error {
	_, cursoID, _, err := s.repo.GetActividadTipo(ctx, actividadID)
	if err != nil {
		return err
	}
	ownerID, err := s.repo.GetCursoDocente(ctx, cursoID)
	if err != nil {
		return err
	}
	if ownerID != docenteID {
		return ErrForbidden
	}
	return nil
}

// isDuplicateKey detects PostgreSQL unique violation errors.
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "unique")
}
