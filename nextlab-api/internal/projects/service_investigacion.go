package projects

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/Elder2003/nextlab-api/internal/alerts"
)

type InvestigacionService struct {
	repo        *Repository
	mc          *minio.Client
	bucket      string
	alertWriter alerts.AlertWriter
}

func newInvestigacionService(repo *Repository, mc *minio.Client, bucket string, alertWriter alerts.AlertWriter) *InvestigacionService {
	return &InvestigacionService{repo: repo, mc: mc, bucket: bucket, alertWriter: alertWriter}
}

// CreateStage adds a stage to an investigacion actividad.
func (s *InvestigacionService) CreateStage(ctx context.Context, docenteID, actividadID uuid.UUID, name string, displayOrder int, deadline time.Time, description string) (*InvestigacionStage, error) {
	if err := s.requireDocenteOwner(ctx, docenteID, actividadID); err != nil {
		return nil, err
	}
	if err := s.requireTipo(ctx, actividadID, "investigacion"); err != nil {
		return nil, err
	}
	stage, err := s.repo.InsertStage(ctx, actividadID, name, displayOrder, deadline, description)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return stage, nil
}

// ListStages returns stages in display order.
func (s *InvestigacionService) ListStages(ctx context.Context, actividadID uuid.UUID) ([]InvestigacionStage, error) {
	list, err := s.repo.ListStages(ctx, actividadID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []InvestigacionStage{}
	}
	return list, nil
}

// UpdateStage updates a stage's fields; docente only.
func (s *InvestigacionService) UpdateStage(ctx context.Context, docenteID, actividadID, stageID uuid.UUID, name *string, displayOrder *int, deadline *time.Time, description *string) (*InvestigacionStage, error) {
	if err := s.requireDocenteOwner(ctx, docenteID, actividadID); err != nil {
		return nil, err
	}
	stage, err := s.repo.UpdateStage(ctx, stageID, name, displayOrder, deadline, description)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return stage, nil
}

// DeleteStage deletes a stage if no submissions exist.
func (s *InvestigacionService) DeleteStage(ctx context.Context, docenteID, actividadID, stageID uuid.UUID) error {
	if err := s.requireDocenteOwner(ctx, docenteID, actividadID); err != nil {
		return err
	}
	has, err := s.repo.StageHasSubmissions(ctx, stageID)
	if err != nil {
		return err
	}
	if has {
		return ErrStageHasSubmissions
	}
	return s.repo.DeleteStage(ctx, stageID)
}

// SubmitToStage uploads a file to MinIO and records the stage submission (idempotent).
func (s *InvestigacionService) SubmitToStage(ctx context.Context, studentID, actividadID, stageID uuid.UUID, header *multipart.FileHeader) (*StageSubmission, error) {
	_, cursoID, _, err := s.repo.GetActividadTipo(ctx, actividadID)
	if err != nil {
		return nil, err
	}
	enrolled, err := s.repo.StudentEnrolledInCurso(ctx, studentID, cursoID)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, ErrForbidden
	}

	stage, err := s.repo.GetStage(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage.ActividadID != actividadID {
		return nil, ErrNotFound
	}

	isLate := time.Now().UTC().After(stage.Deadline.UTC())

	f, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("open upload: %w", err)
	}
	defer f.Close()

	fileKey := fmt.Sprintf("stages/%s/%s/%s", stageID, studentID, uuid.New().String())
	ext := strings.ToLower(header.Filename[strings.LastIndex(header.Filename, ".")+1:])
	if ext != "" {
		fileKey += "." + ext
	}

	_, err = s.mc.PutObject(ctx, s.bucket, fileKey, f, header.Size,
		minio.PutObjectOptions{ContentType: header.Header.Get("Content-Type")})
	if err != nil {
		return nil, fmt.Errorf("store file: %w", err)
	}

	sub, err := s.repo.UpsertStageSubmission(ctx, stageID, studentID, fileKey, header.Filename, isLate)
	if err != nil {
		return nil, err
	}

	if s.alertWriter != nil {
		docenteID, _ := s.repo.GetCursoDocente(ctx, cursoID)
		aid := actividadID
		sid := studentID
		_ = s.alertWriter.WriteAlert(ctx, alerts.Alert{
			DocenteID:   docenteID,
			ActividadID: &aid,
			StudentID:   &sid,
			Type:        alerts.AlertSubmissionReceived,
			Payload:     map[string]any{"stage_id": stageID, "filename": header.Filename},
		})
		if isLate {
			_ = s.alertWriter.WriteAlert(ctx, alerts.Alert{
				DocenteID:   docenteID,
				ActividadID: &aid,
				StudentID:   &sid,
				Type:        alerts.AlertLateSubmission,
				Advisory:    true,
				Payload:     map[string]any{"stage_id": stageID, "filename": header.Filename},
			})
		}
	}

	return sub, nil
}

