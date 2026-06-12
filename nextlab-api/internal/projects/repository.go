package projects

import (
	"context"
	"encoding/json"
	"errors"
	"time"

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

// ─── Actividad helpers ────────────────────────────────────────────────────────

// GetActividadTipo returns the tipo string and curso_id for the given actividad.
func (r *Repository) GetActividadTipo(ctx context.Context, actividadID uuid.UUID) (tipo string, cursoID uuid.UUID, deadline time.Time, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT tipo, curso_id, deadline FROM actividades WHERE id = $1`,
		actividadID).Scan(&tipo, &cursoID, &deadline)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", uuid.Nil, time.Time{}, ErrNotFound
	}
	return
}

// GetCursoDocente returns the docente_id for a curso.
func (r *Repository) GetCursoDocente(ctx context.Context, cursoID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT docente_id FROM cursos WHERE id = $1`, cursoID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return id, err
}

// StudentEnrolledInCurso checks if a student is enrolled in the given curso.
func (r *Repository) StudentEnrolledInCurso(ctx context.Context, studentID, cursoID uuid.UUID) (bool, error) {
	var enrolled bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM inscripciones WHERE user_id=$1 AND curso_id=$2 AND is_active=true)`,
		studentID, cursoID).Scan(&enrolled)
	return enrolled, err
}

// GetEnrolledStudents returns all enrolled students in a curso with their display names.
func (r *Repository) GetEnrolledStudents(ctx context.Context, cursoID uuid.UUID) ([]struct {
	StudentID uuid.UUID
	Name      string
}, error) {
	rows, err := r.db.Query(ctx,
		`SELECT i.user_id, u.email
		 FROM inscripciones i
		 JOIN users u ON u.id = i.user_id
		 WHERE i.curso_id = $1 AND i.is_active = true
		 ORDER BY u.email`,
		cursoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []struct {
		StudentID uuid.UUID
		Name      string
	}
	for rows.Next() {
		var s struct {
			StudentID uuid.UUID
			Name      string
		}
		if err := rows.Scan(&s.StudentID, &s.Name); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// ─── Investigacion Stages ─────────────────────────────────────────────────────

func (r *Repository) InsertStage(ctx context.Context, actividadID uuid.UUID, name string, displayOrder int, deadline time.Time, description string) (*InvestigacionStage, error) {
	s := &InvestigacionStage{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO investigacion_stages (actividad_id, name, display_order, deadline, description)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, actividad_id, name, display_order, deadline, description, created_at`,
		actividadID, name, displayOrder, deadline, description,
	).Scan(&s.ID, &s.ActividadID, &s.Name, &s.DisplayOrder, &s.Deadline, &s.Description, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *Repository) ListStages(ctx context.Context, actividadID uuid.UUID) ([]InvestigacionStage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, actividad_id, name, display_order, deadline, description, created_at
		 FROM investigacion_stages WHERE actividad_id = $1 ORDER BY display_order, name`,
		actividadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []InvestigacionStage
	for rows.Next() {
		var s InvestigacionStage
		if err := rows.Scan(&s.ID, &s.ActividadID, &s.Name, &s.DisplayOrder, &s.Deadline, &s.Description, &s.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *Repository) GetStage(ctx context.Context, stageID uuid.UUID) (*InvestigacionStage, error) {
	s := &InvestigacionStage{}
	err := r.db.QueryRow(ctx,
		`SELECT id, actividad_id, name, display_order, deadline, description, created_at
		 FROM investigacion_stages WHERE id = $1`, stageID).
		Scan(&s.ID, &s.ActividadID, &s.Name, &s.DisplayOrder, &s.Deadline, &s.Description, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *Repository) UpdateStage(ctx context.Context, stageID uuid.UUID, name *string, displayOrder *int, deadline *time.Time, description *string) (*InvestigacionStage, error) {
	s := &InvestigacionStage{}
	err := r.db.QueryRow(ctx,
		`UPDATE investigacion_stages SET
		   name          = COALESCE($2, name),
		   display_order = COALESCE($3, display_order),
		   deadline      = COALESCE($4, deadline),
		   description   = COALESCE($5, description)
		 WHERE id = $1
		 RETURNING id, actividad_id, name, display_order, deadline, description, created_at`,
		stageID, name, displayOrder, deadline, description,
	).Scan(&s.ID, &s.ActividadID, &s.Name, &s.DisplayOrder, &s.Deadline, &s.Description, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *Repository) StageHasSubmissions(ctx context.Context, stageID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM stage_submissions WHERE stage_id = $1)`, stageID).Scan(&exists)
	return exists, err
}