// GetStageProgress builds the per-student progress matrix for an actividad.
func (s *InvestigacionService) GetStageProgress(ctx context.Context, docenteID, actividadID uuid.UUID) (*StageProgressResponse, error) {
	if err := s.requireDocenteOwner(ctx, docenteID, actividadID); err != nil {
		return nil, err
	}

	_, cursoID, _, err := s.repo.GetActividadTipo(ctx, actividadID)
	if err != nil {
		return nil, err
	}

	stages, err := s.repo.ListStages(ctx, actividadID)
	if err != nil {
		return nil, err
	}

	students, err := s.repo.GetEnrolledStudents(ctx, cursoID)
	if err != nil {
		return nil, err
	}

	submissions, err := s.repo.ListStageSubmissionsForActividad(ctx, actividadID)
	if err != nil {
		return nil, err
	}

	// Index submissions by (stageID, studentID)
	type key struct {
		stageID   uuid.UUID
		studentID uuid.UUID
	}
	subMap := make(map[key]*StageSubmission, len(submissions))
	for i := range submissions {
		subMap[key{submissions[i].StageID, submissions[i].StudentID}] = &submissions[i]
	}

	stageIndex := make([]uuid.UUID, len(stages))
	for i, st := range stages {
		stageIndex[i] = st.ID
	}

	var studentProgress []StudentProgress
	for _, stu := range students {
		var entries []StageProgressEntry
		for _, sid := range stageIndex {
			sub := subMap[key{sid, stu.StudentID}]
			e := StageProgressEntry{StageID: sid}
			if sub == nil {
				e.Status = StageStatusPending
			} else {
				if sub.IsLate {
					e.Status = StageStatusSubmittedLate
				} else {
					e.Status = StageStatusSubmitted
				}
				t := sub.SubmittedAt
				e.SubmittedAt = &t
				e.IsLate = sub.IsLate
				e.Grade = sub.Grade
			}
			entries = append(entries, e)
		}
		studentProgress = append(studentProgress, StudentProgress{
			StudentID:   stu.StudentID,
			StudentName: stu.Name,
			Stages:      entries,
		})
	}

	return &StageProgressResponse{
		ActividadID: actividadID,
		Stages:      stages,
		Students:    studentProgress,
	}, nil
}

// GradeStageSubmission assigns a grade to a student's stage submission.
func (s *InvestigacionService) GradeStageSubmission(ctx context.Context, docenteID, actividadID, stageID, studentID uuid.UUID, grade float64, gradeNote string) (*StageSubmission, error) {
	if err := s.requireDocenteOwner(ctx, docenteID, actividadID); err != nil {
		return nil, err
	}
	if grade < 0 || grade > 100 {
		return nil, fmt.Errorf("grade must be 0–100")
	}
	return s.repo.GradeStageSubmission(ctx, stageID, studentID, docenteID, grade, gradeNote)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (s *InvestigacionService) requireDocenteOwner(ctx context.Context, docenteID, actividadID uuid.UUID) error {
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

func (s *InvestigacionService) requireTipo(ctx context.Context, actividadID uuid.UUID, expected string) error {
	tipo, _, _, err := s.repo.GetActividadTipo(ctx, actividadID)
	if err != nil {
		return err
	}
	if tipo != expected {
		return ErrWrongTipo
	}
	return nil
}