func (r *Repository) DeleteStage(ctx context.Context, stageID uuid.UUID) error {
	cmd, err := r.db.Exec(ctx, `DELETE FROM investigacion_stages WHERE id = $1`, stageID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CountStagesForActividad(ctx context.Context, actividadID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM investigacion_stages WHERE actividad_id = $1`, actividadID).Scan(&n)
	return n, err
}

// ─── Stage Submissions ────────────────────────────────────────────────────────

func (r *Repository) UpsertStageSubmission(ctx context.Context, stageID, studentID uuid.UUID, fileKey, originalFilename string, isLate bool) (*StageSubmission, error) {
	s := &StageSubmission{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO stage_submissions (stage_id, student_id, file_key, original_filename, is_late)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (stage_id, student_id) DO UPDATE SET
		   file_key = EXCLUDED.file_key,
		   original_filename = EXCLUDED.original_filename,
		   is_late = EXCLUDED.is_late,
		   submitted_at = now()
		 RETURNING id, stage_id, student_id, file_key, original_filename, is_late, grade, grade_note, graded_at, graded_by, submitted_at`,
		stageID, studentID, fileKey, originalFilename, isLate,
	).Scan(&s.ID, &s.StageID, &s.StudentID, &s.FileKey, &s.OriginalFilename, &s.IsLate,
		&s.Grade, &s.GradeNote, &s.GradedAt, &s.GradedBy, &s.SubmittedAt)
	return s, err
}

func (r *Repository) GetStageSubmission(ctx context.Context, stageID, studentID uuid.UUID) (*StageSubmission, error) {
	s := &StageSubmission{}
	err := r.db.QueryRow(ctx,
		`SELECT id, stage_id, student_id, file_key, original_filename, is_late, grade, grade_note, graded_at, graded_by, submitted_at
		 FROM stage_submissions WHERE stage_id=$1 AND student_id=$2`,
		stageID, studentID,
	).Scan(&s.ID, &s.StageID, &s.StudentID, &s.FileKey, &s.OriginalFilename, &s.IsLate,
		&s.Grade, &s.GradeNote, &s.GradedAt, &s.GradedBy, &s.SubmittedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// ListStageSubmissionsForActividad returns all submissions for all stages of an actividad.
func (r *Repository) ListStageSubmissionsForActividad(ctx context.Context, actividadID uuid.UUID) ([]StageSubmission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT ss.id, ss.stage_id, ss.student_id, ss.file_key, ss.original_filename,
		        ss.is_late, ss.grade, ss.grade_note, ss.graded_at, ss.graded_by, ss.submitted_at
		 FROM stage_submissions ss
		 JOIN investigacion_stages ist ON ist.id = ss.stage_id
		 WHERE ist.actividad_id = $1`,
		actividadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []StageSubmission
	for rows.Next() {
		var s StageSubmission
		if err := rows.Scan(&s.ID, &s.StageID, &s.StudentID, &s.FileKey, &s.OriginalFilename,
			&s.IsLate, &s.Grade, &s.GradeNote, &s.GradedAt, &s.GradedBy, &s.SubmittedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *Repository) GradeStageSubmission(ctx context.Context, stageID, studentID, gradedBy uuid.UUID, grade float64, gradeNote string) (*StageSubmission, error) {
	s := &StageSubmission{}
	err := r.db.QueryRow(ctx,
		`UPDATE stage_submissions SET grade=$3, grade_note=$4, graded_by=$5, graded_at=now()
		 WHERE stage_id=$1 AND student_id=$2
		 RETURNING id, stage_id, student_id, file_key, original_filename, is_late, grade, grade_note, graded_at, graded_by, submitted_at`,
		stageID, studentID, grade, gradeNote, gradedBy,
	).Scan(&s.ID, &s.StageID, &s.StudentID, &s.FileKey, &s.OriginalFilename, &s.IsLate,
		&s.Grade, &s.GradeNote, &s.GradedAt, &s.GradedBy, &s.SubmittedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

// ─── Git Submissions ──────────────────────────────────────────────────────────

func (r *Repository) UpsertGitSubmission(ctx context.Context, actividadID, studentID uuid.UUID, repoURL string) (*GitSubmission, error) {
	s := &GitSubmission{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO git_submissions (actividad_id, student_id, repo_url)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (actividad_id, student_id) DO UPDATE SET
		   repo_url = EXCLUDED.repo_url, updated_at = now()
		 RETURNING id, actividad_id, student_id, repo_url, submitted_at, updated_at`,
		actividadID, studentID, repoURL,
	).Scan(&s.ID, &s.ActividadID, &s.StudentID, &s.RepoURL, &s.SubmittedAt, &s.UpdatedAt)
	return s, err
}

func (r *Repository) GetGitSubmission(ctx context.Context, actividadID, studentID uuid.UUID) (*GitSubmission, error) {
	s := &GitSubmission{}
	err := r.db.QueryRow(ctx,
		`SELECT id, actividad_id, student_id, repo_url, submitted_at, updated_at
		 FROM git_submissions WHERE actividad_id=$1 AND student_id=$2`,
		actividadID, studentID,
	).Scan(&s.ID, &s.ActividadID, &s.StudentID, &s.RepoURL, &s.SubmittedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// ─── Git Analyses ─────────────────────────────────────────────────────────────

func (r *Repository) InsertGitAnalysis(ctx context.Context, gitSubmissionID uuid.UUID) (*GitAnalysis, error) {
	a := &GitAnalysis{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO git_analyses (git_submission_id) VALUES ($1)
		 RETURNING id, git_submission_id, status, commit_count, authors, first_commit_at, last_commit_at,
		           deadline_rush_pct, consistency_score, flags, llm_advisory, llm_status, fail_reason,
		           grade, override_reason, graded_by, graded_at, created_at, updated_at`,
		gitSubmissionID,
	).Scan(&a.ID, &a.GitSubmissionID, &a.Status, &a.CommitCount, (*jsonArray)(&a.Authors),
		&a.FirstCommitAt, &a.LastCommitAt, &a.DeadlineRushPct, &a.ConsistencyScore,
		(*jsonArray)(&a.Flags), &a.LLMAdvisory, &a.LLMStatus, &a.FailReason,
		&a.Grade, &a.OverrideReason, &a.GradedBy, &a.GradedAt, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (r *Repository) GetLatestGitAnalysis(ctx context.Context, gitSubmissionID uuid.UUID) (*GitAnalysis, error) {
	a := &GitAnalysis{}
	err := r.db.QueryRow(ctx,
		`SELECT id, git_submission_id, status, commit_count, authors, first_commit_at, last_commit_at,
		        deadline_rush_pct, consistency_score, flags, llm_advisory, llm_status, fail_reason,
		        grade, override_reason, graded_by, graded_at, created_at, updated_at
		 FROM git_analyses WHERE git_submission_id = $1 ORDER BY created_at DESC LIMIT 1`,
		gitSubmissionID,
	).Scan(&a.ID, &a.GitSubmissionID, &a.Status, &a.CommitCount, (*jsonArray)(&a.Authors),
		&a.FirstCommitAt, &a.LastCommitAt, &a.DeadlineRushPct, &a.ConsistencyScore,
		(*jsonArray)(&a.Flags), &a.LLMAdvisory, &a.LLMStatus, &a.FailReason,
		&a.Grade, &a.OverrideReason, &a.GradedBy, &a.GradedAt, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func (r *Repository) UpdateGitAnalysisSuccess(ctx context.Context, id uuid.UUID, commitCount int, authors []string, firstCommit, lastCommit *time.Time, deadlineRushPct float64, consistencyScore int, flags []string) error {
	authorsJSON, _ := json.Marshal(authors)
	flagsJSON, _ := json.Marshal(flags)
	_, err := r.db.Exec(ctx,
		`UPDATE git_analyses SET
		   status = 'success', commit_count=$2, authors=$3, first_commit_at=$4, last_commit_at=$5,
		   deadline_rush_pct=$6, consistency_score=$7, flags=$8, updated_at=now()
		 WHERE id = $1`,
		id, commitCount, authorsJSON, firstCommit, lastCommit, deadlineRushPct, consistencyScore, flagsJSON)
	return err
}

func (r *Repository) UpdateGitAnalysisLLM(ctx context.Context, id uuid.UUID, advisory, llmStatus string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE git_analyses SET llm_advisory=$2, llm_status=$3, updated_at=now() WHERE id=$1`,
		id, advisory, llmStatus)
	return err
}

func (r *Repository) UpdateGitAnalysisFailed(ctx context.Context, id uuid.UUID, failReason string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE git_analyses SET status='failed', fail_reason=$2, llm_status='failed', updated_at=now() WHERE id=$1`,
		id, failReason)
	return err
}

func (r *Repository) GradeGitAnalysis(ctx context.Context, id, gradedBy uuid.UUID, grade float64, overrideReason string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE git_analyses SET grade=$2, override_reason=$3, graded_by=$4, graded_at=now(), updated_at=now() WHERE id=$1`,
		id, grade, overrideReason, gradedBy)
	return err
}

// ─── Git Analysis Jobs ────────────────────────────────────────────────────────

func (r *Repository) InsertGitAnalysisJob(ctx context.Context, gitAnalysisID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO git_analysis_jobs (git_analysis_id) VALUES ($1)`, gitAnalysisID)
	return err
}

// ClaimNextGitAnalysisJob atomically claims the oldest queued job and returns it with joined repo_url and actividad_id.
func (r *Repository) ClaimNextGitAnalysisJob(ctx context.Context) (*GitAnalysisJob, error) {
	j := &GitAnalysisJob{}
	err := r.db.QueryRow(ctx,
		`UPDATE git_analysis_jobs
		 SET status='processing', started_at=now()
		 WHERE id = (
		   SELECT j.id FROM git_analysis_jobs j
		   JOIN git_analyses a ON a.id = j.git_analysis_id
		   WHERE j.status = 'queued' AND j.dead_at IS NULL
		   ORDER BY j.queued_at
		   FOR UPDATE SKIP LOCKED
		   LIMIT 1
		 )
		 RETURNING id, git_analysis_id, status, retries, max_retries, queued_at, started_at, completed_at`,
	).Scan(&j.ID, &j.GitAnalysisID, &j.Status, &j.Retries, &j.MaxRetries, &j.QueuedAt, &j.StartedAt, &j.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Load repo_url and actividad_id for worker use
	err = r.db.QueryRow(ctx,
		`SELECT gs.repo_url, gs.actividad_id
		 FROM git_submissions gs
		 JOIN git_analyses ga ON ga.git_submission_id = gs.id
		 WHERE ga.id = $1`,
		j.GitAnalysisID,
	).Scan(&j.RepoURL, &j.ActividadID)
	return j, err
}

func (r *Repository) CompleteGitAnalysisJob(ctx context.Context, jobID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE git_analysis_jobs SET status='done', completed_at=now() WHERE id=$1`, jobID)
	return err
}

func (r *Repository) FailGitAnalysisJob(ctx context.Context, jobID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE git_analysis_jobs SET status='failed', retries=retries+1, completed_at=now() WHERE id=$1`, jobID)
	return err
}

// MarkGitAnalysisJobDead permanently kills a git analysis job that has exhausted its retries.
func (r *Repository) MarkGitAnalysisJobDead(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE git_analysis_jobs SET status='dead', dead_at=NOW(), last_error=$2 WHERE id=$1`, jobID, errMsg)
	return err
}

// ─── Defense Appointments ─────────────────────────────────────────────────────

func (r *Repository) UpsertDefenseAppointment(ctx context.Context, actividadID, studentID uuid.UUID, scheduledAt time.Time) (*DefenseAppointment, error) {
	d := &DefenseAppointment{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO defense_appointments (actividad_id, student_id, scheduled_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (actividad_id, student_id) DO UPDATE SET
		   scheduled_at = EXCLUDED.scheduled_at, updated_at = now()
		 RETURNING id, actividad_id, student_id, scheduled_at, grade, graded_by, graded_at, created_at, updated_at`,
		actividadID, studentID, scheduledAt,
	).Scan(&d.ID, &d.ActividadID, &d.StudentID, &d.ScheduledAt,
		&d.Grade, &d.GradedBy, &d.GradedAt, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *Repository) GetDefenseAppointment(ctx context.Context, actividadID, studentID uuid.UUID) (*DefenseAppointment, error) {
	d := &DefenseAppointment{}
	err := r.db.QueryRow(ctx,
		`SELECT id, actividad_id, student_id, scheduled_at, grade, graded_by, graded_at, created_at, updated_at
		 FROM defense_appointments WHERE actividad_id=$1 AND student_id=$2`,
		actividadID, studentID,
	).Scan(&d.ID, &d.ActividadID, &d.StudentID, &d.ScheduledAt,
		&d.Grade, &d.GradedBy, &d.GradedAt, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return d, err
}

func (r *Repository) GradeDefenseAppointment(ctx context.Context, actividadID, studentID, gradedBy uuid.UUID, grade float64) (*DefenseAppointment, error) {
	d := &DefenseAppointment{}
	err := r.db.QueryRow(ctx,
		`UPDATE defense_appointments SET grade=$3, graded_by=$4, graded_at=now(), updated_at=now()
		 WHERE actividad_id=$1 AND student_id=$2
		 RETURNING id, actividad_id, student_id, scheduled_at, grade, graded_by, graded_at, created_at, updated_at`,
		actividadID, studentID, grade, gradedBy,
	).Scan(&d.ID, &d.ActividadID, &d.StudentID, &d.ScheduledAt,
		&d.Grade, &d.GradedBy, &d.GradedAt, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

// ─── jsonArray scan helper ────────────────────────────────────────────────────

// jsonArray is a scan target that reads a JSONB column into []string.
type jsonArray []string

func (j *jsonArray) Scan(src any) error {
	if src == nil {
		*j = nil
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return nil
	}
	return json.Unmarshal(b, j)
}
